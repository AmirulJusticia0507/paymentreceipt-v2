// Package basestore berisi adapter yang menghubungkan query sqlc ke
// interface generic milik package masterdata untuk tabel base.
package basestore

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/masterdata"
)

// BaseStore mengimplementasikan masterdata.Store untuk tabel base.
type BaseStore struct {
	q *dbgen.Queries
}

// NewBaseStore membuat adapter base.
func NewBaseStore(q *dbgen.Queries) *BaseStore {
	return &BaseStore{q: q}
}

// List mengembalikan halaman base beserta totalnya.
func (s *BaseStore) List(ctx context.Context, in masterdata.ListInput) ([]dbgen.Base, int64, error) {
	params := dbgen.ListBasesParams{
		Q:      nullableText(in.Query),
		Limit:  in.Limit,
		Offset: in.Offset,
	}

	items, err := s.q.ListBases(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.q.CountBases(ctx, params.Q)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// GetByID mengambil satu base.
func (s *BaseStore) GetByID(ctx context.Context, id int64) (dbgen.Base, error) {
	return s.q.GetBaseByID(ctx, id)
}

// Create membuat base baru.
func (s *BaseStore) Create(ctx context.Context, in masterdata.WriteInput) (dbgen.Base, error) {
	return s.q.CreateBase(ctx, dbgen.CreateBaseParams{
		Kode:       in.Kode,
		Nama:       in.Nama,
		Keterangan: in.Keterangan,
	})
}

// Update mengubah base.
func (s *BaseStore) Update(ctx context.Context, id int64, in masterdata.WriteInput) (dbgen.Base, error) {
	return s.q.UpdateBase(ctx, dbgen.UpdateBaseParams{
		Kode:       in.Kode,
		Nama:       in.Nama,
		Keterangan: in.Keterangan,
		ID:         id,
	})
}

// SoftDelete menonaktifkan base.
func (s *BaseStore) SoftDelete(ctx context.Context, id int64) (dbgen.Base, error) {
	return s.q.SoftDeleteBase(ctx, id)
}

// CountUsage menghitung pemakaian base di request base.
func (s *BaseStore) CountUsage(ctx context.Context, id int64) (int64, error) {
	total, err := s.q.CountBaseUsage(ctx, id)
	return int64(total), err
}

// nullableText mengubah string kosong menjadi NULL (lihat masterdatastore).
func nullableText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}
