-- name: StoreRefreshToken :one
-- Menyimpan refresh token sebagai HASH, bukan token aslinya.
-- Kalau database bocor, attacker tetap tidak punya token yang bisa dipakai.
INSERT INTO refresh_tokens (user_id, token_hash, expires_at, user_agent, ip_address)
VALUES (@user_id, @token_hash, @expires_at, @user_agent, @ip_address)
RETURNING id, user_id, token_hash, expires_at, revoked_at, user_agent, ip_address, created_at;

-- name: GetActiveRefreshToken :one
-- Hanya mengembalikan token yang belum dicabut dan belum kedaluwarsa.
-- revoked_at IS NOT NULL dijoins supaya token yang sudah dipakai (rotasi)
-- tidak bisa dipakai dua kali.
SELECT
    rt.id, rt.user_id, rt.token_hash, rt.expires_at, rt.revoked_at,
    rt.user_agent, rt.ip_address, rt.created_at,
    u.username, u.email, u.full_name, u.role, u.is_active AS user_is_active
FROM refresh_tokens rt
JOIN users u ON u.id = rt.user_id
WHERE rt.token_hash = @token_hash
  AND rt.revoked_at IS NULL
  AND rt.expires_at > now()
  AND u.is_active = TRUE;

-- name: RevokeRefreshToken :exec
-- Revoke satu token (logout).
UPDATE refresh_tokens
SET revoked_at = now()
WHERE token_hash = @token_hash
  AND revoked_at IS NULL;

-- name: RevokeAllUserRefreshTokens :exec
-- Dipakai saat logout dari semua perangkat, atau saat mendeteksi penyalahgunaan
-- (token lama dipakai ulang - indikasi token dicuri).
UPDATE refresh_tokens
SET revoked_at = now()
WHERE user_id = @user_id
  AND revoked_at IS NULL;

-- name: DeleteExpiredRefreshTokens :execrows
-- Cleanup token kedaluwarsa. Dijadwalkan berkala (T12.x).
DELETE FROM refresh_tokens
WHERE expires_at < @cutoff;