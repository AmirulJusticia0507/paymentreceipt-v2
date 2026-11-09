-- Catat aktor dan konteks setiap mutasi API yang berhasil.
-- Body request sengaja tidak disimpan agar rahasia atau data pribadi tidak tersalin.
CREATE TABLE audit_log (
    id           BIGSERIAL PRIMARY KEY,
    actor_id     BIGINT NOT NULL,
    http_method  VARCHAR(8) NOT NULL,
    request_path TEXT NOT NULL,
    status_code  SMALLINT NOT NULL,
    request_id   VARCHAR(64) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT audit_log_actor_fk FOREIGN KEY (actor_id)
        REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT audit_log_method_check CHECK (http_method IN ('POST', 'PUT', 'PATCH', 'DELETE')),
    CONSTRAINT audit_log_success_check CHECK (status_code >= 200 AND status_code < 300)
);

CREATE INDEX idx_audit_log_created ON audit_log (created_at DESC, id DESC);
CREATE INDEX idx_audit_log_actor ON audit_log (actor_id, created_at DESC);
