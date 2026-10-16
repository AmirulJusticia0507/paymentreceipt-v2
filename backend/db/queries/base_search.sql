-- name: SearchBases :many
-- Endpoint autocomplete untuk memilih base saat membuat request base.
--
-- Menggantikan searchbase.php legacy, yang menerima ?term lalu mengembalikan
-- JSON. Perbedaannya di sini: batas hasil selalu 10 supaya response tidak ikut
-- membesar saat pengguna mengetik satu huruf saja.
--
-- ILIKE '%...%' tidak bisa memakai index B-tree. Index trigram GIN di tabel base
-- (dibuat di migrasi 0006) membuatnya tetap cepat pada tabel kecil seperti
-- yang dipakai aplikasi internal ini.
SELECT
    id, kode, nama, keterangan
FROM base
WHERE is_active = TRUE
  AND (
      nama ILIKE '%' || sqlc.arg('q')::text || '%'
      OR kode ILIKE '%' || sqlc.arg('q')::text || '%'
  )
ORDER BY nama ASC
LIMIT sqlc.arg('limit');