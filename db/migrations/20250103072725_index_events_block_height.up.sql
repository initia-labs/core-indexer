CREATE INDEX ix_finalize_block_events_event_key_block_height_desc ON public.finalize_block_events (event_key, block_height DESC);

CREATE INDEX ix_transactions_events_event_key_block_height_desc
    ON public.transaction_events (event_key, block_height DESC);
