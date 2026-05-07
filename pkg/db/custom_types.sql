-- Since GORM & Atlas cannot handle creating custom types programmatically, we have to define them here.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'accounttype') THEN
        CREATE TYPE accounttype AS ENUM (
            'BaseAccount',
            'InterchainAccount',
            'ModuleAccount',
            'ContinuousVestingAccount',
            'DelayedVestingAccount',
            'ClawbackVestingAccount',
            'ContractAccount'
            );
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'upgradepolicy') THEN
        CREATE TYPE upgradepolicy AS ENUM (
            'Arbitrary',
            'Compatible',
            'Immutable'
            );
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'commit_signature_type') THEN
        CREATE TYPE commit_signature_type AS ENUM (
            'PROPOSE',
            'VOTE',
            'ABSENT'
            );
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'slashtype') THEN
        CREATE TYPE slashtype AS ENUM (
            'Jailed',
            'Slashed',
            'Unjailed'
            );
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'finalize_block_events_mode') THEN
        CREATE TYPE finalize_block_events_mode AS ENUM (
            'BeginBlock',
            'EndBlock'
            );
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'proposalstatus') THEN
        CREATE TYPE proposalstatus AS ENUM (
            'Nil',
            'DepositPeriod',
            'VotingPeriod',
            'Passed',
            'Rejected',
            'Failed',
            'Inactive',
            'Cancelled'
            );
    END IF;
END $$;
