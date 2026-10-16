-- name: ListRequestBases :many
-- Daftar request base aktif. base_id hanya jadi filter: halaman umum
-- menampilkannya tanpa filter, halaman yang dibuka dari sebuah base
-- memfilternya ke isi base tersebut.
--
-- base_id harus lewat sqlc.narg, bukan @base_id. Parametrnya wajib nullable.
-- Dengan @base_id, sqlc membuat field Go bertipe int64 dan meneruskan 0 saat
-- filter tidak dipakai - sehingga SQL-nya jadi "0 IS NULL OR base_id = 0".
-- 0 IS NULL bernilai false, jadi daftar tanpa filter selalu kosong. Gejalanya
-- seperti filter base_id rusak, padahal filter itu sendiri benar.
SELECT
    rb.id, rb.kode, rb.nama, rb.jabatan, rb.base_id, rb.keterangan,
    rb.is_active, rb.created_at, rb.updated_at
FROM request_base rb
WHERE rb.is_active = TRUE
  AND (
      sqlc.narg('base_id')::bigint IS NULL
      OR rb.base_id = sqlc.narg('base_id')::bigint
  )
  AND (
      sqlc.narg('q')::text IS NULL
      OR rb.kode ILIKE '%' || sqlc.narg('q')::text || '%'
      OR rb.nama ILIKE '%' || sqlc.narg('q')::text || '%'
  )
ORDER BY rb.nama ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountRequestBases :one
SELECT count(*)
FROM request_base rb
WHERE rb.is_active = TRUE
  AND (
      sqlc.narg('base_id')::bigint IS NULL
      OR rb.base_id = sqlc.narg('base_id')::bigint
  )
  AND (
      sqlc.narg('q')::text IS NULL
      OR rb.kode ILIKE '%' || sqlc.narg('q')::text || '%'
      OR rb.nama ILIKE '%' || sqlc.narg('q')::text || '%'
  );

-- name: CreateRequestBase :one
INSERT INTO request_base (kode, nama, jabatan, base_id, keterangan)
VALUES (@kode, @nama, @jabatan, @base_id, @keterangan)
RETURNING id, kode, nama, jabatan, base_id, keterangan, is_active, created_at, updated_at;

-- name: UpdateRequestBase :one
UPDATE request_base
SET kode = @kode,
    nama = @nama,
    jabatan = @jabatan,
    base_id = @base_id,
    keterangan = @keterangan
WHERE id = @id AND is_active = TRUE
RETURNING id, kode, nama, jabatan, base_id, keterangan, is_active, created_at, updated_at;

-- name: SoftDeleteRequestBase :one
UPDATE request_base
SET is_active = FALSE
WHERE id = @id AND is_active = TRUE
RETURNING id, kode, nama, jabatan, base_id, keterangan, is_active, created_at, updated_at;

-- name: ReactivateRequestBase :one
UPDATE request_base
SET is_active = TRUE
WHERE id = @id AND is_active = FALSE
RETURNING id, kode, nama, jabatan, base_id, keterangan, is_active, created_at, updated_at;

-- name: GetRequestBaseByID :one
SELECT id, kode, nama, jabatan, base_id, keterangan, is_active, created_at, updated_at
FROM request_base
WHERE id = @id AND is_active = TRUE;

