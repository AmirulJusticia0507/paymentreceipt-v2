-- 0014_archive_flag.down.sql
ALTER TABLE payment_receipt
    DROP COLUMN archived_at;
