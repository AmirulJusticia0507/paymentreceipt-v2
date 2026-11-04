-- 0015_requester_role.up.sql
-- Tambah role 'requester' ke enum user_role untuk staf pemohon PR departemen.

ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'requester';
