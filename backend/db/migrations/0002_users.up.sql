-- 0002_users.up.sql
-- Tabel user + refresh token.
--
-- Aplikasi legacy TIDAK PUNYA autentikasi sama sekali: tidak ada tabel user,
-- tidak ada session guard, tidak ada role enforcement. Semua ini baru.

CREATE TYPE user_role AS ENUM (
    'admin',      -- akses penuh, termasuk master data & user management
    'ga_staff',   -- General Affairs: create/edit/delete payment receipt & arsip PR
    'approver',   -- lihat semua, ubah status jadi checked/finished
    'viewer'      -- read-only
);

CREATE TABLE users (
    id             BIGSERIAL PRIMARY KEY,
    username       CITEXT       NOT NULL,
    email          CITEXT       NOT NULL,
    -- argon2id encoded string (PHC format). Panjang 97 byte untuk default param.
    password_hash  TEXT         NOT NULL,
    full_name      VARCHAR(150) NOT NULL DEFAULT '',
    role           user_role    NOT NULL DEFAULT 'viewer',
    is_active      BOOLEAN      NOT NULL DEFAULT TRUE,
    last_login_at  TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT users_username_key UNIQUE (username),
    CONSTRAINT users_email_key    UNIQUE (email),
    CONSTRAINT users_username_format CHECK (username ~ '^[a-zA-Z0-9._-]{3,50}$'),
    CONSTRAINT users_email_format    CHECK (email ~ '^[^@\s]+@[^@\s]+\.[^@\s]+$'),
    CONSTRAINT users_fullname_len    CHECK (char_length(full_name) <= 150)
);

CREATE INDEX idx_users_role      ON users (role)     WHERE is_active;
CREATE INDEX idx_users_active    ON users (is_active);
CREATE INDEX idx_users_last_login ON users (last_login_at DESC NULLS LAST);

-- Refresh token disimpan sebagai HASH, bukan plaintext.
-- Kalau database bocor, attacker tidak langsung mendapat token yang valid.
CREATE TABLE refresh_tokens (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL,
    token_hash TEXT        NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    user_agent TEXT,
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT refresh_tokens_hash_key UNIQUE (token_hash),
    CONSTRAINT refresh_tokens_user_fk  FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE,
    -- revoked_at hanya bisa diisi manual (revocation), tidak boleh di-set di masa depan
    CONSTRAINT refresh_tokens_revoked_not_future CHECK (
        revoked_at IS NULL OR revoked_at <= NOW()
    ),
    CONSTRAINT refresh_tokens_expires_future CHECK (expires_at > created_at)
);

-- Lookup saat refresh: WHERE token_hash = $1
-- Index unik sudah menutupnya, tapi ini untuk cleanup token kedaluwarsa.
CREATE INDEX idx_refresh_tokens_expires  ON refresh_tokens (expires_at);
CREATE INDEX idx_refresh_tokens_user     ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_active   ON refresh_tokens (user_id, expires_at)
    WHERE revoked_at IS NULL;