package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/config"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
)

// Identity adalah identitas terautentikasi yang melekat pada request.
type Identity struct {
	UserID   int64
	Username string
	Role     string
}

// HasRole melaporkan apakah user punya salah satu dari role yang diminta.
func (i Identity) HasRole(roles ...string) bool {
	for _, r := range roles {
		if i.Role == r {
			return true
		}
	}
	return false
}

// IsAdmin melaporkan apakah user adalah admin.
func (i Identity) IsAdmin() bool { return i.Role == RoleAdmin }

type ctxKey int

const ctxKeyIdentity ctxKey = iota

// WithIdentity menyimpan identity ke context.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKeyIdentity, id)
}

// IdentityFrom membaca identity dari context.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ctxKeyIdentity).(Identity)
	return id, ok
}

// Nama role. Dipakai sebagai konstanta supaya tidak ada typo string.
const (
	RoleAdmin    = "admin"
	RoleGAStaff  = "ga_staff"
	RoleApprover = "approver"
	RoleViewer   = "viewer"
)

// Authenticator memverifikasi JWT pada setiap request.
//
// Ini adalah inti dari perbaikan terhadap legacy: aplikasi PHP lama tidak punya
// penjaga akses sama sekali, sehingga endpoint hapus bisa dipanggil siapa saja
// yang mengetahui URL-nya. Di sini setiap route tertutup wajib melewati
// middleware ini.
type Authenticator struct {
	tokens  *TokenManager
	cookies CookieOptions
}

// NewAuthenticator membuat Authenticator.
func NewAuthenticator(tokens *TokenManager, cookies CookieOptions) *Authenticator {
	return &Authenticator{tokens: tokens, cookies: cookies}
}

// Middleware mewajibkan request punya access token yang valid.
//
// Token expired dikembalikan sebagai 401 dengan kode khusus refresh_required,
// supaya frontend tahu harus mencoba refresh dulu - dan tidak menganggapnya
// sebagai sesi yang benar-benar habis.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := AccessTokenFromRequest(r, a.cookies.AccessName)
		if tokenStr == "" {
			httputil.Problemf(w, r, apperror.Unauthorized(
				"Anda harus masuk terlebih dahulu"))
			return
		}

		claims, err := a.tokens.ParseAccess(tokenStr)
		if err != nil {
			httputil.Problemf(w, r, translateTokenError(err))
			return
		}

		userID, err := claims.UserID()
		if err != nil {
			httputil.Problemf(w, r, apperror.Unauthorized("Sesi tidak valid, silakan masuk ulang"))
			return
		}

		identity := Identity{
			UserID:   userID,
			Username: claims.Username,
			Role:     claims.Role,
		}

		next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), identity)))
	})
}

// RequireRole mewajibkan user punya salah satu role tertentu.
//
// Middleware ini dipasang SESUDAH Authenticator. Untuk route yang butuh login
// tanpa pembatasan role, cukup pakai Authenticator saja.
func (a *Authenticator) RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := IdentityFrom(r.Context())
			if !ok {
				// Middleware dipasang tanpa Authenticator di depannya - ini
				// kesalahan konfigurasi router, bukan kesalahan user.
				httputil.Problemf(w, r, apperror.Internal(
					errNoIdentity))
				return
			}

			if !allowed[identity.Role] {
				httputil.Problemf(w, r, apperror.Forbidden(
					"Peran Anda tidak memiliki izin untuk aksi ini"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

var errNoIdentity = &configError{"identity tidak ada di context"}

type configError struct{ msg string }

func (e *configError) Error() string { return e.msg }

// translateTokenError mengubah error token menjadi respons yang bisa ditindaklanjuti.
func translateTokenError(err error) error {
	switch {
	case errors.Is(err, ErrTokenExpired):
		// Kode khusus supaya frontend tahu harus refresh, bukan langsung logout.
		return apperror.New(apperror.KindUnauthorized, "token_expired",
			"Sesi Anda sudah kedaluwarsa, silakan masuk ulang")

	case errors.Is(err, ErrTokenWrongType):
		return apperror.New(apperror.KindUnauthorized, "token_wrong_type",
			"Jenis token tidak sesuai untuk akses ini")

	case errors.Is(err, ErrTokenNotYetValid):
		return apperror.New(apperror.KindUnauthorized, "token_not_yet_valid",
			"Sesi Anda belum aktif, silakan coba lagi")

	default:
		return apperror.New(apperror.KindUnauthorized, "token_invalid",
			"Sesi tidak valid, silakan masuk ulang")
	}
}

// CookieOptionsFrom membuat opsi cookie dari konfigurasi.
func CookieOptionsFrom(cfg config.AuthConfig) CookieOptions {
	return NewCookieOptions(cfg)
}
