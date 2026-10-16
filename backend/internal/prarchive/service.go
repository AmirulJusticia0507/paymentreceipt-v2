// Package prarchive mengelola arsip Purchase Requisition dari payment receipt
// yang sudah selesai (finished).
package prarchive

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
)

// PrArchive adalah proyeksi API-facing untuk satu arsip PR, termasuk label
// join dari payment receipt dan user approver.
type PrArchive struct {
	ID               int64     `json:"id"`
	Kode             string    `json:"kode"`
	PaymentReceiptID int64     `json:"payment_receipt_id"`
	ReceiptKode      string    `json:"receipt_kode"`
	TanggalSelesai   time.Time `json:"tanggal_selesai"`
	NamaRequest      string    `json:"nama_request"`
	ManagerApproved  string    `json:"manager_approved"`
	ApprovedBy       *int64    `json:"approved_by"`
	ApprovedByNama   string    `json:"approved_by_nama"`
	Description      string    `json:"description"`
	Quantity         int32     `json:"quantity"`
	HargaRencana     string    `json:"harga_rencana"`
	HargaAktual      string    `json:"harga_aktual"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Filters untuk daftar arsip PR.
type Filters struct {
	DateFrom, DateTo *time.Time
	Query            string
	ApproverID       int64
	Limit, Offset    int32
}

// CreateInput untuk membuat arsip PR baru (biasanya lewat bulk archive).
type CreateInput struct {
	Kode             string    `json:"kode"`
	PaymentReceiptID int64     `json:"payment_receipt_id"`
	TanggalSelesai   time.Time `json:"tanggal_selesai"`
	NamaRequest      string    `json:"nama_request"`
	ManagerApproved  string    `json:"manager_approved"`
	ApprovedBy       *int64    `json:"approved_by"`
	Description      string    `json:"description"`
	Quantity         int32     `json:"quantity"`
	HargaRencana     string    `json:"harga_rencana"`
	HargaAktual      string    `json:"harga_aktual"`
}

// Store persists dan reads pr_archive.
type Store interface {
	List(ctx context.Context, f Filters) ([]PrArchive, int64, error)
	Get(ctx context.Context, id int64) (PrArchive, error)
	Create(ctx context.Context, in CreateInput) (PrArchive, error)
	Update(ctx context.Context, id int64, in CreateInput) (PrArchive, error)
	Delete(ctx context.Context, id int64) error
}

// Service mengkoordinasi use case arsip PR.
type Service struct{ store Store }

func NewService(store Store) *Service { return &Service{store: store} }

func (s *Service) List(ctx context.Context, f Filters) ([]PrArchive, int64, error) {
	return s.store.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id int64) (PrArchive, error) {
	item, err := s.store.Get(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return PrArchive{}, apperror.New(apperror.KindNotFound, "pr_archive.not_found", "PR archive tidak ditemukan")
	}
	return item, err
}

func (s *Service) Update(ctx context.Context, id int64, in CreateInput) (PrArchive, error) {
	return s.store.Update(ctx, id, in)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.store.Delete(ctx, id)
}

func (s *Service) Create(ctx context.Context, in CreateInput) (PrArchive, error) {
	in.NamaRequest = strings.TrimSpace(in.NamaRequest)
	in.Description = strings.TrimSpace(in.Description)
	in.ManagerApproved = strings.TrimSpace(in.ManagerApproved)

	validation := apperror.New(apperror.KindValidation, "pr_archive.validation", "Data PR archive belum valid")
	if in.PaymentReceiptID <= 0 {
		validation.WithField("payment_receipt_id", "Payment receipt wajib dipilih", "required")
	}
	if in.NamaRequest == "" {
		validation.WithField("nama_request", "Nama request wajib diisi", "required")
	}
	if in.HargaAktual == "" {
		validation.WithField("harga_aktual", "Harga aktual wajib diisi", "required")
	}
	if len(validation.Fields) > 0 {
		return PrArchive{}, validation
	}
	return s.store.Create(ctx, in)
}

// ErrNotFound menunjukkan arsip PR tidak ada.
var ErrNotFound = errors.New("pr archive not found")
