package paymentreceipt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
)

type fakeStore struct {
	created      CreateInput
	createCalls  int
	getErr       error
	receipt      Receipt
	historyCalls int
	historyNote  string
}

func (f *fakeStore) List(context.Context, Filters) ([]Receipt, int64, error) { return nil, 0, nil }
func (f *fakeStore) Get(context.Context, int64) (Receipt, error)             { return f.receipt, f.getErr }

func (f *fakeStore) Create(_ context.Context, in CreateInput) (Receipt, error) {
	f.created = in
	f.createCalls++
	return Receipt{ID: 1, Kode: "PR-2026-000001"}, nil
}

func (f *fakeStore) Update(_ context.Context, id int64, in CreateInput) (Receipt, error) {
	return Receipt{ID: id, Kode: "PR-2026-000001"}, nil
}
func (f *fakeStore) Delete(_ context.Context, id int64) error { return nil }
func (f *fakeStore) UpdateStatus(_ context.Context, id int64, status string) (Receipt, error) {
	return Receipt{ID: id, Kode: "PR-2026-000001", Status: status}, nil
}

func (f *fakeStore) InsertStatusHistory(_ context.Context, receiptID int64, from, to string, changedBy int64, note string) error {
	f.historyCalls++
	f.historyNote = note
	return nil
}

func (f *fakeStore) ListStatusHistory(_ context.Context, receiptID int64) ([]StatusHistory, error) {
	return nil, nil
}

func (f *fakeStore) GetRequestDepartmentID(_ context.Context, requestID int64) (int64, error) {
	return 2, nil
}

func TestServiceCreate_ValidatesRequiredAndNumericFields(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)
	_, err := service.Create(context.Background(), CreateInput{TanggalPembelian: time.Now().AddDate(0, 0, 1), JumlahItems: 0, HargaItems: "abc"})
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %v", err)
	}
	if len(appErr.Fields) < 5 {
		t.Fatalf("expected multiple field errors, got %#v", appErr.Fields)
	}
	if store.createCalls != 0 {
		t.Fatal("store must not be called for invalid input")
	}
}

func TestServiceCreate_TrimsAndCreates(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)
	date := time.Now().Add(-time.Hour)
	got, err := service.Create(context.Background(), CreateInput{RequestID: 1, DepartmentID: 2, TanggalPembelian: date, DeskripsiItems: "  Laptop  ", JumlahItems: 1, HargaItems: "1000"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 1 || store.created.DeskripsiItems != "Laptop" {
		t.Fatalf("unexpected result: %#v %#v", got, store.created)
	}
}

func TestServiceGet_MapsNotFound(t *testing.T) {
	_, err := NewService(&fakeStore{getErr: ErrNotFound}).Get(context.Background(), 99)
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceChangeStatus_CancellationRequiresReason(t *testing.T) {
	store := &fakeStore{receipt: Receipt{ID: 5, Status: "checked"}}
	_, err := NewService(store).ChangeStatus(context.Background(), 5, ChangeStatusInput{
		Status: "cancelled", Note: "  ", ChangedBy: 1,
	})
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindValidation {
		t.Fatalf("ChangeStatus() error = %v, want validation error", err)
	}
	if store.historyCalls != 0 {
		t.Fatal("status history must not be written without a rejection reason")
	}
}

func TestServiceChangeStatus_CancellationStoresReason(t *testing.T) {
	store := &fakeStore{receipt: Receipt{ID: 5, Status: "checked"}}
	_, err := NewService(store).ChangeStatus(context.Background(), 5, ChangeStatusInput{
		Status: "cancelled", Note: "Tidak sesuai anggaran", ChangedBy: 1,
	})
	if err != nil {
		t.Fatalf("ChangeStatus() error = %v", err)
	}
	if store.historyCalls != 1 {
		t.Fatalf("status history calls = %d, want 1", store.historyCalls)
	}
	if store.historyNote != "Tidak sesuai anggaran" {
		t.Fatalf("status history note = %q, want rejection reason", store.historyNote)
	}
}

func TestServiceBulkChangeStatus_CancellationRequiresReason(t *testing.T) {
	store := &fakeStore{receipt: Receipt{ID: 5, Status: "checked"}}
	_, err := NewService(store).BulkChangeStatus(context.Background(), []int64{5}, "cancelled", 1, "")
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindValidation {
		t.Fatalf("BulkChangeStatus() error = %v, want validation error", err)
	}
	if store.historyCalls != 0 {
		t.Fatal("bulk status history must not be written without a rejection reason")
	}
}
