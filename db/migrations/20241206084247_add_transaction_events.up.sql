CREATE TABLE IF NOT EXISTS public.transaction_events (
    transaction_hash VARCHAR NOT NULL,
    block_height BIGINT NOT NULL,
    event_key VARCHAR NOT NULL,
    event_value VARCHAR NOT NULL,
    event_index INTEGER NOT NULL,
    PRIMARY KEY (transaction_hash, block_height, event_index)
);

GRANT SELECT ON public.transaction_events TO readonly;

CREATE TYPE public.finalize_block_events_mode AS ENUM (
    'BeginBlock',
    'EndBlock'
    );

CREATE TABLE public.finalize_block_events (
   block_height BIGINT NOT NULL,
   event_key VARCHAR NOT NULL,
   event_value VARCHAR NOT NULL,
   event_index INTEGER NOT NULL,
   mode public.finalize_block_events_mode NOT NULL,
   PRIMARY KEY (block_height, event_index)
);

GRANT SELECT ON public.finalize_block_events TO readonly;
