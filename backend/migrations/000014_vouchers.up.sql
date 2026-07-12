-- Platform-level subscription discount vouchers (super-admin managed).
CREATE TABLE vouchers (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code           TEXT NOT NULL,
    discount_type  TEXT NOT NULL CHECK (discount_type IN ('fixed', 'percentage')),
    discount_value BIGINT NOT NULL CHECK (discount_value > 0), -- centavos (fixed) | percent 1-100 (percentage)
    applies_to     TEXT NOT NULL DEFAULT 'all' CHECK (applies_to IN ('all', 'monthly', 'yearly')),
    max_uses       INT,          -- NULL = unlimited
    used_count     INT NOT NULL DEFAULT 0,
    expires_at     TIMESTAMPTZ,  -- NULL = no expiry
    active         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);

-- One live voucher per code, case-insensitive.
CREATE UNIQUE INDEX vouchers_code_key ON vouchers (upper(code)) WHERE deleted_at IS NULL;

-- Record which voucher a payment redeemed and how much it saved.
ALTER TABLE subscription_payments
    ADD COLUMN voucher_id        UUID REFERENCES vouchers(id),
    ADD COLUMN discount_centavos BIGINT NOT NULL DEFAULT 0;
