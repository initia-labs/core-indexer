CREATE TABLE transfer_alert_batches (
    chain_id TEXT NOT NULL,
    block_height BIGINT NOT NULL,
    transaction_hash TEXT NOT NULL,
    events JSONB NOT NULL,
    evaluated_at TIMESTAMPTZ,
    PRIMARY KEY (chain_id, block_height, transaction_hash)
);

CREATE INDEX ix_transfer_alert_batches_pending
    ON transfer_alert_batches (chain_id, block_height, transaction_hash)
    WHERE evaluated_at IS NULL;

CREATE INDEX ix_transfer_alerts_pending
    ON transfer_alerts (created_at, id)
    WHERE sent_at IS NULL;
