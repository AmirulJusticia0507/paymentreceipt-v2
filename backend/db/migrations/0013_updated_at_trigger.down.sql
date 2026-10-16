-- 0013_updated_at_trigger.down.sql
DROP TRIGGER IF EXISTS trg_pr_archive_updated_at     ON pr_archive;
DROP TRIGGER IF EXISTS trg_payment_receipt_updated_at ON payment_receipt;
DROP TRIGGER IF EXISTS trg_sources_updated_at         ON sources;
DROP TRIGGER IF EXISTS trg_request_base_updated_at   ON request_base;
DROP TRIGGER IF EXISTS trg_requests_updated_at       ON requests;
DROP TRIGGER IF EXISTS trg_base_updated_at           ON base;
DROP TRIGGER IF EXISTS trg_categories_updated_at     ON categories;
DROP TRIGGER IF EXISTS trg_satuan_updated_at         ON satuan;
DROP TRIGGER IF EXISTS trg_department_updated_at     ON department;
DROP TRIGGER IF EXISTS trg_users_updated_at          ON users;

DROP FUNCTION IF EXISTS set_updated_at();