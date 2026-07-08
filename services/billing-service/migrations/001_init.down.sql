DROP TRIGGER IF EXISTS invoices_updated_at ON invoices;
DROP TRIGGER IF EXISTS subscriptions_updated_at ON subscriptions;
DROP TRIGGER IF EXISTS billing_plans_updated_at ON billing_plans;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_usage_records_resource_type;
DROP INDEX IF EXISTS idx_usage_records_org_id;
DROP TABLE IF EXISTS usage_records;

DROP INDEX IF EXISTS idx_invoices_subscription_id;
DROP INDEX IF EXISTS idx_invoices_org_id;
DROP TABLE IF EXISTS invoices;

DROP INDEX IF EXISTS idx_subscriptions_status;
DROP INDEX IF EXISTS idx_subscriptions_org_id;
DROP TABLE IF EXISTS subscriptions;

DROP TABLE IF EXISTS billing_plans;
