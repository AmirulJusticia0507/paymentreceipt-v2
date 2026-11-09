-- Pengaitan satu akun pemohon ke department yang menjadi ruang lingkup datanya.
ALTER TABLE users
    ADD COLUMN department_id BIGINT,
    ADD CONSTRAINT users_department_fk FOREIGN KEY (department_id)
        REFERENCES department (id) ON DELETE RESTRICT;

CREATE INDEX idx_users_department ON users (department_id)
    WHERE department_id IS NOT NULL;
