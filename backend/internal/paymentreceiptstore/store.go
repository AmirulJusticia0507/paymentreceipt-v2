// Package paymentreceiptstore adapts sqlc queries to payment receipt use cases.
package paymentreceiptstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/paymentreceipt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Store struct{ q *dbgen.Queries }

func New(q *dbgen.Queries) *Store { return &Store{q: q} }
func text(v string) pgtype.Text   { return pgtype.Text{String: v, Valid: v != ""} }
func id(v int64) pgtype.Int8      { return pgtype.Int8{Int64: v, Valid: v > 0} }
func optID(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}
	return id(*v)
}

func date(v *time.Time) pgtype.Date {
	if v == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *v, Valid: true}
}

func status(v string) dbgen.NullPaymentStatus {
	valid := v == "pending" || v == "checked" || v == "processed" || v == "finished" || v == "cancelled"
	return dbgen.NullPaymentStatus{PaymentStatus: dbgen.PaymentStatus(v), Valid: valid}
}

func intPtr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

func numericString(v pgtype.Numeric) string {
	if !v.Valid {
		return ""
	}
	value, err := v.Value()
	if err != nil {
		return ""
	}
	return fmt.Sprint(value)
}

func numericPtr(v pgtype.Numeric) *string {
	value := numericString(v)
	if value == "" {
		return nil
	}
	return &value
}

func fromList(v dbgen.ListPaymentReceiptsRow) paymentreceipt.Receipt {
	return paymentreceipt.Receipt{ID: v.ID, Kode: v.Kode, RequestID: v.RequestID, RequestKode: v.RequestKode, RequestNama: v.RequestNama, DepartmentID: v.DepartmentID, DepartmentKode: v.DepartmentKode, DepartmentNama: v.DepartmentNama, SourceID: intPtr(v.SourceID), SourceMode: v.SourceMode, SourceNamaBarang: v.SourceNamaBarang.String, CategoryID: intPtr(v.CategoryID), CategoryNama: v.CategoryNama.String, UnitID: intPtr(v.UnitID), UnitNama: v.UnitNama.String, TanggalPembelian: v.TanggalPembelian.Time, Usage: v.Usage, DeskripsiItems: v.DeskripsiItems, JumlahItems: v.JumlahItems, HargaItems: v.Hargaitems, TotalHarga: v.TotalHarga, PriceEstimate: &v.PriceEstimate, Keterangan: v.Keterangan, Status: string(v.Status), InputDate: v.Inputdate, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}

func fromGet(v dbgen.GetPaymentReceiptRow) paymentreceipt.Receipt {
	return paymentreceipt.Receipt{ID: v.ID, Kode: v.Kode, RequestID: v.RequestID, RequestKode: v.RequestKode, RequestNama: v.RequestNama, DepartmentID: v.DepartmentID, DepartmentKode: v.DepartmentKode, DepartmentNama: v.DepartmentNama, SourceID: intPtr(v.SourceID), SourceMode: v.SourceMode, SourceNamaBarang: v.SourceNamaBarang.String, CategoryID: intPtr(v.CategoryID), CategoryNama: v.CategoryNama.String, UnitID: intPtr(v.UnitID), UnitNama: v.UnitNama.String, TanggalPembelian: v.TanggalPembelian.Time, Usage: v.Usage, DeskripsiItems: v.DeskripsiItems, JumlahItems: v.JumlahItems, HargaItems: v.Hargaitems, TotalHarga: v.TotalHarga, PriceEstimate: &v.PriceEstimate, Keterangan: v.Keterangan, Status: string(v.Status), InputDate: v.Inputdate, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}

func (s *Store) List(ctx context.Context, f paymentreceipt.Filters) ([]paymentreceipt.Receipt, int64, error) {
	p := dbgen.ListPaymentReceiptsParams{Status: status(f.Status), DepartmentID: id(f.DepartmentID), RequestID: id(f.RequestID), DateFrom: date(f.DateFrom), DateTo: date(f.DateTo), Q: text(f.Query), Sort: f.Sort, Limit: f.Limit, Offset: f.Offset}
	rows, err := s.q.ListPaymentReceipts(ctx, p)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountPaymentReceipts(ctx, dbgen.CountPaymentReceiptsParams{Status: p.Status, DepartmentID: p.DepartmentID, RequestID: p.RequestID, DateFrom: p.DateFrom, DateTo: p.DateTo, Q: p.Q})
	if err != nil {
		return nil, 0, err
	}
	out := make([]paymentreceipt.Receipt, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromList(row))
	}
	return out, total, nil
}

func (s *Store) Get(ctx context.Context, id int64) (paymentreceipt.Receipt, error) {
	row, err := s.q.GetPaymentReceipt(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return paymentreceipt.Receipt{}, paymentreceipt.ErrNotFound
	}
	if err != nil {
		return paymentreceipt.Receipt{}, err
	}
	return fromGet(row), nil
}

func numeric(v *string) pgtype.Numeric {
	if v == nil || *v == "" {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	if err := n.Scan(*v); err != nil {
		return pgtype.Numeric{}
	}
	return n
}

func (s *Store) Create(ctx context.Context, in paymentreceipt.CreateInput) (paymentreceipt.Receipt, error) {
	created, err := s.q.CreatePaymentReceipt(ctx, dbgen.CreatePaymentReceiptParams{RequestID: in.RequestID, DepartmentID: in.DepartmentID, SourceID: optID(in.SourceID), CategoryID: optID(in.CategoryID), UnitID: optID(in.UnitID), TanggalPembelian: pgtype.Date{Time: in.TanggalPembelian, Valid: true}, Usage: in.Usage, DeskripsiItems: in.DeskripsiItems, JumlahItems: in.JumlahItems, Hargaitems: in.HargaItems, PriceEstimate: numeric(in.PriceEstimate), Keterangan: in.Keterangan})
	if err != nil {
		return paymentreceipt.Receipt{}, err
	}
	return s.Get(ctx, created.ID)
}

func (s *Store) Update(ctx context.Context, id int64, in paymentreceipt.CreateInput) (paymentreceipt.Receipt, error) {
	created, err := s.q.UpdatePaymentReceipt(ctx, dbgen.UpdatePaymentReceiptParams{
		RequestID: in.RequestID, DepartmentID: in.DepartmentID, SourceID: optID(in.SourceID), CategoryID: optID(in.CategoryID), UnitID: optID(in.UnitID),
		TanggalPembelian: pgtype.Date{Time: in.TanggalPembelian, Valid: true}, Usage: in.Usage, DeskripsiItems: in.DeskripsiItems,
		JumlahItems: in.JumlahItems, Hargaitems: in.HargaItems, PriceEstimate: numeric(in.PriceEstimate), Keterangan: in.Keterangan, ID: id,
	})
	if err != nil {
		return paymentreceipt.Receipt{}, err
	}
	return s.Get(ctx, created.ID)
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	return s.q.DeletePaymentReceipt(ctx, id)
}

func (s *Store) UpdateStatus(ctx context.Context, id int64, status string) (paymentreceipt.Receipt, error) {
	created, err := s.q.UpdatePaymentReceiptStatus(ctx, dbgen.UpdatePaymentReceiptStatusParams{ID: id, Status: dbgen.PaymentStatus(status)})
	if err != nil {
		return paymentreceipt.Receipt{}, err
	}
	return s.Get(ctx, created.ID)
}

func (s *Store) InsertStatusHistory(ctx context.Context, receiptID int64, from, to string, changedBy int64, note string) error {
	_, err := s.q.CreateStatusHistory(ctx, dbgen.CreateStatusHistoryParams{
		PaymentReceiptID: receiptID, FromStatus: nullableStatus(from), ToStatus: dbgen.PaymentStatus(to), ChangedBy: changedBy, Note: note,
	})
	return err
}

func (s *Store) ListStatusHistory(ctx context.Context, receiptID int64) ([]paymentreceipt.StatusHistory, error) {
	rows, err := s.q.ListStatusHistoryByReceipt(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	out := make([]paymentreceipt.StatusHistory, 0, len(rows))
	for _, row := range rows {
		var from *string
		if row.FromStatus.Valid {
			s := string(row.FromStatus.PaymentStatus)
			from = &s
		}
		out = append(out, paymentreceipt.StatusHistory{
			ID: row.ID, PaymentReceiptID: row.PaymentReceiptID, FromStatus: from,
			ToStatus: string(row.ToStatus), ChangedBy: row.ChangedBy, ChangedByNama: row.ChangedByUsername,
			Note: row.Note, CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

func (s *Store) GetRequestDepartmentID(ctx context.Context, requestID int64) (int64, error) {
	r, err := s.q.GetRequestByID(ctx, requestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, paymentreceipt.ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	return r.DepartmentID, nil
}

func nullableStatus(s string) dbgen.NullPaymentStatus {
	valid := s == "pending" || s == "checked" || s == "processed" || s == "finished" || s == "cancelled"
	r := dbgen.NullPaymentStatus{Valid: valid}
	if valid {
		r.PaymentStatus = dbgen.PaymentStatus(s)
	}
	return r
}
