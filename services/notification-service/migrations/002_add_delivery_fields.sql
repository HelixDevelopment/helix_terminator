-- Adds the delivery `target` column (recipient email address for
-- channel=email, destination URL for channel=webhook) and extends the
-- status CHECK constraint with the honest "pending_provider_unconfigured"
-- state used by not-yet-configured delivery providers (push/FCM/APNs) so we
-- never fabricate a false "sent"/"delivered" status.
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS target VARCHAR(1000);

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_status_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_status_check
    CHECK (status IN ('pending', 'sent', 'delivered', 'failed', 'pending_provider_unconfigured'));
