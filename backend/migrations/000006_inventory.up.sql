-- Inventory: units, items, recipes (BOM), append-only movement ledger.

CREATE TABLE units (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants (id),
    name         TEXT NOT NULL,
    abbreviation TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_units_tenant_name ON units (tenant_id, lower(name)) WHERE deleted_at IS NULL;

CREATE TABLE inventory_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants (id),
    name          TEXT NOT NULL,
    type          TEXT NOT NULL DEFAULT 'ingredient', -- ingredient | finished_good
    unit_id       UUID NOT NULL REFERENCES units (id),
    current_stock NUMERIC(14,3) NOT NULL DEFAULT 0,
    reorder_level NUMERIC(14,3) NOT NULL DEFAULT 0,
    cost_per_unit BIGINT NOT NULL DEFAULT 0, -- centavos
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_inventory_items_tenant_name ON inventory_items (tenant_id, lower(name)) WHERE deleted_at IS NULL;

-- BOM: what one unit of a product consumes.
CREATE TABLE recipe_items (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants (id),
    product_id        UUID NOT NULL REFERENCES products (id),
    inventory_item_id UUID NOT NULL REFERENCES inventory_items (id),
    qty               NUMERIC(14,3) NOT NULL CHECK (qty > 0),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_recipe_items_unique ON recipe_items (tenant_id, product_id, inventory_item_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_recipe_items_product ON recipe_items (tenant_id, product_id) WHERE deleted_at IS NULL;

-- Append-only ledger; every stock change goes through here.
CREATE TABLE inventory_movements (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants (id),
    item_id        UUID NOT NULL REFERENCES inventory_items (id),
    movement_type  TEXT NOT NULL, -- stock_in | stock_out | adjustment | sale | po_receive | refund_return | waste
    qty_delta      NUMERIC(14,3) NOT NULL, -- signed
    qty_before     NUMERIC(14,3) NOT NULL,
    qty_after      NUMERIC(14,3) NOT NULL,
    unit_cost      BIGINT NOT NULL DEFAULT 0,
    reference_type TEXT NOT NULL DEFAULT '', -- order | purchase_order | manual
    reference_id   TEXT NOT NULL DEFAULT '',
    notes          TEXT NOT NULL DEFAULT '',
    performed_by   UUID NOT NULL REFERENCES users (id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_inventory_movements_item ON inventory_movements (tenant_id, item_id, created_at DESC);
CREATE INDEX idx_inventory_movements_ref ON inventory_movements (tenant_id, reference_type, reference_id);
