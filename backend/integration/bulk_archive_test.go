//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// insertReceipt membuat satu payment receipt berstatus yang diminta, lengkap
// dengan department dan request induknya.
func (e *apiTestEnv) insertReceipt(t *testing.T, status string) int64 {
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

// TestBulkArchive_KodeHasilGenerateDikembalikan memastikan respons 201 memuat
// kode PR hasil sekuens database ketika body tidak menyediakan kode.
//
// Regression: respons sebelumnya mengembalikan body.Kode (string kosong) -
// kode yang benar-benar tersimpan di pr_archive tidak pernah sampai ke klien.
func TestBulkArchive_KodeHasilGenerateDikembalikan(t *testing.T) {
	e := newAPITestEnv(t)
	receiptID := e.insertReceipt(t, "finished")

	rec := e.postJSON(t, "/api/v1/pr-archives/bulk-archive", `{
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
	e := newAPITestEnv(t)
	receiptID := e.insertReceipt(t, "pending")

	rec := e.postJSON(t, "/api/v1/pr-archives/bulk-archive", `{
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
