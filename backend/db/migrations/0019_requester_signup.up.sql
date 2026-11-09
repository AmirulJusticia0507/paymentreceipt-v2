-- Allow self-registered requester accounts to omit email and record a phone.
ALTER TABLE users
    ALTER COLUMN email DROP NOT NULL,
    DROP CONSTRAINT users_email_format,
    ADD CONSTRAINT users_email_format CHECK (
        email IS NULL OR email ~ '^[^@\s]+@[^@\s]+\.[^@\s]+$'
    ),
    ADD COLUMN phone VARCHAR(30),
    ADD CONSTRAINT users_phone_format CHECK (
        phone IS NULL OR phone ~ '^[0-9+()\- ]{8,30}$'
    );
