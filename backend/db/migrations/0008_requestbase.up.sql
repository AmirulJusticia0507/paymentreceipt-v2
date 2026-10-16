-- 0008_requestbase.up.sql
-- Pengajuan turunan dari `base`: misalnya base "Perlengkapan Kantor" ->
-- request_base "Pengadaan Komputer" dengan jabatan penanggung jawab.
--
-- Legacy: `requestbase` (regbase_id, namarequestbase, jabatan, kode_base).
-- Perhatikan inkonsistensi nama: kolomnya `namarequestbase` (tanpa pemisah) sementara
-- tabel lain pakai `nama_request`. Di sini konsisten pakai `nama`.

CREATE TABLE request_base (
    id          BIGSERIAL PRIMARY KEY,
    kode        VARCHAR(30)  NOT NULL,
    nama        VARCHAR(150) NOT NULL,
    jabatan     VARCHAR(100) NOT NULL DEFAULT '',
    base_id     BIGINT       NOT NULL,
    keterangan  TEXT         NOT NULL DEFAULT '',
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT request_base_kode_key UNIQUE (kode),
    CONSTRAINT request_base_kode_format CHECK (kode ~ '^[A-Z0-9-]{2,30}$'),
    CONSTRAINT request_base_base_fk FOREIGN KEY (base_id)
        REFERENCES base (id) ON DELETE RESTRICT
);

CREATE INDEX idx_request_base_base    ON request_base (base_id);
CREATE INDEX idx_request_base_active  ON request_base (is_active);
CREATE INDEX idx_request_base_nama_trgm ON request_base USING GIN (nama gin_trgm_ops);