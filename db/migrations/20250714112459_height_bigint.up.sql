-- Modify "blocks" table
ALTER TABLE "public"."blocks" ALTER COLUMN "height" TYPE bigint;
-- Modify "collection_mutation_events" table
ALTER TABLE "public"."collection_mutation_events" ALTER COLUMN "block_height" TYPE bigint;
-- Modify "collection_transactions" table
ALTER TABLE "public"."collection_transactions" ALTER COLUMN "block_height" TYPE bigint;
-- Modify "collections" table
ALTER TABLE "public"."collections" ALTER COLUMN "block_height" TYPE bigint;
-- Modify "lcd_tx_results" table
ALTER TABLE "public"."lcd_tx_results" ALTER COLUMN "block_height" TYPE bigint;
-- Modify "module_histories" table
ALTER TABLE "public"."module_histories" ALTER COLUMN "block_height" TYPE bigint;
-- Modify "module_transactions" table
ALTER TABLE "public"."module_transactions" ALTER COLUMN "block_height" TYPE bigint;
-- Modify "nft_histories" table
ALTER TABLE "public"."nft_histories" ALTER COLUMN "block_height" TYPE bigint;
-- Modify "nft_mutation_events" table
ALTER TABLE "public"."nft_mutation_events" ALTER COLUMN "block_height" TYPE bigint;
-- Modify "nft_transactions" table
ALTER TABLE "public"."nft_transactions" ALTER COLUMN "block_height" TYPE bigint;
-- Modify "opinit_transactions" table
ALTER TABLE "public"."opinit_transactions" ALTER COLUMN "block_height" TYPE bigint;
-- Modify "proposals" table
ALTER TABLE "public"."proposals" ALTER COLUMN "resolved_height" TYPE bigint, ALTER COLUMN "created_height" TYPE bigint;
-- Modify "tracking" table
ALTER TABLE "public"."tracking" ALTER COLUMN "latest_informative_block_height" TYPE bigint;
