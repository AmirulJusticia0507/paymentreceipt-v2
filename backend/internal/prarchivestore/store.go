// Package prarchivestore mengadaptasi query sqlc ke interface Store milik
// package prarchive.
package prarchivestore

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/prarchive"
)

type Store struct{ q *dbgen.Queries }

func New(q *dbgen.Queries) *Store { return &Store{q: q} }

func textPtr(v *string) pgtype.Text {
	if v == nil || *v == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *v, Valid: true}
}

func int64Ptr(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *v, Valid: true}
}

func datePtr(v *time.Time) pgtype.Timestamptz {
	if v == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *v, Valid: true}
}

func intPtr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

func numericPtr(v *string) pgtype.Numeric {
	if v == nil || *v == "" {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	if err := n.Scan(*v); err != nil {
		return pgtype.Numeric{}
	}
	return n
}

func fromList(r dbgen.ListPrArchivesRow) prarchive.PrArchive {
	approvedByNama := ""
	if s, ok := r.ApprovedByUsername.(string); ok {
		approvedByNama = s
	}
	return prarchive.PrArchive{
		ID:               r.ID,
		Kode:             r.Kode,
		PaymentReceiptID: r.PaymentReceiptID,
		ReceiptKode:      r.ReceiptKode,
		TanggalSelesai:   r.TanggalSelesai,
		NamaRequest:      r.NamaRequest,
		ManagerApproved:  r.ManagerApproved,
		ApprovedBy:       intPtr(r.ApprovedBy),
		ApprovedByNama:   approvedByNama,
		Description:      r.Description,
		Quantity:         r.Quantity,
		HargaRencana:     r.HargaRencana,
		HargaAktual:      r.HargaAktual,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func fromGet(r dbgen.GetPrArchiveRow) prarchive.PrArchive {
	approvedByNama := ""
	if s, ok := r.ApprovedByUsername.(string); ok {
		approvedByNama = s
	}
	return prarchive.PrArchive{
		ID:               r.ID,
		Kode:             r.Kode,
		PaymentReceiptID: r.PaymentReceiptID,
		ReceiptKode:      r.ReceiptKode,
		TanggalSelesai:   r.TanggalSelesai,
		NamaRequest:      r.NamaRequest,
		ManagerApproved:  r.ManagerApproved,
		ApprovedBy:       intPtr(r.ApprovedBy),
		ApprovedByNama:   approvedByNama,
		Description:      r.Description,
		Quantity:         r.Quantity,
		HargaRencana:     r.HargaRencana,
		HargaAktual:      r.HargaAktual,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func fromPrArchive(r dbgen.PrArchive) prarchive.PrArchive {
	return prarchive.PrArchive{
		ID:               r.ID,
		Kode:             r.Kode,
		PaymentReceiptID: r.PaymentReceiptID,
		TanggalSelesai:   r.TanggalSelesai,
		NamaRequest:      r.NamaRequest,
		ManagerApproved:  r.ManagerApproved,
		ApprovedBy:       intPtr(r.ApprovedBy),
		Description:      r.Description,
		Quantity:         r.Quantity,
		HargaRencana:     r.HargaRencana,
		HargaAktual:      r.HargaAktual,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func stringPtr(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func (s *Store) List(ctx context.Context, f prarchive.Filters) ([]prarchive.PrArchive, int64, error) {
	params := dbgen.ListPrArchivesParams{
		DateFrom: datePtr(f.DateFrom),
		DateTo:   datePtr(f.DateTo),
		Q:        textPtr(&f.Query),
		Approver: int64Ptr(&f.ApproverID),
		Limit:    f.Limit,
		Offset:   f.Offset,
	}

	rows, err := s.q.ListPrArchives(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.q.CountPrArchives(ctx, dbgen.CountPrArchivesParams{
		DateFrom: params.DateFrom,
		DateTo:   params.DateTo,
		Q:        params.Q,
		Approver: params.Approver,
	})
	if err != nil {
		return nil, 0, err
	}

	out := make([]prarchive.PrArchive, 0, len(rows))
	for _, r := range rows {
		out = append(out, fromList(r))
	}
	return out, total, nil
}

func (s *Store) Get(ctx context.Context, id int64) (prarchive.PrArchive, error) {
	r, err := s.q.GetPrArchive(ctx, id)
	if err != nil {
		return prarchive.PrArchive{}, err
	}
	return fromGet(r), nil
}

func (s *Store) Create(ctx context.Context, in prarchive.CreateInput) (prarchive.PrArchive, error) {
	r, err := s.q.CreatePrArchive(ctx, dbgen.CreatePrArchiveParams{
		Kode:             in.Kode,
		PaymentReceiptID: in.PaymentReceiptID,
		TanggalSelesai:   in.TanggalSelesai,
		NamaRequest:      in.NamaRequest,
		ManagerApproved:  in.ManagerApproved,
		ApprovedBy:       int64Ptr(in.ApprovedBy),
		Description:      in.Description,
		Quantity:         in.Quantity,
		HargaRencana:     in.HargaRencana,
		HargaAktual:      in.HargaAktual,
	})
	if err != nil {
		return prarchive.PrArchive{}, err
	}
	// Setelah create, ambil data lengkap dengan join.
	return s.Get(ctx, r.ID)
}

func (s *Store) Update(ctx context.Context, id int64, in prarchive.CreateInput) (prarchive.PrArchive, error) {
	r, err := s.q.UpdatePrArchive(ctx, dbgen.UpdatePrArchiveParams{
		ID:             id,
		NamaRequest:      in.NamaRequest,
		ManagerApproved:  in.ManagerApproved,
		Description:      in.Description,
		Quantity:         in.Quantity,
		HargaRencana:     in.HargaRencana,
		HargaAktual:      in.HargaAktual,
	})
	if err != nil {
		return prarchive.PrArchive{}, err
	}
	return fromPrArchive(r), nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	return s.q.DeletePrArchive(ctx, id)
}
