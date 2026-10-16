-- 0011_pr_archive.up.sql
-- Arsip Purchase Requisition: PR yang sudah disetujui dan siap dicetak.
--
-- Legacy: `archieved` (id_archieved, payments_id, tanggal_selesai, namarequestPR,
-- managerapproved, description, quantity, hargarencana, hargaaktual).
--
-- Bug legacy yang relevan di sini:
--   COUNT(quantity*hargarencana=hargaaktual) AS Total
-- dipakai di PR/data_PR.php untuk "menghitung" total. COUNT() di sini menghitung
--JUMLAH BARIS yang memenuhi kondisi, bukan menjumlahkan nilainya - hasilnya 0 atau 1,
-- bukan total harga. Total yang ditampilkan di halaman PR legacy selalu salah.
--
-- Di sini total = harga_rencana (dijumlahkan di query, bukan dihitung dari COUNT).
-- `harga_aktual` diisi dari total payment receipt terkait (T7.3).

CREATE TABLE pr_archive (
    id                BIGSERIAL PRIMARY KEY,
    kode              VARCHAR(30)   NOT NULL,
    payment_receipt_id BIGINT       NOT NULL,
    tanggal_selesai   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    nama_request      VARCHAR(150)  NOT NULL,
    manager_approved  VARCHAR(150)  NOT NULL DEFAULT '',
    approved_by       BIGINT,          -- users.id, NULL kalau belum ada user-nya
    description       TEXT            NOT NULL DEFAULT '',
    quantity          INTEGER         NOT NULL DEFAULT 0,
    harga_rencana     NUMERIC(15,2)   NOT NULL DEFAULT 0,
    harga_aktual      NUMERIC(15,2)   NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT pr_archive_kode_key UNIQUE (kode),
    CONSTRAINT pr_archive_receipt_fk FOREIGN KEY (payment_receipt_id)
        REFERENCES payment_receipt (id) ON DELETE RESTRICT,
    -- Manager boleh punya user atau tidak (data lama tidak punya)
    CONSTRAINT pr_archive_approved_by_fk FOREIGN KEY (approved_by)
        REFERENCES users (id) ON DELETE SET NULL,

    CONSTRAINT pr_archive_quantity_positive   CHECK (quantity >= 0),
    CONSTRAINT pr_archive_rencana_nonneg      CHECK (harga_rencana >= 0),
    CONSTRAINT pr_archive_aktual_nonneg       CHECK (harga_aktual >= 0)
);

CREATE INDEX idx_pr_archive_receipt   ON pr_archive (payment_receipt_id);
CREATE INDEX idx_pr_archive_approver  ON pr_archive (approved_by);
CREATE INDEX idx_pr_archive_tanggal   ON pr_archive (tanggal_selesai DESC, id DESC);
CREATE INDEX idx_pr_archive_manager_trgm ON pr_archive USING GIN (manager_approved gin_trgm_ops);

CREATE SEQUENCE IF NOT EXISTS seq_pr_archive_kode START 1;