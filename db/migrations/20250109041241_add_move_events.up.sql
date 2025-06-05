CREATE TABLE IF NOT EXISTS public.move_events (
    type_tag VARCHAR NOT NULL,
    data JSONB NOT NULL,
    block_height BIGINT NOT NULL,
    transaction_hash VARCHAR NOT NULL,
    event_index INTEGER NOT NULL,
    PRIMARY KEY (transaction_hash, block_height, event_index)
);

GRANT SELECT ON public.move_events TO readonly;

CREATE INDEX ix_move_events_type_tag_block_height_desc
    ON public.move_events (type_tag, block_height DESC);
