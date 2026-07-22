DROP INDEX IF EXISTS idx_subscriptions_external_subscription_id;

ALTER TABLE subscriptions
    DROP COLUMN IF EXISTS external_customer_id,
    DROP COLUMN IF EXISTS external_subscription_id,
    DROP COLUMN IF EXISTS provider;
