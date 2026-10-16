-- 0013_updated_at_trigger.up.sql
-- Set updated_at = NOW() otomatis di setiap tabel yang punya kolom itu.
--
-- Bukan di-set manual di service layer karena:
--   1. Raw query atau batch sqlc yang lupa set updated_at tetap aman
--   2. Tidak perlu mengingat setiap query UPDATE
--
-- Kalau updated_at diisi dari application code, ada kalanya "lupa" - dan kolomnya
-- jadi tidak akurat tanpa error yang terlihat. Trigger menutup celah itu.

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$;

-- Satu trigger per tabel; nama trigger sama, cukup satu fungsi.
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_department_updated_at
    BEFORE UPDATE ON department
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_satuan_updated_at
    BEFORE UPDATE ON satuan
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_categories_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_base_updated_at
    BEFORE UPDATE ON base
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_requests_updated_at
    BEFORE UPDATE ON requests
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_request_base_updated_at
    BEFORE UPDATE ON request_base
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_sources_updated_at
    BEFORE UPDATE ON sources
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_payment_receipt_updated_at
    BEFORE UPDATE ON payment_receipt
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_pr_archive_updated_at
    BEFORE UPDATE ON pr_archive
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();