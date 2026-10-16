-- 0010_payment_receipt.up.sql
-- Tabel utama: transaksi pembelian.
--
-- Legacy: `generalaffairspayment`, kolom-kolomnya:
--   GApayment_id, request_id, department_id, source_id, tanggal_pembelian,
--   categories (teks), kategori_id (FK), usagegeneralaffair, deskripsi_items,
--   jumlah_items, units_id, hargaitems, total_harga, price_estimate,
--   keterangan, inputdate, status
--
-- Empat masalah yang diperbaiki di sini:
--
-- 1. DUPLIKAT KATEGORI: ada `categories` (teks bebas) DAN `kategori_id` (FK) -
--    dua sumber kebenaran untuk hal yang sama. Data bisa tidak sinkron.
--    -> hanya `category_id` FK.
--
-- 2. TOTAL DARI CLIENT: legacy menerima `$_POST['total_harga']` dan NEVER
--    menghitung ulang di server. Siapa pun bisa mengirim total palsu.
--    -> GENERATED column. Database yang menghitung, tidak bisa dipalsukan.
--
-- 3. STATUS VARCHAR BEBAS: legacy pakai `CASE status WHEN 'Checked' THEN ...`
--    di setiap query. Nilai di luar daftar itu menghasilkan NULL, bukan error.
--    -> enum + CHECK.
--
-- 4. TIDAK ADA RIWAYAT: perubahan status di legacy menimpa nilai lama. Tidak ada
--    jejak siapa yang mengubah apa dan kapan.
--    -> tabel status_history (dipisah ke migrasi 0011).

CREATE TYPE payment_status AS ENUM (
    'pending',    -- baru dibuat
    'checked',    -- sudah diverifikasi
    'processed',  -- sedang diproses (pembelian berjalan)
    'finished',   -- selesai
    'cancelled'   -- dibatalkan (terminal)
);

CREATE TABLE payment_receipt (
    id                 BIGSERIAL PRIMARY KEY,
    -- Nomor dokumen. Legacy tidak punya ini sama sekali - itu sebabnya PDF-nya
    -- tidak bisa merujuk transaksi tertentu.
    kode               VARCHAR(30)  NOT NULL,
    request_id         BIGINT       NOT NULL,
    department_id      BIGINT       NOT NULL,
    source_id          BIGINT,
    category_id        BIGINT,
    unit_id            BIGINT,
    tanggal_pembelian  DATE         NOT NULL,
    usage              VARCHAR(150) NOT NULL DEFAULT '',
    deskripsi_items    VARCHAR(255) NOT NULL,
    jumlah_items       INTEGER      NOT NULL,
    hargaitems         NUMERIC(15,2) NOT NULL,
    -- GENERATED: client tidak bisa mengirim nilai, dan tidak bisa dipalsukan.
    total_harga        NUMERIC(15,2)
        GENERATED ALWAYS AS (jumlah_items * hargaitems) STORED,
    price_estimate     NUMERIC(15,2),
    keterangan         TEXT         NOT NULL DEFAULT '',
    status             payment_status NOT NULL DEFAULT 'pending',
    inputdate          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT payment_receipt_kode_key UNIQUE (kode),

    CONSTRAINT payment_receipt_request_fk FOREIGN KEY (request_id)
        REFERENCES requests (id) ON DELETE RESTRICT,
    CONSTRAINT payment_receipt_department_fk FOREIGN KEY (department_id)
        REFERENCES department (id) ON DELETE RESTRICT,
    CONSTRAINT payment_receipt_source_fk FOREIGN KEY (source_id)
        REFERENCES sources (id) ON DELETE RESTRICT,
    CONSTRAINT payment_receipt_category_fk FOREIGN KEY (category_id)
        REFERENCES categories (id) ON DELETE RESTRICT,
    CONSTRAINT payment_receipt_unit_fk FOREIGN KEY (unit_id)
        REFERENCES satuan (id) ON DELETE RESTRICT,

    -- Aturan nilai dasar. Diperkuat lagi di service layer (T6.10).
    CONSTRAINT payment_receipt_qty_positive  CHECK (jumlah_items > 0),
    CONSTRAINT payment_receipt_price_positive CHECK (hargaitems > 0),
    CONSTRAINT payment_receipt_estimate_positive CHECK (
        price_estimate IS NULL OR price_estimate >= 0
    ),
    CONSTRAINT payment_receipt_deskripsi_len CHECK (char_length(deskripsi_items) BETWEEN 1 AND 255),
    CONSTRAINT payment_receipt_usage_len    CHECK (char_length(usage) <= 150),
    -- Tanggal pembelian tidak boleh di masa depan
    CONSTRAINT payment_receipt_date_not_future CHECK (tanggal_pembelian <= CURRENT_DATE)
);

-- Default filter halaman utama: filter status + urut tanggal DESC
CREATE INDEX idx_payment_receipt_status  ON payment_receipt (status);
CREATE INDEX idx_payment_receipt_tanggal ON payment_receipt (tanggal_pembelian DESC, id DESC);

-- FK index: PostgreSQL TIDAK otomatis meng-index kolom FK
CREATE INDEX idx_payment_receipt_request    ON payment_receipt (request_id);
CREATE INDEX idx_payment_receipt_department ON payment_receipt (department_id);
CREATE INDEX idx_payment_receipt_source     ON payment_receipt (source_id);
CREATE INDEX idx_payment_receipt_category   ON payment_receipt (category_id);
CREATE INDEX idx_payment_receipt_unit       ON payment_receipt (unit_id);

-- Filter kombinasi yang paling sering dipakai user
CREATE INDEX idx_payment_receipt_dept_status ON payment_receipt (department_id, status);

-- Autocomplete pada deskripsi item
CREATE INDEX idx_payment_receipt_deskripsi_trgm
    ON payment_receipt USING GIN (deskripsi_items gin_trgm_ops);

-- Kode dokumen berikutnya: PR-2026-000123
CREATE SEQUENCE IF NOT EXISTS seq_payment_receipt_kode START 1;