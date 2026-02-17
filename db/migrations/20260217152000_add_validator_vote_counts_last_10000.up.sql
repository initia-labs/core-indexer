-- Add last_10000 column to validator_vote_counts table to store pre-calculated vote counts for last 10,000 blocks
-- This avoids expensive on-demand counting queries in GetValidatorBlockStats
ALTER TABLE validator_vote_counts ADD COLUMN last_10000 integer NOT NULL DEFAULT 0;
