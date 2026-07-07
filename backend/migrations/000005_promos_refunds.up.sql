-- Discounts, coupons, split bills, refunds, voids.

CREATE TABLE discounts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants (id),
    name              TEXT NOT NULL,
    type              TEXT NOT NULL, -- percent | fixed
    percent_value     NUMERIC(5,2) NOT NULL DEFAULT 0, -- when type=percent
    amount_value      BIGINT NOT NULL DEFAULT 0,       -- centavos, when type=fixed
    requires_approval BOOLEAN NOT NULL DEFAULT FALSE,
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX idx_discounts_tenant ON discounts (tenant_id) WHERE deleted_at IS NULL;

CREATE TABLE coupons (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants (id),
    code             TEXT NOT NULL,
    discount_type    TEXT NOT NULL, -- percent | fixed
    percent_value    NUMERIC(5,2) NOT NULL DEFAULT 0,
    amount_value     BIGINT NOT NULL DEFAULT 0,
    min_order_amount BIGINT NOT NULL DEFAULT 0,
    max_uses         INT NOT NULL DEFAULT 0, -- 0 = unlimited
    uses_count       INT NOT NULL DEFAULT 0,
    valid_from       TIMESTAMPTZ,
    valid_to         TIMESTAMPTZ,
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_coupons_tenant_code ON coupons (tenant_id, upper(code)) WHERE deleted_at IS NULL;

CREATE TABLE coupon_redemptions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants (id),
    coupon_id   UUID NOT NULL REFERENCES coupons (id),
    order_id    UUID NOT NULL REFERENCES orders (id),
    customer_id UUID,
    released_at TIMESTAMPTZ, -- set when the order is voided
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_coupon_redemptions_coupon ON coupon_redemptions (tenant_id, coupon_id);
CREATE INDEX idx_coupon_redemptions_order ON coupon_redemptions (tenant_id, order_id);

-- Order-level promo references + split bills.
ALTER TABLE orders ADD COLUMN discount_id UUID REFERENCES discounts (id);
ALTER TABLE orders ADD COLUMN coupon_id UUID REFERENCES coupons (id);

CREATE TABLE order_splits (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants (id),
    order_id     UUID NOT NULL REFERENCES orders (id),
    split_number INT NOT NULL,
    amount       BIGINT NOT NULL CHECK (amount > 0),
    status       TEXT NOT NULL DEFAULT 'pending', -- pending | paid
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_order_splits_number ON order_splits (tenant_id, order_id, split_number) WHERE deleted_at IS NULL;

ALTER TABLE payments ADD COLUMN split_id UUID REFERENCES order_splits (id);

CREATE TABLE refunds (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants (id),
    order_id      UUID NOT NULL REFERENCES orders (id),
    refund_number BIGINT NOT NULL,
    reason        TEXT NOT NULL,
    amount        BIGINT NOT NULL CHECK (amount > 0),
    refunded_by   UUID NOT NULL REFERENCES users (id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refunds_order ON refunds (tenant_id, order_id);

CREATE TABLE refund_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants (id),
    refund_id     UUID NOT NULL REFERENCES refunds (id),
    order_item_id UUID NOT NULL REFERENCES order_items (id),
    qty           INT NOT NULL CHECK (qty > 0),
    amount        BIGINT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refund_items_refund ON refund_items (tenant_id, refund_id);
