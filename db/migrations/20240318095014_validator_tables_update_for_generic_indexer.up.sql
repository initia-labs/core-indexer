ALTER TABLE public.validators ADD COLUMN is_active boolean;
ALTER TABLE public.validators ADD COLUMN consensus_pubkey varchar;

ALTER TABLE public.validators DROP COLUMN id;

CREATE TYPE public.commit_signature_type AS ENUM (
    'PROPOSE',
    'VOTE',
    'ABSENT'
);
CREATE TABLE IF NOT EXISTS public.validator_commit_signatures (
    validator_address varchar REFERENCES public.validators (operator_address) ,
    block_height bigint NOT NULL,
    vote public.commit_signature_type NOT NULL,
    PRIMARY KEY(validator_address, block_height)
);
GRANT SELECT ON public.validator_commit_signatures TO readonly;

CREATE TABLE IF NOT EXISTS public.validator_vote_counts (
    validator_address varchar REFERENCES public.validators (operator_address) PRIMARY KEY,
    last_100 integer NOT NULL
);
GRANT SELECT ON public.validator_vote_counts TO readonly;

CREATE TABLE IF NOT EXISTS public.validator_historical_powers (
      validator_address varchar REFERENCES public.validators (operator_address),
      tokens json NOT NULL,
      voting_power bigint NOT NULL,
      hour_rounded_timestamp timestamp NOT NULL,
      timestamp timestamp NOT NULL,
      CONSTRAINT unique_validator_historicail_power UNIQUE (validator_address, hour_rounded_timestamp)
);
GRANT SELECT ON public.validator_historical_powers TO readonly;

CREATE TABLE IF NOT EXISTS public.validator_bonded_token_changes (
     validator_address varchar NOT NULL REFERENCES public.validators (operator_address),
     transaction_id bigint NOT NULL REFERENCES public.transactions (id),
     block_height bigint NOT NULL REFERENCES public.blocks (height),
     tokens json NOT NULL
);
GRANT SELECT ON public.validator_bonded_token_changes TO readonly;

CREATE TABLE IF NOT EXISTS public.validator_slash_events (
     validator_address varchar NOT NULL REFERENCES public.validators (operator_address),
     block_height bigint NOT NULL REFERENCES public.blocks (height),
     is_jailed boolean NOT NULL
);
GRANT SELECT ON public.validator_slash_events TO readonly;
