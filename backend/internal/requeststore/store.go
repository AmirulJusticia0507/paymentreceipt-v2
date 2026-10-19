// Package requeststore berisi adapter yang menghubungkan query sqlc ke
// interface generic milik package masterdata untuk tabel requests.
//
// Request berbeda dari department/satuan/categories karena punya foreign key
// ke department dan filter list dapat dibatasi per departemen.
package requeststore

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/masterdata"
)

// RequestStore mengimplementasikan masterdata.Store untuk tabel requests.
type RequestStore struct {
	q *dbgen.Queries
}

// NewRequestStore membuat adapter requests.
func NewRequestStore(q *dbgen.Queries) *RequestStore {
	return &RequestStore{q: q}
}

// List mengembalikan halaman request beserta totalnya. Saat DepartmentID
// diisi, hanya request milik departemen tersebut yang dikembalikan.
func (s *RequestStore) List(ctx context.Context, in masterdata.ListInput) ([]dbgen.Request, int64, error) {
	params := dbgen.ListRequestsParams{
		Q:            nullableText(in.Query),
		DepartmentID: in.DepartmentID,
		Limit:        in.Limit,
		Offset:       in.Offset,
	}

	items, err := s.q.ListRequests(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	// Count menerima params sendiri, bukan memakai params List, supaya limit dan
	// offset yang tidak ada artinya untuk COUNT tidak ikut terbawa.
	total, err := s.q.CountRequests(ctx, dbgen.CountRequestsParams{
		DepartmentID: params.DepartmentID,
		Q:            params.Q,
	})
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// GetByID mengambil satu request.
func (s *RequestStore) GetByID(ctx context.Context, id int64) (dbgen.Request, error) {
	return s.q.GetRequestByID(ctx, id)
}

// Create membuat request baru.
func (s *RequestStore) Create(ctx context.Context, in masterdata.WriteInput) (dbgen.Request, error) {
	return s.q.CreateRequest(ctx, dbgen.CreateRequestParams{
		Kode:         in.Kode,
		Nama:         in.Nama,
		DepartmentID: in.DepartmentID,
		Keterangan:   in.Keterangan,
	})
}

// Update mengubah request.
func (s *RequestStore) Update(ctx context.Context, id int64, in masterdata.WriteInput) (dbgen.Request, error) {
	return s.q.UpdateRequest(ctx, dbgen.UpdateRequestParams{
		Kode:         in.Kode,
		Nama:         in.Nama,
		DepartmentID: in.DepartmentID,
		Keterangan:   in.Keterangan,
		ID:           id,
	})
}

// SoftDelete menonaktifkan request.
func (s *RequestStore) SoftDelete(ctx context.Context, id int64) (dbgen.Request, error) {
	return s.q.SoftDeleteRequest(ctx, id)
}

// CountUsage menghitung pemakaian request di payment receipt.
func (s *RequestStore) CountUsage(ctx context.Context, id int64) (int64, error) {
	total, err := s.q.CountRequestUsage(ctx, id)
	return int64(total), err
}

// nullableText mengubah string kosong menjadi NULL (lihat masterdatastore).
func nullableText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// nullableInt64 dihapus (unused) setelah refactor; pgtype.Int8{} dipakai langsung.
