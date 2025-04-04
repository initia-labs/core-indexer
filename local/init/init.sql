CREATE DATABASE hasura_metadata;

CREATE TABLE IF NOT EXISTS transaction_events(
    transaction_hash varchar,
    block_height     bigint,
    event_key        varchar,
    event_value      varchar,
    event_index      integer
);

CREATE TABLE IF NOT EXISTS finalize_block_events(
    block_height bigint,
    event_key    varchar,
    event_value  varchar,
    event_index  integer,
    mode         varchar
);

CREATE TABLE IF NOT EXISTS move_events(
    type_tag varchar,
    data jsonb,
    block_height bigint,
    transaction_hash varchar,
    event_index integer
);

INSERT INTO transaction_events(transaction_hash, block_height, event_key, event_value, event_index) VALUES ('7cbd3c3abc1790b92a8ddcfcb666d31891a09b7bea2861d92d444ca07a8859b7', 7197856, 'execute.sender', '0x1,0x50fbce14436f1bfa9b30576dbb8a5a0c1e249dd3', 0);
