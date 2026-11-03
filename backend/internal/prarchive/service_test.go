package prarchive

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
)

type mockStore struct {
	items      map[int64]PrArchive
	listFunc   func(ctx context.Context, f Filters) ([]PrArchive, int64, error)
	getFunc    func(ctx context.Context, id int64) (PrArchive, error)
	createFunc func(ctx context.Context, in CreateInput) (PrArchive, error)
	updateFunc func(ctx context.Context, id int64, in CreateInput) (PrArchive, error)
	deleteFunc func(ctx context.Context, id int64) error
}

func (m *mockStore) List(ctx context.Context, f Filters) ([]PrArchive, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, f)
	}
	return nil, 0, nil
}

func (m *mockStore) Get(ctx context.Context, id int64) (PrArchive, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	item, ok := m.items[id]
	if !ok {
		return PrArchive{}, ErrNotFound
	}
	return item, nil
}

func (m *mockStore) Create(ctx context.Context, in CreateInput) (PrArchive, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, in)
	}
	res := PrArchive{
		ID:               int64(len(m.items) + 1),
		Kode:             in.Kode,
		PaymentReceiptID: in.PaymentReceiptID,
		NamaRequest:      in.NamaRequest,
		Description:      in.Description,
		HargaAktual:      in.HargaAktual,
		CreatedAt:        time.Now(),
	}
	if m.items == nil {
		m.items = make(map[int64]PrArchive)
	}
	m.items[res.ID] = res
	return res, nil
}

func (m *mockStore) Update(ctx context.Context, id int64, in CreateInput) (PrArchive, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, in)
	}
	item, ok := m.items[id]
	if !ok {
		return PrArchive{}, ErrNotFound
	}
	item.NamaRequest = in.NamaRequest
	item.Description = in.Description
	m.items[id] = item
	return item, nil
}

func (m *mockStore) Delete(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	delete(m.items, id)
	return nil
}

func TestService_Get_SuccessAndNotFound(t *testing.T) {
	store := &mockStore{
		items: map[int64]PrArchive{
			1: {ID: 1, Kode: "PR-001", NamaRequest: "ATK GA"},
		},
	}
	svc := NewService(store)

	ctx := context.Background()

	// Happy path
	item, err := svc.Get(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Kode != "PR-001" {
		t.Errorf("got kode %q, want PR-001", item.Kode)
	}

	// Not found
	_, err = svc.Get(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent ID, got nil")
	}
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindNotFound {
		t.Errorf("expected KindNotFound apperror, got: %v", err)
	}
}

func TestService_Create_Validation(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)
	ctx := context.Background()

	tests := []struct {
		name      string
		input     CreateInput
		wantField string
	}{
		{
			name: "missing payment_receipt_id",
			input: CreateInput{
				PaymentReceiptID: 0,
				NamaRequest:      "Test Request",
				HargaAktual:      "100000",
			},
			wantField: "payment_receipt_id",
		},
		{
			name: "missing nama_request",
			input: CreateInput{
				PaymentReceiptID: 10,
				NamaRequest:      "   ",
				HargaAktual:      "100000",
			},
			wantField: "nama_request",
		},
		{
			name: "missing harga_aktual",
			input: CreateInput{
				PaymentReceiptID: 10,
				NamaRequest:      "Test Request",
				HargaAktual:      "",
			},
			wantField: "harga_aktual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(ctx, tt.input)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			var appErr *apperror.Error
			if !errors.As(err, &appErr) || appErr.Kind != apperror.KindValidation {
				t.Fatalf("expected validation apperror, got %v", err)
			}
			found := false
			for _, f := range appErr.Fields {
				if f.Field == tt.wantField {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected field %q in validation error, got %+v", tt.wantField, appErr.Fields)
			}
		})
	}
}

func TestService_Create_Success(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)
	ctx := context.Background()

	in := CreateInput{
		Kode:             "PR-2026-001",
		PaymentReceiptID: 5,
		NamaRequest:      "Pengadaan ATK",
		Description:      "Kertas A4 dan Ballpoint",
		HargaAktual:      "250000",
	}

	res, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("unexpected error creating PR archive: %v", err)
	}
	if res.ID == 0 {
		t.Errorf("expected generated ID > 0, got %d", res.ID)
	}
	if res.NamaRequest != "Pengadaan ATK" {
		t.Errorf("got nama_request %q, want Pengadaan ATK", res.NamaRequest)
	}
}

func TestService_ListAndUpdateAndDelete(t *testing.T) {
	store := &mockStore{
		items: map[int64]PrArchive{
			1: {ID: 1, Kode: "PR-001", NamaRequest: "Item 1"},
		},
	}
	svc := NewService(store)
	ctx := context.Background()

	// List
	store.listFunc = func(_ context.Context, f Filters) ([]PrArchive, int64, error) {
		return []PrArchive{{ID: 1, Kode: "PR-001"}}, 1, nil
	}
	items, total, err := svc.List(ctx, Filters{})
	if err != nil {
		t.Fatalf("unexpected error listing: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Errorf("expected 1 item, got %d items (total %d)", len(items), total)
	}

	// Update
	upd, err := svc.Update(ctx, 1, CreateInput{NamaRequest: "Updated"})
	if err != nil {
		t.Fatalf("unexpected error updating: %v", err)
	}
	if upd.NamaRequest != "Updated" {
		t.Errorf("got %q, want Updated", upd.NamaRequest)
	}

	// Delete
	if err := svc.Delete(ctx, 1); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}
}
