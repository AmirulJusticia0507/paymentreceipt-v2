-- 0003_department.up.sql
-- Unit kerja pemohon.
--
-- Legacy: tabel `department` dengan kolom nama_department & keterangandepartment,
-- tanpa kode, tanpa unique constraint, tanpa soft delete.

CREATE TABLE department (
    id          BIGSERIAL PRIMARY KEY,
    kode        VARCHAR(30)  NOT NULL,
    nama        VARCHAR(100) NOT NULL,
    keterangan  VARCHAR(255) NOT NULL DEFAULT '',
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT department_kode_key UNIQUE (kode),
    CONSTRAINT department_kode_format CHECK (kode ~ '^[A-Z0-9-]{2,30}$')
);

-- List endpoint selalu filter is_active dan sort by nama.
CREATE INDEX idx_department_active_nama ON department (nama) WHERE is_active;