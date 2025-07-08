package flusher

import (
	"context"
	"errors"

	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	vmtypes "github.com/initia-labs/movevm/types"
	"github.com/jackc/pgx/v5"

	"github.com/alleslabs/initia-mono/generic-indexer/common"
	"github.com/alleslabs/initia-mono/generic-indexer/db"
)

var (
	Bech32PrefixAccAddr = "init"
	Bech32PrefixCons    = Bech32PrefixAccAddr + "valcons"
)

func Bech32ValConsPub(val bytes.HexBytes) (string, error) {
	consensusAddress, err := bech32.ConvertAndEncode(Bech32PrefixCons, val)
	if err != nil {
		return "", err
	}
	return consensusAddress, err
}

// ProcessCommitSignatureVote checks all validator votes from the last commit signature
// to determine which validators voted as "absent" or "proposed" a block. It returns
// a mapping of validator consensus addresses to their respective votes.
func (f *Flusher) ProcessCommitSignatureVote(sigs []types.CommitSig) (map[string]db.BlockVote, error) {
	commitSigs := make(map[string]db.BlockVote)
	for _, commitSig := range sigs {
		if commitSig.ValidatorAddress.String() == "" {
			continue
		}
		consensusAddress, err := Bech32ValConsPub(commitSig.ValidatorAddress)
		if err != nil {
			return nil, err
		}

		switch commitSig.BlockIDFlag {
		case types.BlockIDFlagAbsent:
			commitSigs[consensusAddress] = db.ABSENT
		case types.BlockIDFlagCommit:
			commitSigs[consensusAddress] = db.VOTE
		}
	}
	return commitSigs, nil
}

// findNewValidatorsToUpdate checks for new validators by comparing the last commit signatures
// from the latest block with the existing validators in the database. It returns a list of validators
// that need to be added or updated in the database based on the new signatures.
func (f *Flusher) findNewValidatorsToUpdate(commitSigs map[string]db.BlockVote) map[string]bool {
	newVals := make(map[string]bool)
	for consensusAddr := range commitSigs {
		_, ok := f.validators[consensusAddr]
		if !ok {
			newVals[consensusAddr] = true
		}
	}
	return newVals
}

// updateNewValidator is a method of the ValidatorFlusher type and is responsible for updating information
// about new validators in the database within a PostgreSQL transaction (pgx.Tx).
// It takes a map of new validator consensus addresses (newValidator) and processes them as follows:
//  1. Fetches the existing validator relations from the database and stores them in 'existingRelations'.
//  2. Iterates over the new validator consensus addresses and processes each one:
//     a. Checks if the new validator exists in 'existingRelations'.
//     b. If the new validator is not in 'existingRelations', inserts a new validator relation record
//     in the database with the consensus address and other relevant data.
//  3. After processing all new validators, returns a slice of updated validator relations (both new and existing).
func (f *Flusher) updateNewValidator(dbTx pgx.Tx, newValidator map[string]bool) error {
	ctx := context.Background()

	err := f.loadValidatorToCache(dbTx)
	if err != nil {
		return err
	}

	valInfos, err := f.rpcClient.ValidatorInfos(ctx, "BOND_STATUS_BONDED")
	if err != nil {
		logger.Error().Msgf("Error cannot get Validator info from all RPC endpoints: %v", err)
		return err
	}
	for _, valInfo := range *valInfos {
		err := valInfo.UnpackInterfaces(f.encodingConfig.InterfaceRegistry)
		if err != nil {
			return err
		}
		consAddr, err := valInfo.GetConsAddr()
		if err != nil {
			return errors.Join(ErrorNonRetryable, err)
		}
		_, ok := newValidator[consAddr.String()]
		if ok {
			acc, _ := sdk.ValAddressFromBech32(valInfo.OperatorAddress)
			addr := sdk.AccAddress(acc)
			vmAddr, _ := vmtypes.NewAccountAddressFromBytes(addr)
			err = db.GetAccountOrInsertIfNotExist(ctx, dbTx, addr.String(), vmAddr.String())
			if err != nil {
				return err
			}
			logger.Info().Msgf("Inserting new validator: %v (%v)", addr.String(), consAddr)
			v := db.NewValidator(valInfo, addr.String(), consAddr)
			err = db.UpsertValidators(
				ctx,
				dbTx,
				&[]db.Validator{v},
			)
			if err != nil {
				return err
			}
			// we load new validators to cache here manually
			f.validators[consAddr.String()] = db.ValidatorRelation{
				OperatorAddress:  valInfo.OperatorAddress,
				ConsensusAddress: consAddr.String(),
			}
		}
	}

	return nil
}

func (f *Flusher) loadValidatorToCache(dbTx pgx.Tx) error {
	vals, err := db.QueryValidatorRelations(context.Background(), dbTx)
	f.validators = make(map[string]db.ValidatorRelation)
	if err != nil {
		return err
	}

	for _, val := range vals {
		f.validators[val.ConsensusAddress] = val
	}
	logger.Info().Msgf("Total validators loaded to cache = %d", len(f.validators))

	return nil
}

func (f *Flusher) processValidator(parentCtx context.Context, block *common.BlockMsg) error {
	span, ctx := common.StartSentrySpan(parentCtx, "processValidator", "Parse validator signatures from block and insert into DB")
	defer span.Finish()

	// If block.LastCommit is nil, it means there's no last commit in the Kafka message,
	// indicating that this message represents an old version. In this case,
	// we skip processing the message without raising an error, as our intention is to ignore old versions.
	if block.LastCommit == nil || block.BlockProposer == nil {
		logger.Info().Int64("height", block.Height).Msgf("Skipping processing of this block")
		return nil
	}

	logger.Info().Msgf("Processing validator commit signatures from block: %d", block.Height)

	dbTx, err := f.dbClient.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		logger.Error().Int64("height", block.Height).Msgf("Error beginning transaction: %v", err)
		return err
	}
	defer dbTx.Rollback(ctx)

	sigs, err := f.ProcessCommitSignatureVote(block.LastCommit.Signatures)
	if err != nil {
		logger.Error().Int64("height", block.Height).Msgf("Error processsing block commit signature from db: %v", err)
		return err
	}

	newValidator := f.findNewValidatorsToUpdate(sigs)
	if len(newValidator) > 0 {
		err = f.updateNewValidator(dbTx, newValidator)
		logger.Info().Msgf("Total new validators amount = %d", len(newValidator))
		if err != nil {
			return err
		}
	}
	logger.Info().Msgf("Proposer for this round is: %v => %v", *block.BlockProposer.ConsensusAddress, f.validators["initvalcons1v844rl3j404khzckxdrwcjp25zukc07gq3fag2"])
	err = db.InsertValidatorCommitSignatureForProposer(ctx, dbTx, f.validators[*block.BlockProposer.ConsensusAddress].OperatorAddress, block.Height)
	if err != nil {
		logger.Error().Int64("height", block.Height).Msgf("Error inserting commmit signature for block proposer: %v", err)
		return err
	}
	dbSigs := make([]db.ValidatorCommitSignatures, 0)
	for _, val := range f.validators {
		// Check if there is a validator's vote for the given consensus address (val.ConsensusAddress).
		// If the validator's vote is not found (ok is false), we skip this validator because they have not committed
		// evidence on this block. We do this to avoid including votes from validators who are not in the active set
		// in the database. This helps ensure that only active validators' votes are considered.
		vote, ok := sigs[val.ConsensusAddress]
		if !ok {
			continue
		}

		dbSigs = append(dbSigs, db.NewValidatorCommitSignatures(val.OperatorAddress, block.LastCommit.Height, vote))
	}
	err = db.InsertValidatorCommitSignatures(ctx, dbTx, &dbSigs)
	if err != nil {
		logger.Error().Int64("height", block.Height).Msgf("Error committing transaction: %v", err)
		return err
	}

	err = dbTx.Commit(ctx)
	if err != nil {
		logger.Error().Int64("height", block.Height).Msgf("Error committing transaction: %v", err)
		return err
	}

	logger.Info().Int64("height", block.Height).Msgf("Successfully flushed validator: %d", block.Height)

	return nil
}
