-- name: CreateAuditLog :exec
INSERT INTO audit_log (actor_id, http_method, request_path, status_code, request_id)
VALUES (@actor_id, @http_method, @request_path, @status_code, @request_id);

-- name: ListAuditLogs :many
SELECT a.id, a.actor_id, u.username, u.full_name, a.http_method,
       a.request_path, a.status_code, a.request_id, a.created_at
FROM audit_log a
JOIN users u ON u.id = a.actor_id
ORDER BY a.created_at DESC, a.id DESC
LIMIT sqlc.arg('page_limit') OFFSET sqlc.arg('page_offset');

-- name: CountAuditLogs :one
SELECT count(*) FROM audit_log;
