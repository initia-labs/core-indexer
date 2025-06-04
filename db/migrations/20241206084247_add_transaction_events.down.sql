REVOKE SELECT ON public.transaction_events FROM readonly;
REVOKE SELECT ON public.finalize_block_events FROM readonly;

DROP TABLE IF EXISTS public.transaction_events CASCADE;
DROP TABLE IF EXISTS public.finalize_block_events CASCADE;
DROP TYPE IF EXISTS public.finalize_block_events_mode CASCADE;