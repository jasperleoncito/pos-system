-- In-app notifications and per-user email preferences.

CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants (id),
    user_id    UUID NOT NULL REFERENCES users (id),
    type       TEXT NOT NULL, -- low_stock | attendance | daily_summary | system
    title      TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    link       TEXT NOT NULL DEFAULT '', -- app path, e.g. /inventory
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user ON notifications (tenant_id, user_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications (tenant_id, user_id) WHERE read_at IS NULL;

CREATE TABLE notification_prefs (
    tenant_id            UUID NOT NULL REFERENCES tenants (id),
    user_id              UUID NOT NULL REFERENCES users (id),
    email_low_stock      BOOLEAN NOT NULL DEFAULT TRUE,
    email_attendance     BOOLEAN NOT NULL DEFAULT TRUE,
    email_daily_summary  BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, user_id)
);
