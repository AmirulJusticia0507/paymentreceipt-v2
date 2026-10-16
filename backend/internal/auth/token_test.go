package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func testSecret() []byte { return []byte("test-secret-yang-cukup-panjang-32-byte") }

func testManager(ttl time.Duration) *TokenManager {
	return &TokenManager{
		secret:     testSecret(),
		issuer:     "paymentreceipt-v2-test",
		accessTTL:  ttl,
		refreshTTL: 7 * 24 * time.Hour,
		leeway:     0,
	}
}

func TestIssueAndParseAccess(t *testing.T) {
	m := testManager(15 * time.Minute)

	token, exp, err := m.IssueAccess(42, "budi", RoleGAStaff)
	if err != nil {
		t.Fatalf("IssueAccess() error = %v", err)
	}
	if time.Until(exp) > 15*time.Minute+time.Second {
		t.Errorf("expiry = %v, want ~15m", exp)
	}

	claims, err := m.ParseAccess(token)
	if err != nil {
		t.Fatalf("ParseAccess() error = %v", err)
	}

	if claims.Username != "budi" {
		t.Errorf("Username = %q, want budi", claims.Username)
	}
	if claims.Role != RoleGAStaff {
		t.Errorf("Role = %q, want %s", claims.Role, RoleGAStaff)
	}
	if claims.TokenType != TokenTypeAccess {
		t.Errorf("TokenType = %q, want %s", claims.TokenType, TokenTypeAccess)
	}

	uid, err := claims.UserID()
	if err != nil {
		t.Fatalf("UserID() error = %v", err)
	}
	if uid != 42 {
		t.Errorf("UserID() = %d, want 42", uid)
	}
}

func TestParseRejectsWrongTokenType(t *testing.T) {
	m := testManager(15 * time.Minute)

	refresh, _, err := m.IssueRefresh(1, "budi", RoleViewer)
	if err != nil {
		t.Fatalf("IssueRefresh() error = %v", err)
	}

	// Refresh token TIDAK BOLEH dipakai sebagai access token. Kalau boleh,
	// rantai refresh bisa dilewati begitu saja.
	if _, err := m.ParseAccess(refresh); err != ErrTokenWrongType {
		t.Errorf("ParseAccess(refreshToken) error = %v, want ErrTokenWrongType", err)
	}

	// Tapi sebagai refresh token sendiri harus valid.
	if _, err := m.Parse(refresh); err != nil {
		t.Errorf("Parse(refreshToken) error = %v, want nil", err)
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	m := testManager(-1 * time.Minute) // TTL negatif = sudah kedaluwarsa

	token, _, err := m.IssueAccess(1, "budi", RoleViewer)
	if err != nil {
		t.Fatalf("IssueAccess() error = %v", err)
	}

	if _, err := m.ParseAccess(token); err != ErrTokenExpired {
		t.Errorf("ParseAccess(expired) error = %v, want ErrTokenExpired", err)
	}
}

func TestParseRejectsTamperedToken(t *testing.T) {
	m := testManager(15 * time.Minute)

	token, _, _ := m.IssueAccess(1, "budi", RoleViewer)

	// Ubah satu karakter di payload. Signature tidak akan cocok lagi.
	tampered := token[:len(token)-3] + "abc"

	if _, err := m.ParseAccess(tampered); err == nil {
		t.Error("ParseAccess(token dimodifikasi) error = nil, want error")
	}
}

func TestParseRejectsTokenFromDifferentSecret(t *testing.T) {
	issuer := testManager(15 * time.Minute)
	token, _, _ := issuer.IssueAccess(1, "budi", RoleAdmin)

	other := testManager(15 * time.Minute)
	other.secret = []byte("secret-berbeda-sama-sekali-32-byte-ok")

	if _, err := other.ParseAccess(token); err == nil {
		t.Error("ParseAccess(token dari secret lain) error = nil, want error")
	}
}

// TestParseRejectsAlgNone adalah uji keamanan penting.
//
// Kerentanan "alg: none" adalah kelas kerentanan JWT yang muncul di banyak
// library: kalau verifikasi tidak membatasi algoritme, penyerang bisa mengirim
// token dengan header {"alg":"none"} dan signature kosong, lalu server menerimanya
// sebagai token sah.
func TestParseRejectsAlgNone(t *testing.T) {
	m := testManager(15 * time.Minute)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "1",
			Issuer:    m.issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Username:  "penyerang",
		Role:      RoleAdmin,
		TokenType: TokenTypeAccess,
	}

	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("bangun token alg=none: %v", err)
	}

	if _, err := m.ParseAccess(unsigned); err == nil {
		t.Error("ParseAccess(alg=none) error = nil, want error - ini kerentanan kritis")
	}
}

func TestParseRejectsWrongIssuer(t *testing.T) {
	m := testManager(15 * time.Minute)
	token, _, _ := m.IssueAccess(1, "budi", RoleViewer)

	other := testManager(15 * time.Minute)
	other.issuer = "aplikasi-lain"

	if _, err := other.ParseAccess(token); err == nil {
		t.Error("ParseAccess(token dari issuer lain) error = nil, want error")
	}
}

func TestUserIDRejectsBadSubject(t *testing.T) {
	tests := []struct {
		name    string
		subject string
	}{
		{"kosong", ""},
		{"bukan angka", "budi"},
		{"negatif", "-5"},
		{"nol", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: tt.subject}}
			if _, err := c.UserID(); err == nil {
				t.Errorf("UserID(%q) error = nil, want error", tt.subject)
			}
		})
	}
}

// --- Cookie ---------------------------------------------------------------

func testCookies() CookieOptions {
	return CookieOptions{
		Secure:      true,
		SameSite:    http.SameSiteLaxMode,
		AccessName:  "pr_access_token",
		RefreshName: "pr_refresh_token",
	}
}

func TestSetAccessCookieHasSecureFlags(t *testing.T) {
	o := testCookies()
	w := httptest.NewRecorder()
	o.SetAccessCookie(w, "token-abc", time.Now().Add(time.Hour))

	c := w.Result().Cookies()
	if len(c) != 1 {
		t.Fatalf("len(cookies) = %d, want 1", len(c))
	}

	got := c[0]
	if !got.HttpOnly {
		t.Error("HttpOnly = false, want true - tanpa ini token bisa dicuri XSS")
	}
	if !got.Secure {
		t.Error("Secure = false, want true - tanpa ini token bocor lewat HTTP")
	}
	if got.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", got.SameSite)
	}
	if got.Value != "token-abc" {
		t.Errorf("Value = %q, want token-abc", got.Value)
	}
}

func TestRefreshCookiePathIsScoped(t *testing.T) {
	// Refresh cookie dibatasi ke /api/v1/auth supaya tidak ikut terkirim ke
	// setiap request - mengurangi permukaan serangannya.
	o := testCookies()
	w := httptest.NewRecorder()
	o.SetRefreshCookie(w, "refresh-abc", time.Now().Add(24*time.Hour))

	got := w.Result().Cookies()[0]
	if got.Path != "/api/v1/auth" {
		t.Errorf("Path = %q, want /api/v1/auth", got.Path)
	}
	if !got.HttpOnly {
		t.Error("HttpOnly = false, want true")
	}
}

func TestClearCookiesExpiresThem(t *testing.T) {
	o := testCookies()
	w := httptest.NewRecorder()
	o.ClearAccessCookie(w)
	o.ClearRefreshCookie(w)

	cookies := w.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("len(cookies) = %d, want 2", len(cookies))
	}
	for _, c := range cookies {
		if c.Value != "" {
			t.Errorf("cookie %q Value = %q, want kosong", c.Name, c.Value)
		}
		if c.MaxAge >= 0 {
			t.Errorf("cookie %q MaxAge = %d, want negatif (hapus)", c.Name, c.MaxAge)
		}
	}
}

func TestAccessTokenFromRequestPrefersCookie(t *testing.T) {
	o := testCookies()

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: o.AccessName, Value: "dari-cookie"})
	r.Header.Set("Authorization", "Bearer dari-header")

	got := AccessTokenFromRequest(r, o.AccessName)
	if got != "dari-cookie" {
		t.Errorf("AccessTokenFromRequest() = %q, want dari-cookie (cookie diprioritaskan)", got)
	}
}

func TestAccessTokenFromRequestFallsBackToHeader(t *testing.T) {
	o := testCookies()

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer dari-header")

	if got := AccessTokenFromRequest(r, o.AccessName); got != "dari-header" {
		t.Errorf("AccessTokenFromRequest() = %q, want dari-header", got)
	}
}

func TestAccessTokenFromRequestEmpty(t *testing.T) {
	o := testCookies()
	r := httptest.NewRequest("GET", "/", nil)

	if got := AccessTokenFromRequest(r, o.AccessName); got != "" {
		t.Errorf("AccessTokenFromRequest() = %q, want kosong", got)
	}
}

func TestParseSameSite(t *testing.T) {
	tests := []struct {
		in   string
		want http.SameSite
	}{
		{"lax", http.SameSiteLaxMode},
		{"Strict", http.SameSiteStrictMode},
		{"none", http.SameSiteNoneMode},
		{"tidak-dikenal", http.SameSiteLaxMode}, // default aman
	}

	for _, tt := range tests {
		if got := parseSameSite(tt.in); got != tt.want {
			t.Errorf("parseSameSite(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestIdentityRoles(t *testing.T) {
	i := Identity{Role: RoleAdmin}
	if !i.IsAdmin() {
		t.Error("IsAdmin() = false, want true")
	}
	if !i.HasRole(RoleAdmin, RoleGAStaff) {
		t.Error("HasRole(admin) = false, want true")
	}
	if i.HasRole(RoleViewer) {
		t.Error("HasRole(viewer) = true, want false")
	}
}
