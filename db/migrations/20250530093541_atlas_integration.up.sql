CREATE TABLE IF NOT EXISTS "public"."schema_migrations" ("version" bigserial NOT NULL, "dirty" boolean NOT NULL, PRIMARY KEY ("version"));

ALTER TABLE "public"."accounts" RENAME CONSTRAINT "accounts_vm_address_id_fkey" TO "fk_accounts_vm_address";

ALTER TABLE "public"."account_transactions" RENAME CONSTRAINT "account_transactions_account_id_fkey" TO "fk_account_transactions_account";
ALTER TABLE "public"."account_transactions" RENAME CONSTRAINT "account_transactions_block_height_fkey" TO "fk_account_transactions_block";
ALTER TABLE "public"."account_transactions" RENAME CONSTRAINT "account_transactions_transaction_id_fkey" TO "fk_account_transactions_transaction";

ALTER TABLE "public"."blocks" RENAME CONSTRAINT "blocks_proposer_fkey" TO "fk_blocks_proposer_validator";

ALTER TABLE "public"."collections" RENAME CONSTRAINT "collections_block_height_fkey" TO "fk_collections_block";
ALTER TABLE "public"."collections" RENAME CONSTRAINT "collections_collection_fkey" TO "fk_collections_collection_addr";
ALTER TABLE "public"."collections" RENAME CONSTRAINT "collections_creator_fkey" TO "fk_collections_creator_addr";

ALTER TABLE "public"."collection_mutation_events" RENAME CONSTRAINT "collection_mutation_events_block_height_fkey" TO "fk_collection_mutation_events_block";
ALTER TABLE "public"."collection_mutation_events" RENAME CONSTRAINT "collection_mutation_events_collection_id_fkey" TO "fk_collection_mutation_events_collection";
ALTER TABLE "public"."collection_mutation_events" RENAME CONSTRAINT "collection_mutation_events_proposal_id_fkey" TO "fk_collection_mutation_events_proposal";
ALTER TABLE "public"."collection_mutation_events" RENAME CONSTRAINT "collection_mutation_events_tx_id_fkey" TO "fk_collection_mutation_events_transaction";

ALTER TABLE "public"."collection_proposals" RENAME CONSTRAINT "collection_proposals_collection_id_fkey" TO "fk_collection_proposals_collection";
ALTER TABLE "public"."collection_proposals" RENAME CONSTRAINT "collection_proposals_nft_id_fkey" TO "fk_collection_proposals_nft";
ALTER TABLE "public"."collection_proposals" RENAME CONSTRAINT "collection_proposals_proposal_id_fkey" TO "fk_collection_proposals_proposal";

ALTER TABLE "public"."collection_transactions" RENAME CONSTRAINT "collection_transactions_block_height_fkey" TO "fk_collection_transactions_block";
ALTER TABLE "public"."collection_transactions" RENAME CONSTRAINT "collection_transactions_collection_id_fkey" TO "fk_collection_transactions_collection";
ALTER TABLE "public"."collection_transactions" RENAME CONSTRAINT "collection_transactions_nft_id_fkey" TO "fk_collection_transactions_nft";
ALTER TABLE "public"."collection_transactions" RENAME CONSTRAINT "collection_transactions_tx_id_fkey" TO "fk_collection_transactions_transaction";

ALTER TABLE "public"."lcd_tx_results" RENAME CONSTRAINT "lcd_tx_results_block_height_fkey" TO "fk_lcd_tx_results_block";
ALTER TABLE "public"."lcd_tx_results" RENAME CONSTRAINT "lcd_tx_results_transaction_id_fkey" TO "fk_lcd_tx_results_transaction";

ALTER TABLE "public"."modules" RENAME CONSTRAINT "modules_publish_tx_id_fkey" TO "fk_modules_publish_tx";
ALTER TABLE "public"."modules" RENAME CONSTRAINT "modules_publisher_id_fkey" TO "fk_modules_publisher";

ALTER TABLE "public"."module_histories" RENAME CONSTRAINT "module_histories_block_height_fkey" TO "fk_module_histories_block";
ALTER TABLE "public"."module_histories" RENAME CONSTRAINT "module_histories_module_id_fkey" TO "fk_module_histories_module";
ALTER TABLE "public"."module_histories" RENAME CONSTRAINT "module_histories_proposal_id_fkey" TO "fk_module_histories_proposal";
ALTER TABLE "public"."module_histories" RENAME CONSTRAINT "module_histories_tx_id_fkey" TO "fk_module_histories_transaction";

ALTER TABLE "public"."module_proposals" RENAME CONSTRAINT "module_proposals_module_id_fkey" TO "fk_module_proposals_module";
ALTER TABLE "public"."module_proposals" RENAME CONSTRAINT "module_proposals_proposal_id_fkey" TO "fk_module_proposals_proposal";

ALTER TABLE "public"."module_transactions" RENAME CONSTRAINT "module_transactions_block_height_fkey" TO "fk_module_transactions_block";
ALTER TABLE "public"."module_transactions" RENAME CONSTRAINT "module_transactions_module_id_fkey" TO "fk_module_transactions_module";
ALTER TABLE "public"."module_transactions" RENAME CONSTRAINT "module_transactions_tx_id_fkey" TO "fk_module_transactions_transaction";

ALTER TABLE "public"."nfts" RENAME CONSTRAINT "nfts_collection_fkey" TO "fk_nfts_collection_ref";
ALTER TABLE "public"."nfts" RENAME CONSTRAINT "nfts_nft_fkey" TO "fk_nfts_nft_addr";
ALTER TABLE "public"."nfts" RENAME CONSTRAINT "nfts_owner_fkey" TO "fk_nfts_owner_addr";
ALTER TABLE "public"."nfts" RENAME CONSTRAINT "nfts_proposal_id_fkey" TO "fk_nfts_proposal";
ALTER TABLE "public"."nfts" RENAME CONSTRAINT "nfts_tx_id_fkey" TO "fk_nfts_tx_ref";

ALTER TABLE "public"."nft_histories" RENAME CONSTRAINT "nft_histories_block_height_fkey" TO "fk_nft_histories_block";
ALTER TABLE "public"."nft_histories" RENAME CONSTRAINT "nft_histories_from_fkey" TO "fk_nft_histories_from_addr";
ALTER TABLE "public"."nft_histories" RENAME CONSTRAINT "nft_histories_nft_id_fkey" TO "fk_nft_histories_nft";
ALTER TABLE "public"."nft_histories" RENAME CONSTRAINT "nft_histories_proposal_id_fkey" TO "fk_nft_histories_proposal";
ALTER TABLE "public"."nft_histories" RENAME CONSTRAINT "nft_histories_to_fkey" TO "fk_nft_histories_to_addr";
ALTER TABLE "public"."nft_histories" RENAME CONSTRAINT "nft_histories_tx_id_fkey" TO "fk_nft_histories_transaction";

ALTER TABLE "public"."nft_mutation_events" RENAME CONSTRAINT "nft_mutation_events_block_height_fkey" TO "fk_nft_mutation_events_block";
ALTER TABLE "public"."nft_mutation_events" RENAME CONSTRAINT "nft_mutation_events_nft_id_fkey" TO "fk_nft_mutation_events_nft";
ALTER TABLE "public"."nft_mutation_events" RENAME CONSTRAINT "nft_mutation_events_proposal_id_fkey" TO "fk_nft_mutation_events_proposal";
ALTER TABLE "public"."nft_mutation_events" RENAME CONSTRAINT "nft_mutation_events_tx_id_fkey" TO "fk_nft_mutation_events_transaction";

ALTER TABLE "public"."nft_proposals" RENAME CONSTRAINT "nft_proposals_nft_id_fkey" TO "fk_nft_proposals_nft";
ALTER TABLE "public"."nft_proposals" RENAME CONSTRAINT "nft_proposals_proposal_id_fkey" TO "fk_nft_proposals_proposal";

ALTER TABLE "public"."nft_transactions" RENAME CONSTRAINT "nft_transactions_block_height_fkey" TO "fk_nft_transactions_block";
ALTER TABLE "public"."nft_transactions" RENAME CONSTRAINT "nft_transactions_nft_id_fkey" TO "fk_nft_transactions_nft";
ALTER TABLE "public"."nft_transactions" RENAME CONSTRAINT "nft_transactions_tx_id_fkey" TO "fk_nft_transactions_transaction";

ALTER TABLE "public"."opinit_transactions" RENAME CONSTRAINT "opinit_transactions_block_height_fkey" TO "fk_opinit_transactions_block";
ALTER TABLE "public"."opinit_transactions" RENAME CONSTRAINT "opinit_transactions_tx_id_fkey" TO "fk_opinit_transactions_transaction";

ALTER TABLE "public"."proposals" RENAME CONSTRAINT "proposals_created_height_fkey" TO "fk_proposals_created_block";
ALTER TABLE "public"."proposals" RENAME CONSTRAINT "proposals_created_tx_fkey" TO "fk_proposals_created_tx_ref";
ALTER TABLE "public"."proposals" RENAME CONSTRAINT "proposals_proposer_id_fkey" TO "fk_proposals_proposer";
ALTER TABLE "public"."proposals" RENAME CONSTRAINT "proposals_resolved_height_fkey" TO "fk_proposals_resolved_block";

ALTER TABLE "public"."proposal_deposits" RENAME CONSTRAINT "proposal_deposits_depositor_fkey" TO "fk_proposal_deposits_depositor_account";
ALTER TABLE "public"."proposal_deposits" RENAME CONSTRAINT "proposal_deposits_proposal_id_fkey" TO "fk_proposal_deposits_proposal";
ALTER TABLE "public"."proposal_deposits" RENAME CONSTRAINT "proposal_deposits_transaction_id_fkey" TO "fk_proposal_deposits_transaction";

ALTER TABLE "public"."proposal_votes" RENAME CONSTRAINT "proposal_votes_proposal_id_fkey" TO "fk_proposal_votes_proposal";
ALTER TABLE "public"."proposal_votes" RENAME CONSTRAINT "proposal_votes_transaction_id_fkey" TO "fk_proposal_votes_transaction";
ALTER TABLE "public"."proposal_votes" RENAME CONSTRAINT "proposal_votes_validator_address_fkey" TO "fk_proposal_votes_validator";
ALTER TABLE "public"."proposal_votes" RENAME CONSTRAINT "proposal_votes_voter_fkey" TO "fk_proposal_votes_voter_account";

ALTER TABLE "public"."proposal_votes_legacy" RENAME CONSTRAINT "proposal_votes_legacy_transaction_id_fkey" TO "fk_proposal_votes_legacy_transaction";
ALTER TABLE "public"."proposal_votes_legacy" RENAME CONSTRAINT "proposal_votes_legacy_voter_fkey" TO "fk_proposal_votes_legacy_voter_account";
ALTER TABLE "public"."proposal_votes_legacy" RENAME CONSTRAINT "proposal_votes_proposal_id_fkey" TO "fk_proposal_votes_legacy_proposal";
ALTER TABLE "public"."proposal_votes_legacy" RENAME CONSTRAINT "proposal_votes_validator_address_fkey" TO "fk_proposal_votes_legacy_validator";

-- should be bigint anyway
ALTER TABLE "public"."transactions" ALTER COLUMN "block_height" TYPE bigint;

ALTER TABLE "public"."transactions" RENAME CONSTRAINT "transactions_block_height_fkey" TO "fk_transactions_block";
ALTER TABLE "public"."transactions" RENAME CONSTRAINT "transactions_sender_fkey" TO "fk_transactions_sender_account";

ALTER TABLE "public"."validators" RENAME CONSTRAINT "validators_account_id_fkey" TO "fk_validators_account";

ALTER TABLE "public"."validator_bonded_token_changes" RENAME CONSTRAINT "validator_bonded_token_changes_block_height_fkey" TO "fk_validator_bonded_token_changes_block";
ALTER TABLE "public"."validator_bonded_token_changes" RENAME CONSTRAINT "validator_bonded_token_changes_transaction_id_fkey" TO "fk_validator_bonded_token_changes_transaction";
ALTER TABLE "public"."validator_bonded_token_changes" RENAME CONSTRAINT "validator_bonded_token_changes_validator_address_fkey" TO "fk_validator_bonded_token_changes_validator";

ALTER TABLE "public"."validator_commit_signatures" RENAME CONSTRAINT "validator_commit_signatures_validator_address_fkey" TO "fk_validator_commit_signatures_validator";

ALTER TABLE "public"."validator_historical_powers" RENAME CONSTRAINT "validator_historical_powers_validator_address_fkey" TO "fk_validator_historical_powers_validator";

ALTER TABLE "public"."validator_slash_events" RENAME CONSTRAINT "validator_slash_events_block_height_fkey" TO "fk_validator_slash_events_block";
ALTER TABLE "public"."validator_slash_events" RENAME CONSTRAINT "validator_slash_events_validator_address_fkey" TO "fk_validator_slash_events_validator";

ALTER TABLE "public"."validator_vote_counts" RENAME CONSTRAINT "validator_vote_counts_validator_address_fkey" TO "fk_validator_vote_counts_validator";

-- PRIMARY KEYS ORDER

ALTER TABLE "public"."move_events" DROP CONSTRAINT "move_events_pkey", ADD PRIMARY KEY ("block_height", "transaction_hash", "event_index");

ALTER TABLE "public"."transaction_events" DROP CONSTRAINT "transaction_events_pkey", ADD PRIMARY KEY ("block_height", "transaction_hash", "event_index");
