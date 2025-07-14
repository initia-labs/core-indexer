package statetracker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmosgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/initia-labs/initia/app/params"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/initia-labs/core-indexer/informative-indexer/flusher/types"
	"github.com/initia-labs/core-indexer/informative-indexer/flusher/utils"
	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

// StateUpdateManager tracks entities that need to be synchronized with the blockchain state
// through RPC queries. It maintains sets of validators and modules that have been modified
// and need their latest state to be fetched from the chain.
type StateUpdateManager struct {
	// validators tracks validator addresses that need their state to be synchronized
	Validators map[string]bool

	// TODO: refactor value type
	// modules tracks Move modules that need their state to be synchronized.
	// The string pointer value is the transaction hash where the module was published.
	// A nil value indicates the module was not published in the current transaction.
	Modules map[vmapi.ModuleInfoResponse]*string

	// dbBatchInsert handles database batch operations
	dbBatchInsert *DBBatchInsert

	// encodingConfig is used for encoding/decoding validator information
	encodingConfig *params.EncodingConfig

	// height is the height of the block to be used for RPC queries
	height *int64

	ProposalsToUpdate     map[int32]string
	CollectionsToUpdate   map[string]bool
	NftsToUpdate          map[string]bool
	ProposalStatusChanges map[int32]db.Proposal
}

func NewStateUpdateManager(
	dbBatchInsert *DBBatchInsert,
	encodingConfig *params.EncodingConfig,
	height *int64,
) *StateUpdateManager {
	return &StateUpdateManager{
		Validators:            make(map[string]bool),
		Modules:               make(map[vmapi.ModuleInfoResponse]*string),
		dbBatchInsert:         dbBatchInsert,
		encodingConfig:        encodingConfig,
		height:                height,
		ProposalsToUpdate:     make(map[int32]string),
		CollectionsToUpdate:   make(map[string]bool),
		NftsToUpdate:          make(map[string]bool),
		ProposalStatusChanges: make(map[int32]db.Proposal),
	}
}

func (s *StateUpdateManager) UpdateState(ctx context.Context, rpcClient cosmosrpc.CosmosJSONRPCHub) error {
	// TODO: add retry logic
	if err := s.updateValidators(ctx, rpcClient); err != nil {
		return err
	}

	if err := s.updateModules(ctx, rpcClient); err != nil {
		return err
	}

	if err := s.updateCollections(ctx, rpcClient); err != nil {
		return err
	}

	if err := s.updateNfts(ctx, rpcClient); err != nil {
		return err
	}

	if err := s.updateProposals(ctx, rpcClient); err != nil {
		return err
	}

	return nil
}

func (s *StateUpdateManager) updateProposals(ctx context.Context, rpcClient cosmosrpc.CosmosJSONRPCHub) error {
	for proposalID, txID := range s.ProposalsToUpdate {
		proposal, err := rpcClient.Proposal(ctx, proposalID, s.height)
		if err != nil {
			return fmt.Errorf("failed to query proposal: %w", err)
		}

		proposalInfo := proposal.Proposal
		if err := proposalInfo.UnpackInterfaces(s.encodingConfig.Codec); err != nil {
			return fmt.Errorf("failed to unpack interfaces proposal: %w", err)
		}

		proposalStatus := utils.ParseProposalStatus(proposalInfo.GetStatus())
		rawProposalMsgs, err := proposalInfo.GetMsgs()
		if err != nil {
			return fmt.Errorf("failed to get proposal msgs: %w", err)
		}

		proposalType := ""
		proposalTypes := make([]string, 0)
		proposalMsgs := make([]map[string]any, 0)
		for idx, proposalMsg := range rawProposalMsgs {
			if idx == 0 {
				proposalType = sdk.MsgTypeURL(proposalMsg)
			}
			proposalTypes = append(proposalTypes, sdk.MsgTypeURL(proposalMsg))

			var proposalMsgJsDict map[string]any
			proposalMsgJson, err := codec.ProtoMarshalJSON(proposalMsg, nil)
			if err != nil {
				return fmt.Errorf("failed to marshal proposal msg: %w", err)
			}
			err = json.Unmarshal(proposalMsgJson, &proposalMsgJsDict)
			if err != nil {
				return fmt.Errorf("failed to unmarshal proposal msg: %w", err)
			}
			proposalMsgJsDict["@type"] = sdk.MsgTypeURL(proposalMsg)
			proposalMsgs = append(proposalMsgs, proposalMsgJsDict)
		}

		msgsJson, err := json.Marshal(proposalMsgs)
		if err != nil {
			return fmt.Errorf("failed to marshal proposal msgs: %w", err)
		}

		contentJSON, err := json.Marshal(map[string]any{
			"messages": proposalMsgs,
			"metadata": proposalInfo.GetMetadata(),
		})
		if err != nil {
			return fmt.Errorf("failed to marshal content: %w", err)
		}

		proposalTypesJSON, err := json.Marshal(proposalTypes)
		if err != nil {
			return fmt.Errorf("failed to marshal proposal types: %w", err)
		}

		s.dbBatchInsert.proposals[proposalID] = db.Proposal{
			ID:                     proposalID,
			Title:                  proposalInfo.GetTitle(),
			Description:            proposalInfo.GetSummary(),
			Status:                 string(proposalStatus),
			SubmitTime:             *proposalInfo.SubmitTime,
			DepositEndTime:         *proposalInfo.DepositEndTime,
			TotalDeposit:           db.JSON("[]"),
			Messages:               db.JSON(msgsJson),
			Content:                db.JSON(contentJSON),
			Version:                "v1",
			Yes:                    0,
			No:                     0,
			Abstain:                0,
			NoWithVeto:             0,
			IsExpedited:            proposalInfo.GetExpedited(),
			IsEmergency:            proposalInfo.GetEmergency(),
			EmergencyStartTime:     proposalInfo.GetEmergencyStartTime(),
			EmergencyNextTallyTime: proposalInfo.GetEmergencyNextTallyTime(),
			FailedReason:           "",
			CreatedHeight:          int32(*s.height),
			CreatedTx:              txID,
			ProposerID:             proposalInfo.Proposer,
			ProposalRoute:          cosmosgovtypes.RouterKey,
			Type:                   proposalType,
			Types:                  proposalTypesJSON,
		}
	}

	for proposalID, proposal := range s.ProposalStatusChanges {
		if !utils.IsProposalPruned(proposal.Status) {
			res, err := rpcClient.Proposal(ctx, proposalID, s.height)
			if err != nil {
				return nil
			}

			proposalInfo := res.Proposal
			proposal.VotingTime = proposalInfo.VotingStartTime
			proposal.VotingEndTime = proposalInfo.VotingEndTime

			if proposalInfo.FinalTallyResult.V1TallyResult != nil {
				tally := proposalInfo.FinalTallyResult.V1TallyResult
				counts := map[string]*int64{
					"abstain":      &proposal.Abstain,
					"yes":          &proposal.Yes,
					"no":           &proposal.No,
					"no_with_veto": &proposal.NoWithVeto,
				}
				for option, count := range map[string]string{
					"abstain":      tally.AbstainCount,
					"yes":          tally.YesCount,
					"no":           tally.NoCount,
					"no_with_veto": tally.NoWithVetoCount,
				} {
					parsed, err := strconv.ParseInt(count, 10, 64)
					if err != nil {
						return fmt.Errorf("failed to parse %s count: %w", option, err)
					}
					*counts[option] = parsed
				}
				if proposal.ResolvedVotingPower == nil {
					totalVestingPower, err := strconv.ParseInt(proposalInfo.FinalTallyResult.TotalVestingPower, 10, 64)
					if err != nil {
						return fmt.Errorf("failed to parse total vesting power: %w", err)
					}
					totalStakingPower, err := strconv.ParseInt(proposalInfo.FinalTallyResult.TotalStakingPower, 10, 64)
					if err != nil {
						return fmt.Errorf("failed to parse total staking power: %w", err)
					}
					resolveVotingPower := totalVestingPower + totalStakingPower
					proposal.ResolvedVotingPower = &resolveVotingPower
				}
			}
		}
		s.dbBatchInsert.ProposalStatusChanges[proposalID] = proposal
	}
	return nil
}

func (s *StateUpdateManager) updateValidators(ctx context.Context, rpcClient cosmosrpc.CosmosJSONRPCHub) error {
	validatorAddresses := make([]string, 0, len(s.Validators))
	for addr := range s.Validators {
		validatorAddresses = append(validatorAddresses, addr)
	}

	return s.syncValidators(ctx, rpcClient, validatorAddresses)
}

func (s *StateUpdateManager) updateModules(ctx context.Context, rpcClient cosmosrpc.CosmosJSONRPCHub) error {
	modules := make([]vmapi.ModuleInfoResponse, 0, len(s.Modules))
	publishTxIds := make([]*string, 0, len(s.Modules))
	for module, publishTxId := range s.Modules {
		publishTxIds = append(publishTxIds, publishTxId)
		modules = append(modules, module)
	}

	return s.syncModules(ctx, rpcClient, modules, publishTxIds)
}

func (s *StateUpdateManager) syncValidators(ctx context.Context, rpcClient cosmosrpc.CosmosJSONRPCHub, validatorAddresses []string) error {
	for _, validatorAddr := range validatorAddresses {
		valAcc, err := sdk.ValAddressFromBech32(validatorAddr)
		if err != nil {
			return fmt.Errorf("failed to convert validator address: %w", err)
		}

		accAddr := sdk.AccAddress(valAcc)
		vmAddr, _ := vmtypes.NewAccountAddressFromBytes(accAddr)

		s.dbBatchInsert.AddAccounts(db.Account{
			Address:   accAddr.String(),
			VMAddress: db.VMAddress{VMAddress: vmAddr.String()},
			Type:      string(db.BaseAccount),
		})

		validator, err := rpcClient.Validator(ctx, validatorAddr, s.height)
		if err != nil {
			return fmt.Errorf("failed to fetch validator data: %w", err)
		}

		valInfo := validator.Validator
		if err := valInfo.UnpackInterfaces(s.encodingConfig.InterfaceRegistry); err != nil {
			return fmt.Errorf("failed to unpack validator info: %w", err)
		}

		consAddr, err := valInfo.GetConsAddr()
		if err != nil {
			return errors.Join(types.ErrorNonRetryable, err)
		}

		s.dbBatchInsert.AddValidators(
			db.NewValidator(
				valInfo,
				accAddr.String(),
				consAddr,
			),
		)
	}

	return nil
}

func (s *StateUpdateManager) syncModules(ctx context.Context, rpcClient cosmosrpc.CosmosJSONRPCHub, modules []vmapi.ModuleInfoResponse, publishTxIds []*string) error {
	for idx, module := range modules {
		moduleInfo, err := rpcClient.Module(ctx, module.Address.String(), module.Name, s.height)
		if err != nil {
			return fmt.Errorf("failed to fetch module info: %w", err)
		}

		var publishTxId *string
		if publishTxIds[idx] != nil {
			publishTxId = publishTxIds[idx]
		}

		s.dbBatchInsert.AddModule(db.Module{
			Name:                module.Name,
			ModuleEntryExecuted: 0,
			IsVerify:            false,
			PublishTxID:         publishTxId,
			PublisherID:         module.Address.String(),
			ID:                  db.GetModuleID(module),
			Digest:              parser.GetModuleDigest(moduleInfo.RawBytes),
			UpgradePolicy:       db.GetUpgradePolicy(moduleInfo.UpgradePolicy),
		})
	}

	return nil
}

func (s *StateUpdateManager) updateCollections(ctx context.Context, rpcClient cosmosrpc.CosmosJSONRPCHub) error {
	if len(s.CollectionsToUpdate) == 0 {
		return nil
	}

	for collection := range s.CollectionsToUpdate {
		resource, err := rpcClient.Resource(ctx, collection, types.CollectionStructType, s.height)
		if err != nil {
			return fmt.Errorf("failed to fetch collection resource: %w", err)
		}
		nft, err := parser.DecodeResource[parser.CollectionResource](resource.Resource.MoveResource)
		if err != nil {
			return fmt.Errorf("failed to decode collection resource: %w", err)
		}

		if existingCollection, exists := s.dbBatchInsert.Collections[collection]; exists {
			// Update existing collection
			existingCollection.URI = nft.Data.URI
			existingCollection.Description = nft.Data.Description
			existingCollection.Name = nft.Data.Name
			s.dbBatchInsert.Collections[collection] = existingCollection
		} else {
			return fmt.Errorf("collection not found: %s", collection)
		}
	}

	return nil
}

func (s *StateUpdateManager) updateNfts(ctx context.Context, rpcClient cosmosrpc.CosmosJSONRPCHub) error {
	if len(s.NftsToUpdate) == 0 {
		return nil
	}

	// TODO: remove query when 0x1::nft::Nft upgrade is done
	for nftId := range s.NftsToUpdate {
		resource, err := rpcClient.Resource(ctx, nftId, types.NftStructType, s.height)
		if err != nil {
			return fmt.Errorf("failed to fetch nft: %w", err)
		}

		nft, err := parser.DecodeResource[parser.NftResource](resource.Resource.MoveResource)
		if err != nil {
			return fmt.Errorf("failed to decode nft: %w", err)
		}

		if existingNft, exists := s.dbBatchInsert.Nfts[nftId]; exists {
			existingNft.URI = nft.Data.URI
			existingNft.Description = nft.Data.Description
			existingNft.TokenID = nft.Data.TokenID
			s.dbBatchInsert.Nfts[nftId] = existingNft
		} else {
			return fmt.Errorf("nft not found: %s", nftId)
		}
	}
	return nil
}
