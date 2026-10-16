-- 0012_status_history.up.sql
-- Audit trail perubahan status payment receipt.
--
-- Legacy: TIDAK ADA. `UPDATE generalaffairspayment SET status = '$status'` menimpa
-- nilai lama. Tidak ada cara tahu siapa yang mengubah status dari apa ke apa, kapan,
-- atau berapa kali. Untuk sistem yang menangani persetujuan dan butuh audit, ini masalah serius.
--
-- Setiap transisi status (dan pembuatan awal) dicatat di sini.

CREATE TABLE status_history (
    id                 BIGSERIAL PRIMARY KEY,
    payment_receipt_id BIGINT        NOT NULL,
    -- NULL saat pembuatan (dari "tidak ada" ke pending)
    from_status        payment_status,
    to_status          payment_status NOT NULL,
    changed_by         BIGINT         NOT NULL,
    note               TEXT           NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT status_history_receipt_fk FOREIGN KEY (payment_receipt_id)
        REFERENCES payment_receipt (id) ON DELETE CASCADE,
    CONSTRAINT status_history_user_fk FOREIGN KEY (changed_by)
        REFERENCES users (id) ON DELETE RESTRICT,

    -- Transisi tidak boleh "tidak berubah diri" (pending -> pending)
    CONSTRAINT status_history_changed CHECK (
        from_status IS NULL OR from_status <> to_status
    )
);

-- Timeline per receipt (T6.17): ORDER BY created_at DESC
CREATE INDEX idx_status_history_receipt ON status_history (payment_receipt_id, created_at DESC);
CREATE INDEX idx_status_history_user    ON status_history (changed_by);

-- Tabel ini grows tanpa batas. Index ini mendukung T12.x: hapus entri lama
-- setelah waktu tertentu (retention policy).
CREATE INDEX idx_status_history_created ON status_history (created_at);