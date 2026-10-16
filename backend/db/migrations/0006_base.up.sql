-- 0006_base.up.sql
-- Kategori pengajuan dasar.
--
-- Legacy: tabel `base` punya DUA kolom id - `base_id` dan `kode_base` - yang isinya
-- merujuk konsep sama. Legacy juga punya file `basename/data_basename.php` DAN
-- `sources/data_basename.php` dengan nama file identik tapi isi berbeda (hasil copy-paste).
-- Di sini cukup satu `kode`.

CREATE TABLE base (
    id          BIGSERIAL PRIMARY KEY,
    kode        VARCHAR(30)  NOT NULL,
    nama        VARCHAR(150) NOT NULL,
    keterangan  TEXT         NOT NULL DEFAULT '',
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT base_kode_key UNIQUE (kode),
    CONSTRAINT base_kode_format CHECK (kode ~ '^[A-Z0-9-]{2,30}$')
);

-- Endpoint autocomplete: WHERE basename LIKE '%term%' ORDER BY nama
-- Trigram GIN index mempercepat ILIKE '%...%'
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_base_nama_trgm ON base USING GIN (nama gin_trgm_ops);
CREATE INDEX idx_base_active      ON base (is_active);