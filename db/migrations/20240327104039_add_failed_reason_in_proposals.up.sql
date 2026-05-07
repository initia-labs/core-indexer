ALTER TABLE public.proposals ADD COLUMN failed_reason varchar default '';
ALTER TABLE public.proposals ALTER COLUMN failed_reason set not null;
