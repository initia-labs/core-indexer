ALTER TABLE public.module_histories ADD COLUMN proposal_id INTEGER NULL ;
ALTER TABLE public.module_histories ADD CONSTRAINT module_histories_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES public.proposals(id);
ALTER TABLE public.module_histories ADD COLUMN tx_id INTEGER NULL ;
ALTER TABLE public.module_histories ADD CONSTRAINT module_histories_tx_id_fkey FOREIGN KEY (tx_id) REFERENCES public.transactions(id);

ALTER TABLE public.nfts ADD COLUMN proposal_id INTEGER NULL ;
ALTER TABLE public.nfts ADD CONSTRAINT nfts_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES public.proposals(id);
ALTER TABLE public.nfts ADD COLUMN tx_id INTEGER NULL ;
ALTER TABLE public.nfts ADD CONSTRAINT nfts_tx_id_fkey FOREIGN KEY (tx_id) REFERENCES public.transactions(id);

ALTER TABLE public.nft_histories ADD COLUMN proposal_id INTEGER NULL ;
ALTER TABLE public.nft_histories ADD CONSTRAINT nft_histories_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES public.proposals(id);
ALTER TABLE public.nft_histories ADD COLUMN tx_id INTEGER NULL ;
ALTER TABLE public.nft_histories ADD CONSTRAINT nft_histories_tx_id_fkey FOREIGN KEY (tx_id) REFERENCES public.transactions(id);


ALTER TABLE public.nft_mutation_events ADD COLUMN proposal_id INTEGER NULL ;
ALTER TABLE public.nft_mutation_events ADD CONSTRAINT nft_mutation_events_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES public.proposals(id);
ALTER TABLE public.nft_mutation_events ADD COLUMN tx_id INTEGER NULL ;
ALTER TABLE public.nft_mutation_events ADD CONSTRAINT nft_mutation_events_tx_id_fkey FOREIGN KEY (tx_id) REFERENCES public.transactions(id);


ALTER TABLE public.collection_mutation_events ADD COLUMN proposal_id INTEGER NULL ;
ALTER TABLE public.collection_mutation_events ADD CONSTRAINT collection_mutation_events_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES public.proposals(id);
ALTER TABLE public.collection_mutation_events ADD COLUMN tx_id INTEGER NULL ;
ALTER TABLE public.collection_mutation_events ADD CONSTRAINT collection_mutation_events_tx_id_fkey FOREIGN KEY (tx_id) REFERENCES public.transactions(id);
