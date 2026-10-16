-- name: ListDepartments :many
-- Daftar department aktif untuk halaman list.
-- Filter q mendukung pencarian sebagian pada kode dan nama.
SELECT
    id, kode, nama, keterangan, is_active, created_at, updated_at
FROM department
WHERE is_active = TRUE
  AND (
      sqlc.narg('q')::text IS NULL
      OR kode ILIKE '%' || sqlc.narg('q')::text || '%'
      OR nama ILIKE '%' || sqlc.narg('q')::text || '%'
  )
ORDER BY nama ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountDepartments :one
SELECT count(*)
FROM department
WHERE is_active = TRUE
  AND (
      sqlc.narg('q')::text IS NULL
      OR kode ILIKE '%' || sqlc.narg('q')::text || '%'
      OR nama ILIKE '%' || sqlc.narg('q')::text || '%'
  );

-- name: GetDepartmentByID :one
SELECT id, kode, nama, keterangan, is_active, created_at, updated_at
FROM department
WHERE id = @id;

-- name: GetDepartmentByKode :one
SELECT id, kode, nama, keterangan, is_active, created_at, updated_at
FROM department
WHERE kode = @kode;

-- name: CreateDepartment :one
-- RETURNING dipakai supaya data yang tersimpan dikembalikan tanpa query kedua.
-- Lebih hemat satu round-trip dan menutup celah TOCTOU antara insert lalu select.
INSERT INTO department (kode, nama, keterangan)
VALUES (@kode, @nama, @keterangan)
RETURNING id, kode, nama, keterangan, is_active, created_at, updated_at;

-- name: UpdateDepartment :one
UPDATE department
SET kode = @kode,
    nama = @nama,
    keterangan = @keterangan
WHERE id = @id AND is_active = TRUE
RETURNING id, kode, nama, keterangan, is_active, created_at, updated_at;

-- name: SoftDeleteDepartment :one
-- Soft delete, bukan DELETE. Department yang sudah dipakai oleh request atau
-- payment receipt tidak boleh hilang - nilai historisnya masih dibutuhkan
-- untuk laporan dan PDF.
UPDATE department
SET is_active = FALSE
WHERE id = @id AND is_active = TRUE
RETURNING id, kode, nama, keterangan, is_active, created_at, updated_at;

-- name: ReactivateDepartment :one
-- Mengembalikan department non-aktif ke daftar aktif. Berguna ketika user salah
-- menonaktifkan lalu menyesal.
UPDATE department
SET is_active = TRUE
WHERE id = @id AND is_active = FALSE
RETURNING id, kode, nama, keterangan, is_active, created_at, updated_at;

-- name: CountDepartmentUsage :one
-- Berapa kali department dipakai. Dipakai untuk memberi pesan yang jelas saat
-- delete ditolak, alih-alih "duplicate key value violates foreign key".
-- Qualifier dipakai di kedua subquery karena @id yang sama dipakai dua kali
-- dan tanpa qualify kolomnya jadi ambigu bagi parser.
SELECT
    (SELECT count(*) FROM requests         r WHERE r.department_id = @id::bigint) +
    (SELECT count(*) FROM payment_receipt p WHERE p.department_id = @id::bigint) AS total;