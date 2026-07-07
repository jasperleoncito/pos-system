-- Tenancy & authentication foundation.

CREATE TABLE tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    slug          TEXT NOT NULL,
    owner_user_id UUID,
    status        TEXT NOT NULL DEFAULT 'active', -- active | suspended
    currency      TEXT NOT NULL DEFAULT 'PHP',
    timezone      TEXT NOT NULL DEFAULT 'Asia/Manila',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_tenants_slug ON tenants (slug) WHERE deleted_at IS NULL;

CREATE TABLE tenant_settings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants (id),
    logo_key        TEXT NOT NULL DEFAULT '',
    logo_thumb_key  TEXT NOT NULL DEFAULT '',
    favicon_keys    JSONB NOT NULL DEFAULT '{}',
    primary_color   TEXT NOT NULL DEFAULT '#DC2626',
    secondary_color TEXT NOT NULL DEFAULT '#F87171',
    accent_color    TEXT NOT NULL DEFAULT '#CA8A04',
    receipt_header  TEXT NOT NULL DEFAULT '',
    receipt_footer  TEXT NOT NULL DEFAULT '',
    contact_number  TEXT NOT NULL DEFAULT '',
    facebook        TEXT NOT NULL DEFAULT '',
    website         TEXT NOT NULL DEFAULT '',
    address         TEXT NOT NULL DEFAULT '',
    tax_label       TEXT NOT NULL DEFAULT '',
    tax_id          TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_tenant_settings_tenant ON tenant_settings (tenant_id) WHERE deleted_at IS NULL;

CREATE TABLE users (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email             TEXT NOT NULL,
    password_hash     TEXT NOT NULL,
    full_name         TEXT NOT NULL,
    phone             TEXT NOT NULL DEFAULT '',
    avatar_key        TEXT NOT NULL DEFAULT '',
    is_super_admin    BOOLEAN NOT NULL DEFAULT FALSE,
    email_verified_at TIMESTAMPTZ,
    status            TEXT NOT NULL DEFAULT 'active', -- active | disabled
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_users_email ON users (lower(email)) WHERE deleted_at IS NULL;

CREATE TABLE tenant_users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants (id),
    user_id    UUID NOT NULL REFERENCES users (id),
    role       TEXT NOT NULL, -- owner | manager | cashier | kitchen | employee
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_tenant_users_unique ON tenant_users (tenant_id, user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_users_user ON tenant_users (user_id) WHERE deleted_at IS NULL;

CREATE TABLE device_sessions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID NOT NULL REFERENCES users (id),
    refresh_token_hash TEXT NOT NULL,
    device_name        TEXT NOT NULL DEFAULT '',
    user_agent         TEXT NOT NULL DEFAULT '',
    ip                 TEXT NOT NULL DEFAULT '',
    last_used_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at         TIMESTAMPTZ NOT NULL,
    revoked_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ
);

CREATE INDEX idx_device_sessions_user ON device_sessions (user_id) WHERE revoked_at IS NULL AND deleted_at IS NULL;
CREATE INDEX idx_device_sessions_token ON device_sessions (refresh_token_hash) WHERE revoked_at IS NULL AND deleted_at IS NULL;

CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID,
    user_id     UUID,
    action      TEXT NOT NULL,
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id   TEXT NOT NULL DEFAULT '',
    before      JSONB,
    after       JSONB,
    ip          TEXT NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_tenant_created ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_user ON audit_logs (user_id, created_at DESC);
