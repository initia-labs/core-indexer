CREATE TABLE transfer_alerts (
    id TEXT PRIMARY KEY,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ
);

CREATE TABLE transfer_alert_cursors (
    chain_id TEXT PRIMARY KEY,
    height BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
