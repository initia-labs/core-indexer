-- vm_address
alter table public.accounts add column new_vm_address_id varchar;
update public.accounts set new_vm_address_id = vm_addresses.vm_address from public.vm_addresses where accounts.vm_address_id = vm_addresses.id;
alter table public.accounts drop column vm_address_id;
alter table public.accounts rename column new_vm_address_id to vm_address_id;
alter table public.accounts add constraint accounts_vm_address_id_fkey foreign key (vm_address_id) references public.vm_addresses (vm_address);

alter table public.collections add column new_vm_address_id varchar;
update public.collections set new_vm_address_id = vm_addresses.vm_address from public.vm_addresses where collections.collection = vm_addresses.id;
alter table public.collections drop column collection;
alter table public.collections rename column new_vm_address_id to collection;
alter table public.collections add constraint collections_collection_fkey foreign key (collection) references public.vm_addresses (vm_address);

alter table public.collections add column new_vm_address_id varchar;
update public.collections set new_vm_address_id = vm_addresses.vm_address from public.vm_addresses where collections.creator = vm_addresses.id;
alter table public.collections drop column creator;
alter table public.collections rename column new_vm_address_id to creator;
alter table public.collections add constraint collections_creator_fkey foreign key (creator) references public.vm_addresses (vm_address);

alter table public.modules add column new_vm_address_id varchar;
update public.modules set new_vm_address_id = vm_addresses.vm_address from public.vm_addresses where modules.publisher_id = vm_addresses.id;
alter table public.modules drop column publisher_id;
alter table public.modules rename column new_vm_address_id to publisher_id;
alter table public.modules add constraint modules_publisher_id_fkey foreign key (publisher_id) references public.vm_addresses (vm_address);

alter table public.nft_histories add column new_vm_address_id varchar;
update public.nft_histories set new_vm_address_id = vm_addresses.vm_address from public.vm_addresses where nft_histories."from" = vm_addresses.id;
alter table public.nft_histories drop column "from";
alter table public.nft_histories rename column new_vm_address_id to "from";
alter table public.nft_histories add constraint nft_histories_from_fkey foreign key ("from") references public.vm_addresses (vm_address);

alter table public.nft_histories add column new_vm_address_id varchar;
update public.nft_histories set new_vm_address_id = vm_addresses.vm_address from public.vm_addresses where nft_histories."to" = vm_addresses.id;
alter table public.nft_histories drop column "to";
alter table public.nft_histories rename column new_vm_address_id to "to";
alter table public.nft_histories add constraint nft_histories_to_fkey foreign key ("to") references public.vm_addresses (vm_address);

alter table public.nfts add column new_vm_address_id varchar;
update public.nfts set new_vm_address_id = vm_addresses.vm_address from public.vm_addresses where nfts."owner" = vm_addresses.id;
alter table public.nfts drop column "owner";
alter table public.nfts rename column new_vm_address_id to "owner";
alter table public.nfts add constraint nfts_owner_fkey foreign key ("owner") references public.vm_addresses (vm_address);

alter table public.nfts add column new_vm_address_id varchar;
update public.nfts set new_vm_address_id = vm_addresses.vm_address from public.vm_addresses where nfts.nft = vm_addresses.id;
alter table public.nfts drop column nft;
alter table public.nfts rename column new_vm_address_id to nft;
alter table public.nfts add constraint nfts_nft_fkey foreign key (nft) references public.vm_addresses (vm_address);

alter table public.vm_addresses drop column id;
alter table public.vm_addresses add primary key (vm_address);

-- collection 
alter table public.collection_mutation_events drop constraint collection_mutation_events_collection_id_fkey;
alter table public.collection_proposals drop constraint collection_proposals_collection_id_fkey;
alter table public.collection_transactions drop constraint collection_transactions_collection_id_fkey;
alter table public.nfts drop constraint nfts_collection_fkey;

alter table public.collections drop constraint collections_pkey;
alter table public.collections add primary key (collection);

alter table public.collection_mutation_events add column new_collection_id varchar;
update public.collection_mutation_events set new_collection_id = collections.collection from public.collections where collection_mutation_events.collection_id = collections.id;
alter table public.collection_mutation_events drop column collection_id;
alter table public.collection_mutation_events rename column new_collection_id to collection_id;
alter table public.collection_mutation_events add constraint collection_mutation_events_collection_id_fkey foreign key (collection_id) references public.collections (collection);

alter table public.collection_proposals add column new_collection_id varchar;
update public.collection_proposals set new_collection_id = collections.collection from public.collections where collection_proposals.collection_id = collections.id;
alter table public.collection_proposals drop column collection_id;
alter table public.collection_proposals rename column new_collection_id to collection_id;
alter table public.collection_proposals add constraint collection_proposals_collection_id_fkey foreign key (collection_id) references public.collections (collection);

alter table public.collection_transactions add column new_collection_id varchar;
update public.collection_transactions set new_collection_id = collections.collection from public.collections where collection_transactions.collection_id = collections.id;
alter table public.collection_transactions drop column collection_id;
alter table public.collection_transactions rename column new_collection_id to collection_id;
alter table public.collection_transactions add constraint collection_transactions_collection_id_fkey foreign key (collection_id) references public.collections (collection);

alter table public.nfts add column new_collection_id varchar;
update public.nfts set new_collection_id = collections.collection from public.collections where nfts.collection = collections.id;
alter table public.nfts drop column collection;
alter table public.nfts rename column new_collection_id to collection;
alter table public.nfts add constraint nfts_collection_fkey foreign key (collection) references public.collections (collection);

alter table public.collections drop column id;
alter table public.collections rename column collection to id;

-- module
alter table public.module_histories drop constraint module_histories_module_id_fkey;
alter table public.module_proposals drop constraint module_proposals_module_id_fkey;
alter table public.module_transactions drop constraint module_transactions_module_id_fkey;

alter table public.modules add column new_id varchar;
update public.modules set new_id = publisher_id || '::' || name;
alter table public.modules add primary key (new_id);

alter table public.module_histories add column new_module_id varchar;
update public.module_histories set new_module_id = modules.new_id from public.modules where module_histories.module_id = modules.id;
alter table public.module_histories drop column module_id;
alter table public.module_histories rename column new_module_id to module_id;
alter table public.module_histories add constraint module_histories_module_id_fkey foreign key (module_id) references public.modules (new_id);

alter table public.module_proposals add column new_module_id varchar;
update public.module_proposals set new_module_id = modules.new_id from public.modules where module_proposals.module_id = modules.id;
alter table public.module_proposals drop column module_id;
alter table public.module_proposals rename column new_module_id to module_id;
alter table public.module_proposals add constraint module_proposals_module_id_fkey foreign key (module_id) references public.modules (new_id);

alter table public.module_transactions add column new_module_id varchar;
update public.module_transactions set new_module_id = modules.new_id from public.modules where module_transactions.module_id = modules.id;
alter table public.module_transactions drop column module_id;
alter table public.module_transactions rename column new_module_id to module_id;
alter table public.module_transactions add constraint module_transactions_module_id_fkey foreign key (module_id) references public.modules (new_id);

alter table public.modules drop column id;
alter table public.modules rename column new_id to id;

-- nft
alter table public.collection_proposals drop constraint collection_proposals_nft_id_fkey;
alter table public.collection_transactions drop constraint collection_transactions_nft_id_fkey;
alter table public.nft_histories drop constraint nft_histories_nft_id_fkey;
alter table public.nft_mutation_events drop constraint nft_mutation_events_nft_id_fkey;
alter table public.nft_proposals drop constraint nft_proposals_nft_id_fkey;
alter table public.nft_transactions drop constraint nft_transactions_nft_id_fkey;

alter table public.nfts drop constraint nfts_pkey;
alter table public.nfts add primary key (nft);

alter table public.collection_proposals add column new_nft_id varchar;
update public.collection_proposals set new_nft_id = nfts.nft from public.nfts where collection_proposals.nft_id = nfts.id;
alter table public.collection_proposals drop column nft_id;
alter table public.collection_proposals rename column new_nft_id to nft_id;
alter table public.collection_proposals add constraint collection_proposals_nft_id_fkey foreign key (nft_id) references public.nfts (nft);

alter table public.collection_transactions add column new_nft_id varchar;
update public.collection_transactions set new_nft_id = nfts.nft from public.nfts where collection_transactions.nft_id = nfts.id;
alter table public.collection_transactions drop column nft_id;
alter table public.collection_transactions rename column new_nft_id to nft_id;
alter table public.collection_transactions add constraint collection_transactions_nft_id_fkey foreign key (nft_id) references public.nfts (nft);

alter table public.nft_histories add column new_nft_id varchar;
update public.nft_histories set new_nft_id = nfts.nft from public.nfts where nft_histories.nft_id = nfts.id;
alter table public.nft_histories drop column nft_id;
alter table public.nft_histories rename column new_nft_id to nft_id;
alter table public.nft_histories add constraint nft_histories_nft_id_fkey foreign key (nft_id) references public.nfts (nft);

alter table public.nft_mutation_events add column new_nft_id varchar;
update public.nft_mutation_events set new_nft_id = nfts.nft from public.nfts where nft_mutation_events.nft_id = nfts.id;
alter table public.nft_mutation_events drop column nft_id;
alter table public.nft_mutation_events rename column new_nft_id to nft_id;
alter table public.nft_mutation_events add constraint nft_mutation_events_nft_id_fkey foreign key (nft_id) references public.nfts (nft);

alter table public.nft_proposals add column new_nft_id varchar;
update public.nft_proposals set new_nft_id = nfts.nft from public.nfts where nft_proposals.nft_id = nfts.id;
alter table public.nft_proposals drop column nft_id;
alter table public.nft_proposals rename column new_nft_id to nft_id;
alter table public.nft_proposals add constraint nft_proposals_nft_id_fkey foreign key (nft_id) references public.nfts (nft);

alter table public.nft_transactions add column new_nft_id varchar;
update public.nft_transactions set new_nft_id = nfts.nft from public.nfts where nft_transactions.nft_id = nfts.id;
alter table public.nft_transactions drop column nft_id;
alter table public.nft_transactions rename column new_nft_id to nft_id;
alter table public.nft_transactions add constraint nft_transactions_nft_id_fkey foreign key (nft_id) references public.nfts (nft);

alter table public.nfts drop column id;
alter table public.nfts rename column nft to id;
