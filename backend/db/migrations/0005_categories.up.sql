-- 0005_categories.up.sql
-- Kategori barang.
--
-- Sama seperti `satuan`: di legacy tabel ini hanya di-SELECT lewat JOIN, tidak pernah ada
-- INSERT/UPDATE/DELETE. Kategori jadi tidak bisa dikelola user.

CREATE TABLE categories (
    id          BIGSERIAL PRIMARY KEY,
    kode        VARCHAR(30)  NOT NULL,
    nama        VARCHAR(100) NOT NULL,
    keterangan  VARCHAR(255) NOT NULL DEFAULT '',
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT categories_kode_key UNIQUE (kode),
    CONSTRAINT categories_kode_format CHECK (kode ~ '^[A-Z0-9-]{2,30}$')
);

CREATE INDEX idx_categories_active_nama ON categories (nama) WHERE is_active;