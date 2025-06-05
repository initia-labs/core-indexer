package flusher

import (
	"context"

	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
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
func (f *Flusher) ProcessCommitSignatureVote(sigs []types.CommitSig) (map[string]db.CommitSignatureType, error) {
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

func (f *Flusher) loadValidatorsToCache(ctx context.Context) error {
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
	if block.LastCommit == nil {
		logger.Info().Int64("height", block.Height).Msgf("Skipping processing of this block")
		return nil
	}

	logger.Info().Msgf("Processing validator commit signatures from block: %d", block.Height)

	sigs, err := f.ProcessCommitSignatureVote(block.LastCommit.Signatures)
	if err != nil {
		logger.Error().Int64("height", block.Height).Msgf("Error processsing block commit signature from db: %v", err)
		return err
	}

	proposer, ok := f.validators[block.ProposerConsensusAddress]
	if !ok {
		logger.Info().Msgf("Update validators cache")
		if err := f.loadValidatorsToCache(ctx); err != nil {
			logger.Error().Int64("height", block.Height).Msgf("Error loading validators to cache: %v", err)
			return err
		}
	}
	logger.Info().Msgf("Proposer for this round is: %v => %v", block.ProposerConsensusAddress, proposer.OperatorAddress)
	if err := f.dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// use for test only
		// {
		// 	vals := make([]db.Validator, 0)
		// 	accs := make([]db.Account, 0)
		// 	vmAddrs := make([]db.VMAddress, 0)
		// 	for _, val := range f.validators {
		// 		valAcc, err := sdk.ValAddressFromBech32(val.OperatorAddress)
		// 		if err != nil {
		// 			return fmt.Errorf("failed to convert validator address: %w", err)
		// 		}

		// 		accAddr := sdk.AccAddress(valAcc)
		// 		vmAddr, _ := vmtypes.NewAccountAddressFromBytes(accAddr)

		// 		if err := val.UnpackInterfaces(f.encodingConfig.InterfaceRegistry); err != nil {
		// 			return fmt.Errorf("failed to unpack validator info: %w", err)
		// 		}

		// 		consAddr, err := val.GetConsAddr()
		// 		if err != nil {
		// 			return errors.Join(ErrorNonRetryable, err)
		// 		}
		// 		vals = append(vals, db.NewValidator(val, accAddr.String(), consAddr))
		// 		accs = append(accs, db.Account{
		// 			Address:     accAddr.String(),
		// 			VMAddressID: vmAddr.String(),
		// 			Type:        string(db.BaseAccount),
		// 		})
		// 		vmAddrs = append(vmAddrs, db.VMAddress{
		// 			VMAddress: vmAddr.String(),
		// 		})
		// 	}

		// 	err = db.InsertVMAddressIgnoreConflict(ctx, dbTx, vmAddrs)
		// 	if err != nil {
		// 		logger.Error().Int64("height", block.Height).Msgf("Error inserting VM addresses: %v", err)
		// 		return err
		// 	}

		// 	err = db.InsertAccountIgnoreConflict(ctx, dbTx, accs)
		// 	if err != nil {
		// 		logger.Error().Int64("height", block.Height).Msgf("Error inserting accounts: %v", err)
		// 		return err
		// 	}

		// 	err = db.InsertValidatorIgnoreConflict(ctx, dbTx, vals)
		// 	if err != nil {
		// 		logger.Error().Int64("height", block.Height).Msgf("Error inserting validators: %v", err)
		// 		return err
		// 	}
		// }

		err = db.InsertValidatorCommitSignatureForProposer(ctx, dbTx, proposer.OperatorAddress, block.Height)
		if err != nil {
			logger.Error().Int64("height", block.Height).Msgf("Error inserting commmit signature for block proposer: %v", err)
			return err
		}
		dbSigs := make([]db.ValidatorCommitSignature, 0)
		for consAddr, val := range f.validators {
			// Check if there is a validator's vote for the given consensus address (val.ConsensusAddress).
			// If the validator's vote is not found (ok is false), we skip this validator because they have not committed
			// evidence on this block. We do this to avoid including votes from validators who are not in the active set
			// in the database. This helps ensure that only active validators' votes are considered.
			vote, ok := sigs[consAddr]
			if !ok {
				continue
			}

			dbSigs = append(dbSigs, db.ValidatorCommitSignature{
				ValidatorAddress: val.OperatorAddress,
				BlockHeight:      block.LastCommit.Height,
				Vote:             string(vote),
			})
		}
		err = db.InsertValidatorCommitSignatures(ctx, dbTx, &dbSigs)
		if err != nil {
			logger.Error().Int64("height", block.Height).Msgf("Error inserting validator commit signatures: %v", err)
			return err
		}

		return nil
	}); err != nil {
		logger.Error().Int64("height", block.Height).Msgf("Error committing transaction: %v", err)
		return err
	}

	logger.Info().Int64("height", block.Height).Msgf("Successfully flushed validator: %d", block.Height)

	return nil
}
