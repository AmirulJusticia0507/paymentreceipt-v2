-- name: CreateStatusHistory :one
INSERT INTO status_history (payment_receipt_id, from_status, to_status, changed_by, note)
VALUES (@payment_receipt_id, @from_status, @to_status, @changed_by, @note)
RETURNING id, payment_receipt_id, from_status, to_status, changed_by, note, created_at;

-- name: ListStatusHistoryByReceipt :many
SELECT
    h.id, h.payment_receipt_id, h.from_status, h.to_status,
    h.changed_by, u.username AS changed_by_username, h.note, h.created_at
FROM status_history h
JOIN users u ON u.id = h.changed_by
WHERE h.payment_receipt_id = @payment_receipt_id
ORDER BY h.created_at ASC, h.id ASC;
