-- 1
alter table public.transactions add column new_id varchar;
update public.transactions set new_id = UPPER(ENCODE(hash, 'hex')) || '/' || block_height;
alter table public.transactions drop constraint transactions_pkey;
alter table public.transactions add primary key (new_id);

-- 2
alter table public.proposals add column new_tx_id varchar;
update public.proposals set new_tx_id = transactions.new_id from public.transactions where proposals.created_tx = transactions.id;
alter table public.proposals drop column created_tx;
alter table public.proposals rename column new_tx_id to created_tx;
alter table public.proposals add constraint proposals_created_tx_fkey foreign key (created_tx) references public.transactions (new_id);

alter table public.account_transactions add column new_tx_id varchar;
update public.account_transactions set new_tx_id = transactions.new_id from public.transactions where account_transactions.transaction_id = transactions.id;
alter table public.account_transactions drop column transaction_id;
alter table public.account_transactions rename column new_tx_id to transaction_id;
alter table public.account_transactions add constraint account_transactions_transaction_id_fkey foreign key (transaction_id) references public.transactions (new_id);

alter table public.lcd_tx_results add column new_tx_id varchar;
update public.lcd_tx_results set new_tx_id = transactions.new_id from public.transactions where lcd_tx_results.transaction_id = transactions.id;
alter table public.lcd_tx_results drop column transaction_id;
alter table public.lcd_tx_results rename column new_tx_id to transaction_id;
alter table public.lcd_tx_results add constraint lcd_tx_results_transaction_id_fkey foreign key (transaction_id) references public.transactions (new_id);

alter table public.proposal_deposits add column new_tx_id varchar;
update public.proposal_deposits set new_tx_id = transactions.new_id from public.transactions where proposal_deposits.transaction_id = transactions.id;
alter table public.proposal_deposits drop column transaction_id;
alter table public.proposal_deposits rename column new_tx_id to transaction_id;
alter table public.proposal_deposits add constraint proposal_deposits_transaction_id_fkey foreign key (transaction_id) references public.transactions (new_id);

alter table public.proposal_votes add column new_tx_id varchar;
update public.proposal_votes set new_tx_id = transactions.new_id from public.transactions where proposal_votes.transaction_id = transactions.id;
alter table public.proposal_votes drop column transaction_id;
alter table public.proposal_votes rename column new_tx_id to transaction_id;
alter table public.proposal_votes add constraint proposal_votes_transaction_id_fkey foreign key (transaction_id) references public.transactions (new_id);

alter table public.proposal_votes_legacy add column new_tx_id varchar;
update public.proposal_votes_legacy set new_tx_id = transactions.new_id from public.transactions where proposal_votes_legacy.transaction_id = transactions.id;
alter table public.proposal_votes_legacy drop column transaction_id;
alter table public.proposal_votes_legacy rename column new_tx_id to transaction_id;
alter table public.proposal_votes_legacy add constraint proposal_votes_legacy_transaction_id_fkey foreign key (transaction_id) references public.transactions (new_id);

alter table public.validator_bonded_token_changes add column new_tx_id varchar;
update public.validator_bonded_token_changes set new_tx_id = transactions.new_id from public.transactions where validator_bonded_token_changes.transaction_id = transactions.id;
alter table public.validator_bonded_token_changes drop column transaction_id;
alter table public.validator_bonded_token_changes rename column new_tx_id to transaction_id;
alter table public.validator_bonded_token_changes add constraint validator_bonded_token_changes_transaction_id_fkey foreign key (transaction_id) references public.transactions (new_id);

alter table public.collection_transactions add column new_tx_id varchar;
update public.collection_transactions set new_tx_id = transactions.new_id from public.transactions where collection_transactions.tx_id = transactions.id;
alter table public.collection_transactions drop column tx_id;
alter table public.collection_transactions rename column new_tx_id to tx_id;
alter table public.collection_transactions add constraint collection_transactions_tx_id_fkey foreign key (tx_id) references public.transactions (new_id);

alter table public.module_transactions add column new_tx_id varchar;
update public.module_transactions set new_tx_id = transactions.new_id from public.transactions where module_transactions.tx_id = transactions.id;
alter table public.module_transactions drop column tx_id;
alter table public.module_transactions rename column new_tx_id to tx_id;
alter table public.module_transactions add constraint module_transactions_tx_id_fkey foreign key (tx_id) references public.transactions (new_id);

alter table public.nft_transactions add column new_tx_id varchar;
update public.nft_transactions set new_tx_id = transactions.new_id from public.transactions where nft_transactions.tx_id = transactions.id;
alter table public.nft_transactions drop column tx_id;
alter table public.nft_transactions rename column new_tx_id to tx_id;
alter table public.nft_transactions add constraint nft_transactions_tx_id_fkey foreign key (tx_id) references public.transactions (new_id);

alter table public.opinit_transactions add column new_tx_id varchar;
update public.opinit_transactions set new_tx_id = transactions.new_id from public.transactions where opinit_transactions.tx_id = transactions.id;
alter table public.opinit_transactions drop column tx_id;
alter table public.opinit_transactions rename column new_tx_id to tx_id;
alter table public.opinit_transactions add constraint opinit_transactions_tx_id_fkey foreign key (tx_id) references public.transactions (new_id);

alter table public.module_histories add column new_tx_id varchar;
update public.module_histories set new_tx_id = transactions.new_id from public.transactions where module_histories.tx_id = transactions.id;
alter table public.module_histories drop column tx_id;
alter table public.module_histories rename column new_tx_id to tx_id;
alter table public.module_histories add constraint module_histories_tx_id_fkey foreign key (tx_id) references public.transactions (new_id);

alter table public.nfts add column new_tx_id varchar;
update public.nfts set new_tx_id = transactions.new_id from public.transactions where nfts.tx_id = transactions.id;
alter table public.nfts drop column tx_id;
alter table public.nfts rename column new_tx_id to tx_id;
alter table public.nfts add constraint nfts_tx_id_fkey foreign key (tx_id) references public.transactions (new_id);

alter table public.nft_histories add column new_tx_id varchar;
update public.nft_histories set new_tx_id = transactions.new_id from public.transactions where nft_histories.tx_id = transactions.id;
alter table public.nft_histories drop column tx_id;
alter table public.nft_histories rename column new_tx_id to tx_id;
alter table public.nft_histories add constraint nft_histories_tx_id_fkey foreign key (tx_id) references public.transactions (new_id);

alter table public.nft_mutation_events add column new_tx_id varchar;
update public.nft_mutation_events set new_tx_id = transactions.new_id from public.transactions where nft_mutation_events.tx_id = transactions.id;
alter table public.nft_mutation_events drop column tx_id;
alter table public.nft_mutation_events rename column new_tx_id to tx_id;
alter table public.nft_mutation_events add constraint nft_mutation_events_tx_id_fkey foreign key (tx_id) references public.transactions (new_id);

alter table public.collection_mutation_events add column new_tx_id varchar;
update public.collection_mutation_events set new_tx_id = transactions.new_id from public.transactions where collection_mutation_events.tx_id = transactions.id;
alter table public.collection_mutation_events drop column tx_id;
alter table public.collection_mutation_events rename column new_tx_id to tx_id;
alter table public.collection_mutation_events add constraint collection_mutation_events_tx_id_fkey foreign key (tx_id) references public.transactions (new_id);

alter table public.modules add column new_tx_id varchar;
update public.modules set new_tx_id = transactions.new_id from public.transactions where modules.publish_tx_id = transactions.id;
alter table public.modules drop column publish_tx_id;
alter table public.modules rename column new_tx_id to publish_tx_id;
alter table public.modules add constraint modules_publish_tx_id_fkey foreign key (publish_tx_id) references public.transactions (new_id);

-- 3
alter table public.transactions drop column id;
alter table public.transactions rename column new_id to id;
