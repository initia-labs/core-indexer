ALTER TABLE public.proposals DROP COLUMN IF EXISTS emergency_next_tally_time;
ALTER TABLE public.proposals DROP COLUMN IF EXISTS emergency_start_time;
ALTER TABLE public.proposals DROP COLUMN IF EXISTS is_emergency;
