-- Migration: 000077_add_manjo_payment_gateway
-- Description: Add Manjo (manjo.co.id) as a second QRIS payment gateway next to Midtrans.
--
-- * tenant_configs gets per-tenant Manjo credentials and the tenant's default gateway.
--   manjo_client_secret and manjo_private_key are Vault-encrypted by tenant-service,
--   like the Midtrans keys.
-- * payment_transactions records which gateway created each payment plus the Manjo
--   references the query and refund APIs need. Existing rows are Midtrans payments.
-- * payment_refunds keeps the refund history of guest order payments.

ALTER TABLE tenant_configs
ADD COLUMN payment_gateway VARCHAR(20) NOT NULL DEFAULT 'midtrans' CHECK (
    payment_gateway IN ('midtrans', 'manjo')
),
ADD COLUMN manjo_client_id TEXT,
ADD COLUMN manjo_client_secret TEXT,
ADD COLUMN manjo_private_key TEXT,
ADD COLUMN manjo_merchant_id TEXT,
ADD COLUMN manjo_partner_id TEXT,
ADD COLUMN manjo_environment VARCHAR(20) NOT NULL DEFAULT 'sandbox' CHECK (
    manjo_environment IN ('sandbox', 'production')
);

COMMENT ON COLUMN tenant_configs.payment_gateway IS 'Default QRIS payment gateway for guest orders: midtrans or manjo';
COMMENT ON COLUMN tenant_configs.manjo_client_id IS 'Manjo client ID, sent as X-CLIENT-KEY when requesting an access token';
COMMENT ON COLUMN tenant_configs.manjo_client_secret IS 'Manjo client secret for HMAC-SHA512 request signatures (Vault-encrypted)';
COMMENT ON COLUMN tenant_configs.manjo_private_key IS 'PEM RSA private key for the SHA256withRSA access token signature (Vault-encrypted)';
COMMENT ON COLUMN tenant_configs.manjo_merchant_id IS 'Manjo merchant ID (merchantId in request bodies)';
COMMENT ON COLUMN tenant_configs.manjo_partner_id IS 'Manjo X-PARTNER-ID; falls back to the client ID when empty';
COMMENT ON COLUMN tenant_configs.manjo_environment IS 'Manjo environment: sandbox or production';

ALTER TABLE payment_transactions
ADD COLUMN payment_gateway VARCHAR(20) NOT NULL DEFAULT 'midtrans' CHECK (
    payment_gateway IN ('midtrans', 'manjo')
),
ADD COLUMN gateway_reference_no VARCHAR(64),
ADD COLUMN gateway_external_id VARCHAR(64);

COMMENT ON COLUMN payment_transactions.payment_gateway IS 'Gateway that created this payment';
COMMENT ON COLUMN payment_transactions.midtrans_transaction_id IS 'Gateway transaction ID: Midtrans transaction_id, or Manjo referenceNo';
COMMENT ON COLUMN payment_transactions.midtrans_order_id IS 'Merchant reference sent to the gateway (order reference / Manjo partnerReferenceNo)';
COMMENT ON COLUMN payment_transactions.gateway_reference_no IS 'Manjo referenceNo returned by QR MPM generate';
COMMENT ON COLUMN payment_transactions.gateway_external_id IS 'X-EXTERNAL-ID sent with the Manjo generate request (originalExternalId for refunds)';

CREATE TABLE payment_refunds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_transaction_id UUID NOT NULL REFERENCES payment_transactions(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    payment_gateway VARCHAR(20) NOT NULL CHECK (payment_gateway IN ('midtrans', 'manjo')),
    partner_refund_no VARCHAR(64) NOT NULL UNIQUE,
    gateway_refund_no VARCHAR(128),
    amount INTEGER NOT NULL CHECK (amount > 0),
    reason TEXT NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('processing', 'success', 'failed')),
    failure_reason TEXT,
    response_payload JSONB,
    requested_by_user_id VARCHAR(64),
    requested_by_name VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payment_refunds_order_id ON payment_refunds (order_id, created_at DESC);
CREATE INDEX idx_payment_refunds_payment_transaction_id ON payment_refunds (payment_transaction_id);

-- At most one refund in flight or done per payment: refunds are full refunds.
CREATE UNIQUE INDEX uq_payment_refunds_active_per_payment
    ON payment_refunds (payment_transaction_id)
    WHERE status IN ('processing', 'success');

COMMENT ON TABLE payment_refunds IS 'Refunds of guest order QRIS payments, requested by owners';
