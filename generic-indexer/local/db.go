package local

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateSweeperTablesIfNotExist(db *pgxpool.Pool) error {
	_, err := db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS blocks (
			height INTEGER PRIMARY KEY,
			timestamp TIMESTAMP WITHOUT TIME ZONE NOT NULL,
			proposer CHARACTER VARYING NOT NULL,
			hash BYTEA NOT NULL
		);
	`)

	if err != nil {
		return err
	}

	return nil
}

func CreateValidatorCronIfNotExist(db *pgxpool.Pool, window int64) error {
	// LastNBlockValidatorVotes
	_, err := db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS validator_vote_counts (
			validator_address varchar REFERENCES validators (operator_address) PRIMARY KEY,
			last_100 integer
		);
	`)

	if err != nil {
		return err
	}
	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS validator_historical_powers (
			validator_address varchar REFERENCES validators (operator_address),
			voting_power bigint,
			hour_rounded_timestamp timestamp,
			timestamp timestamp,
			CONSTRAINT unique_validator_historicail_power UNIQUE (validator_address, hour_rounded_timestamp)
		);
	`)
	return err
}

func CreateValidatorFlusherTablesIfNotExist(db *pgxpool.Pool) error {
	// _, err := db.Exec(context.Background(), `
	// 	CREATE TYPE "commit_signature_type" AS ENUM (
	// 		'PROPOSE',
	// 		'VOTE',
	// 		'ABSENT'
	// 	);
	// `)

	// if err != nil {
	// 	return err
	// }

	_, err := db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS validator_commit_signatures (
			validator_address varchar REFERENCES validators (operator_address) ,
			block_height integer,
			vote commit_signature_type,
			PRIMARY KEY(validator_address, block_height)
			);
	`)

	if err != nil {
		return err
	}

	return nil
}

func CreateFlusherTablesIfNotExist(db *pgxpool.Pool) error {
	err := CreateSweeperTablesIfNotExist(db)
	if err != nil {
		return err
	}

	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS transactions (
			id SERIAL PRIMARY KEY,
			hash BYTEA NOT NULL, 
			block_height INTEGER NOT NULL, 
			gas_used INTEGER NOT NULL, 
			gas_limit INTEGER NOT NULL, 
			gas_fee CHARACTER VARYING NOT NULL, 
			err_msg CHARACTER VARYING, 
			success BOOLEAN NOT NULL, 
			sender INTEGER NOT NULL, 
			memo CHARACTER VARYING NOT NULL, 
			messages JSONB NOT NULL,
			is_ibc BOOLEAN NOT NULL,
			is_store_code BOOLEAN NOT NULL,
			is_instantiate BOOLEAN NOT NULL,
			is_execute BOOLEAN NOT NULL,
			is_send BOOLEAN NOT NULL,
			is_update_admin BOOLEAN NOT NULL,
			is_clear_admin BOOLEAN NOT NULL,
			is_migrate BOOLEAN NOT NULL
		);
	`)

	if err != nil {
		return err
	}

	_, err = db.Exec(context.Background(), `
		CREATE TYPE accounttype AS ENUM (
			'BaseAccount'
		);
	`)

	if err != nil {
		return err
	}

	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS accounts (
			id SERIAL UNIQUE,
			address CHARACTER VARYING PRIMARY KEY,
			type accounttype DEFAULT 'BaseAccount'::accounttype,
			name character varying
		);
	`)

	if err != nil {
		return err
	}

	return nil
}
