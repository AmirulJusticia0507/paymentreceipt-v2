//go:build integration

package integration_test

// harness_test menyediakan server lengkap (router + middleware + auth) untuk
// integration test: skema test yang termigrasi, pool database, config valid,
// dan token admin. Baru dibangun per test karena applyMigrations melakukan
// DROP SCHEMA di awal setiap pemanggilan.

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/config"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/database"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/router"
)

type apiTestEnv struct {
	handler    http.Handler
	db         *database.DB
	conn       *pgx.Conn
	cfg        *config.Config
	adminID    int64
	adminToken string
}

func newAPITestEnv(t *testing.T) *apiTestEnv {
	t.Helper()

	conn := openTestDatabase(t)
	applyMigrations(t, conn)

	ctx := context.Background()
	db, err := database.New(ctx, config.DBConfig{
		URL:                os.Getenv("TEST_DATABASE_URL"),
		MaxConns:           4,
		ConnectTimeout:     10 * time.Second,
		StatementCacheMode: "describe",
	})
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	t.Cleanup(db.Close)

	cfg := &config.Config{
		App:    config.AppConfig{Env: "test", Name: "paymentreceipt-test", BaseURL: "http://localhost"},
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, MaxRequestBody: 2 << 20},
		Auth: config.AuthConfig{
			JWTSecret:         []byte("kunci-test-integration-yang-cukup-panjang-0123456789"),
			Issuer:            "paymentreceipt-test",
			AccessTTL:         15 * time.Minute,
			RefreshTTL:        168 * time.Hour,
			CookieName:        "pr_access_token",
			RefreshCookieName: "pr_refresh_token",
			Argon2Memory:      19456,
			Argon2Time:        2,
			Argon2Threads:     1,
			Argon2KeyLen:      32,
			Argon2SaltLen:     16,
		},
		AI: config.AIConfig{MaxRequestBody: 10 << 20, Timeout: time.Second},
	}

	handler := router.New(router.Deps{
		Config:  cfg,
		DB:      db,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Queries: dbgen.New(db.Pool()),
	})

	// User admin: pemilik token, sumber identity di handler, dan FK approved_by
	// untuk test arsip.
	var adminID int64
	if err := conn.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role)
		VALUES ($1, $2, $3, 'admin') RETURNING id`,
		"admin-test", "admin-test@test.example", "$argon2id$test-placeholder",
	).Scan(&adminID); err != nil {
		t.Fatalf("insert admin fixture: %v", err)
	}

	adminToken, _, err := auth.NewTokenManager(cfg.Auth).IssueAccess(adminID, "admin-test", auth.RoleAdmin)
	if err != nil {
		t.Fatalf("issue admin token: %v", err)
	}

	return &apiTestEnv{
		handler:    handler,
		db:         db,
		conn:       conn,
		cfg:        cfg,
		adminID:    adminID,
		adminToken: adminToken,
	}
}

// request memanggil endpoint lewat router penuh - middleware auth, body limit,
// request log - persis seperti klien sungguhan. Bearer kosong berarti tanpa
// token autentikasi.
func (e *apiTestEnv) request(t *testing.T, method, path, body, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

// postJSON memanggil endpoint POST dengan token admin.
func (e *apiTestEnv) postJSON(t *testing.T, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.request(t, http.MethodPost, path, body, e.adminToken)
}
