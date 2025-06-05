ALTER TABLE public.proposals ADD is_emergency BOOL DEFAULT FALSE NOT NULL;
ALTER TABLE public.proposals ADD emergency_start_time TIMESTAMP;
ALTER TABLE public.proposals ADD emergency_next_tally_time TIMESTAMP;
