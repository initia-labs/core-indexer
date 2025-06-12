package flusher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmosgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/initia-labs/initia/app/params"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

// StateUpdateManager tracks entities that need to be synchronized with the blockchain state
// through RPC queries. It maintains sets of validators and modules that have been modified
// and need their latest state to be fetched from the chain.
type StateUpdateManager struct {
	// validators tracks validator addresses that need their state to be synchronized
	validators map[string]bool

	// modules tracks Move modules that need their state to be synchronized.
	// The string pointer value is the transaction hash where the module was published.
	// A nil value indicates the module was not published in the current transaction.
	modules map[vmapi.ModuleInfoResponse]*string

	// dbBatchInsert handles database batch operations
	dbBatchInsert *DBBatchInsert

	// encodingConfig is used for encoding/decoding validator information
	encodingConfig *params.EncodingConfig

	// height is the height of the block to be used for RPC queries
	height *int64

	proposalsToUpdate   map[int32]string
	collectionsToUpdate map[string]bool
	nftsToUpdate        map[string]bool
}

func NewStateUpdateManager(
	dbBatchInsert *DBBatchInsert,
	encodingConfig *params.EncodingConfig,
	height *int64,
) *StateUpdateManager {
	return &StateUpdateManager{
		validators:          make(map[string]bool),
		modules:             make(map[vmapi.ModuleInfoResponse]*string),
		dbBatchInsert:       dbBatchInsert,
		encodingConfig:      encodingConfig,
		height:              height,
		collectionsToUpdate: make(map[string]bool),
		nftsToUpdate:        make(map[string]bool),
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
	for proposalID, txID := range s.proposalsToUpdate {
		proposal, err := rpcClient.Proposal(ctx, proposalID, s.height)
		if err != nil {
			return fmt.Errorf("failed to fetch proposal info: %w", err)
		}

		proposalInfo := proposal.Proposal
		rawProposalMsgs, _ := proposalInfo.GetMsgs()

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

		contentJSON, err := json.Marshal(map[string]any{
			"messages": proposalMsgs,
			"metadata": proposalInfo.GetMetadata(),
		})
		if err != nil {
			return fmt.Errorf("failed to marshal content: %w", err)
		}

		totalDepositJSON, err := json.Marshal(proposalInfo.GetTotalDeposit())
		if err != nil {
			return fmt.Errorf("failed to marshal total deposit: %w", err)
		}

		proposalTypesJSON, err := json.Marshal(proposalTypes)
		if err != nil {
			return fmt.Errorf("failed to marshal proposal types: %w", err)
		}

		s.dbBatchInsert.proposals[proposalID] = db.Proposal{
			ID:                     proposalID,
			Title:                  proposalInfo.GetTitle(),
			Description:            proposalInfo.GetSummary(),
			Status:                 proposalInfo.GetStatus().String(),
			SubmitTime:             *proposalInfo.SubmitTime,
			DepositEndTime:         *proposalInfo.DepositEndTime,
			TotalDeposit:           db.JSON(totalDepositJSON),
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
	return nil
}

func (s *StateUpdateManager) updateValidators(ctx context.Context, rpcClient cosmosrpc.CosmosJSONRPCHub) error {
	validatorAddresses := make([]string, 0, len(s.validators))
	for addr := range s.validators {
		validatorAddresses = append(validatorAddresses, addr)
	}

	return s.syncValidators(ctx, rpcClient, validatorAddresses)
}

func (s *StateUpdateManager) updateModules(ctx context.Context, rpcClient cosmosrpc.CosmosJSONRPCHub) error {
	modules := make([]vmapi.ModuleInfoResponse, 0, len(s.modules))
	publishTxIds := make([]*string, 0, len(s.modules))
	for module, publishTxId := range s.modules {
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
			return errors.Join(ErrorNonRetryable, err)
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

		publishTxId := ""
		if publishTxIds[idx] != nil {
			publishTxId = *publishTxIds[idx]
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
	if len(s.collectionsToUpdate) == 0 {
		return nil
	}

	for collection := range s.collectionsToUpdate {
		resource, err := rpcClient.Resource(ctx, collection, CollectionStructType, s.height)
		if err != nil {
			return fmt.Errorf("failed to fetch collection resource: %w", err)
		}
		nft, err := parser.DecodeResource[parser.CollectionResource](resource.Resource.MoveResource)
		if err != nil {
			return fmt.Errorf("failed to decode collection resource: %w", err)
		}

		if existingCollection, exists := s.dbBatchInsert.collections[collection]; exists {
			// Update existing collection
			existingCollection.URI = nft.Data.URI
			existingCollection.Description = nft.Data.Description
			existingCollection.Name = nft.Data.Name
			s.dbBatchInsert.collections[collection] = existingCollection
		} else {
			return fmt.Errorf("collection not found: %s", collection)
		}
	}

	return nil
}

func (s *StateUpdateManager) updateNfts(ctx context.Context, rpcClient cosmosrpc.CosmosJSONRPCHub) error {
	if len(s.nftsToUpdate) == 0 {
		return nil
	}

	// TODO: remove query when 0x1::nft::Nft upgrade is done
	for nftId := range s.nftsToUpdate {
		resource, err := rpcClient.Resource(ctx, nftId, NftStructType, s.height)
		if err != nil {
			return fmt.Errorf("failed to fetch nft: %w", err)
		}

		nft, err := parser.DecodeResource[parser.NftResource](resource.Resource.MoveResource)
		if err != nil {
			return fmt.Errorf("failed to decode nft: %w", err)
		}

		if existingNft, exists := s.dbBatchInsert.nfts[nftId]; exists {
			existingNft.URI = nft.Data.URI
			existingNft.Description = nft.Data.Description
			existingNft.TokenID = nft.Data.TokenID
			s.dbBatchInsert.nfts[nftId] = existingNft
		} else {
			return fmt.Errorf("nft not found: %s", nftId)
		}
	}
	return nil
}
