-- name: ListRequests :many
-- Daftar request aktif untuk halaman list.
-- Filter q mendukung pencarian sebagian pada kode dan nama.
-- Filter department_id dipakai saat halaman request dibuka dari konteks
-- departemen tertentu.
SELECT
    id, kode, nama, department_id, keterangan, is_active, created_at, updated_at
FROM requests
WHERE is_active = TRUE
  AND (@department_id::bigint IS NULL OR department_id = @department_id::bigint)
  AND (
      sqlc.narg('q')::text IS NULL
      OR kode ILIKE '%' || sqlc.narg('q')::text || '%'
      OR nama ILIKE '%' || sqlc.narg('q')::text || '%'
  )
ORDER BY nama ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountRequests :one
SELECT count(*)
FROM requests
WHERE is_active = TRUE
  AND (@department_id::bigint IS NULL OR department_id = @department_id::bigint)
  AND (
      sqlc.narg('q')::text IS NULL
      OR kode ILIKE '%' || sqlc.narg('q')::text || '%'
      OR nama ILIKE '%' || sqlc.narg('q')::text || '%'
  );

-- name: GetRequestByID :one
SELECT id, kode, nama, department_id, keterangan, is_active, created_at, updated_at
FROM requests
WHERE id = @id AND is_active = TRUE;

-- name: GetRequestByKode :one
SELECT id, kode, nama, department_id, keterangan, is_active, created_at, updated_at
FROM requests
WHERE kode = @kode;

-- name: CreateRequest :one
INSERT INTO requests (kode, nama, department_id, keterangan)
VALUES (@kode, @nama, @department_id, @keterangan)
RETURNING id, kode, nama, department_id, keterangan, is_active, created_at, updated_at;

-- name: UpdateRequest :one
UPDATE requests
SET kode = @kode,
    nama = @nama,
    department_id = @department_id,
    keterangan = @keterangan
WHERE id = @id AND is_active = TRUE
RETURNING id, kode, nama, department_id, keterangan, is_active, created_at, updated_at;

-- name: SoftDeleteRequest :one
UPDATE requests
SET is_active = FALSE
WHERE id = @id AND is_active = TRUE
RETURNING id, kode, nama, department_id, keterangan, is_active, created_at, updated_at;

-- name: ReactivateRequest :one
UPDATE requests
SET is_active = TRUE
WHERE id = @id AND is_active = FALSE
RETURNING id, kode, nama, department_id, keterangan, is_active, created_at, updated_at;

-- name: CountRequestUsage :one
SELECT count(*) AS total FROM payment_receipt WHERE request_id = @id::bigint;
