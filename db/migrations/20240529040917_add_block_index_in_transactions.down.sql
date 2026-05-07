DROP INDEX ix_transactions_block_height_block_index;
ALTER TABLE transactions DROP COLUMN block_index;
