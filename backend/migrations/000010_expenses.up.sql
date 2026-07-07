-- Operating expenses feed the profit calculation on the dashboard.

CREATE TABLE expenses (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants (id),
    category     TEXT NOT NULL DEFAULT 'other', -- rent | utilities | supplies | salaries | other
    description  TEXT NOT NULL,
    amount       BIGINT NOT NULL CHECK (amount > 0), -- centavos
    expense_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_by   UUID NOT NULL REFERENCES users (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_expenses_tenant_date ON expenses (tenant_id, expense_date DESC) WHERE deleted_at IS NULL;
