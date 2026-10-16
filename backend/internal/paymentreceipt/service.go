// Package paymentreceipt implements payment receipt use cases.
package paymentreceipt

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
)

// Receipt is the API-facing payment receipt projection, including joined labels.
type Receipt struct {
	ID               int64     `json:"id"`
	Kode             string    `json:"kode"`
	RequestID        int64     `json:"request_id"`
	RequestKode      string    `json:"request_kode"`
	RequestNama      string    `json:"request_nama"`
	DepartmentID     int64     `json:"department_id"`
	DepartmentKode   string    `json:"department_kode"`
	DepartmentNama   string    `json:"department_nama"`
	SourceID         *int64    `json:"source_id"`
	SourceMode       string    `json:"source_mode"`
	SourceNamaBarang string    `json:"source_nama_barang"`
	CategoryID       *int64    `json:"category_id"`
	CategoryNama     string    `json:"category_nama"`
	UnitID           *int64    `json:"unit_id"`
	UnitNama         string    `json:"unit_nama"`
	TanggalPembelian time.Time `json:"tanggal_pembelian"`
	Usage            string    `json:"usage"`
	DeskripsiItems   string    `json:"deskripsi_items"`
	JumlahItems      int32     `json:"jumlah_items"`
	HargaItems       string    `json:"harga_items"`
	TotalHarga       string    `json:"total_harga"`
	PriceEstimate    *string   `json:"price_estimate"`
	Keterangan       string    `json:"keterangan"`
	Status           string    `json:"status"`
	InputDate        time.Time `json:"input_date"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Filters controls list filtering and pagination.
type Filters struct {
	Status                  string
	DepartmentID, RequestID int64
	DateFrom, DateTo        *time.Time
	Query, Sort             string
	Limit, Offset           int32
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

// Store persists and reads receipts.
type Store interface {
	List(context.Context, Filters) ([]Receipt, int64, error)
	Get(context.Context, int64) (Receipt, error)
	Create(context.Context, CreateInput) (Receipt, error)
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
	if in.DeskripsiItems == "" {
		validation.WithField("deskripsi_items", "Deskripsi item wajib diisi", "required")
	}
	if len(in.DeskripsiItems) > 255 {
		validation.WithField("deskripsi_items", "Deskripsi item maksimal 255 karakter", "maxlen")
	}
	if in.JumlahItems <= 0 {
		validation.WithField("jumlah_items", "Jumlah harus lebih dari 0", "positive")
	}
	if price, err := strconv.ParseFloat(in.HargaItems, 64); err != nil || price <= 0 {
		validation.WithField("harga_items", "Harga harus berupa angka lebih dari 0", "positive")
	}
	if in.PriceEstimate != nil {
		if estimate, err := strconv.ParseFloat(strings.TrimSpace(*in.PriceEstimate), 64); err != nil || estimate < 0 {
			validation.WithField("price_estimate", "Estimasi harga harus berupa angka nol atau lebih", "min")
		}
	}
	if len(validation.Fields) > 0 {
		return Receipt{}, validation
	}
	return s.store.Create(ctx, in)
}

// ErrNotFound indicates that a receipt does not exist.
var ErrNotFound = errors.New("payment receipt not found")
