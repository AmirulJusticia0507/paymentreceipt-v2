//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
)

func TestReceiptNotifications_ReadPersistsUntilReceiptChanges(t *testing.T) {
	e := newAPITestEnv(t)
	ctx := context.Background()

	var departmentID, userID, requestID, receiptID int64
	if err := e.conn.QueryRow(ctx, `INSERT INTO department (kode, nama) VALUES ('NTF-TEST', 'Notification Test') RETURNING id`).Scan(&departmentID); err != nil {
		t.Fatal(err)
	}
	if err := e.conn.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role, department_id)
		VALUES ('requester-notification', 'requester-notification@test.example', 'test-hash', 'requester', $1)
		RETURNING id`, departmentID).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := e.conn.QueryRow(ctx, `
		INSERT INTO requests (kode, nama, department_id)
		VALUES ('REQ-NTF-TEST', 'Notification Test', $1) RETURNING id`, departmentID).Scan(&requestID); err != nil {
		t.Fatal(err)
	}
	if err := e.conn.QueryRow(ctx, `
		INSERT INTO payment_receipt (kode, request_id, department_id, tanggal_pembelian, deskripsi_items, jumlah_items, hargaitems)
		VALUES ('PR-NTF-TEST', $1, $2, CURRENT_DATE, 'Notification Test', 1, 100)
		RETURNING id`, requestID, departmentID).Scan(&receiptID); err != nil {
		t.Fatal(err)
	}
	token, _, err := auth.NewTokenManager(e.cfg.Auth).IssueAccess(userID, "requester-notification", auth.RoleRequester, departmentID)
	if err != nil {
		t.Fatal(err)
	}

	list := func() []int64 {
		t.Helper()
		rec := e.request(t, http.MethodGet, "/api/v1/notifications/receipts", "", token)
		if rec.Code != http.StatusOK {
			t.Fatalf("list notifications status = %d, body=%s", rec.Code, rec.Body.String())
		}
		var body struct {
			Data []struct {
				ID int64 `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode notifications: %v", err)
		}
		ids := make([]int64, 0, len(body.Data))
		for _, item := range body.Data {
			ids = append(ids, item.ID)
		}
		return ids
	}

	if ids := list(); len(ids) != 1 || ids[0] != receiptID {
		t.Fatalf("initial notifications = %v, want receipt %d", ids, receiptID)
	}
	read := e.request(t, http.MethodPost, fmt.Sprintf("/api/v1/notifications/receipts/%d/read", receiptID), "", token)
	if read.Code != http.StatusNoContent {
		t.Fatalf("mark read status = %d, body=%s", read.Code, read.Body.String())
	}
	if ids := list(); len(ids) != 0 {
		t.Fatalf("notifications after read = %v, want none", ids)
	}

	if _, err := e.conn.Exec(ctx,
		"UPDATE payment_receipt SET status = 'checked' WHERE id = $1", receiptID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := e.conn.Exec(ctx, `
		INSERT INTO status_history (payment_receipt_id, from_status, to_status, changed_by, note)
		VALUES ($1, 'pending', 'checked', $2, 'updated after read')`, receiptID, userID); err != nil {
		t.Fatal(err)
	}
	if ids := list(); len(ids) != 1 || ids[0] != receiptID {
		t.Fatalf("notifications after new activity = %v, want receipt %d", ids, receiptID)
	}
}
