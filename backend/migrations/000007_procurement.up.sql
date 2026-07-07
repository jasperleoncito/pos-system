-- Suppliers, purchase orders, low-stock alerts.

CREATE TABLE suppliers (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants (id),
    name           TEXT NOT NULL,
    contact_person TEXT NOT NULL DEFAULT '',
    phone          TEXT NOT NULL DEFAULT '',
    email          TEXT NOT NULL DEFAULT '',
    address        TEXT NOT NULL DEFAULT '',
    notes          TEXT NOT NULL DEFAULT '',
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_suppliers_tenant_name ON suppliers (tenant_id, lower(name)) WHERE deleted_at IS NULL;

CREATE TABLE po_counters (
    tenant_id UUID PRIMARY KEY REFERENCES tenants (id),
    counter   BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE purchase_orders (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants (id),
    po_number   BIGINT NOT NULL,
    supplier_id UUID NOT NULL REFERENCES suppliers (id),
    status      TEXT NOT NULL DEFAULT 'draft', -- draft | ordered | partially_received | received | cancelled
    notes       TEXT NOT NULL DEFAULT '',
    total       BIGINT NOT NULL DEFAULT 0, -- centavos
    created_by  UUID NOT NULL REFERENCES users (id),
    received_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_po_tenant_number ON purchase_orders (tenant_id, po_number);
CREATE INDEX idx_po_tenant_status ON purchase_orders (tenant_id, status, created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE purchase_order_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants (id),
    po_id        UUID NOT NULL REFERENCES purchase_orders (id),
    item_id      UUID NOT NULL REFERENCES inventory_items (id),
    qty_ordered  NUMERIC(14,3) NOT NULL CHECK (qty_ordered > 0),
    qty_received NUMERIC(14,3) NOT NULL DEFAULT 0,
    unit_cost    BIGINT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_po_items_po ON purchase_order_items (tenant_id, po_id);

CREATE TABLE stock_alerts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants (id),
    item_id         UUID NOT NULL REFERENCES inventory_items (id),
    alert_type      TEXT NOT NULL DEFAULT 'low_stock', -- low_stock | out_of_stock
    stock_at_alert  NUMERIC(14,3) NOT NULL DEFAULT 0,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One open alert per item.
CREATE UNIQUE INDEX idx_stock_alerts_open ON stock_alerts (tenant_id, item_id) WHERE acknowledged_at IS NULL;
