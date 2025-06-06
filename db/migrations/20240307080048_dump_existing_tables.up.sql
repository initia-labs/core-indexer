--
-- PostgreSQL database dump
--

-- Dumped from database version 15.5
-- Dumped by pg_dump version 15.6 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: accounttype; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.accounttype AS ENUM (
    'BaseAccount',
    'InterchainAccount',
    'ModuleAccount',
    'ContinuousVestingAccount',
    'DelayedVestingAccount',
    'ClawbackVestingAccount',
    'ContractAccount'
    );


ALTER TYPE public.accounttype OWNER TO postgres;

--
-- Name: proposalstatus; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.proposalstatus AS ENUM (
    'Nil',
    'DepositPeriod',
    'VotingPeriod',
    'Passed',
    'Rejected',
    'Failed',
    'Inactive',
    'Cancelled'
    );


ALTER TYPE public.proposalstatus OWNER TO postgres;

--
-- Name: upgradepolicy; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.upgradepolicy AS ENUM (
    'Arbitrary',
    'Compatible',
    'Immutable'
    );


ALTER TYPE public.upgradepolicy OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: proposals; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.proposals (
                                  id integer NOT NULL,
                                  proposer_id integer,
                                  type character varying NOT NULL,
                                  title character varying NOT NULL,
                                  description character varying NOT NULL,
                                  proposal_route character varying NOT NULL,
                                  status public.proposalstatus NOT NULL,
                                  submit_time timestamp without time zone NOT NULL,
                                  deposit_end_time timestamp without time zone NOT NULL,
                                  voting_time timestamp without time zone,
                                  voting_end_time timestamp without time zone,
                                  content json,
                                  total_deposit json NOT NULL,
                                  yes bigint NOT NULL,
                                  no bigint NOT NULL,
                                  abstain bigint NOT NULL,
                                  no_with_veto bigint NOT NULL,
                                  is_expedited boolean NOT NULL,
                                  version character varying NOT NULL,
                                  resolved_height integer,
                                  types json NOT NULL,
                                  messages json NOT NULL,
                                  created_tx integer,
                                  created_height integer,
                                  metadata character varying NOT NULL
);


ALTER TABLE public.proposals OWNER TO postgres;

--
-- Name: search_proposals(character varying[]); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.search_proposals(types_param character varying[] DEFAULT NULL::character varying[]) RETURNS SETOF public.proposals
    LANGUAGE sql STABLE
AS $$
SELECT *
FROM proposals
WHERE types_param IS NULL OR types_param::text[] <@ ARRAY(SELECT json_array_elements_text(types));
$$;


ALTER FUNCTION public.search_proposals(types_param character varying[]) OWNER TO postgres;

--
-- Name: account_transactions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.account_transactions (
                                             transaction_id integer NOT NULL,
                                             account_id integer NOT NULL,
                                             is_signer boolean NOT NULL,
                                             block_height integer NOT NULL
);


ALTER TABLE public.account_transactions OWNER TO postgres;

--
-- Name: accounts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.accounts (
                                 id integer NOT NULL,
                                 address character varying NOT NULL,
                                 vm_address_id integer NOT NULL,
                                 type public.accounttype,
                                 name character varying
);


ALTER TABLE public.accounts OWNER TO postgres;

--
-- Name: blocks; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.blocks (
                               height integer NOT NULL,
                               "timestamp" timestamp without time zone NOT NULL,
                               proposer character varying,
                               hash bytea NOT NULL
);


ALTER TABLE public.blocks OWNER TO postgres;

--
-- Name: blocks_height_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.blocks_height_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.blocks_height_seq OWNER TO postgres;

--
-- Name: blocks_height_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.blocks_height_seq OWNED BY public.blocks.height;


--
-- Name: collection_mutation_events; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.collection_mutation_events (
                                                   collection_id integer NOT NULL,
                                                   mutated_field_name character varying NOT NULL,
                                                   old_value character varying NOT NULL,
                                                   new_value character varying NOT NULL,
                                                   block_height integer NOT NULL,
                                                   remark json NOT NULL
);


ALTER TABLE public.collection_mutation_events OWNER TO postgres;

--
-- Name: collection_proposals; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.collection_proposals (
                                             collection_id integer NOT NULL,
                                             nft_id integer,
                                             proposal_id integer NOT NULL
);


ALTER TABLE public.collection_proposals OWNER TO postgres;

--
-- Name: collection_transactions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.collection_transactions (
                                                collection_id integer NOT NULL,
                                                nft_id integer,
                                                tx_id integer NOT NULL,
                                                is_nft_transfer boolean NOT NULL,
                                                is_nft_mint boolean NOT NULL,
                                                is_nft_burn boolean NOT NULL,
                                                is_collection_create boolean NOT NULL,
                                                block_height integer NOT NULL
);


ALTER TABLE public.collection_transactions OWNER TO postgres;

--
-- Name: collections; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.collections (
                                    id integer NOT NULL,
                                    block_height integer NOT NULL,
                                    collection integer NOT NULL,
                                    uri character varying NOT NULL,
                                    description character varying NOT NULL,
                                    creator integer NOT NULL,
                                    name character varying NOT NULL
);


ALTER TABLE public.collections OWNER TO postgres;

--
-- Name: lcd_tx_results; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.lcd_tx_results (
                                       block_height integer NOT NULL,
                                       transaction_id integer NOT NULL,
                                       result json NOT NULL
);


ALTER TABLE public.lcd_tx_results OWNER TO postgres;

--
-- Name: module_histories; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.module_histories (
                                         module_id integer NOT NULL,
                                         upgrade_policy public.upgradepolicy NOT NULL,
                                         block_height integer NOT NULL,
                                         remark json NOT NULL
);


ALTER TABLE public.module_histories OWNER TO postgres;

--
-- Name: module_proposals; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.module_proposals (
                                         module_id integer NOT NULL,
                                         proposal_id integer NOT NULL
);


ALTER TABLE public.module_proposals OWNER TO postgres;

--
-- Name: module_transactions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.module_transactions (
                                            tx_id integer NOT NULL,
                                            module_id integer NOT NULL,
                                            is_entry boolean NOT NULL,
                                            block_height integer NOT NULL
);


ALTER TABLE public.module_transactions OWNER TO postgres;

--
-- Name: modules; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.modules (
                                id integer NOT NULL,
                                publisher_id integer NOT NULL,
                                name character varying NOT NULL,
                                upgrade_policy public.upgradepolicy NOT NULL,
                                publish_tx_id integer,
                                module_entry_executed integer NOT NULL,
                                is_verify boolean NOT NULL
);


ALTER TABLE public.modules OWNER TO postgres;

--
-- Name: nft_histories; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.nft_histories (
                                      nft_id integer NOT NULL,
                                      "from" integer NOT NULL,
                                      "to" integer NOT NULL,
                                      block_height integer NOT NULL,
                                      remark json NOT NULL
);


ALTER TABLE public.nft_histories OWNER TO postgres;

--
-- Name: nft_mutation_events; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.nft_mutation_events (
                                            nft_id integer NOT NULL,
                                            mutated_field_name character varying NOT NULL,
                                            old_value character varying NOT NULL,
                                            new_value character varying NOT NULL,
                                            block_height integer NOT NULL,
                                            remark json NOT NULL
);


ALTER TABLE public.nft_mutation_events OWNER TO postgres;

--
-- Name: nft_proposals; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.nft_proposals (
                                      nft_id integer NOT NULL,
                                      proposal_id integer NOT NULL
);


ALTER TABLE public.nft_proposals OWNER TO postgres;

--
-- Name: nft_transactions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.nft_transactions (
                                         nft_id integer NOT NULL,
                                         tx_id integer NOT NULL,
                                         is_nft_transfer boolean NOT NULL,
                                         is_nft_mint boolean NOT NULL,
                                         is_nft_burn boolean NOT NULL,
                                         block_height integer NOT NULL
);


ALTER TABLE public.nft_transactions OWNER TO postgres;

--
-- Name: nfts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.nfts (
                             id integer NOT NULL,
                             collection integer NOT NULL,
                             owner integer NOT NULL,
                             nft integer NOT NULL,
                             uri character varying NOT NULL,
                             description character varying NOT NULL,
                             is_burned boolean NOT NULL,
                             token_id character varying NOT NULL,
                             remark json NOT NULL
);


ALTER TABLE public.nfts OWNER TO postgres;

--
-- Name: opinit_transactions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.opinit_transactions (
                                            tx_id integer NOT NULL,
                                            bridge_id integer NOT NULL,
                                            block_height integer NOT NULL
);


ALTER TABLE public.opinit_transactions OWNER TO postgres;

--
-- Name: proposal_deposits; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.proposal_deposits (
                                          proposal_id integer NOT NULL,
                                          transaction_id integer,
                                          depositor integer NOT NULL,
                                          amount json NOT NULL
);


ALTER TABLE public.proposal_deposits OWNER TO postgres;

--
-- Name: proposal_votes; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.proposal_votes (
                                       proposal_id integer NOT NULL,
                                       transaction_id integer,
                                       voter integer NOT NULL,
                                       is_vote_weighted boolean NOT NULL,
                                       is_validator boolean NOT NULL,
                                       validator_address character varying,
                                       yes numeric NOT NULL,
                                       no numeric NOT NULL,
                                       abstain numeric NOT NULL,
                                       no_with_veto numeric NOT NULL
);


ALTER TABLE public.proposal_votes OWNER TO postgres;

--
-- Name: proposal_votes_legacy; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.proposal_votes_legacy (
                                              proposal_id integer NOT NULL,
                                              transaction_id integer,
                                              voter integer NOT NULL,
                                              is_vote_weighted boolean NOT NULL,
                                              is_validator boolean NOT NULL,
                                              validator_address character varying,
                                              yes numeric NOT NULL,
                                              no numeric NOT NULL,
                                              abstain numeric NOT NULL,
                                              no_with_veto numeric NOT NULL
);


ALTER TABLE public.proposal_votes_legacy OWNER TO postgres;

--
-- Name: proposals_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.proposals_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.proposals_id_seq OWNER TO postgres;

--
-- Name: proposals_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.proposals_id_seq OWNED BY public.proposals.id;


--
-- Name: seq_account_id; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.seq_account_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.seq_account_id OWNER TO postgres;

--
-- Name: seq_collection_id; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.seq_collection_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.seq_collection_id OWNER TO postgres;

--
-- Name: seq_module_id; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.seq_module_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.seq_module_id OWNER TO postgres;

--
-- Name: seq_nft_id; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.seq_nft_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.seq_nft_id OWNER TO postgres;

--
-- Name: seq_transaction_id; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.seq_transaction_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.seq_transaction_id OWNER TO postgres;

--
-- Name: seq_validator_id; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.seq_validator_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.seq_validator_id OWNER TO postgres;

--
-- Name: seq_vm_address_id; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.seq_vm_address_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.seq_vm_address_id OWNER TO postgres;

--
-- Name: tracking; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tracking (
                                 chain_id character varying NOT NULL,
                                 topic character varying NOT NULL,
                                 kafka_offset integer NOT NULL,
                                 replay_topic character varying NOT NULL,
                                 replay_offset integer NOT NULL
);


ALTER TABLE public.tracking OWNER TO postgres;

--
-- Name: transactions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.transactions (
                                     id integer NOT NULL,
                                     hash bytea NOT NULL,
                                     block_height integer NOT NULL,
                                     gas_used integer NOT NULL,
                                     gas_limit integer NOT NULL,
                                     gas_fee character varying NOT NULL,
                                     err_msg character varying,
                                     success boolean NOT NULL,
                                     sender integer NOT NULL,
                                     memo character varying NOT NULL,
                                     messages json NOT NULL,
                                     is_ibc boolean NOT NULL,
                                     is_send boolean NOT NULL,
                                     is_move_publish boolean NOT NULL,
                                     is_move_execute_event boolean NOT NULL,
                                     is_move_execute boolean NOT NULL,
                                     is_move_upgrade boolean NOT NULL,
                                     is_move_script boolean NOT NULL,
                                     is_nft_transfer boolean NOT NULL,
                                     is_nft_mint boolean NOT NULL,
                                     is_nft_burn boolean NOT NULL,
                                     is_collection_create boolean NOT NULL,
                                     is_opinit boolean NOT NULL,
                                     is_instantiate boolean NOT NULL,
                                     is_migrate boolean NOT NULL,
                                     is_update_admin boolean NOT NULL,
                                     is_clear_admin boolean NOT NULL,
                                     is_store_code boolean NOT NULL
);


ALTER TABLE public.transactions OWNER TO postgres;

--
-- Name: validators; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.validators (
                                   id integer NOT NULL,
                                   account_id integer NOT NULL,
                                   operator_address character varying NOT NULL,
                                   consensus_address character varying NOT NULL,
                                   voting_powers json NOT NULL,
                                   voting_power bigint NOT NULL,
                                   moniker character varying NOT NULL,
                                   identity character varying NOT NULL,
                                   website character varying NOT NULL,
                                   details character varying NOT NULL,
                                   commission_rate character varying NOT NULL,
                                   commission_max_rate character varying NOT NULL,
                                   commission_max_change character varying NOT NULL,
                                   jailed boolean NOT NULL
);


ALTER TABLE public.validators OWNER TO postgres;

--
-- Name: vm_addresses; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.vm_addresses (
                                     id integer NOT NULL,
                                     vm_address character varying NOT NULL
);


ALTER TABLE public.vm_addresses OWNER TO postgres;

--
-- Name: blocks height; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.blocks ALTER COLUMN height SET DEFAULT nextval('public.blocks_height_seq'::regclass);


--
-- Name: proposals id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposals ALTER COLUMN id SET DEFAULT nextval('public.proposals_id_seq'::regclass);


--
-- Name: account_transactions account_transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_transactions
    ADD CONSTRAINT account_transactions_pkey PRIMARY KEY (transaction_id, account_id);


--
-- Name: accounts accounts_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_id_key UNIQUE (id);


--
-- Name: accounts accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (address);


--
-- Name: blocks blocks_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT blocks_pkey PRIMARY KEY (height);


--
-- Name: collections collections_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collections
    ADD CONSTRAINT collections_pkey PRIMARY KEY (id);


--
-- Name: modules modules_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.modules
    ADD CONSTRAINT modules_id_key UNIQUE (id);


--
-- Name: modules modules_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.modules
    ADD CONSTRAINT modules_pkey PRIMARY KEY (publisher_id, name);


--
-- Name: nfts nfts_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nfts
    ADD CONSTRAINT nfts_pkey PRIMARY KEY (id);


--
-- Name: proposals proposals_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposals
    ADD CONSTRAINT proposals_pkey PRIMARY KEY (id);


--
-- Name: tracking tracking_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tracking
    ADD CONSTRAINT tracking_pkey PRIMARY KEY (chain_id);


--
-- Name: transactions transactions_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_id_key UNIQUE (id);


--
-- Name: transactions transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_pkey PRIMARY KEY (hash, block_height);


--
-- Name: validators validators_account_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.validators
    ADD CONSTRAINT validators_account_id_key UNIQUE (account_id);


--
-- Name: validators validators_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.validators
    ADD CONSTRAINT validators_id_key UNIQUE (id);


--
-- Name: validators validators_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.validators
    ADD CONSTRAINT validators_pkey PRIMARY KEY (operator_address);


--
-- Name: vm_addresses vm_addresses_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.vm_addresses
    ADD CONSTRAINT vm_addresses_pkey PRIMARY KEY (id);


--
-- Name: vm_addresses vm_addresses_vm_address_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.vm_addresses
    ADD CONSTRAINT vm_addresses_vm_address_key UNIQUE (vm_address);


--
-- Name: ix_blocks_timestamp; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_blocks_timestamp ON public.blocks USING btree ("timestamp");


--
-- Name: ix_collection_mutation_events_block_height; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_collection_mutation_events_block_height ON public.collection_mutation_events USING btree (block_height);


--
-- Name: ix_collection_transactions_block_height; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_collection_transactions_block_height ON public.collection_transactions USING btree (block_height);


--
-- Name: ix_collections_block_height; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_collections_block_height ON public.collections USING btree (block_height);


--
-- Name: ix_lcd_tx_results_block_height; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_lcd_tx_results_block_height ON public.lcd_tx_results USING btree (block_height);


--
-- Name: ix_lcd_tx_results_transaction_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_lcd_tx_results_transaction_id ON public.lcd_tx_results USING btree (transaction_id);


--
-- Name: ix_module_histories_block_height; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_module_histories_block_height ON public.module_histories USING btree (block_height);


--
-- Name: ix_module_histories_module_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_module_histories_module_id ON public.module_histories USING btree (module_id);


--
-- Name: ix_module_transactions_block_height; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_module_transactions_block_height ON public.module_transactions USING btree (block_height);


--
-- Name: ix_nft_histories_block_height; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_nft_histories_block_height ON public.nft_histories USING btree (block_height);


--
-- Name: ix_nft_mutation_events_block_height; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_nft_mutation_events_block_height ON public.nft_mutation_events USING btree (block_height);


--
-- Name: ix_nft_transactions_block_height; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_nft_transactions_block_height ON public.nft_transactions USING btree (block_height);


--
-- Name: ix_opinit_transactions_block_height; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_opinit_transactions_block_height ON public.opinit_transactions USING btree (block_height);


--
-- Name: ix_proposal_votes_transaction_id_desc; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_proposal_votes_transaction_id_desc ON public.proposal_votes USING btree (proposal_id, transaction_id DESC);


--
-- Name: ix_proposals_resolved_height; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_proposals_resolved_height ON public.proposals USING btree (resolved_height);


--
-- Name: ix_transactions_block_height; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX ix_transactions_block_height ON public.transactions USING btree (block_height);


--
-- Name: account_transactions account_transactions_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_transactions
    ADD CONSTRAINT account_transactions_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id);


--
-- Name: account_transactions account_transactions_block_height_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_transactions
    ADD CONSTRAINT account_transactions_block_height_fkey FOREIGN KEY (block_height) REFERENCES public.blocks(height);


--
-- Name: account_transactions account_transactions_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_transactions
    ADD CONSTRAINT account_transactions_transaction_id_fkey FOREIGN KEY (transaction_id) REFERENCES public.transactions(id);


--
-- Name: accounts accounts_vm_address_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_vm_address_id_fkey FOREIGN KEY (vm_address_id) REFERENCES public.vm_addresses(id);


--
-- Name: blocks blocks_proposer_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT blocks_proposer_fkey FOREIGN KEY (proposer) REFERENCES public.validators(operator_address);


--
-- Name: collection_mutation_events collection_mutation_events_block_height_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collection_mutation_events
    ADD CONSTRAINT collection_mutation_events_block_height_fkey FOREIGN KEY (block_height) REFERENCES public.blocks(height);


--
-- Name: collection_mutation_events collection_mutation_events_collection_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collection_mutation_events
    ADD CONSTRAINT collection_mutation_events_collection_id_fkey FOREIGN KEY (collection_id) REFERENCES public.collections(id);


--
-- Name: collection_proposals collection_proposals_collection_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collection_proposals
    ADD CONSTRAINT collection_proposals_collection_id_fkey FOREIGN KEY (collection_id) REFERENCES public.collections(id);


--
-- Name: collection_proposals collection_proposals_nft_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collection_proposals
    ADD CONSTRAINT collection_proposals_nft_id_fkey FOREIGN KEY (nft_id) REFERENCES public.nfts(id);


--
-- Name: collection_proposals collection_proposals_proposal_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collection_proposals
    ADD CONSTRAINT collection_proposals_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES public.proposals(id);


--
-- Name: collection_transactions collection_transactions_block_height_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collection_transactions
    ADD CONSTRAINT collection_transactions_block_height_fkey FOREIGN KEY (block_height) REFERENCES public.blocks(height);


--
-- Name: collection_transactions collection_transactions_collection_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collection_transactions
    ADD CONSTRAINT collection_transactions_collection_id_fkey FOREIGN KEY (collection_id) REFERENCES public.collections(id);


--
-- Name: collection_transactions collection_transactions_nft_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collection_transactions
    ADD CONSTRAINT collection_transactions_nft_id_fkey FOREIGN KEY (nft_id) REFERENCES public.nfts(id);


--
-- Name: collection_transactions collection_transactions_tx_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collection_transactions
    ADD CONSTRAINT collection_transactions_tx_id_fkey FOREIGN KEY (tx_id) REFERENCES public.transactions(id);


--
-- Name: collections collections_block_height_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collections
    ADD CONSTRAINT collections_block_height_fkey FOREIGN KEY (block_height) REFERENCES public.blocks(height);


--
-- Name: collections collections_collection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collections
    ADD CONSTRAINT collections_collection_fkey FOREIGN KEY (collection) REFERENCES public.vm_addresses(id);


--
-- Name: collections collections_creator_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collections
    ADD CONSTRAINT collections_creator_fkey FOREIGN KEY (creator) REFERENCES public.vm_addresses(id);


--
-- Name: lcd_tx_results lcd_tx_results_block_height_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.lcd_tx_results
    ADD CONSTRAINT lcd_tx_results_block_height_fkey FOREIGN KEY (block_height) REFERENCES public.blocks(height);


--
-- Name: lcd_tx_results lcd_tx_results_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.lcd_tx_results
    ADD CONSTRAINT lcd_tx_results_transaction_id_fkey FOREIGN KEY (transaction_id) REFERENCES public.transactions(id);


--
-- Name: module_histories module_histories_block_height_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.module_histories
    ADD CONSTRAINT module_histories_block_height_fkey FOREIGN KEY (block_height) REFERENCES public.blocks(height);


--
-- Name: module_histories module_histories_module_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.module_histories
    ADD CONSTRAINT module_histories_module_id_fkey FOREIGN KEY (module_id) REFERENCES public.modules(id);


--
-- Name: module_proposals module_proposals_module_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.module_proposals
    ADD CONSTRAINT module_proposals_module_id_fkey FOREIGN KEY (module_id) REFERENCES public.modules(id);


--
-- Name: module_proposals module_proposals_proposal_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.module_proposals
    ADD CONSTRAINT module_proposals_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES public.proposals(id);


--
-- Name: module_transactions module_transactions_block_height_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.module_transactions
    ADD CONSTRAINT module_transactions_block_height_fkey FOREIGN KEY (block_height) REFERENCES public.blocks(height);


--
-- Name: module_transactions module_transactions_module_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.module_transactions
    ADD CONSTRAINT module_transactions_module_id_fkey FOREIGN KEY (module_id) REFERENCES public.modules(id);


--
-- Name: module_transactions module_transactions_tx_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.module_transactions
    ADD CONSTRAINT module_transactions_tx_id_fkey FOREIGN KEY (tx_id) REFERENCES public.transactions(id);


--
-- Name: modules modules_publish_tx_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.modules
    ADD CONSTRAINT modules_publish_tx_id_fkey FOREIGN KEY (publish_tx_id) REFERENCES public.transactions(id);


--
-- Name: modules modules_publisher_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.modules
    ADD CONSTRAINT modules_publisher_id_fkey FOREIGN KEY (publisher_id) REFERENCES public.vm_addresses(id);


--
-- Name: nft_histories nft_histories_block_height_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nft_histories
    ADD CONSTRAINT nft_histories_block_height_fkey FOREIGN KEY (block_height) REFERENCES public.blocks(height);


--
-- Name: nft_histories nft_histories_from_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nft_histories
    ADD CONSTRAINT nft_histories_from_fkey FOREIGN KEY ("from") REFERENCES public.vm_addresses(id);


--
-- Name: nft_histories nft_histories_nft_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nft_histories
    ADD CONSTRAINT nft_histories_nft_id_fkey FOREIGN KEY (nft_id) REFERENCES public.nfts(id);


--
-- Name: nft_histories nft_histories_to_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nft_histories
    ADD CONSTRAINT nft_histories_to_fkey FOREIGN KEY ("to") REFERENCES public.vm_addresses(id);


--
-- Name: nft_mutation_events nft_mutation_events_block_height_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nft_mutation_events
    ADD CONSTRAINT nft_mutation_events_block_height_fkey FOREIGN KEY (block_height) REFERENCES public.blocks(height);


--
-- Name: nft_mutation_events nft_mutation_events_nft_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nft_mutation_events
    ADD CONSTRAINT nft_mutation_events_nft_id_fkey FOREIGN KEY (nft_id) REFERENCES public.nfts(id);


--
-- Name: nft_proposals nft_proposals_nft_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nft_proposals
    ADD CONSTRAINT nft_proposals_nft_id_fkey FOREIGN KEY (nft_id) REFERENCES public.nfts(id);


--
-- Name: nft_proposals nft_proposals_proposal_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nft_proposals
    ADD CONSTRAINT nft_proposals_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES public.proposals(id);


--
-- Name: nft_transactions nft_transactions_block_height_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nft_transactions
    ADD CONSTRAINT nft_transactions_block_height_fkey FOREIGN KEY (block_height) REFERENCES public.blocks(height);


--
-- Name: nft_transactions nft_transactions_nft_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nft_transactions
    ADD CONSTRAINT nft_transactions_nft_id_fkey FOREIGN KEY (nft_id) REFERENCES public.nfts(id);


--
-- Name: nft_transactions nft_transactions_tx_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nft_transactions
    ADD CONSTRAINT nft_transactions_tx_id_fkey FOREIGN KEY (tx_id) REFERENCES public.transactions(id);


--
-- Name: nfts nfts_collection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nfts
    ADD CONSTRAINT nfts_collection_fkey FOREIGN KEY (collection) REFERENCES public.collections(id);


--
-- Name: nfts nfts_nft_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nfts
    ADD CONSTRAINT nfts_nft_fkey FOREIGN KEY (nft) REFERENCES public.vm_addresses(id);


--
-- Name: nfts nfts_owner_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.nfts
    ADD CONSTRAINT nfts_owner_fkey FOREIGN KEY (owner) REFERENCES public.vm_addresses(id);


--
-- Name: opinit_transactions opinit_transactions_block_height_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.opinit_transactions
    ADD CONSTRAINT opinit_transactions_block_height_fkey FOREIGN KEY (block_height) REFERENCES public.blocks(height);


--
-- Name: opinit_transactions opinit_transactions_tx_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.opinit_transactions
    ADD CONSTRAINT opinit_transactions_tx_id_fkey FOREIGN KEY (tx_id) REFERENCES public.transactions(id);


--
-- Name: proposal_deposits proposal_deposits_depositor_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposal_deposits
    ADD CONSTRAINT proposal_deposits_depositor_fkey FOREIGN KEY (depositor) REFERENCES public.accounts(id);


--
-- Name: proposal_deposits proposal_deposits_proposal_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposal_deposits
    ADD CONSTRAINT proposal_deposits_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES public.proposals(id);


--
-- Name: proposal_deposits proposal_deposits_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposal_deposits
    ADD CONSTRAINT proposal_deposits_transaction_id_fkey FOREIGN KEY (transaction_id) REFERENCES public.transactions(id);


--
-- Name: proposal_votes_legacy proposal_votes_proposal_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposal_votes_legacy
    ADD CONSTRAINT proposal_votes_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES public.proposals(id);


--
-- Name: proposal_votes proposal_votes_proposal_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposal_votes
    ADD CONSTRAINT proposal_votes_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES public.proposals(id);


--
-- Name: proposal_votes_legacy proposal_votes_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposal_votes_legacy
    ADD CONSTRAINT proposal_votes_transaction_id_fkey FOREIGN KEY (transaction_id) REFERENCES public.transactions(id);


--
-- Name: proposal_votes proposal_votes_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposal_votes
    ADD CONSTRAINT proposal_votes_transaction_id_fkey FOREIGN KEY (transaction_id) REFERENCES public.transactions(id);


--
-- Name: proposal_votes_legacy proposal_votes_validator_address_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposal_votes_legacy
    ADD CONSTRAINT proposal_votes_validator_address_fkey FOREIGN KEY (validator_address) REFERENCES public.validators(operator_address);


--
-- Name: proposal_votes proposal_votes_validator_address_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposal_votes
    ADD CONSTRAINT proposal_votes_validator_address_fkey FOREIGN KEY (validator_address) REFERENCES public.validators(operator_address);


--
-- Name: proposal_votes_legacy proposal_votes_voter_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposal_votes_legacy
    ADD CONSTRAINT proposal_votes_voter_fkey FOREIGN KEY (voter) REFERENCES public.accounts(id);


--
-- Name: proposal_votes proposal_votes_voter_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposal_votes
    ADD CONSTRAINT proposal_votes_voter_fkey FOREIGN KEY (voter) REFERENCES public.accounts(id);


--
-- Name: proposals proposals_created_height_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposals
    ADD CONSTRAINT proposals_created_height_fkey FOREIGN KEY (created_height) REFERENCES public.blocks(height);


--
-- Name: proposals proposals_created_tx_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposals
    ADD CONSTRAINT proposals_created_tx_fkey FOREIGN KEY (created_tx) REFERENCES public.transactions(id);


--
-- Name: proposals proposals_proposer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposals
    ADD CONSTRAINT proposals_proposer_id_fkey FOREIGN KEY (proposer_id) REFERENCES public.accounts(id);


--
-- Name: proposals proposals_resolved_height_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proposals
    ADD CONSTRAINT proposals_resolved_height_fkey FOREIGN KEY (resolved_height) REFERENCES public.blocks(height);


--
-- Name: transactions transactions_block_height_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_block_height_fkey FOREIGN KEY (block_height) REFERENCES public.blocks(height);


--
-- Name: transactions transactions_sender_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_sender_fkey FOREIGN KEY (sender) REFERENCES public.accounts(id);


--
-- Name: validators validators_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.validators
    ADD CONSTRAINT validators_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id);


--
-- Name: TABLE proposals; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.proposals TO readonly;


--
-- Name: FUNCTION search_proposals(types_param character varying[]); Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON FUNCTION public.search_proposals(types_param character varying[]) TO readonly;


--
-- Name: TABLE account_transactions; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.account_transactions TO readonly;


--
-- Name: TABLE accounts; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.accounts TO readonly;


--
-- Name: TABLE blocks; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.blocks TO readonly;


--
-- Name: TABLE collection_mutation_events; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.collection_mutation_events TO readonly;


--
-- Name: TABLE collection_proposals; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.collection_proposals TO readonly;


--
-- Name: TABLE collection_transactions; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.collection_transactions TO readonly;


--
-- Name: TABLE collections; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.collections TO readonly;


--
-- Name: TABLE lcd_tx_results; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.lcd_tx_results TO readonly;


--
-- Name: TABLE module_histories; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.module_histories TO readonly;


--
-- Name: TABLE module_proposals; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.module_proposals TO readonly;


--
-- Name: TABLE module_transactions; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.module_transactions TO readonly;


--
-- Name: TABLE modules; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.modules TO readonly;


--
-- Name: TABLE nft_histories; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.nft_histories TO readonly;


--
-- Name: TABLE nft_mutation_events; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.nft_mutation_events TO readonly;


--
-- Name: TABLE nft_proposals; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.nft_proposals TO readonly;


--
-- Name: TABLE nft_transactions; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.nft_transactions TO readonly;


--
-- Name: TABLE nfts; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.nfts TO readonly;


--
-- Name: TABLE opinit_transactions; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.opinit_transactions TO readonly;


--
-- Name: TABLE proposal_deposits; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.proposal_deposits TO readonly;


--
-- Name: TABLE proposal_votes; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.proposal_votes TO readonly;


--
-- Name: TABLE proposal_votes_legacy; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.proposal_votes_legacy TO readonly;


--
-- Name: TABLE tracking; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.tracking TO readonly;


--
-- Name: TABLE transactions; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.transactions TO readonly;


--
-- Name: TABLE validators; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.validators TO readonly;


--
-- Name: TABLE vm_addresses; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.vm_addresses TO readonly;


--
-- Name: DEFAULT PRIVILEGES FOR TABLES; Type: DEFAULT ACL; Schema: public; Owner: postgres
--

ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public GRANT SELECT ON TABLES  TO readonly;


--
-- PostgreSQL database dump complete
--
