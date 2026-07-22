-- Adds 'slack' to the channel CHECK constraint on both `notifications` and
-- `notification_preferences`, following the exact ADD-COLUMN-was-not-
-- needed pattern of 002_add_delivery_fields.up.sql (the `target` column
-- 002 already added is reused verbatim by channel=slack — see
-- internal/model/model.go's Target doc comment: destination Slack channel
-- ID).
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_channel_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_channel_check
    CHECK (channel IN ('email', 'in_app', 'push', 'webhook', 'slack'));

ALTER TABLE notification_preferences DROP CONSTRAINT IF EXISTS notification_preferences_channel_check;
ALTER TABLE notification_preferences ADD CONSTRAINT notification_preferences_channel_check
    CHECK (channel IN ('email', 'in_app', 'push', 'webhook', 'slack'));
