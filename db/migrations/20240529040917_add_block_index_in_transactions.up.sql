ALTER TABLE public.transactions ADD COLUMN block_index int NOT NULL DEFAULT 0;

CREATE INDEX ix_transactions_block_height_block_index ON public.transactions(block_height desc, block_index desc);
