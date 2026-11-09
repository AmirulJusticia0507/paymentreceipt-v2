-- Status notifikasi dibaca per akun; aktivitas receipt baru membuat notifikasi baru.
CREATE TABLE receipt_notification_reads (
    user_id BIGINT NOT NULL,
    payment_receipt_id BIGINT NOT NULL,
    activity_at TIMESTAMPTZ NOT NULL,
    read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT receipt_notification_reads_pkey PRIMARY KEY (user_id, payment_receipt_id),
    CONSTRAINT receipt_notification_reads_user_fk FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT receipt_notification_reads_receipt_fk FOREIGN KEY (payment_receipt_id)
        REFERENCES payment_receipt (id) ON DELETE CASCADE
);
