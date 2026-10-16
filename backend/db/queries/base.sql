-- name: ListBases :many
-- Daftar base aktif untuk halaman list.
-- Filter q mendukung pencarian sebagian pada kode dan nama.
SELECT
    id, kode, nama, keterangan, is_active, created_at, updated_at
FROM base
WHERE is_active = TRUE
  AND (
      sqlc.narg('q')::text IS NULL
      OR kode ILIKE '%' || sqlc.narg('q')::text || '%'
      OR nama ILIKE '%' || sqlc.narg('q')::text || '%'
  )
ORDER BY nama ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountBases :one
SELECT count(*)
FROM base
WHERE is_active = TRUE
  AND (
      sqlc.narg('q')::text IS NULL
      OR kode ILIKE '%' || sqlc.narg('q')::text || '%'
      OR nama ILIKE '%' || sqlc.narg('q')::text || '%'
  );

-- name: GetBaseByID :one
SELECT id, kode, nama, keterangan, is_active, created_at, updated_at
FROM base
WHERE id = @id;

-- name: GetBaseByKode :one
SELECT id, kode, nama, keterangan, is_active, created_at, updated_at
FROM base
WHERE kode = @kode;

-- name: CreateBase :one
-- RETURNING dipakai supaya data yang tersimpan dikembalikan tanpa query kedua.
INSERT INTO base (kode, nama, keterangan)
VALUES (@kode, @nama, @keterangan)
RETURNING id, kode, nama, keterangan, is_active, created_at, updated_at;

-- name: UpdateBase :one
UPDATE base
SET kode = @kode,
    nama = @nama,
    keterangan = @keterangan
WHERE id = @id AND is_active = TRUE
RETURNING id, kode, nama, keterangan, is_active, created_at, updated_at;

-- name: SoftDeleteBase :one
-- Soft delete, bukan DELETE. Base yang sudah dipakai oleh request base tidak
-- boleh hilang - nilai historisnya masih dibutuhkan untuk pelacakan pengajuan.
UPDATE base
SET is_active = FALSE
WHERE id = @id AND is_active = TRUE
RETURNING id, kode, nama, keterangan, is_active, created_at, updated_at;

-- name: ReactivateBase :one
-- Mengembalikan base non-aktif ke daftar aktif.
UPDATE base
SET is_active = TRUE
WHERE id = @id AND is_active = FALSE
RETURNING id, kode, nama, keterangan, is_active, created_at, updated_at;

-- name: CountBaseUsage :one
-- Berapa banyak request base yang merujuk ke base ini. Dipakai untuk memberi
-- pesan yang jelas saat delete ditolak, alih-alih foreign key violation.
SELECT count(*) AS total FROM request_base WHERE base_id = @id::bigint;
