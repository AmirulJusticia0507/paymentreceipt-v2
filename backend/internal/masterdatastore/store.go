// Package masterdatastore berisi adapter yang menghubungkan query sqlc ke
// interface generic milik package masterdata.
//
// Satu file per tabel karena nama fungsi sqlc berbeda-beda
// (CreateDepartment vs CreateSatuan). Yang-genrik ada di lapisan atas; di sini
// hanya menerjemahkan.
package masterdatastore

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/masterdata"
)

// --- Department ----------------------------------------------------------

// DepartmentStore mengimplementasikan masterdata.Store untuk tabel department.
type DepartmentStore struct {
	q *dbgen.Queries
}

// NewDepartmentStore membuat adapter department.
func NewDepartmentStore(q *dbgen.Queries) *DepartmentStore {
	return &DepartmentStore{q: q}
}

// List mengembalikan halaman department beserta totalnya.
func (s *DepartmentStore) List(ctx context.Context, in masterdata.ListInput) ([]dbgen.Department, int64, error) {
	params := dbgen.ListDepartmentsParams{
		Q:      nullableText(in.Query),
		Limit:  in.Limit,
		Offset: in.Offset,
	}

	items, err := s.q.ListDepartments(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.q.CountDepartments(ctx, params.Q)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// GetByID mengambil satu department.
func (s *DepartmentStore) GetByID(ctx context.Context, id int64) (dbgen.Department, error) {
	return s.q.GetDepartmentByID(ctx, id)
}

// Create membuat department baru.
func (s *DepartmentStore) Create(ctx context.Context, in masterdata.WriteInput) (dbgen.Department, error) {
	return s.q.CreateDepartment(ctx, dbgen.CreateDepartmentParams{
		Kode:       in.Kode,
		Nama:       in.Nama,
		Keterangan: in.Keterangan,
	})
}

// Update mengubah department.
func (s *DepartmentStore) Update(ctx context.Context, id int64, in masterdata.WriteInput) (dbgen.Department, error) {
	return s.q.UpdateDepartment(ctx, dbgen.UpdateDepartmentParams{
		Kode:       in.Kode,
		Nama:       in.Nama,
		Keterangan: in.Keterangan,
		ID:         id,
	})
}

// SoftDelete menonaktifkan department.
func (s *DepartmentStore) SoftDelete(ctx context.Context, id int64) (dbgen.Department, error) {
	return s.q.SoftDeleteDepartment(ctx, id)
}

// CountUsage menghitung pemakaian department di request dan payment receipt.
func (s *DepartmentStore) CountUsage(ctx context.Context, id int64) (int64, error) {
	total, err := s.q.CountDepartmentUsage(ctx, id)
	return int64(total), err
}

// --- Satuan --------------------------------------------------------------

// SatuanStore mengimplementasikan masterdata.Store untuk tabel satuan.
type SatuanStore struct {
	q *dbgen.Queries
}

// NewSatuanStore membuat adapter satuan.
func NewSatuanStore(q *dbgen.Queries) *SatuanStore {
	return &SatuanStore{q: q}
}

// List mengembalikan halaman satuan beserta totalnya.
func (s *SatuanStore) List(ctx context.Context, in masterdata.ListInput) ([]dbgen.Satuan, int64, error) {
	params := dbgen.ListSatuanParams{
		Q:      nullableText(in.Query),
		Limit:  in.Limit,
		Offset: in.Offset,
	}

	items, err := s.q.ListSatuan(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.q.CountSatuan(ctx, params.Q)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// GetByID mengambil satu satuan.
func (s *SatuanStore) GetByID(ctx context.Context, id int64) (dbgen.Satuan, error) {
	return s.q.GetSatuanByID(ctx, id)
}

// Create membuat satuan baru.
func (s *SatuanStore) Create(ctx context.Context, in masterdata.WriteInput) (dbgen.Satuan, error) {
	return s.q.CreateSatuan(ctx, dbgen.CreateSatuanParams{
		Kode: in.Kode,
		Nama: in.Nama,
	})
}

// Update mengubah satuan.
//
// Keterangan sengaja DIABAIKAN: tabel satuan tidak punya kolom itu. Memabdikan
// field yang tidak ada akan membuat user percaya datanya tersimpan padahal
// tidak.
func (s *SatuanStore) Update(ctx context.Context, id int64, in masterdata.WriteInput) (dbgen.Satuan, error) {
	return s.q.UpdateSatuan(ctx, dbgen.UpdateSatuanParams{
		Kode: in.Kode,
		Nama: in.Nama,
		ID:   id,
	})
}

// SoftDelete menonaktifkan satuan.
func (s *SatuanStore) SoftDelete(ctx context.Context, id int64) (dbgen.Satuan, error) {
	return s.q.SoftDeleteSatuan(ctx, id)
}

// CountUsage menghitung pemakaian satuan di payment receipt.
func (s *SatuanStore) CountUsage(ctx context.Context, id int64) (int64, error) {
	total, err := s.q.CountSatuanUsage(ctx, id)
	return int64(total), err
}

// --- Categories ----------------------------------------------------------

// CategoryStore mengimplementasikan masterdata.Store untuk tabel categories.
type CategoryStore struct {
	q *dbgen.Queries
}

// NewCategoryStore membuat adapter kategori.
func NewCategoryStore(q *dbgen.Queries) *CategoryStore {
	return &CategoryStore{q: q}
}

// List mengembalikan halaman kategori beserta totalnya.
func (s *CategoryStore) List(ctx context.Context, in masterdata.ListInput) ([]dbgen.Category, int64, error) {
	params := dbgen.ListCategoriesParams{
		Q:      nullableText(in.Query),
		Limit:  in.Limit,
		Offset: in.Offset,
	}

	items, err := s.q.ListCategories(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.q.CountCategories(ctx, params.Q)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// GetByID mengambil satu kategori.
func (s *CategoryStore) GetByID(ctx context.Context, id int64) (dbgen.Category, error) {
	return s.q.GetCategoryByID(ctx, id)
}

// Create membuat kategori baru.
func (s *CategoryStore) Create(ctx context.Context, in masterdata.WriteInput) (dbgen.Category, error) {
	return s.q.CreateCategory(ctx, dbgen.CreateCategoryParams{
		Kode:       in.Kode,
		Nama:       in.Nama,
		Keterangan: in.Keterangan,
	})
}

// Update mengubah kategori.
func (s *CategoryStore) Update(ctx context.Context, id int64, in masterdata.WriteInput) (dbgen.Category, error) {
	return s.q.UpdateCategory(ctx, dbgen.UpdateCategoryParams{
		Kode:       in.Kode,
		Nama:       in.Nama,
		Keterangan: in.Keterangan,
		ID:         id,
	})
}

// SoftDelete menonaktifkan kategori.
func (s *CategoryStore) SoftDelete(ctx context.Context, id int64) (dbgen.Category, error) {
	return s.q.SoftDeleteCategory(ctx, id)
}

// CountUsage menghitung pemakaian kategori di payment receipt.
func (s *CategoryStore) CountUsage(ctx context.Context, id int64) (int64, error) {
	total, err := s.q.CountCategoryUsage(ctx, id)
	return int64(total), err
}

// --- helper --------------------------------------------------------------

// nullableText mengubah string kosong menjadi NULL.
//
// NULL dan string kosong berbeda maknanya di SQL: query di db/queries memakai
// "sqlc.narg('q')::text IS NULL OR ... ILIKE", jadi string kosong akan ikut
// dicocokkan dan user mengetik spasi akan melihat hasil tak terduga.
func nullableText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}
