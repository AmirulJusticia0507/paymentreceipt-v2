//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func openTestDatabase(t *testing.T) *pgx.Conn {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL wajib diisi untuk integration test")
	}

	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	if !strings.Contains(strings.ToLower(config.Database), "test") {
		t.Fatalf("database %q ditolak: nama test database wajib mengandung 'test'", config.Database)
	}

	conn, err := pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

func applyMigrations(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration test path")
	}
	migrationDir := filepath.Join(filepath.Dir(currentFile), "..", "db", "migrations")
	files, err := filepath.Glob(filepath.Join(migrationDir, "*.up.sql"))
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("tidak ada migration di %s", migrationDir)
	}
	sort.Strings(files)

	ctx := context.Background()
	if _, err := conn.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatalf("reset test schema: %v", err)
	}
	for _, file := range files {
		sql, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration %s: %v", filepath.Base(file), err)
		}
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply migration %s: %v", filepath.Base(file), err)
		}
	}
}

func TestMigrations_PaymentReceiptConstraintsAndGeneratedTotal(t *testing.T) {
	conn := openTestDatabase(t)
	applyMigrations(t, conn)
	ctx := context.Background()

	var departmentID int64
	err := conn.QueryRow(ctx,
		"INSERT INTO department (kode, nama) VALUES ($1, $2) RETURNING id",
		"GA-TEST", "General Affairs Test",
	).Scan(&departmentID)
	if err != nil {
		t.Fatalf("insert department fixture: %v", err)
	}

	var requestID int64
	err = conn.QueryRow(ctx,
		"INSERT INTO requests (kode, nama, department_id) VALUES ($1, $2, $3) RETURNING id",
		"REQ-TEST", "Integration Request", departmentID,
	).Scan(&requestID)
	if err != nil {
		t.Fatalf("insert request fixture: %v", err)
	}

	insertReceipt := func(quantity int) (string, error) {
		var total string
		err := conn.QueryRow(ctx, `
			INSERT INTO payment_receipt (
				kode, request_id, department_id, tanggal_pembelian,
				deskripsi_items, jumlah_items, hargaitems
			) VALUES ($1, $2, $3, CURRENT_DATE, $4, $5, $6)
			RETURNING total_harga::text`,
			fmt.Sprintf("PR-TEST-%d", quantity), requestID, departmentID,
			"Kertas", quantity, "12.50",
		).Scan(&total)
		return total, err
	}

	total, err := insertReceipt(3)
	if err != nil {
		t.Fatalf("insert valid payment receipt: %v", err)
	}
	if total != "37.50" {
		t.Errorf("generated total = %q, want 37.50", total)
	}

	_, err = insertReceipt(0)
	var postgresError *pgconn.PgError
	if err == nil || !strings.Contains(err.Error(), "payment_receipt_qty_positive") {
		t.Fatalf("zero quantity error = %v, want qty check violation", err)
	}
	if !errors.As(err, &postgresError) || postgresError.Code != "23514" {
		t.Fatalf("zero quantity SQLSTATE = %v, want 23514", postgresError)
	}
}

// TestUsers_ContactConstraints memastikan akun hasil signup boleh tanpa email
// selama punya nomor HP valid, sementara email yang diisi tetap harus valid.
// Regression test untuk migrasi 0019 yang sempat menulis regex dengan
// backslash ganda sehingga email valid ditolak.
func TestUsers_ContactConstraints(t *testing.T) {
	conn := openTestDatabase(t)
	applyMigrations(t, conn)
	ctx := context.Background()

	insertUser := func(username string, email any, phone any) error {
		_, err := conn.Exec(ctx, `
			INSERT INTO users (username, email, phone, password_hash)
			VALUES ($1, $2, $3, 'test-hash')`,
			username, email, phone,
		)
		return err
	}

	if err := insertUser("signup-null-email", nil, "0812-3456-7890"); err != nil {
		t.Fatalf("email NULL + phone valid harus diterima: %v", err)
	}
	if err := insertUser("signup-valid-email", "user@example.test", nil); err != nil {
		t.Fatalf("email valid tanpa phone harus diterima: %v", err)
	}
	if err := insertUser("signup-invalid-email", "bukan-email", nil); err == nil {
		t.Fatal("email tidak valid harus ditolak")
	}
	if err := insertUser("signup-invalid-phone", nil, "abc"); err == nil {
		t.Fatal("phone tidak valid harus ditolak")
	}
}
