-- 0010_payment_receipt.down.sql
DROP SEQUENCE IF EXISTS seq_payment_receipt_kode;
DROP TABLE IF EXISTS payment_receipt;
DROP TYPE IF EXISTS payment_status;