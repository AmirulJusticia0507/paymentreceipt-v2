-- name: GetUserByUsername :one
-- Mengambil user untuk login. Kolom password_hash ikut diambil karena
-- service butuh memverifikasinya; layer atas TIDAK BOLEH mengirimnya ke klien.
SELECT
    id, username, email, password_hash, full_name, role, is_active,
    last_login_at, created_at, updated_at
FROM users
WHERE username = @username;

-- name: GetUserByID :one
SELECT
    id, username, email, password_hash, full_name, role, is_active,
    last_login_at, created_at, updated_at
FROM users
WHERE id = @id;

-- name: GetUserSafeByID :one
-- Versi tanpa password_hash, untuk ditampilkan di /auth/me dan daftar user.
SELECT
    id, username, email, full_name, role, is_active,
    last_login_at, created_at, updated_at
FROM users
WHERE id = @id;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = @password_hash, updated_at = now()
WHERE id = @id;

-- name: TouchUserLastLogin :exec
UPDATE users
SET last_login_at = now()
WHERE id = @id;

-- name: DeactivateUser :exec
-- Soft delete. User tidak dihapus dari tabel supaya audit trail
-- (status_history, pr_archive) tetap punya rujukan yang valid.
UPDATE users
SET is_active = FALSE
WHERE id = @id;

-- name: CountActiveUsers :one
SELECT count(*) FROM users WHERE is_active = TRUE;

-- name: ListActiveUsers :many
SELECT
    id, username, email, full_name, role, is_active,
    last_login_at, created_at, updated_at
FROM users
WHERE is_active = TRUE
ORDER BY username ASC;

-- name: ListUsers :many
SELECT
    id, username, email, full_name, role, is_active,
    last_login_at, created_at, updated_at
FROM users
ORDER BY is_active DESC, username ASC;