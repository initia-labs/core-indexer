ALTER TABLE public.validators DROP COLUMN is_active;
ALTER TABLE public.validators DROP COLUMN consensus_pubkey;

ALTER TABLE public.validators ADD COLUMN id integer;

DROP TABLE IF EXISTS public.validator_commit_signatures;
DROP TYPE IF EXISTS public.commit_signature_type;

DROP TABLE IF EXISTS public.validator_vote_counts;

DROP TABLE IF EXISTS public.validator_historical_powers;

DROP TABLE IF EXISTS public.validator_bonded_token_changes;

DROP TABLE IF EXISTS public.validator_slash_events;
