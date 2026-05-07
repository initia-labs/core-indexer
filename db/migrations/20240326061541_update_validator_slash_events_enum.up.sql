CREATE TYPE public.slashtype AS ENUM ('Jailed', 'Slashed', 'Unjailed');
ALTER TYPE public.slashtype OWNER TO postgres;

ALTER TABLE public.validator_slash_events ADD COLUMN type public.slashtype;

ALTER TABLE public.validator_slash_events DROP COLUMN is_jailed;
