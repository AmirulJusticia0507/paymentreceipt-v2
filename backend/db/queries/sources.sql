-- name: ListSources :many
SELECT * FROM sources
WHERE is_active = TRUE
  AND (sqlc.narg('source')::source_type IS NULL OR source = sqlc.narg('source')::source_type)
  AND (sqlc.narg('q')::text IS NULL OR nama_barang ILIKE '%' || sqlc.narg('q')::text || '%' OR nama_toko ILIKE '%' || sqlc.narg('q')::text || '%')
ORDER BY nama_barang ASC LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountSources :one
SELECT count(*) FROM sources
WHERE is_active = TRUE
  AND (sqlc.narg('source')::source_type IS NULL OR source = sqlc.narg('source')::source_type)
  AND (sqlc.narg('q')::text IS NULL OR nama_barang ILIKE '%' || sqlc.narg('q')::text || '%' OR nama_toko ILIKE '%' || sqlc.narg('q')::text || '%');

-- name: GetSourceByID :one
SELECT * FROM sources WHERE id = $1;

-- name: CreateSource :one
INSERT INTO sources (source, nama_barang, nama_toko, no_telp, no_rekening, link_pembelian, keterangan)
VALUES (@source, @nama_barang, @nama_toko, @no_telp, @no_rekening, @link_pembelian, @keterangan) RETURNING *;

-- name: UpdateSource :one
UPDATE sources SET source = @source, nama_barang = @nama_barang, nama_toko = @nama_toko, no_telp = @no_telp,
 no_rekening = @no_rekening, link_pembelian = @link_pembelian, keterangan = @keterangan
WHERE id = @id AND is_active = TRUE RETURNING *;

-- name: SoftDeleteSource :execrows
UPDATE sources SET is_active = FALSE WHERE id = $1 AND is_active = TRUE;
