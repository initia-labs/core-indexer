package move

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/types"
	"github.com/initia-labs/core-indexer/informative-indexer/indexer/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

func (p *Processor) handleEvent(event abci.Event) error {
	switch event.Type {
	case movetypes.EventTypeExecute:
		return p.handleMoveExecuteEvent(event)
	case movetypes.EventTypeMove:
		return p.handleMoveEvent(event)
	default:
		return nil
	}
}

// handleMoveExecuteEvent processes Move execution events, recording module executions
func (p *Processor) handleMoveExecuteEvent(event abci.Event) error {
	moduleAddrs := make([]string, 0)
	moduleNames := make([]string, 0)

	for _, attr := range event.Attributes {
		switch attr.Key {
		case movetypes.AttributeKeyModuleAddr:
			moduleAddrs = append(moduleAddrs, attr.Value)
		case movetypes.AttributeKeyModuleName:
			moduleNames = append(moduleNames, attr.Value)
		}
	}

	for idx, moduleAddr := range moduleAddrs {
		addr, err := vmtypes.NewAccountAddress(moduleAddr)
		if err != nil {
			return fmt.Errorf("invalid module address: %w", err)
		}
		p.txProcessor.modulesInTx[vmapi.ModuleInfoResponse{Address: addr, Name: moduleNames[idx]}] = false
	}
	return nil
}

// handleMoveEvent processes Move-specific events, routing them to appropriate handlers
func (p *Processor) handleMoveEvent(event abci.Event) error {
	if value, found := utils.FindAttribute(event.Attributes, movetypes.AttributeKeyTypeTag); found {
		switch value {
		case types.ModulePublishedEventKey:
			return p.handlePublishEvent(event)
		case types.CreateCollectionEventKey:
			return p.handleCollectionCreateEvent(event)
		case types.CollectionMutationEventKey:
			return p.handleCollectionMutationEvent(event)
		case types.NftMutationEventKey:
			return p.handleNftMutationEvent(event)
		case types.CollectionMintEventKey:
			return p.handleCollectionMintEvent(event)
		case types.CollectionBurnEventKey:
			return p.handleCollectionBurnEvent(event)
		case types.ObjectCreateEventKey:
			return p.handleObjectCreateEvent(event)
		case types.ObjectTransferEventKey:
			return p.handleObjectTransferEvent(event)
		}
	}
	return nil
}

// handlePublishEvent processes module publish events, recording new modules
func (p *Processor) handlePublishEvent(event abci.Event) error {
	p.txProcessor.txData.IsMovePublish = true
	if value, found := utils.FindAttribute(event.Attributes, movetypes.AttributeKeyData); found {
		module, upgradePolicy, err := parser.DecodePublishModuleData(value)
		if err != nil {
			return fmt.Errorf("failed to decode publish module data: %w", err)
		}
		p.newModules[module] = p.txProcessor.txData.ID
		p.modulePublishedEvents = append(p.modulePublishedEvents, db.ModuleHistory{
			ModuleID:      db.GetModuleID(module),
			BlockHeight:   p.txProcessor.txData.BlockHeight,
			Remark:        db.JSON("{}"),
			ProposalID:    nil,
			TxID:          &p.txProcessor.txData.ID,
			UpgradePolicy: db.GetUpgradePolicy(upgradePolicy),
		})
	}
	return nil
}

// handleCollectionCreateEvent processes collection creation events
func (p *Processor) handleCollectionCreateEvent(event abci.Event) error {
	return utils.HandleEventWithKey(event, movetypes.AttributeKeyData, &p.txProcessor.txData.IsCollectionCreate, func(e types.CreateCollectionEvent) {
		// TODO: can be filled with a new event
		p.newCollections[e.Collection] = db.Collection{
			ID:          e.Collection,
			Creator:     e.Creator,
			Name:        e.Name,
			BlockHeight: p.Height,
			URI:         "",
			Description: "",
		}

		p.collectionTransactions = append(p.collectionTransactions, db.CollectionTransaction{
			CollectionID:       e.Collection,
			IsCollectionCreate: true,
			BlockHeight:        p.Height,
			TxID:               p.txProcessor.txData.ID,
			NftID:              nil,
		})
	})
}

// handleCollectionMintEvent processes NFT minting events
func (p *Processor) handleCollectionMintEvent(event abci.Event) error {
	return utils.HandleEventWithKey(event, movetypes.AttributeKeyData, &p.txProcessor.txData.IsNftMint, func(e types.CollectionMintEvent) {
		// TODO: can be filled with a new event
		p.newNfts[e.Nft] = db.Nft{
			TokenID:    e.TokenID,
			Remark:     db.JSON("{}"),
			ProposalID: nil,
			TxID:       &p.txProcessor.txData.ID,
			ID:         e.Nft,
			Collection: e.Collection,
			IsBurned:   false,
			Owner:      p.objectOwners[e.Nft],

			// temporary fields
			Description: "",
			URI:         "",
		}

		p.mintedNftTransactions = append(p.mintedNftTransactions, db.NewNftMintTransaction(e.Nft, p.txProcessor.txData.ID, p.Height))
		p.collectionTransactions = append(p.collectionTransactions, db.CollectionTransaction{
			CollectionID: e.Collection,
			IsNftMint:    true,
			BlockHeight:  p.Height,
			TxID:         p.txProcessor.txData.ID,
			NftID:        &e.Nft,
		})
	})
}

func (p *Processor) handleCollectionBurnEvent(event abci.Event) error {
	return utils.HandleEventWithKey(event, movetypes.AttributeKeyData, &p.txProcessor.txData.IsNftBurn, func(e types.CollectionBurnEvent) {
		p.burnedNftTransactions = append(p.burnedNftTransactions, db.NewNftBurnTransaction(e.Nft, p.txProcessor.txData.ID, p.Height))
		p.collectionTransactions = append(p.collectionTransactions, db.CollectionTransaction{
			CollectionID: e.Collection,
			IsNftBurn:    true,
			BlockHeight:  p.Height,
			TxID:         p.txProcessor.txData.ID,
			NftID:        &e.Nft,
		})
	})
}

// handleCollectionMutationEvent processes collection mutation events
func (p *Processor) handleCollectionMutationEvent(event abci.Event) error {
	return utils.HandleEventWithKey(event, movetypes.AttributeKeyData, nil, func(e types.CollectionMutationEvent) {
		p.collectionMutationEvents = append(p.collectionMutationEvents, db.CollectionMutationEvent{
			CollectionID:     e.Collection,
			MutatedFieldName: e.MutatedFieldName,
			OldValue:         e.OldValue,
			NewValue:         e.NewValue,
			Remark:           db.JSON("{}"),
			TxID:             p.txProcessor.txData.ID,
			BlockHeight:      p.Height,
		})
	})
}

func (p *Processor) handleNftMutationEvent(event abci.Event) error {
	return utils.HandleEventWithKey(event, movetypes.AttributeKeyData, nil, func(e types.NftMutationEvent) {
		p.nftMutationEvents = append(p.nftMutationEvents, db.NftMutationEvent{
			NftID:            e.Nft,
			MutatedFieldName: e.MutatedFieldName,
			OldValue:         e.OldValue,
			NewValue:         e.NewValue,
			Remark:           db.JSON("{}"),
			TxID:             p.txProcessor.txData.ID,
			BlockHeight:      p.Height,
		})
	})
}

func (p *Processor) handleObjectCreateEvent(event abci.Event) error {
	return utils.HandleEventWithKey(event, movetypes.AttributeKeyData, nil, func(e types.ObjectCreateEvent) {
		p.objectOwners[e.Object] = e.Owner
	})
}

// handleObjectTransferEvent processes object transfer events
func (p *Processor) handleObjectTransferEvent(event abci.Event) error {
	return utils.HandleEventWithKey(event, movetypes.AttributeKeyData, &p.txProcessor.txData.IsNftTransfer, func(e types.ObjectTransferEvent) {
		p.objectOwners[e.Object] = e.To

		// in case the nft is created and tranferred in the same tx
		if nft, ok := p.newNfts[e.Object]; ok {
			nft.Owner = e.To
			p.newNfts[e.Object] = nft
		}
	})
}
