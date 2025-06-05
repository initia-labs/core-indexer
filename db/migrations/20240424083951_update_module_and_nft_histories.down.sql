ALTER TABLE public.module_histories DROP COLUMN tx_id;
ALTER TABLE public.module_histories DROP COLUMN proposal_id;

ALTER TABLE public.nfts DROP COLUMN tx_id;
ALTER TABLE public.nfts DROP COLUMN proposal_id;

ALTER TABLE public.nft_histories DROP COLUMN tx_id;
ALTER TABLE public.nft_histories DROP COLUMN proposal_id;

ALTER TABLE public.nft_mutation_events DROP COLUMN tx_id;
ALTER TABLE public.nft_mutation_events DROP COLUMN proposal_id;

ALTER TABLE public.collection_mutation_events DROP COLUMN tx_id;
ALTER TABLE public.collection_mutation_events DROP COLUMN proposal_id;
