-- Reverses 002_add_delivery_fields.up.sql ONLY: restores the original
-- 4-value status CHECK constraint (drops the "pending_provider_unconfigured"
-- honest-delivery-provider-not-configured state added by 002) and drops the
-- delivery `target` column.
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_status_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_status_check
    CHECK (status IN ('pending', 'sent', 'delivered', 'failed'));

ALTER TABLE notifications DROP COLUMN IF EXISTS target;
