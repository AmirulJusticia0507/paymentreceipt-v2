package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/config"
)

// CookieOptions berisi konfigurasi cookie untuk token.
//
// Set cookie WAJIB HttpOnly. Tanpa itu, satu XSS sudah cukup untuk mencuri
// token lewat document.cookie - inilah alasan utama token tidak disimpan di
// localStorage (lihat ADR-0003).
type CookieOptions struct {
	Domain      string
	Secure      bool
	SameSite    http.SameSite
	AccessName  string
	RefreshName string
}

// NewCookieOptions membuat opsi cookie dari konfigurasi aplikasi.
func NewCookieOptions(cfg config.AuthConfig) CookieOptions {
	return CookieOptions{
		Domain:      cfg.CookieDomain,
		Secure:      cfg.CookieSecure,
		SameSite:    parseSameSite(cfg.CookieSameSite),
		AccessName:  cfg.CookieName,
		RefreshName: cfg.RefreshCookieName,
	}
}

func parseSameSite(s string) http.SameSite {
	switch strings.ToLower(s) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

// SetAccessCookie menulis access token ke cookie.
func (o CookieOptions) SetAccessCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     o.AccessName,
		Value:    token,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		Path:     "/",
		Domain:   o.Domain,
		Secure:   o.Secure,
		HttpOnly: true,
		SameSite: o.SameSite,
	})
}

// SetRefreshCookie menulis refresh token ke cookie.
//
// Path dibatasi ke /api/v1/auth agar cookie ini tidak ikut terkirim ke setiap
// request API dan tidak pernah terekspos ke endpoint lain.
func (o CookieOptions) SetRefreshCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     o.RefreshName,
		Value:    token,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		Path:     "/api/v1/auth",
		Domain:   o.Domain,
		Secure:   o.Secure,
		HttpOnly: true,
		SameSite: o.SameSite,
	})
}

// ClearAccessCookie menghapus access token.
func (o CookieOptions) ClearAccessCookie(w http.ResponseWriter) {
	o.clear(w, o.AccessName, "/")
}

// ClearRefreshCookie menghapus refresh token.
func (o CookieOptions) ClearRefreshCookie(w http.ResponseWriter) {
	o.clear(w, o.RefreshName, "/api/v1/auth")
}

func (o CookieOptions) clear(w http.ResponseWriter, name, path string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Path:     path,
		Domain:   o.Domain,
		Secure:   o.Secure,
		HttpOnly: true,
		SameSite: o.SameSite,
	})
}

// AccessTokenFromRequest mengambil access token dari request.
//
// Sumbernya berurutan: cookie dulu, lalu header Authorization. Cookie
// dipilih supaya frontend tidak perlu menyentuh token sama sekali - JavaScript
// tidak bisa membaca cookie HttpOnly, jadi token benar-benar tidak pernah ada
// di sisi klien.
func AccessTokenFromRequest(r *http.Request, cookieName string) string {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return c.Value
	}

	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	const prefix = "Bearer "
	if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		return auth[len(prefix):]
	}

	return ""
}

// RefreshTokenFromRequest mengambil refresh token dari cookie.
func RefreshTokenFromRequest(r *http.Request, cookieName string) string {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return c.Value
}
