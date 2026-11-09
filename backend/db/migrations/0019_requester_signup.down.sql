ALTER TABLE users
    DROP CONSTRAINT users_phone_format,
    DROP COLUMN phone,
    DROP CONSTRAINT users_email_format,
    ALTER COLUMN email SET NOT NULL,
    ADD CONSTRAINT users_email_format CHECK (email ~ '^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$');
