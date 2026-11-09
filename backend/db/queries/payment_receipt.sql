-- name: ListPaymentReceipts :many
SELECT
    p.id, p.kode, p.request_id, r.kode AS request_kode, r.nama AS request_nama,
    p.department_id, d.kode AS department_kode, d.nama AS department_nama,
    p.source_id, COALESCE(s.source::text, '')::text AS source_mode, s.nama_barang AS source_nama_barang,
    p.category_id, c.nama AS category_nama, p.unit_id, u.nama AS unit_nama,
    p.tanggal_pembelian, p.usage, p.deskripsi_items, p.jumlah_items,
    p.hargaitems, p.total_harga::text AS total_harga, COALESCE(p.price_estimate::text, '')::text AS price_estimate, p.keterangan,
    p.status, p.inputdate, p.created_at, p.updated_at
FROM payment_receipt p
JOIN requests r ON r.id = p.request_id
JOIN department d ON d.id = p.department_id
LEFT JOIN sources s ON s.id = p.source_id
LEFT JOIN categories c ON c.id = p.category_id
LEFT JOIN satuan u ON u.id = p.unit_id
WHERE (sqlc.narg('status')::payment_status IS NULL OR p.status = sqlc.narg('status')::payment_status)
  AND (sqlc.narg('department_id')::bigint IS NULL OR p.department_id = sqlc.narg('department_id')::bigint)
  AND (sqlc.narg('request_id')::bigint IS NULL OR p.request_id = sqlc.narg('request_id')::bigint)
  AND (sqlc.narg('date_from')::date IS NULL OR p.tanggal_pembelian >= sqlc.narg('date_from')::date)
  AND (sqlc.narg('date_to')::date IS NULL OR p.tanggal_pembelian <= sqlc.narg('date_to')::date)
  AND (sqlc.narg('q')::text IS NULL OR p.kode ILIKE '%' || sqlc.narg('q')::text || '%' OR p.deskripsi_items ILIKE '%' || sqlc.narg('q')::text || '%' OR r.nama ILIKE '%' || sqlc.narg('q')::text || '%')
ORDER BY
  CASE WHEN sqlc.arg('sort')::text = 'tanggal_asc' THEN p.tanggal_pembelian END ASC,
  CASE WHEN sqlc.arg('sort')::text = 'kode_asc' THEN p.kode END ASC,
  CASE WHEN sqlc.arg('sort')::text = 'kode_desc' THEN p.kode END DESC,
  CASE WHEN sqlc.arg('sort')::text = 'total_asc' THEN p.total_harga END ASC,
  CASE WHEN sqlc.arg('sort')::text = 'total_desc' THEN p.total_harga END DESC,
  p.tanggal_pembelian DESC, p.id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountPaymentReceipts :one
SELECT count(*) FROM payment_receipt p
JOIN requests r ON r.id = p.request_id
WHERE (sqlc.narg('status')::payment_status IS NULL OR p.status = sqlc.narg('status')::payment_status)
  AND (sqlc.narg('department_id')::bigint IS NULL OR p.department_id = sqlc.narg('department_id')::bigint)
  AND (sqlc.narg('request_id')::bigint IS NULL OR p.request_id = sqlc.narg('request_id')::bigint)
  AND (sqlc.narg('date_from')::date IS NULL OR p.tanggal_pembelian >= sqlc.narg('date_from')::date)
  AND (sqlc.narg('date_to')::date IS NULL OR p.tanggal_pembelian <= sqlc.narg('date_to')::date)
  AND (sqlc.narg('q')::text IS NULL OR p.kode ILIKE '%' || sqlc.narg('q')::text || '%' OR p.deskripsi_items ILIKE '%' || sqlc.narg('q')::text || '%' OR r.nama ILIKE '%' || sqlc.narg('q')::text || '%');

-- name: GetPaymentReceipt :one
SELECT
    p.id, p.kode, p.request_id, r.kode AS request_kode, r.nama AS request_nama,
    p.department_id, d.kode AS department_kode, d.nama AS department_nama,
    p.source_id, COALESCE(s.source::text, '')::text AS source_mode, s.nama_barang AS source_nama_barang,
    p.category_id, c.nama AS category_nama, p.unit_id, u.nama AS unit_nama,
    p.tanggal_pembelian, p.usage, p.deskripsi_items, p.jumlah_items,
    p.hargaitems, p.total_harga::text AS total_harga, COALESCE(p.price_estimate::text, '')::text AS price_estimate, p.keterangan,
    p.status, p.inputdate, p.created_at, p.updated_at
FROM payment_receipt p
JOIN requests r ON r.id = p.request_id
JOIN department d ON d.id = p.department_id
LEFT JOIN sources s ON s.id = p.source_id
LEFT JOIN categories c ON c.id = p.category_id
LEFT JOIN satuan u ON u.id = p.unit_id
WHERE p.id = $1;

-- name: CreatePaymentReceipt :one
INSERT INTO payment_receipt (
  kode, request_id, department_id, source_id, category_id, unit_id,
  tanggal_pembelian, usage, deskripsi_items, jumlah_items, hargaitems,
  price_estimate, keterangan
) VALUES (
  'PR-' || to_char(CURRENT_DATE, 'YYYY') || '-' || lpad(nextval('seq_payment_receipt_kode')::text, 6, '0'),
  @request_id, @department_id, @source_id, @category_id, @unit_id,
  @tanggal_pembelian, @usage, @deskripsi_items, @jumlah_items, @hargaitems,
  @price_estimate, @keterangan
)
RETURNING *;

-- name: UpdatePaymentReceipt :one
UPDATE payment_receipt
SET request_id = @request_id,
    department_id = @department_id,
    source_id = @source_id,
    category_id = @category_id,
    unit_id = @unit_id,
    tanggal_pembelian = @tanggal_pembelian,
    usage = @usage,
    deskripsi_items = @deskripsi_items,
    jumlah_items = @jumlah_items,
    hargaitems = @hargaitems,
    price_estimate = @price_estimate,
    keterangan = @keterangan
WHERE id = @id
RETURNING id, kode, request_id, department_id, source_id, category_id, unit_id,
    tanggal_pembelian, usage, deskripsi_items, jumlah_items, hargaitems,
    total_harga, price_estimate, keterangan, status, inputdate, created_at, updated_at;

-- name: UpdatePaymentReceiptStatus :one
UPDATE payment_receipt
SET status = @status
WHERE id = @id
RETURNING id, kode, request_id, department_id, source_id, category_id, unit_id,
    tanggal_pembelian, usage, deskripsi_items, jumlah_items, hargaitems,
    total_harga, price_estimate, keterangan, status, inputdate, created_at, updated_at;

-- name: DeletePaymentReceipt :exec
DELETE FROM payment_receipt WHERE id = @id;

-- name: ListReceiptNotifications :many
WITH receipt_activity AS (
    SELECT p.id, p.kode, p.status, p.department_id, d.nama AS department_nama,
           GREATEST(p.updated_at, COALESCE(MAX(h.created_at), p.created_at)) AS last_activity
    FROM payment_receipt p
    JOIN department d ON d.id = p.department_id
    LEFT JOIN status_history h ON h.payment_receipt_id = p.id
    WHERE p.status IN ('pending', 'checked', 'processed')
      AND (sqlc.narg('department_id')::bigint IS NULL OR p.department_id = sqlc.narg('department_id')::bigint)
    GROUP BY p.id, d.nama
)
SELECT a.id, a.kode, a.status, a.department_id, a.department_nama,
       GREATEST(0, CURRENT_DATE - a.last_activity::date)::int AS days_waiting
FROM receipt_activity a
LEFT JOIN receipt_notification_reads nr
  ON nr.user_id = sqlc.arg('user_id') AND nr.payment_receipt_id = a.id
WHERE nr.activity_at IS NULL OR nr.activity_at <> a.last_activity
ORDER BY a.last_activity ASC, a.id ASC
LIMIT 100;

-- name: MarkReceiptNotificationRead :one
WITH receipt_activity AS (
    SELECT p.id, GREATEST(p.updated_at, COALESCE(MAX(h.created_at), p.created_at)) AS last_activity
    FROM payment_receipt p
    LEFT JOIN status_history h ON h.payment_receipt_id = p.id
    WHERE p.id = sqlc.arg('payment_receipt_id')
      AND (sqlc.narg('department_id')::bigint IS NULL OR p.department_id = sqlc.narg('department_id')::bigint)
    GROUP BY p.id
)
INSERT INTO receipt_notification_reads (user_id, payment_receipt_id, activity_at, read_at)
SELECT sqlc.arg('user_id'), a.id, a.last_activity, NOW()
FROM receipt_activity a
ON CONFLICT (user_id, payment_receipt_id) DO UPDATE
SET activity_at = EXCLUDED.activity_at, read_at = EXCLUDED.read_at
RETURNING payment_receipt_id;
