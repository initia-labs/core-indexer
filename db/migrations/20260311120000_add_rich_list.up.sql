-- RichList: top token holders per denom (id = rank, denom, amount)
CREATE TABLE IF NOT EXISTS public.rich_list (
    id     bigint       NOT NULL,
    denom  text         NOT NULL,
    amount numeric      NOT NULL,
    PRIMARY KEY (id, denom)
);

CREATE INDEX IF NOT EXISTS rich_list_denom_amount ON public.rich_list (denom, amount DESC);

GRANT SELECT ON public.rich_list TO readonly;

-- RichListStatus: single row storing last indexed height
CREATE TABLE IF NOT EXISTS public.rich_list_status (
    height bigint NOT NULL PRIMARY KEY
);

GRANT SELECT ON public.rich_list_status TO readonly;
