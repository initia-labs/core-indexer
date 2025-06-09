CREATE INDEX ix_collection_transactions_collection_id ON public.collection_transactions(collection_id);
CREATE INDEX ix_collection_transactions_tx_id ON public.collection_transactions(tx_id);
CREATE INDEX ix_collection_transactions_nft_id ON public.collection_transactions(nft_id);
CREATE INDEX ix_collections_creator ON public.collections(creator);
