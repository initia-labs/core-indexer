package indexer

import (
	"context"
	"fmt"

	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"gorm.io/gorm"

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
func (f *Indexer) ProcessCommitSignatureVote(sigs []types.CommitSig) (map[string]db.CommitSignatureType, error) {
	commitSigs := make(map[string]db.CommitSignatureType)
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
			commitSigs[consensusAddress] = db.Absent
		case types.BlockIDFlagCommit:
			commitSigs[consensusAddress] = db.Vote
		}
	}
	return commitSigs, nil
}

func (f *Indexer) processValidator(parentCtx context.Context, blockResults *mq.BlockResultMsg, proposer *db.ValidatorAddress) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "processValidator", "Parse validator signatures from block and insert into DB")
	defer span.Finish()

	// If block.LastCommit or block.ProposerConsensusAddress is nil, it means there's no last commit in the Kafka message,
	// indicating that this message represents an old version. In this case,
	// we skip processing the message without raising an error, as our intention is to ignore old versions.
	if blockResults.LastCommit == nil {
		logger.Info().Int64("height", blockResults.Height).Msgf("Skipping processing of this block")
		return nil
	}

	logger.Info().Msgf("Processing validator commit signatures from block: %d", blockResults.Height)

	sigs, err := f.ProcessCommitSignatureVote(blockResults.LastCommit.Signatures)
	if err != nil {
		logger.Error().Int64("height", blockResults.Height).Msgf("Error processsing block commit signature from db: %v", err)
		return err
	}

	logger.Info().Msgf("Proposer for this round is: %v => %v", blockResults.ProposerConsensusAddress, proposer.OperatorAddress)
	if err := f.dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		err = db.InsertValidatorCommitSignatureForProposer(ctx, dbTx, proposer.OperatorAddress, blockResults.Height)
		if err != nil {
			logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting commmit signature for block proposer: %v", err)
			return err
		}
		dbSigs := make([]db.ValidatorCommitSignature, 0)
		for consAddr, vote := range sigs {
			val, ok := f.cacher.GetValidatorByConsAddr(consAddr)
			if !ok {
				err := fmt.Errorf("validator not found - %s", consAddr)
				logger.Error().Msgf("Error getting validator for a commit signature: %v", err)
				return err
			}

			dbSigs = append(dbSigs, db.ValidatorCommitSignature{
				ValidatorAddress: val.OperatorAddress,
				BlockHeight:      blockResults.LastCommit.Height,
				Vote:             string(vote),
			})
		}
		err = db.InsertValidatorCommitSignatures(ctx, dbTx, &dbSigs)
		if err != nil {
			logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting validator commit signatures: %v", err)
			return err
		}

		return nil
	}); err != nil {
		logger.Error().Int64("height", blockResults.Height).Msgf("Error committing transaction: %v", err)
		return err
	}

	logger.Info().Int64("height", blockResults.Height).Msgf("Successfully indexed validator: %d", blockResults.Height)

	return nil
}
