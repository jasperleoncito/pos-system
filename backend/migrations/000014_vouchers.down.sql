ALTER TABLE subscription_payments
    DROP COLUMN IF EXISTS voucher_id,
    DROP COLUMN IF EXISTS discount_centavos;

DROP TABLE IF EXISTS vouchers;
