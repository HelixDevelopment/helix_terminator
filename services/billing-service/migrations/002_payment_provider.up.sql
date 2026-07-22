-- 002_payment_provider: adds real-payment-processor bookkeeping columns
-- to subscriptions (Constitution §11.4 anti-bluff covenant — see
-- internal/billing/provider.go for the full rationale). Prior to this
-- migration, subscriptions.status was written unconditionally as
-- 'active' with no reference to any external payment processor
-- whatsoever; a subscription row carried no evidence any processor had
-- ever been contacted. These columns let the DB row itself carry
-- proof-of-real-call: which processor was used (provider), and that
-- processor's own identifiers for the customer and subscription it
-- created (external_customer_id / external_subscription_id).
--
-- provider defaults to 'none' so pre-existing rows (created before this
-- migration, back when no processor was ever contacted at all) are
-- honestly labeled rather than silently backfilled with a fabricated
-- 'stripe' value this migration has no evidence for.
ALTER TABLE subscriptions
    ADD COLUMN provider VARCHAR(50) NOT NULL DEFAULT 'none',
    ADD COLUMN external_subscription_id VARCHAR(255),
    ADD COLUMN external_customer_id VARCHAR(255);

CREATE INDEX idx_subscriptions_external_subscription_id ON subscriptions(external_subscription_id)
    WHERE external_subscription_id IS NOT NULL;
