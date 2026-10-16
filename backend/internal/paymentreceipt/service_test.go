package paymentreceipt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
)

type fakeStore struct {
	created     CreateInput
	createCalls int
	getErr      error
}

func (f *fakeStore) List(context.Context, Filters) ([]Receipt, int64, error) { return nil, 0, nil }
func (f *fakeStore) Get(context.Context, int64) (Receipt, error)             { return Receipt{}, f.getErr }
func (f *fakeStore) Create(_ context.Context, in CreateInput) (Receipt, error) {
	f.created = in
	f.createCalls++
	return Receipt{ID: 1, Kode: "PR-2026-000001"}, nil
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
