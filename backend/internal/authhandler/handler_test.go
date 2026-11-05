package authhandler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/authservice"
)

// newTestHandler membuat Handler dengan svc yang dibungkus lewat helper.
func newTestHandler(svc *authservice.Service, cookies auth.CookieOptions) *Handler {
	return &Handler{svc: svc, cookies: cookies, refreshTTL: 7 * 24 * time.Hour}
}

// cookieOpts adalah opsi cookie minimal untuk test.
var cookieOpts = auth.CookieOptions{
	AccessName:  "pr_access_token",
	RefreshName: "pr_refresh_token",
	Secure:      false,
	SameSite:    http.SameSiteLaxMode,
}

// dummyResult adalah LoginResult minimal untuk test sukses.
var dummyResult = &authservice.LoginResult{
	AccessToken:  "at",
	RefreshToken: "rt",
	ExpiresAt:    time.Now().Add(15 * time.Minute),
	User: authservice.UserInfo{
		ID:       1,
		Username: "admin",
		Role:     "admin",
	},
}

// decodeBody mendekode body JSON ke map untuk inspeksi.
func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v\nbody=%s", err, rec.Body.String())
	}
	return body
}

// ---- Login ----

func TestLogin_MissingUsername(t *testing.T) {
	h := newTestHandler(nil, cookieOpts)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"username":"","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body=%s", rec.Code, rec.Body.String())
	}
}

func TestLogin_MissingPassword(t *testing.T) {
	h := newTestHandler(nil, cookieOpts)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"username":"admin","password":""}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body=%s", rec.Code, rec.Body.String())
	}
}

func TestLogin_MalformedJSON(t *testing.T) {
	h := newTestHandler(nil, cookieOpts)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"username":`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code < 400 {
		t.Fatalf("status = %d, want 4xx; body=%s", rec.Code, rec.Body.String())
	}
}

// ---- Refresh tanpa cookie ----

func TestRefresh_WithoutCookie_ReturnsUnauthorized(t *testing.T) {
	h := newTestHandler(nil, cookieOpts)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
}

// ---- Me tanpa identity di context ----

func TestMe_WithoutIdentityInContext_ReturnsUnauthorized(t *testing.T) {
	h := newTestHandler(nil, cookieOpts)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpdateMe_WithoutIdentityInContext_ReturnsUnauthorized(t *testing.T) {
	h := newTestHandler(nil, cookieOpts)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/me", strings.NewReader(`{"full_name":"Susi Air"}`))
	rec := httptest.NewRecorder()

	h.UpdateMe(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
}

// ---- Cookie ----

// TestCookieFlags memastikan cookie di-set dengan flag HttpOnly.
func TestCookieFlags_HttpOnly(t *testing.T) {
	h := &Handler{cookies: cookieOpts, refreshTTL: 7 * 24 * time.Hour}
	rec := httptest.NewRecorder()

	h.writeLoginResponse(rec, dummyResult, http.StatusOK)

	// Pastikan ada dua Set-Cookie header (access + refresh).
	cookies := rec.Result().Cookies()
	if len(cookies) < 2 {
		t.Fatalf("jumlah Set-Cookie = %d, want >= 2", len(cookies))
	}
	for _, c := range cookies {
		if !c.HttpOnly {
			t.Errorf("cookie %q bukan HttpOnly - rentan XSS", c.Name)
		}
	}
}

// TestCookieFlags_AccessTokenNotInBody memastikan token tidak dikirim di body.
func TestCookieFlags_AccessTokenNotInBody(t *testing.T) {
	h := &Handler{cookies: cookieOpts, refreshTTL: 7 * 24 * time.Hour}
	rec := httptest.NewRecorder()

	h.writeLoginResponse(rec, dummyResult, http.StatusOK)

	body := decodeBody(t, rec)
	if _, hasToken := body["access_token"]; hasToken {
		t.Error("body mengandung 'access_token' - token seharusnya hanya di cookie HttpOnly")
	}
	if _, hasRefresh := body["refresh_token"]; hasRefresh {
		t.Error("body mengandung 'refresh_token' - token seharusnya hanya di cookie HttpOnly")
	}
}

// ---- clientIP ----

func TestClientIP_ExtractsFromRemoteAddr(t *testing.T) {
	tests := []struct {
		remoteAddr string
		want       string
	}{
		{"192.168.1.1:12345", "192.168.1.1"},
		{"[::1]:8080", "::1"},
		{"10.0.0.1:443", "10.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.remoteAddr, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr

			got := clientIP(req)
			if got != tt.want {
				t.Errorf("clientIP(%q) = %q, want %q", tt.remoteAddr, got, tt.want)
			}
		})
	}
}
