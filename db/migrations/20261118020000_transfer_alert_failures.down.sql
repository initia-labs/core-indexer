DROP INDEX ix_transfer_alerts_pending;
ALTER TABLE transfer_alerts DROP COLUMN chain_id;
ALTER TABLE transfer_alerts DROP COLUMN telegram_message;
CREATE INDEX ix_transfer_alerts_pending ON transfer_alerts (created_at, id)
    WHERE sent_at IS NULL;
DROP INDEX ix_transfer_alert_batches_pending;
ALTER TABLE transfer_alert_batches DROP COLUMN evaluation_error;
CREATE INDEX ix_transfer_alert_batches_pending
    ON transfer_alert_batches (chain_id, block_height, transaction_hash)
    WHERE evaluated_at IS NULL;
