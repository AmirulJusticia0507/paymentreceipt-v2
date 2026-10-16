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
