-- Reverses 003_add_slack_channel.up.sql ONLY: restores the pre-003
-- 4-value channel CHECK constraint on both `notifications` and
-- `notification_preferences` (drops 'slack' from the allowed set).
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_channel_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_channel_check
    CHECK (channel IN ('email', 'in_app', 'push', 'webhook'));

ALTER TABLE notification_preferences DROP CONSTRAINT IF EXISTS notification_preferences_channel_check;
ALTER TABLE notification_preferences ADD CONSTRAINT notification_preferences_channel_check
    CHECK (channel IN ('email', 'in_app', 'push', 'webhook'));
