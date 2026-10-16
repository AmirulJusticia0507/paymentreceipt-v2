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
		DepartmentID: nullableInt64(in.DepartmentID),
		Limit:        in.Limit,
		Offset:       in.Offset,
	}

	items, err := s.q.ListRequests(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.q.CountRequests(ctx, params.Q, params.DepartmentID)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// GetRequestByID mengambil satu request.
func (s *RequestStore) GetByID(ctx context.Context, id int64) (dbgen.Request, error) {
	return s.q.GetRequestByID(ctx, id)
}

// Create membuat request baru.
func (s *RequestStore) Create(ctx context.Context, in masterdata.WriteInput) (dbgen.Request, error) {
	return s.q.CreateRequest(ctx, dbgen.CreateRequestParams{
		Kode:        in.Kode,
		Nama:        in.Nama,
		DepartmentID: in.DepartmentID, // placeholder - diganti di bawah
	})
}

// placeholder
const _ = 0
