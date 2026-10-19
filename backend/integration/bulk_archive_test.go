//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
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

// bulkArchiveTestEnv adalah server lengkap (router + middleware + auth) yang
// berjalan terhadap skema test, plus token admin untuk request terautentikasi.
type bulkArchiveTestEnv struct {
	handler http.Handler
	token   string
	conn    *pgx.Conn
}

// newBulkArchiveEnv mereset skema test, membangun router penuh, dan menerbitkan
// token admin. Dipakai per test karena applyMigrations melakukan DROP SCHEMA.
func newBulkArchiveEnv(t *testing.T) *bulkArchiveTestEnv {
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
		App: config.AppConfig{Env: "test", Name: "paymentreceipt-test", BaseURL: "http://localhost"},
		Server: config.ServerConfig{
			Host: "127.0.0.1", Port: 8080,
			MaxRequestBody: 2 << 20,
		},
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

	// User admin: sumber FK approved_by di pr_archive sekaligus pemilik token.
	var userID int64
	if err := conn.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role)
		VALUES ($1, $2, $3, 'admin') RETURNING id`,
		"bulkadmin", "bulkadmin@test.example", "$argon2id$test-placeholder",
	).Scan(&userID); err != nil {
		t.Fatalf("insert user fixture: %v", err)
	}

	token, _, err := auth.NewTokenManager(cfg.Auth).IssueAccess(userID, "bulkadmin", auth.RoleAdmin)
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}

	return &bulkArchiveTestEnv{handler: handler, token: token, conn: conn}
}

// insertReceipt membuat satu payment receipt berstatus yang diminta, lengkap
// dengan department dan request induknya.
func (e *bulkArchiveTestEnv) insertReceipt(t *testing.T, status string) int64 {
	t.Helper()
	ctx := context.Background()

	var departmentID, requestID, receiptID int64
	if err := e.conn.QueryRow(ctx,
		"INSERT INTO department (kode, nama) VALUES ($1, $2) RETURNING id",
		"GA-BULK", "General Affairs Bulk",
	).Scan(&departmentID); err != nil {
		t.Fatalf("insert department: %v", err)
	}
	if err := e.conn.QueryRow(ctx,
		"INSERT INTO requests (kode, nama, department_id) VALUES ($1, $2, $3) RETURNING id",
		"REQ-BULK", "Pengajuan Bulk", departmentID,
	).Scan(&requestID); err != nil {
		t.Fatalf("insert request: %v", err)
	}
	if err := e.conn.QueryRow(ctx, `
		INSERT INTO payment_receipt (
			kode, request_id, department_id, tanggal_pembelian,
			deskripsi_items, jumlah_items, hargaitems
		) VALUES ($1, $2, $3, CURRENT_DATE, $4, $5, $6) RETURNING id`,
		"PR-BULK-1", requestID, departmentID, "Kertas A4", 2, "55000",
	).Scan(&receiptID); err != nil {
		t.Fatalf("insert payment receipt: %v", err)
	}
	if status != "pending" {
		if _, err := e.conn.Exec(ctx,
			"UPDATE payment_receipt SET status = $1::payment_status WHERE id = $2",
			status, receiptID,
		); err != nil {
			t.Fatalf("set status receipt: %v", err)
		}
	}
	return receiptID
}

// postBulkArchive memanggil endpoint persis seperti klien sungguhan: lewat
// router penuh (middleware auth, body limit, request log) dengan Bearer token.
func (e *bulkArchiveTestEnv) postBulkArchive(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pr-archives/bulk-archive", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.token)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

// TestBulkArchive_KodeHasilGenerateDikembalikan memastikan respons 201 memuat
// kode PR hasil sekuens database ketika body tidak menyediakan kode.
//
// Regression: respons sebelumnya mengembalikan body.Kode (string kosong) -
// kode yang benar-benar tersimpan di pr_archive tidak pernah sampai ke klien.
func TestBulkArchive_KodeHasilGenerateDikembalikan(t *testing.T) {
	e := newBulkArchiveEnv(t)
	receiptID := e.insertReceipt(t, "finished")

	rec := e.postBulkArchive(t, `{
		"tanggal_selesai": "2026-10-19T00:00:00Z",
		"nama_request": "Pengajuan Bulk",
		"manager_approved": "Manager Test",
		"description": "",
		"harga_rencana": "100000",
		"payment_receipt_ids": [`+strconv.FormatInt(receiptID, 10)+`]
	}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, ingin 201 (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Kode     string `json:"kode"`
			Status   string `json:"status"`
			Received int    `json:"received"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode respons: %v (body: %s)", err, rec.Body.String())
	}

	if resp.Data.Kode == "" {
		t.Fatal("kode hasil generate tidak boleh kosong di respons")
	}
	if !strings.HasPrefix(resp.Data.Kode, "PR-") {
		t.Fatalf("kode = %q, ingin prefix PR-", resp.Data.Kode)
	}
	if resp.Data.Status != "archived" || resp.Data.Received != 1 {
		t.Fatalf("status/received = %q/%d, ingin archived/1", resp.Data.Status, resp.Data.Received)
	}

	// Respons harus sama dengan kode yang benar-benar tersimpan.
	var stored string
	if err := e.conn.QueryRow(context.Background(),
		"SELECT kode FROM pr_archive WHERE payment_receipt_id = $1", receiptID,
	).Scan(&stored); err != nil {
		t.Fatalf("baca pr_archive: %v", err)
	}
	if stored != resp.Data.Kode {
		t.Fatalf("kode respons = %q, kode di DB = %q", resp.Data.Kode, stored)
	}

	// Receipt ditandai terarsip setelah sukses.
	var archived bool
	if err := e.conn.QueryRow(context.Background(),
		"SELECT archived_at IS NOT NULL FROM payment_receipt WHERE id = $1", receiptID,
	).Scan(&archived); err != nil {
		t.Fatalf("baca archived_at: %v", err)
	}
	if !archived {
		t.Fatal("receipt harus ditandai archived_at setelah bulk-archive sukses")
	}
}

// TestBulkArchive_ReceiptBelumFinishedDitolak memastikan seluruh batch
// dibatalkan dengan 422 dan tidak ada baris pr_archive yang tersimpan ketika
// satu receipt belum berstatus finished.
func TestBulkArchive_ReceiptBelumFinishedDitolak(t *testing.T) {
	e := newBulkArchiveEnv(t)
	receiptID := e.insertReceipt(t, "pending")

	rec := e.postBulkArchive(t, `{
		"tanggal_selesai": "2026-10-19T00:00:00Z",
		"nama_request": "Pengajuan Bulk",
		"manager_approved": "Manager Test",
		"harga_rencana": "100000",
		"payment_receipt_ids": [`+strconv.FormatInt(receiptID, 10)+`]
	}`)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, ingin 422 (body: %s)", rec.Code, rec.Body.String())
	}

	var count int
	if err := e.conn.QueryRow(context.Background(),
		"SELECT count(*) FROM pr_archive",
	).Scan(&count); err != nil {
		t.Fatalf("hitung pr_archive: %v", err)
	}
	if count != 0 {
		t.Fatalf("pr_archive berisi %d baris, ingin 0 - batch gagal total", count)
	}
}
