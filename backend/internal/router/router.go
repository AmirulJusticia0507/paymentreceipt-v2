// Package router menyusun HTTP router aplikasi.
package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/config"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/database"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/health"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/middleware"
)

// New menyusun router lengkap dengan seluruh middleware.
//
// Urutan middleware disengaja:
//
//	Recoverer  -> paling luar, menangkap panic dari semua yang di bawahnya
//	RequestLog -> mencatat request,必须在 Recoverer di bawah supaya request
//	              yang panic tetap tercatat
//	RealIP     -> harus sebelum RateLimit supaya key rate limit memakai IP asli
//	CORS       -> sebelum handler, tapi setelah Recoverer
//	BodyLimit  -> sebelum handler supaya body dibatasi saat dibaca
func New(cfg *config.Config, db *database.DB, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer(logger))
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.RealIP(cfg.App.TrustedProxies))
	r.Use(middleware.CORS(cfg.App))
	r.Use(middleware.SecureHeaders(cfg.App.IsProduction()))
	r.Use(middleware.BodyLimit(cfg.Server.MaxRequestBody))
	r.Use(middleware.Timeout(30 * time.Second))

	// Endpoint operasional. Tidak perlu auth dan tidak boleh dibatasi timeout
	// yang sama dengan endpoint bisnis.
	r.Route("/healthz", func(r chi.Router) {
		r.Get("/", health.Liveness())
	})
	r.Route("/readyz", func(r chi.Router) {
		r.Get("/", health.Readiness(db))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", health.Liveness())

		// Endpoint bisnis akan dipasang di Phase 3+ (T3.x ke atas).
		//
		// Sementara ini, subrouter ini belum punya route. NotFound di bawah
		// yang akan menjawab, sehingga sementara ini semua /api/v1/* akan
		// mendapat 501 "belum diimplementasikan" alih-alih 404 yang menyesatkan.
		//
		// Catatan: jangan pakai r.Route("/*", ...) sebagai catch-all - chi
		// menolak wildcard di tengah segment ("panic: wildcard must be the
		// last value in a route"). NotFound di subrouter adalah cara yang benar.
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
