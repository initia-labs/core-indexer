ALTER TABLE public.validator_slash_events ADD COLUMN is_jailed boolean;

ALTER TABLE public.validator_slash_events DROP COLUMN type;
DROP TYPE slashtype;
