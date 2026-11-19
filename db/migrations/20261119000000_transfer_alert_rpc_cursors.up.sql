CREATE TABLE transfer_alert_rpc_cursors (
    chain_id TEXT PRIMARY KEY,
    height BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
