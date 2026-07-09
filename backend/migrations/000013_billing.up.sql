-- Subscription billing: platform prices, per-tenant subscriptions, and
-- the payment ledger (Xendit invoices + manual admin entries).

-- Singleton price sheet, editable by the super admin.
CREATE TABLE platform_settings (
    id                     INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    monthly_price_centavos BIGINT NOT NULL DEFAULT 80000,
    yearly_price_centavos  BIGINT NOT NULL DEFAULT 800000,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO platform_settings (id) VALUES (1);

CREATE TABLE subscriptions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL UNIQUE REFERENCES tenants (id),
    plan                 TEXT NOT NULL CHECK (plan IN ('monthly', 'yearly')),
    status               TEXT NOT NULL CHECK (status IN ('pending', 'active', 'inactive')),
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT now(),
    current_period_end   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Set when the 3-day renewal notice goes out; reset to NULL on every
    -- period extension so the next cycle notifies again.
    due_notice_sent_at   TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_subscriptions_status_due ON subscriptions (status, current_period_end);

CREATE TABLE subscription_payments (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants (id),
    subscription_id   UUID NOT NULL REFERENCES subscriptions (id),
    plan              TEXT NOT NULL CHECK (plan IN ('monthly', 'yearly')),
    amount            BIGINT NOT NULL, -- centavos
    status            TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'expired')),
    method            TEXT NOT NULL DEFAULT 'xendit' CHECK (method IN ('xendit', 'manual')),
    -- Our reference sent to Xendit as external_id (equals this row's id
    -- for xendit payments); webhook lookups key off this.
    external_id       TEXT NOT NULL UNIQUE,
    xendit_invoice_id  TEXT NOT NULL DEFAULT '',
    xendit_invoice_url TEXT NOT NULL DEFAULT '',
    payment_channel   TEXT NOT NULL DEFAULT '',
    paid_at           TIMESTAMPTZ,
    recorded_by       UUID, -- super admin user for manual entries
    note              TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_subscription_payments_tenant ON subscription_payments (tenant_id, created_at DESC);

-- Existing tenants are auto-enrolled: active monthly, due 30 days out.
INSERT INTO subscriptions (tenant_id, plan, status, current_period_start, current_period_end)
SELECT id, 'monthly', 'active', now(), now() + interval '30 days'
FROM tenants
WHERE deleted_at IS NULL
ON CONFLICT (tenant_id) DO NOTHING;
