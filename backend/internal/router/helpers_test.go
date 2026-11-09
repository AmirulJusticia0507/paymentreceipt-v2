package router

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/config"
)

func TestDepartmentScope_RequesterIsScoped(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(auth.WithIdentity(r.Context(), auth.Identity{Role: auth.RoleRequester, DepartmentID: 12}))
	id, err := departmentScope(r)
	if err != nil || id != 12 {
		t.Fatalf("departmentScope() = %d, %v; want 12, nil", id, err)
	}
}

func TestNew_OperationalReportRouteIsMountedBehindAuthentication(t *testing.T) {
	cfg := &config.Config{
		App:    config.AppConfig{Env: "test", Name: "test", BaseURL: "http://localhost"},
		Server: config.ServerConfig{Host: "localhost", Port: 8080, MaxRequestBody: 1 << 20},
		Auth: config.AuthConfig{
			JWTSecret: []byte("test-secret-long-enough-for-hmac-signing"), Issuer: "test",
			AccessTTL: time.Minute, RefreshTTL: time.Hour, CookieName: "access", RefreshCookieName: "refresh",
		},
	}
	handler := New(Deps{Config: cfg, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/operational?date_from=2026-11-01&date_to=2026-11-09", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d for mounted protected route; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestDepartmentScope_UnassignedRequesterIsRejected(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(auth.WithIdentity(r.Context(), auth.Identity{Role: auth.RoleRequester}))
	if _, err := departmentScope(r); err == nil {
		t.Fatal("departmentScope() error = nil, want unassigned requester rejection")
	}
}

func TestCheckDepartmentAccess_HidesOtherDepartment(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(auth.WithIdentity(r.Context(), auth.Identity{Role: auth.RoleRequester, DepartmentID: 12}))
	if err := checkDepartmentAccess(r, 13); err == nil {
		t.Fatal("checkDepartmentAccess() error = nil, want cross-department rejection")
	}
}
