// Package paymentreceipt implements payment receipt use cases.
package paymentreceipt

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/domain"
)

// Receipt is the API-facing payment receipt projection, including joined labels.
type Receipt struct {
	ID                int64     `json:"id"`
	Kode              string    `json:"kode"`
	RequestID         int64     `json:"request_id"`
	RequestKode       string    `json:"request_kode"`
	RequestNama       string    `json:"request_nama"`
	DepartmentID      int64     `json:"department_id"`
	DepartmentKode    string    `json:"department_kode"`
	DepartmentNama    string    `json:"department_nama"`
	SourceID          *int64    `json:"source_id"`
	SourceMode        string    `json:"source_mode"`
	SourceNamaBarang   string    `json:"source_nama_barang"`
	CategoryID        *int64    `json:"category_id"`
	CategoryNama      string    `json:"category_nama"`
	UnitID            *int64    `json:"unit_id"`
	UnitNama          string    `json:"unit_nama"`
	TanggalPembelian  time.Time `json:"tanggal_pembelian"`
	Usage             string    `json:"usage"`
	DeskripsiItems    string    `json:"deskripsi_items"`
	JumlahItems       int32     `json:"jumlah_items"`
	HargaItems        string    `json:"harga_items"`
	TotalHarga        string    `json:"total_harga"`
	PriceEstimate     *string   `json:"price_estimate"`
	Keterangan        string    `json:"keterangan"`
	Status            string    `json:"status"`
	InputDate         time.Time `json:"input_date"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// StatusHistory merepresentasikan satu entri riwayat perubahan status.
type StatusHistory struct {
	ID              int64     `json:"id"`
	PaymentReceiptID int64    `json:"payment_receipt_id"`
	FromStatus      *string   `json:"from_status"`
	ToStatus        string    `json:"to_status"`
	ChangedBy       int64     `json:"changed_by"`
	ChangedByNama   string    `json:"changed_by_nama"`
	Note            string    `json:"note"`
	CreatedAt       time.Time `json:"created_at"`
}

// Filters controls list filtering and pagination.
type Filters struct {
	Status       string
	DepartmentID int64
	RequestID    int64
	DateFrom     *time.Time
	DateTo       *time.Time
	Query        string
	Sort         string
	Limit        int32
	Offset       int32
}

// CreateInput contains writable fields. Kode, total, and status are server-managed.
type CreateInput struct {
	RequestID        int64     `json:"request_id"`
	DepartmentID     int64     `json:"department_id"`
	SourceID         *int64    `json:"source_id"`
	CategoryID       *int64    `json:"category_id"`
	UnitID           *int64    `json:"unit_id"`
	TanggalPembelian time.Time `json:"-"`
	Usage            string    `json:"usage"`
	DeskripsiItems   string    `json:"deskripsi_items"`
	JumlahItems      int32     `json:"jumlah_items"`
	HargaItems       string    `json:"harga_items"`
	PriceEstimate    *string   `json:"price_estimate"`
	Keterangan       string    `json:"keterangan"`
}

// ChangeStatusInput is the body for status transition.
type ChangeStatusInput struct {
	Status    string `json:"status"`
	ChangedBy int64  `json:"-"`
	Note      string `json:"note"`
}

// Store persists and reads receipts and status history.
type Store interface {
	List(ctx context.Context, f Filters) ([]Receipt, int64, error)
	Get(ctx context.Context, id int64) (Receipt, error)
	Create(ctx context.Context, in CreateInput) (Receipt, error)
	Update(ctx context.Context, id int64, in CreateInput) (Receipt, error)
	Delete(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status string) (Receipt, error)
	InsertStatusHistory(ctx context.Context, receiptID int64, from, to string, changedBy int64, note string) error
	ListStatusHistory(ctx context.Context, receiptID int64) ([]StatusHistory, error)
	GetRequestDepartmentID(ctx context.Context, requestID int64) (int64, error)
}

// Service coordinates payment receipt use cases.
type Service struct{ store Store }

func NewService(store Store) *Service { return &Service{store: store} }

func (s *Service) List(ctx context.Context, f Filters) ([]Receipt, int64, error) {
	return s.store.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id int64) (Receipt, error) {
	item, err := s.store.Get(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Receipt{}, apperror.New(apperror.KindNotFound, "payment_receipt.not_found", "Payment receipt tidak ditemukan")
	}
	return item, err
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Receipt, error) {
	in.Usage, in.DeskripsiItems, in.HargaItems, in.Keterangan = strings.TrimSpace(in.Usage), strings.TrimSpace(in.DeskripsiItems), strings.TrimSpace(in.HargaItems), strings.TrimSpace(in.Keterangan)

	validation := apperror.New(apperror.KindValidation, "payment_receipt.validation", "Data payment receipt belum valid")

	if in.RequestID <= 0 {
		validation.WithField("request_id", "Request wajib dipilih", "required")
	}
	if in.DepartmentID <= 0 {
		validation.WithField("department_id", "Department wajib dipilih", "required")
	}
	if in.TanggalPembelian.IsZero() {
		validation.WithField("tanggal_pembelian", "Tanggal pembelian wajib diisi", "required")
	}
	if in.TanggalPembelian.After(time.Now()) {
		validation.WithField("tanggal_pembelian", "Tanggal pembelian tidak boleh di masa depan", "max")
	}
	if !in.TanggalPembelian.IsZero() && in.TanggalPembelian.Before(time.Now().AddDate(0, 0, -90)) {
		validation.WithField("tanggal_pembelian", "Tanggal pembelian tidak boleh lebih dari 90 hari ke belakang", "range")
	}
	if in.DeskripsiItems == "" {
		validation.WithField("deskripsi_items", "Deskripsi item wajib diisi", "required")
	}
	if len(in.DeskripsiItems) > 255 {
		validation.WithField("deskripsi_items", "Deskripsi item maksimal 255 karakter", "maxlen")
	}
	if in.JumlahItems <= 0 {
		validation.WithField("jumlah_items", "Jumlah harus lebih dari 0", "positive")
	}
	price, priceErr := strconv.ParseFloat(in.HargaItems, 64)
	if priceErr != nil || price <= 0 {
		validation.WithField("harga_items", "Harga harus berupa angka lebih dari 0", "positive")
	}
	if in.HargaItems == "" {
		validation.WithField("harga_items", "Harga wajib diisi", "required")
	}
	if in.Usage != "" && len(in.Usage) > 150 {
		validation.WithField("usage", "Usage maksimal 150 karakter", "maxlen")
	}
	if len(validation.Fields) > 0 {
		return Receipt{}, validation
	}

	// Konsistensi request harus milik department yang dipilih.
	if in.RequestID > 0 && in.DepartmentID > 0 {
		depID, err := s.store.GetRequestDepartmentID(ctx, in.RequestID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				validation.WithField("request_id", "Request tidak ditemukan", "not_found")
				return Receipt{}, validation
			}
			return Receipt{}, apperror.New(apperror.KindInternal, "payment_receipt.request_lookup_failed", "Gagal memvalidasi request")
		}
		if depID != in.DepartmentID {
			validation.WithField("request_id", "Request tidak milik department yang dipilih", "consistency")
			return Receipt{}, validation
		}
	}

	created, err := s.store.Create(ctx, in)
	if err != nil {
		return Receipt{}, err
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, in CreateInput) (Receipt, error) {
	in.Usage, in.DeskripsiItems, in.HargaItems, in.Keterangan = strings.TrimSpace(in.Usage), strings.TrimSpace(in.DeskripsiItems), strings.TrimSpace(in.HargaItems), strings.TrimSpace(in.Keterangan)

	if in.DepartmentID <= 0 {
		in.DepartmentID = 0
	}

	validation := apperror.New(apperror.KindValidation, "payment_receipt.validation", "Data payment receipt belum valid")

	if in.RequestID <= 0 {
		validation.WithField("request_id", "Request wajib dipilih", "required")
	}
	if in.DepartmentID <= 0 {
		validation.WithField("department_id", "Department wajib dipilih", "required")
	}
	if in.TanggalPembelian.IsZero() {
		validation.WithField("tanggal_pembelian", "Tanggal pembelian wajib diisi", "required")
	}
	if in.TanggalPembelian.After(time.Now()) {
		validation.WithField("tanggal_pembelian", "Tanggal pembelian tidak boleh di masa depan", "max")
	}
	if !in.TanggalPembelian.IsZero() && in.TanggalPembelian.Before(time.Now().AddDate(0, 0, -90)) {
		validation.WithField("tanggal_pembelian", "Tanggal pembelian tidak boleh lebih dari 90 hari ke belakang", "range")
	}
	if in.DeskripsiItems == "" {
		validation.WithField("deskripsi_items", "Deskripsi item wajib diisi", "required")
	}
	if len(in.DeskripsiItems) > 255 {
		validation.WithField("deskripsi_items", "Deskripsi item maksimal 255 karakter", "maxlen")
	}
	if in.JumlahItems <= 0 {
		validation.WithField("jumlah_items", "Jumlah harus lebih dari 0", "positive")
	}
	price, priceErr := strconv.ParseFloat(in.HargaItems, 64)
	if priceErr != nil || price <= 0 {
		validation.WithField("harga_items", "Harga harus berupa angka lebih dari 0", "positive")
	}
	if in.HargaItems == "" {
		validation.WithField("harga_items", "Harga wajib diisi", "required")
	}
	if in.Usage != "" && len(in.Usage) > 150 {
		validation.WithField("usage", "Usage maksimal 150 karakter", "maxlen")
	}
	if len(validation.Fields) > 0 {
		return Receipt{}, validation
	}

	if in.RequestID > 0 && in.DepartmentID > 0 {
		depID, err := s.store.GetRequestDepartmentID(ctx, in.RequestID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				validation.WithField("request_id", "Request tidak ditemukan", "not_found")
				return Receipt{}, validation
			}
			return Receipt{}, apperror.New(apperror.KindInternal, "payment_receipt.request_lookup_failed", "Gagal memvalidasi request")
		}
		if depID != in.DepartmentID {
			validation.WithField("request_id", "Request tidak milik department yang dipilih", "consistency")
			return Receipt{}, validation
		}
	}

	item, err := s.store.Update(ctx, id, in)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Receipt{}, apperror.New(apperror.KindNotFound, "payment_receipt.not_found", "Payment receipt tidak ditemukan")
		}
		return Receipt{}, err
	}
	return item, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	item, err := s.store.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "payment_receipt.not_found", "Payment receipt tidak ditemukan")
		}
		return err
	}

	if item.Status == string(domain.StatusProcessed) || item.Status == string(domain.StatusFinished) {
		return apperror.New(apperror.KindConflict, "payment_receipt.in_use", "Payment receipt tidak bisa dihapus karena sudah diproses atau selesai")
	}

	return s.store.Delete(ctx, id)
}

func (s *Service) ChangeStatus(ctx context.Context, id int64, in ChangeStatusInput) (Receipt, error) {
	if !domain.ValidStatus(in.Status) {
		return Receipt{}, apperror.New(apperror.KindValidation, "payment_receipt.invalid_status", "Status tidak valid")
	}

	item, err := s.store.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Receipt{}, apperror.New(apperror.KindNotFound, "payment_receipt.not_found", "Payment receipt tidak ditemukan")
		}
		return Receipt{}, err
	}

	if !domain.CanTransition(domain.PaymentReceiptStatus(item.Status), domain.PaymentReceiptStatus(in.Status)) {
		return Receipt{}, apperror.New(apperror.KindConflict, "payment_receipt.illegal_transition", "Transisi status tidak diizinkan")
	}

	// Insert audit trail sebelum update status.
	if err := s.store.InsertStatusHistory(ctx, id, item.Status, in.Status, in.ChangedBy, in.Note); err != nil {
		return Receipt{}, err
	}

	updated, err := s.store.UpdateStatus(ctx, id, in.Status)
	if err != nil {
		return Receipt{}, err
	}
	return updated, nil
}

func (s *Service) StatusHistory(ctx context.Context, receiptID int64) ([]StatusHistory, error) {
	item, err := s.store.Get(ctx, receiptID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, apperror.New(apperror.KindNotFound, "payment_receipt.not_found", "Payment receipt tidak ditemukan")
		}
		return nil, err
	}
	_ = item
	return s.store.ListStatusHistory(ctx, receiptID)
}

// ErrNotFound indicates that a receipt does not exist.
var ErrNotFound = errors.New("payment receipt not found")