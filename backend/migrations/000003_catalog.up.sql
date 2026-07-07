-- Menu catalog: categories, products, variants, modifiers, taxes.
-- All prices are BIGINT centavos.

CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants (id),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order  INT NOT NULL DEFAULT 0,
    image_key   TEXT NOT NULL DEFAULT '',
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_categories_tenant ON categories (tenant_id, sort_order) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_categories_tenant_name ON categories (tenant_id, lower(name)) WHERE deleted_at IS NULL;

CREATE TABLE taxes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants (id),
    name         TEXT NOT NULL,
    rate_percent NUMERIC(5,2) NOT NULL DEFAULT 0,
    is_inclusive BOOLEAN NOT NULL DEFAULT TRUE,
    is_default   BOOLEAN NOT NULL DEFAULT FALSE,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_taxes_tenant ON taxes (tenant_id) WHERE deleted_at IS NULL;

CREATE TABLE products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants (id),
    category_id     UUID NOT NULL REFERENCES categories (id),
    tax_id          UUID REFERENCES taxes (id),
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    sku             TEXT NOT NULL DEFAULT '',
    base_price      BIGINT NOT NULL DEFAULT 0,
    cost_price      BIGINT NOT NULL DEFAULT 0,
    image_key       TEXT NOT NULL DEFAULT '',
    thumb_key       TEXT NOT NULL DEFAULT '',
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    track_inventory BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_products_tenant_category ON products (tenant_id, category_id, sort_order) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_tenant_name ON products (tenant_id, lower(name)) WHERE deleted_at IS NULL;

CREATE TABLE product_variants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants (id),
    product_id  UUID NOT NULL REFERENCES products (id),
    name        TEXT NOT NULL,
    price_delta BIGINT NOT NULL DEFAULT 0,
    sku         TEXT NOT NULL DEFAULT '',
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_product_variants_product ON product_variants (tenant_id, product_id, sort_order) WHERE deleted_at IS NULL;

CREATE TABLE modifier_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants (id),
    name        TEXT NOT NULL,
    min_select  INT NOT NULL DEFAULT 0,
    max_select  INT NOT NULL DEFAULT 1,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_modifier_groups_tenant ON modifier_groups (tenant_id, sort_order) WHERE deleted_at IS NULL;

CREATE TABLE modifiers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants (id),
    group_id    UUID NOT NULL REFERENCES modifier_groups (id),
    name        TEXT NOT NULL,
    price_delta BIGINT NOT NULL DEFAULT 0,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_modifiers_group ON modifiers (tenant_id, group_id, sort_order) WHERE deleted_at IS NULL;

CREATE TABLE product_modifier_groups (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants (id),
    product_id        UUID NOT NULL REFERENCES products (id),
    modifier_group_id UUID NOT NULL REFERENCES modifier_groups (id),
    sort_order        INT NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_pmg_unique ON product_modifier_groups (tenant_id, product_id, modifier_group_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_pmg_product ON product_modifier_groups (tenant_id, product_id) WHERE deleted_at IS NULL;
