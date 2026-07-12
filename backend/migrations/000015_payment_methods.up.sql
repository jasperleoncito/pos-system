-- Allow super-admin comps ('grant') and voucher-covered activations ('voucher')
-- as payment ledger methods, alongside the existing xendit/manual.
ALTER TABLE subscription_payments DROP CONSTRAINT IF EXISTS subscription_payments_method_check;
ALTER TABLE subscription_payments ADD CONSTRAINT subscription_payments_method_check
    CHECK (method IN ('xendit', 'manual', 'grant', 'voucher'));
