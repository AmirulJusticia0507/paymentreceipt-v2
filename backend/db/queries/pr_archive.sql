-- name: ListPrArchives :many
-- Daftar PR archive dengan filter tanggal, pencarian, dan approver.
-- Join ke payment_receipt untuk harga_aktual cross-check, dan ke user untuk
-- nama approver.
SELECT
    a.id, a.kode, a.payment_receipt_id, p.kode AS receipt_kode,
    a.tanggal_selesai, a.nama_request, a.manager_approved, a.approved_by,
    u.username AS approved_by_username, a.description, a.quantity,
    a.harga_rencana, a.harga_aktual, a.created_at, a.updated_at
FROM pr_archive a
JOIN payment_receipt p ON p.id = a.payment_receipt_id
LEFT JOIN users u ON u.id = a.approved_by
WHERE (sqlc.narg('date_from')::timestamptz IS NULL OR a.tanggal_selesai >= sqlc.narg('date_from')::timestamptz)
  AND (sqlc.narg('date_to')::timestamptz IS NULL OR a.tanggal_selesai <= sqlc.narg('date_to')::timestamptz)
  AND (sqlc.narg('q')::text IS NULL OR a.kode ILIKE '%' || sqlc.narg('q')::text || '%'
       OR a.nama_request ILIKE '%' || sqlc.narg('q')::text || '%'
       OR a.manager_approved ILIKE '%' || sqlc.narg('q')::text || '%'
       OR p.kode ILIKE '%' || sqlc.narg('q')::text || '%')
  AND (sqlc.narg('approver')::bigint IS NULL OR a.approved_by = sqlc.narg('approver')::bigint)
ORDER BY a.tanggal_selesai DESC, a.id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountPrArchives :one
SELECT count(*)
FROM pr_archive a
JOIN payment_receipt p ON p.id = a.payment_receipt_id
LEFT JOIN users u ON u.id = a.approved_by
WHERE (sqlc.narg('date_from')::timestamptz IS NULL OR a.tanggal_selesai >= sqlc.narg('date_from')::timestamptz)
  AND (sqlc.narg('date_to')::timestamptz IS NULL OR a.tanggal_selesai <= sqlc.narg('date_to')::timestamptz)
  AND (sqlc.narg('q')::text IS NULL OR a.kode ILIKE '%' || sqlc.narg('q')::text || '%'
       OR a.nama_request ILIKE '%' || sqlc.narg('q')::text || '%'
       OR a.manager_approved ILIKE '%' || sqlc.narg('q')::text || '%'
       OR p.kode ILIKE '%' || sqlc.narg('q')::text || '%')
  AND (sqlc.narg('approver')::bigint IS NULL OR a.approved_by = sqlc.narg('approver')::bigint);

-- name: GetPrArchive :one
SELECT
    a.id, a.kode, a.payment_receipt_id, p.kode AS receipt_kode,
    a.tanggal_selesai, a.nama_request, a.manager_approved, a.approved_by,
    u.username AS approved_by_username, a.description, a.quantity,
    a.harga_rencana, a.harga_aktual, a.created_at, a.updated_at
FROM pr_archive a
JOIN payment_receipt p ON p.id = a.payment_receipt_id
LEFT JOIN users u ON u.id = a.approved_by
WHERE a.id = @id;

-- name: CreatePrArchive :one
INSERT INTO pr_archive (kode, payment_receipt_id, tanggal_selesai, nama_request,
                        manager_approved, approved_by, description, quantity,
                        harga_rencana, harga_aktual)
VALUES (@kode, @payment_receipt_id, @tanggal_selesai, @nama_request,
        @manager_approved, @approved_by, @description, @quantity,
        @harga_rencana, @harga_aktual)
RETURNING id, kode, payment_receipt_id, tanggal_selesai, nama_request,
          manager_approved, approved_by, description, quantity,
          harga_rencana, harga_aktual, created_at, updated_at;

-- name: UpdatePrArchive :one
UPDATE pr_archive
SET nama_request = @nama_request,
    manager_approved = @manager_approved,
    description = @description,
    quantity = @quantity,
    harga_rencana = @harga_rencana,
    harga_aktual = @harga_aktual
WHERE id = @id
RETURNING id, kode, payment_receipt_id, tanggal_selesai, nama_request,
          manager_approved, approved_by, description, quantity,
          harga_rencana, harga_aktual, created_at, updated_at;

-- name: DeletePrArchive :exec
DELETE FROM pr_archive WHERE id = @id;

-- name: ListUnarchivedFinishedReceipts :many
-- Payment receipt yang sudah finished dan BELUM diarsipkan, siap untuk bulk
-- archive. Urutkan oleh tanggal pembelian.
SELECT
    p.id, p.kode, p.request_id, r.kode AS request_kode, r.nama AS request_nama,
    p.department_id, d.kode AS department_kode, d.nama AS department_nama,
    p.tanggal_pembelian, p.deskripsi_items, p.jumlah_items,
    p.hargaitems, p.total_harga::text AS total_harga, p.status
FROM payment_receipt p
JOIN requests r ON r.id = p.request_id
JOIN department d ON d.id = p.department_id
WHERE p.status = 'finished'::payment_status
  AND p.archived_at IS NULL
ORDER BY p.tanggal_pembelian ASC, p.id ASC;

-- name: MarkReceiptsArchived :exec
-- Tandai sejumlah payment receipt sebagai sudah diarsipkan. Hanya receipt yang
-- belum diarsipkan (archived_at IS NULL) yang diperbarui, jadi aman dipanggil
-- berulang kali.
UPDATE payment_receipt
SET archived_at = NOW()
WHERE id = ANY(@ids::bigint[]) AND archived_at IS NULL;
