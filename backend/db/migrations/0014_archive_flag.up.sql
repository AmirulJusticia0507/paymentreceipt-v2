-- 0014_archive_flag.up.sql
-- Tandai payment receipt yang sudah diarsipkan ke PR, supaya tidak bisa
-- di-archive dua kali.
ALTER TABLE payment_receipt
    ADD COLUMN archived_at timestamp with time zone;

CREATE INDEX idx_payment_receipt_archived_at
    ON payment_receipt (archived_at);
