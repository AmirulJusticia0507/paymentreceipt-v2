-- name: DashboardCounts :one
-- Statistik ringkas: total request aktif, payment receipt, base, dan department.
SELECT
    (SELECT count(*) FROM requests WHERE is_active = TRUE)     AS total_requests,
    (SELECT count(*) FROM payment_receipt)                     AS total_payment_receipts,
    (SELECT count(*) FROM base WHERE is_active = TRUE)         AS total_bases,
    (SELECT count(*) FROM department WHERE is_active = TRUE)   AS total_departments;

-- name: DashboardStatusBreakdown :many
-- Jumlah payment receipt per status seluruh waktu.
SELECT status, count(*) AS count
FROM payment_receipt
GROUP BY status
ORDER BY status;

-- name: DashboardMonthlySpend :many
-- Total belanja per bulan untuk 12 bulan terakhir.
SELECT date_trunc('month', tanggal_pembelian)::date AS bulan,
       coalesce(sum(total_harga), 0)::text AS spend
FROM payment_receipt
WHERE tanggal_pembelian >= (CURRENT_DATE - interval '11 months')::date
GROUP BY 1
ORDER BY 1;

-- name: DashboardTopDepartments :many
-- Department dengan total belanja terbesar (secara default 5 teratas).
SELECT d.id, d.nama,
       coalesce(sum(p.total_harga), 0)::text AS total
FROM department d
LEFT JOIN payment_receipt p ON p.department_id = d.id
GROUP BY d.id, d.nama
ORDER BY coalesce(sum(p.total_harga), 0) DESC, d.nama ASC
LIMIT $1;

-- name: OperationalReport :many
-- Rekap operasional receipt per department dan status untuk periode pilihan.
SELECT d.id AS department_id, d.kode AS department_kode, d.nama AS department_nama,
       p.status, count(*) AS receipt_count,
       coalesce(sum(p.total_harga), 0)::text AS total_amount
FROM payment_receipt p
JOIN department d ON d.id = p.department_id
WHERE p.tanggal_pembelian >= sqlc.arg('date_from')::date
  AND p.tanggal_pembelian <= sqlc.arg('date_to')::date
  AND (sqlc.narg('department_id')::bigint IS NULL
       OR p.department_id = sqlc.narg('department_id')::bigint)
GROUP BY d.id, d.kode, d.nama, p.status
ORDER BY d.nama, p.status;
