// Package requestbasestore berisi adapter yang menghubungkan query sqlc ke
// interface generic milik package masterdata untuk tabel request_base.
//
// Request base berbeda dari department/satuan/categories/base pada satu hal:
// kolom jabatannya. Jabatan adalah penanggung jawab atas pengajuan, bukan field
// wajib pada tabel master lain, jadi ia ikut di WriteInput.
package requestbasestore

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/masterdata"
)

// RequestBaseStore mengimplementasikan masterdata.Store untuk tabel request_base.
type RequestBaseStore struct {
	q *dbgen.Queries
}

// NewRequestBaseStore membuat adapter.
func NewRequestBaseStore(q *dbgen.Queries) *RequestBaseStore {
	return &RequestBaseStore{q: q}
}

// List mengembalikan halaman request base beserta totalnya.
func (s *RequestBaseStore) List(ctx context.Context, in masterdata.ListInput) ([]dbgen.RequestBase, int64, error) {
	params := dbgen.ListRequestBasesParams{
		BaseID: filterBaseID(in.BaseID),
		Q:      nullableText(in.Query),
		Limit:  in.Limit,
		Offset: in.Offset,
	}

	rows, err := s.q.ListRequestBases(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.q.CountRequestBases(ctx, dbgen.CountRequestBasesParams{
		BaseID: params.BaseID,
		Q:      params.Q,
	})
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

// GetByID mengambil satu request base.
func (s *RequestBaseStore) GetByID(ctx context.Context, id int64) (dbgen.RequestBase, error) {
	return s.q.GetRequestBaseByID(ctx, id)
}

// Create membuat request base baru.
func (s *RequestBaseStore) Create(ctx context.Context, in masterdata.WriteInput) (dbgen.RequestBase, error) {
	return s.q.CreateRequestBase(ctx, dbgen.CreateRequestBaseParams{
		Kode:       in.Kode,
		Nama:       in.Nama,
		Jabatan:    in.Jabatan,
		BaseID:     in.BaseID,
		Keterangan: in.Keterangan,
	})
}

// Update mengubah request base.
func (s *RequestBaseStore) Update(ctx context.Context, id int64, in masterdata.WriteInput) (dbgen.RequestBase, error) {
	return s.q.UpdateRequestBase(ctx, dbgen.UpdateRequestBaseParams{
		Kode:       in.Kode,
		Nama:       in.Nama,
		Jabatan:    in.Jabatan,
		BaseID:     in.BaseID,
		Keterangan: in.Keterangan,
		ID:         id,
	})
}

// SoftDelete menonaktifkan request base.
func (s *RequestBaseStore) SoftDelete(ctx context.Context, id int64) (dbgen.RequestBase, error) {
	return s.q.SoftDeleteRequestBase(ctx, id)
}

// CountUsage selalu mengembalikan 0.
//
// request_base belum direferensikan tabel lain - payment_receipt menunjuk ke
// `requests`, bukan ke `request_base`. Jadi belum ada yang bisa menghalangi
// penghapusan, dan melaporkan angka yang tidak berarti akan menyesatkan.
//
// Nilai ini akan jadi tidak nol di Phase 6 kalau payment_receipt diberi FK ke
// request_base. Saat itu fungsi ini harus diganti query sungguhan, dan testnya
// perlu ditulis bersamaan - bukan diam-diam dibiarkan di sini.
func (s *RequestBaseStore) CountUsage(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

// filterBaseID mengubah 0 menjadi NULL.
//
// Query memakai "sqlc.narg('base_id')::bigint IS NULL OR base_id = ...". Nol di
// sini berarti "tanpa filter", jadi harus jadi NULL, bukan angka 0. Kalau 0
// diteruskan apa adanya, SQL-nya jadi "0 IS NULL OR base_id = 0" - cabang
// pertama selalu salah dan cabang kedua tidak akan cocok dengan base mana pun.
// Hasilnya: daftar tanpa filter selalu kosong, dan gejalanya menyuruh kita
// menduga filter base_id-nya yang rusak.
func filterBaseID(id int64) pgtype.Int8 {
	if id <= 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: id, Valid: true}
}

// nullableText mengubah string kosong menjadi NULL.
//
// NULL dan string kosong berbeda maknanya di SQL: query list memakai
// "sqlc.narg('q')::text IS NULL OR ... ILIKE", jadi string kosong akan ikut
// dicocokkan dan user yang mengetik spasi melihat hasil tak terduga.
func nullableText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}
