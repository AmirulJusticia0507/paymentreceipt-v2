-- 0001_extensions.up.sql
-- Ekstensi dasar yang dipakai seluruh skema.

CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS citext;     -- username/email case-insensitive