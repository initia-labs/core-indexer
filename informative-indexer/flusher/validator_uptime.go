package flusher

import (
	"context"

	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	"github.com/jackc/pgx/v5"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
)

func Bech32ValConsPub(val bytes.HexBytes) (string, error) {
	bech32PrefixCons := sdk.GetConfig().GetBech32ConsensusAddrPrefix()
	consensusAddress, err := bech32.ConvertAndEncode(bech32PrefixCons, val)
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

func (f *Flusher) loadValidatorsToCache(ctx context.Context, dbTx pgx.Tx) error {
	// TODO: add retry logic
	valInfos, err := f.rpcClient.ValidatorInfos(ctx, "BOND_STATUS_BONDED")
	f.validators = make(map[string]mstakingtypes.Validator)
	if err != nil {
		return err
	}

	for _, valInfo := range *valInfos {
		err := valInfo.UnpackInterfaces(f.encodingConfig.InterfaceRegistry)
		if err != nil {
			return err
		}
		consAddr, err := valInfo.GetConsAddr()
		if err != nil {
			return err
		}
		f.validators[consAddr.String()] = valInfo
	}
	logger.Info().Msgf("Total validators loaded to cache = %d", len(f.validators))

	return nil
}

func (f *Flusher) processValidator(parentCtx context.Context, block *mq.BlockResultMsg) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "processValidator", "Parse validator signatures from block and insert into DB")
	defer span.Finish()

	// If block.LastCommit or block.ProposerConsensusAddress is nil, it means there's no last commit in the Kafka message,
	// indicating that this message represents an old version. In this case,
	// we skip processing the message without raising an error, as our intention is to ignore old versions.
	if block.LastCommit == nil || block.ProposerConsensusAddress == nil {
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

	proposer, ok := f.validators[*block.ProposerConsensusAddress]
	if !ok {
		logger.Info().Msgf("Proposer for this round is: %v => %v", block.ProposerConsensusAddress, f.validators[*block.ProposerConsensusAddress])
		f.loadValidatorsToCache(ctx, dbTx)
	}
	logger.Info().Msgf("Proposer for this round is: %v => %v", block.ProposerConsensusAddress, proposer)
	err = db.InsertValidatorCommitSignatureForProposer(ctx, dbTx, proposer.OperatorAddress, block.Height)
	if err != nil {
		logger.Error().Int64("height", block.Height).Msgf("Error inserting commmit signature for block proposer: %v", err)
		return err
	}
	dbSigs := make([]db.ValidatorCommitSignatures, 0)
	for consAddr, val := range f.validators {
		// Check if there is a validator's vote for the given consensus address (val.ConsensusAddress).
		// If the validator's vote is not found (ok is false), we skip this validator because they have not committed
		// evidence on this block. We do this to avoid including votes from validators who are not in the active set
		// in the database. This helps ensure that only active validators' votes are considered.
		vote, ok := sigs[consAddr]
		if !ok {
			continue
		}

		dbSigs = append(dbSigs, db.NewValidatorCommitSignatures(val.OperatorAddress, block.LastCommit.Height, vote))
	}
	err = db.InsertValidatorCommitSignatures(ctx, dbTx, &dbSigs)
	if err != nil {
		logger.Error().Int64("height", block.Height).Msgf("Error inserting validator commit signatures: %v", err)
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
