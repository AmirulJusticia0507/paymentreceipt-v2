-- name: GetUserByUsername :one
-- Mengambil user untuk login. Kolom password_hash ikut diambil karena
-- service butuh memverifikasinya; layer atas TIDAK BOLEH mengirimnya ke klien.
SELECT
    id, username, email, password_hash, full_name, role, department_id, is_active,
    last_login_at, created_at, updated_at
FROM users
WHERE username = @username;

-- name: GetUserByID :one
SELECT
    id, username, email, password_hash, full_name, role, department_id, is_active,
    last_login_at, created_at, updated_at
FROM users
WHERE id = @id;

-- name: GetUserSafeByID :one
-- Versi tanpa password_hash, untuk ditampilkan di /auth/me dan daftar user.
SELECT
    id, username, email, full_name, role, is_active, department_id,
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

-- name: UpdateUserFullName :one
-- Pengguna hanya boleh mengubah nama akun miliknya sendiri. Query ini
-- mengembalikan profil aman agar UI dapat langsung memperbarui headernya.
UPDATE users
SET full_name = @full_name, updated_at = now()
WHERE id = @id
RETURNING id, username, email, full_name, role, is_active, department_id, last_login_at, created_at, updated_at;

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
    id, username, email, full_name, role, is_active, department_id,
    last_login_at, created_at, updated_at
FROM users
WHERE is_active = TRUE
ORDER BY username ASC;

-- name: ListUsers :many
SELECT
    id, username, email, full_name, role, is_active, department_id,
    last_login_at, created_at, updated_at
FROM users
ORDER BY is_active DESC, username ASC;

-- name: UpdateRequesterDepartment :one
UPDATE users
SET department_id = @department_id, updated_at = now()
WHERE id = @id AND role = 'requester'::user_role
RETURNING id, username, email, full_name, role, is_active, department_id,
          last_login_at, created_at, updated_at;
