-- 0007_requests.up.sql
-- Pengajuan milik satu department.
--
-- Legacy: `requests` (request_id, nama_request, department_id) tanpa unique constraint
-- dan tanpa soft delete. Relasi ke department tidak punya FK sama sekali - foreign key
-- di legacy hanya "diceritakan" lewat query, tidak ditegakkan database.

CREATE TABLE requests (
    id             BIGSERIAL PRIMARY KEY,
    kode           VARCHAR(30)  NOT NULL,
    nama           VARCHAR(150) NOT NULL,
    department_id  BIGINT       NOT NULL,
    keterangan     TEXT         NOT NULL DEFAULT '',
    is_active      BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT requests_kode_key UNIQUE (kode),
    CONSTRAINT requests_kode_format CHECK (kode ~ '^[A-Z0-9-]{2,30}$'),
    -- ON DELETE RESTRICT (default): department yang sudah punya request tidak boleh dihapus.
    CONSTRAINT requests_department_fk FOREIGN KEY (department_id)
        REFERENCES department (id) ON DELETE RESTRICT
);

CREATE INDEX idx_requests_department ON requests (department_id);
CREATE INDEX idx_requests_active    ON requests (is_active);
CREATE INDEX idx_requests_nama_trgm ON requests USING GIN (nama gin_trgm_ops);