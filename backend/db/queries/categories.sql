-- name: ListCategories :many
SELECT id, kode, nama, keterangan, is_active, created_at, updated_at
FROM categories
WHERE is_active = TRUE
  AND (
      sqlc.narg('q')::text IS NULL
      OR kode ILIKE '%' || sqlc.narg('q')::text || '%'
      OR nama ILIKE '%' || sqlc.narg('q')::text || '%'
  )
ORDER BY nama ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountCategories :one
SELECT count(*)
FROM categories
WHERE is_active = TRUE
  AND (
      sqlc.narg('q')::text IS NULL
      OR kode ILIKE '%' || sqlc.narg('q')::text || '%'
      OR nama ILIKE '%' || sqlc.narg('q')::text || '%'
  );

-- name: GetCategoryByID :one
SELECT id, kode, nama, keterangan, is_active, created_at, updated_at
FROM categories
WHERE id = @id AND is_active = TRUE;

-- name: CreateCategory :one
INSERT INTO categories (kode, nama, keterangan)
VALUES (@kode, @nama, @keterangan)
RETURNING id, kode, nama, keterangan, is_active, created_at, updated_at;

-- name: UpdateCategory :one
UPDATE categories
SET kode = @kode, nama = @nama, keterangan = @keterangan
WHERE id = @id AND is_active = TRUE
RETURNING id, kode, nama, keterangan, is_active, created_at, updated_at;

-- name: SoftDeleteCategory :one
UPDATE categories
SET is_active = FALSE
WHERE id = @id AND is_active = TRUE
RETURNING id, kode, nama, keterangan, is_active, created_at, updated_at;

-- name: ReactivateCategory :one
UPDATE categories
SET is_active = TRUE
WHERE id = @id AND is_active = FALSE
RETURNING id, kode, nama, keterangan, is_active, created_at, updated_at;

-- name: CountCategoryUsage :one
-- @id diberi cast eksplisit karena sqlc tidak bisa menyimpulkan tipenya dari
-- konteks pemakaian di dalam subquery.
SELECT count(*) AS total FROM payment_receipt WHERE category_id = @id::bigint;