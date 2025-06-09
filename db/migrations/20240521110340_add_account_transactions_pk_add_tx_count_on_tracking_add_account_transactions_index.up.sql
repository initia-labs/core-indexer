alter table public.account_transactions add primary key (transaction_id, account_id);
create index ix_module_histories_module_id on public.module_histories (module_id);

alter table public.tracking add column tx_count bigint not null default 0;

create index ix_account_transactions_account_id_block_height_desc on public.account_transactions(account_id, block_height desc);
