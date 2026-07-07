-- Customers, membership tiers, and the loyalty points ledger.

CREATE TABLE customers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants (id),
    full_name       TEXT NOT NULL,
    phone           TEXT NOT NULL DEFAULT '',
    email           TEXT NOT NULL DEFAULT '',
    birthday        DATE,
    notes           TEXT NOT NULL DEFAULT '',
    points_balance  BIGINT NOT NULL DEFAULT 0 CHECK (points_balance >= 0),
    lifetime_points BIGINT NOT NULL DEFAULT 0,
    tier            TEXT NOT NULL DEFAULT 'regular', -- regular | silver | gold | vip
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_customers_tenant_name ON customers (tenant_id, full_name) WHERE deleted_at IS NULL;
-- One profile per phone number when a phone is recorded.
CREATE UNIQUE INDEX idx_customers_tenant_phone ON customers (tenant_id, phone)
    WHERE phone <> '' AND deleted_at IS NULL;

CREATE TABLE loyalty_settings (
    tenant_id         UUID PRIMARY KEY REFERENCES tenants (id),
    is_enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    earn_rate         BIGINT NOT NULL DEFAULT 5000 CHECK (earn_rate > 0),   -- centavos spent per 1 point
    redeem_value      BIGINT NOT NULL DEFAULT 100 CHECK (redeem_value > 0), -- centavos of value per point
    silver_threshold  BIGINT NOT NULL DEFAULT 500,   -- lifetime points
    gold_threshold    BIGINT NOT NULL DEFAULT 1500,
    vip_threshold     BIGINT NOT NULL DEFAULT 4000,
    silver_multiplier NUMERIC(4,2) NOT NULL DEFAULT 1.25,
    gold_multiplier   NUMERIC(4,2) NOT NULL DEFAULT 1.50,
    vip_multiplier    NUMERIC(4,2) NOT NULL DEFAULT 2.00,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE loyalty_transactions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants (id),
    customer_id   UUID NOT NULL REFERENCES customers (id),
    order_id      UUID REFERENCES orders (id),
    type          TEXT NOT NULL, -- earn | redeem | adjust
    points        BIGINT NOT NULL, -- signed delta
    balance_after BIGINT NOT NULL,
    notes         TEXT NOT NULL DEFAULT '',
    created_by    UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_loyalty_tx_customer ON loyalty_transactions (tenant_id, customer_id, created_at DESC);
CREATE INDEX idx_loyalty_tx_order ON loyalty_transactions (tenant_id, order_id) WHERE order_id IS NOT NULL;
