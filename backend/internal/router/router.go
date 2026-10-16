// Package router menyusun HTTP router aplikasi.
package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/authhandler"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/authservice"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/basestore"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/config"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/database"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/health"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/masterdata"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/masterdatastore"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/middleware"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/requeststore"
)

// Deps adalah semua yang dibutuhkan router untuk menyusun handler.
type Deps struct {
	Config  *config.Config
	DB      *database.DB
	Logger  *slog.Logger
	Queries *dbgen.Queries
}

// New menyusun router lengkap dengan seluruh middleware.
//
// Urutan middleware disengaja:
//
//	Recoverer  -> paling luar, menangkap panic dari semua yang di bawahnya
//	RequestLog -> mencatat request, harus DI DALAM Recoverer supaya request
//	              yang panic tetap tercatat
//	RealIP     -> sebelum RateLimit supaya key rate limit memakai IP asli
//	CORS       -> sebelum handler
//	BodyLimit  -> sebelum handler supaya body dibatasi saat dibaca
//
// Perhatikan apa yang TIDAK ada di sini: tidak ada auth guard global. Endpoint
// publik seperti login dan healthcheck harus tetap bisa diakses tanpa token,
// dan menutup semuanya secara global hanya memindahkan masalah ke setiap route.
func New(d Deps) http.Handler {
	cfg := d.Config
	logger := d.Logger

	r := chi.NewRouter()

	r.Use(middleware.Recoverer(logger))
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.RealIP(cfg.App.TrustedProxies))
	r.Use(middleware.CORS(cfg.App))
	r.Use(middleware.SecureHeaders(cfg.App.IsProduction()))
	r.Use(middleware.BodyLimit(cfg.Server.MaxRequestBody))
	r.Use(middleware.Timeout(30 * time.Second))

	// --- Endpoint operasional ---------------------------------------------
	// Tidak perlu auth. Liveness sengaja tidak mengecek database supaya pod tidak
	// di-restart sia-sia ketika masalahnya ada di luar aplikasi.
	r.Get("/healthz", health.Liveness())
	r.Get("/readyz", health.Readiness(d.DB))

	// --- Wiring auth ------------------------------------------------------
	tokens := auth.NewTokenManager(cfg.Auth)
	cookies := auth.NewCookieOptions(cfg.Auth)
	authenticator := auth.NewAuthenticator(tokens, cookies)

	authSvc := authservice.New(
		d.Queries,
		d.Queries,
		tokens,
		cookies,
		auth.ParamsFrom(cfg.Auth),
	)
	authH := authhandler.New(authSvc, cookies, tokens.TTLRefresh())

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", health.Liveness())

		// Endpoint publik.
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authH.Login)
			r.Post("/refresh", authH.Refresh)

			// Logout butuh tahu siapa yang logout. Tanpa auth guard, endpoint ini
			// tidak bisa menemukan token mana yang harus dicabut.
			r.Group(func(r chi.Router) {
				r.Use(authenticator.Middleware)
				r.Post("/logout", authH.Logout)
				r.Get("/me", authH.Me)
			})
		})

		// Seluruh endpoint di bawah WAJIB terautentikasi. Inilah yang hilang
		// di aplikasi legacy: function_paymentreceipt.php?act=deletepayments
		// bisa dipanggil siapa saja tanpa login.
		r.Group(func(r chi.Router) {
			r.Use(authenticator.Middleware)

			// Master data (Phase 4).
			//
			// Dua dari tiga tabel ini TIDAK punya CRUD sama sekali di aplikasi
			// legacy - satuan dan categories hanya di-SELECT lewat JOIN, tanpa
			// form insert maupun handler. User tidak bisa menambah satuan atau
			// kategori baru. Sara ini menutup celah itu.
			mountMasterData(r, authenticator, masterdatastore.NewDepartmentStore(d.Queries), "/departments",
				masterdata.ResourceDescribe{Singular: "Department", Plural: "Department"},
				"request", departmentDTO)

			mountMasterData(r, authenticator, masterdatastore.NewSatuanStore(d.Queries), "/satuan",
				masterdata.ResourceDescribe{Singular: "Satuan", Plural: "Satuan"},
				"payment receipt", satuanDTO)

			mountMasterData(r, authenticator, masterdatastore.NewCategoryStore(d.Queries), "/categories",
				masterdata.ResourceDescribe{Singular: "Kategori", Plural: "Kategori"},
				"payment receipt", categoryDTO)

			mountMasterData(r, authenticator, basestore.NewBaseStore(d.Queries), "/bases",
				masterdata.ResourceDescribe{Singular: "Base", Plural: "Base"},
				"request base", baseDTO)

			// Request (pengajuan department) — filter list per departemen
			// dan department_id wajib saat create/update.
			mountRequests(r, authenticator, d.Queries, requeststore.NewRequestStore(d.Queries), "/requests")

			// Modul bisnis lain menyusul di Phase 5-7.
		})

		r.NotFound(func(w http.ResponseWriter, req *http.Request) {
			httputil.Problemf(w, req, notImplemented(req.URL.Path))
		})
	})

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		httputil.Problemf(w, req, notFound(req.URL.Path))
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		httputil.Problemf(w, req, methodNotAllowed(req.Method))
	})

	return r
}
