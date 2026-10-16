-- name: ListSatuan :many
SELECT id, kode, nama, is_active, created_at, updated_at
FROM satuan
WHERE is_active = TRUE
  AND (
      sqlc.narg('q')::text IS NULL
      OR kode ILIKE '%' || sqlc.narg('q')::text || '%'
      OR nama ILIKE '%' || sqlc.narg('q')::text || '%'
  )
ORDER BY nama ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountSatuan :one
SELECT count(*)
FROM satuan
WHERE is_active = TRUE
  AND (
      sqlc.narg('q')::text IS NULL
      OR kode ILIKE '%' || sqlc.narg('q')::text || '%'
      OR nama ILIKE '%' || sqlc.narg('q')::text || '%'
  );

-- name: GetSatuanByID :one
SELECT id, kode, nama, is_active, created_at, updated_at
FROM satuan
WHERE id = @id;

-- name: CreateSatuan :one
INSERT INTO satuan (kode, nama)
VALUES (@kode, @nama)
RETURNING id, kode, nama, is_active, created_at, updated_at;

-- name: UpdateSatuan :one
UPDATE satuan
SET kode = @kode, nama = @nama
WHERE id = @id AND is_active = TRUE
RETURNING id, kode, nama, is_active, created_at, updated_at;

-- name: SoftDeleteSatuan :one
UPDATE satuan
SET is_active = FALSE
WHERE id = @id AND is_active = TRUE
RETURNING id, kode, nama, is_active, created_at, updated_at;

-- name: ReactivateSatuan :one
UPDATE satuan
SET is_active = TRUE
WHERE id = @id AND is_active = FALSE
RETURNING id, kode, nama, is_active, created_at, updated_at;

-- name: CountSatuanUsage :one
-- Cast eksplisit: tanpa ini sqlc menebak tipenya jadi pgtype.Int8.
SELECT count(*) AS total FROM payment_receipt WHERE unit_id = @id::bigint;