-- POS core: orders, payments, cash drawer. All money is BIGINT centavos.

-- Per-tenant human-friendly order numbering.
CREATE TABLE order_counters (
    tenant_id UUID PRIMARY KEY REFERENCES tenants (id),
    counter   BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants (id),
    order_number    BIGINT NOT NULL,
    order_type      TEXT NOT NULL DEFAULT 'dine_in', -- dine_in | takeout | delivery
    table_number    TEXT NOT NULL DEFAULT '',
    customer_id     UUID,
    cashier_user_id UUID NOT NULL REFERENCES users (id),
    status          TEXT NOT NULL DEFAULT 'open', -- open | held | completed | voided | refunded | partially_refunded
    kitchen_status  TEXT NOT NULL DEFAULT 'pending', -- pending | preparing | ready | completed
    priority        BOOLEAN NOT NULL DEFAULT FALSE,
    subtotal        BIGINT NOT NULL DEFAULT 0,
    discount_total  BIGINT NOT NULL DEFAULT 0,
    tax_total       BIGINT NOT NULL DEFAULT 0, -- informational for inclusive taxes
    total           BIGINT NOT NULL DEFAULT 0,
    tendered        BIGINT NOT NULL DEFAULT 0,
    change          BIGINT NOT NULL DEFAULT 0,
    notes           TEXT NOT NULL DEFAULT '',
    completed_at    TIMESTAMPTZ,
    voided_by       UUID,
    void_reason     TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_orders_tenant_number ON orders (tenant_id, order_number);
CREATE INDEX idx_orders_tenant_status ON orders (tenant_id, status, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_tenant_created ON orders (tenant_id, created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE order_items (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants (id),
    order_id       UUID NOT NULL REFERENCES orders (id),
    product_id     UUID NOT NULL,
    variant_id     UUID,
    -- Snapshots survive later menu edits.
    name           TEXT NOT NULL,
    variant_name   TEXT NOT NULL DEFAULT '',
    unit_price     BIGINT NOT NULL, -- base + variant + modifiers
    qty            INT NOT NULL CHECK (qty > 0),
    discount_amount BIGINT NOT NULL DEFAULT 0,
    tax_amount     BIGINT NOT NULL DEFAULT 0,
    line_total     BIGINT NOT NULL,
    notes          TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'pending', -- per-item kitchen status
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);

CREATE INDEX idx_order_items_order ON order_items (tenant_id, order_id) WHERE deleted_at IS NULL;

CREATE TABLE order_item_modifiers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants (id),
    order_item_id UUID NOT NULL REFERENCES order_items (id),
    modifier_id   UUID NOT NULL,
    group_name    TEXT NOT NULL DEFAULT '',
    name          TEXT NOT NULL,
    price_delta   BIGINT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_order_item_modifiers_item ON order_item_modifiers (tenant_id, order_item_id);

CREATE TABLE payments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants (id),
    order_id     UUID NOT NULL REFERENCES orders (id),
    method       TEXT NOT NULL, -- cash | gcash | card | maya | bank_transfer
    amount       BIGINT NOT NULL CHECK (amount > 0),
    reference_no TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'paid', -- paid | refunded
    received_by  UUID NOT NULL REFERENCES users (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_payments_order ON payments (tenant_id, order_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payments_tenant_created ON payments (tenant_id, created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE cash_drawer_sessions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants (id),
    opened_by     UUID NOT NULL REFERENCES users (id),
    closed_by     UUID,
    opening_float BIGINT NOT NULL DEFAULT 0,
    expected_cash BIGINT NOT NULL DEFAULT 0, -- float + cash in − cash out
    counted_cash  BIGINT,
    variance      BIGINT,
    status        TEXT NOT NULL DEFAULT 'open', -- open | closed
    opened_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at     TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

-- One open drawer per tenant at a time.
CREATE UNIQUE INDEX idx_drawer_one_open ON cash_drawer_sessions (tenant_id) WHERE status = 'open' AND deleted_at IS NULL;

CREATE TABLE cash_movements (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants (id),
    session_id UUID NOT NULL REFERENCES cash_drawer_sessions (id),
    type       TEXT NOT NULL, -- open_float | sale | change | refund | drop | payout
    amount     BIGINT NOT NULL, -- signed: positive into drawer, negative out
    order_id   UUID,
    reason     TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cash_movements_session ON cash_movements (tenant_id, session_id, created_at);

CREATE TABLE order_status_history (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants (id),
    order_id   UUID NOT NULL REFERENCES orders (id),
    field      TEXT NOT NULL DEFAULT 'status', -- status | kitchen_status
    from_value TEXT NOT NULL DEFAULT '',
    to_value   TEXT NOT NULL,
    changed_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_order_status_history_order ON order_status_history (tenant_id, order_id, created_at);
