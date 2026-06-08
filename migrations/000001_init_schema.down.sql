-- Down migration: Rollback Schema

DROP INDEX IF EXISTS idx_outbox_events_status_created;
DROP INDEX IF EXISTS idx_payments_status;
DROP INDEX IF EXISTS idx_payments_customer_id;
DROP INDEX IF EXISTS idx_users_email;

DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS users;
