ALTER TABLE transfer_alert_batches ADD COLUMN evaluation_error TEXT;
DROP INDEX ix_transfer_alert_batches_pending;
CREATE INDEX ix_transfer_alert_batches_pending
    ON transfer_alert_batches (chain_id, block_height, transaction_hash)
    WHERE evaluated_at IS NULL AND evaluation_error IS NULL;

ALTER TABLE transfer_alerts ADD COLUMN telegram_message TEXT;
ALTER TABLE transfer_alerts ADD COLUMN chain_id TEXT;
-- Current transaction/block IDs retain the complete chain prefix, including '/'.
-- Older per-event IDs used a simple chain prefix; retain their delivery records.
UPDATE transfer_alerts SET chain_id = CASE
    WHEN id ~ '/[0-9]+/(tx|block)/[^/]+$'
    THEN regexp_replace(id, '/[0-9]+/(tx|block)/[^/]+$', '')
    ELSE split_part(id, '/', 1)
END;
ALTER TABLE transfer_alerts ALTER COLUMN chain_id SET NOT NULL;
DROP INDEX ix_transfer_alerts_pending;
CREATE INDEX ix_transfer_alerts_pending ON transfer_alerts (chain_id, created_at, id)
    WHERE sent_at IS NULL;
