alter table public.account_transactions add column new_account_id varchar;
update public.account_transactions set new_account_id = accounts.address from public.accounts where account_transactions.account_id = accounts.id;
alter table public.account_transactions drop column account_id;
alter table public.account_transactions rename column new_account_id to account_id;
alter table public.account_transactions add constraint account_transactions_account_id_fkey foreign key (account_id) references public.accounts (address);

alter table public.validators add column new_account_id varchar;
update public.validators set new_account_id = accounts.address from public.accounts where validators.account_id = accounts.id;
alter table public.validators drop column account_id;
alter table public.validators rename column new_account_id to account_id;
alter table public.validators add constraint validators_account_id_fkey foreign key (account_id) references public.accounts (address);

alter table public.proposal_deposits add column new_account_id varchar;
update public.proposal_deposits set new_account_id = accounts.address from public.accounts where proposal_deposits.depositor = accounts.id;
alter table public.proposal_deposits drop column depositor;
alter table public.proposal_deposits rename column new_account_id to depositor;
alter table public.proposal_deposits add constraint proposal_deposits_depositor_fkey foreign key (depositor) references public.accounts (address);

alter table public.proposal_votes_legacy add column new_account_id varchar;
update public.proposal_votes_legacy set new_account_id = accounts.address from public.accounts where proposal_votes_legacy.voter = accounts.id;
alter table public.proposal_votes_legacy drop column voter;
alter table public.proposal_votes_legacy rename column new_account_id to voter;
alter table public.proposal_votes_legacy add constraint proposal_votes_legacy_voter_fkey foreign key (voter) references public.accounts (address);

alter table public.proposal_votes add column new_account_id varchar;
update public.proposal_votes set new_account_id = accounts.address from public.accounts where proposal_votes.voter = accounts.id;
alter table public.proposal_votes drop column voter;
alter table public.proposal_votes rename column new_account_id to voter;
alter table public.proposal_votes add constraint proposal_votes_voter_fkey foreign key (voter) references public.accounts (address);

alter table public.proposals add column new_account_id varchar;
update public.proposals set new_account_id = accounts.address from public.accounts where proposals.proposer_id = accounts.id;
alter table public.proposals drop column proposer_id;
alter table public.proposals rename column new_account_id to proposer_id;
alter table public.proposals add constraint proposals_proposer_id_fkey foreign key (proposer_id) references public.accounts (address);

alter table public.transactions add column new_account_id varchar;
update public.transactions set new_account_id = accounts.address from public.accounts where transactions.sender = accounts.id;
alter table public.transactions drop column sender;
alter table public.transactions rename column new_account_id to sender;
alter table public.transactions add constraint transactions_sender_fkey foreign key (sender) references public.accounts (address);

alter table public.accounts drop column id;
