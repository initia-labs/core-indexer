package move

import (
	vmapi "github.com/initia-labs/movevm/api"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/processors"
	"github.com/initia-labs/core-indexer/pkg/db"
)

var _ processors.Processor = &Processor{}

type TxProcessor struct {
	txData *db.Transaction
	// Event type flags - a transaction can have multiple event types
	modulesInTx map[vmapi.ModuleInfoResponse]bool
}

type Processor struct {
	processors.BaseProcessor
	// Event data collections
	newModules             map[vmapi.ModuleInfoResponse]string
	moduleTransactions     []db.ModuleTransaction
	newCollections         map[string]db.Collection
	collectionTransactions []db.CollectionTransaction
	newNfts                map[string]db.Nft
	mintedNftTransactions  []db.NftTransaction
	burnedNftTransactions  []db.NftTransaction
	objectOwners           map[string]string

	modulePublishedEvents    []db.ModuleHistory
	collectionMutationEvents []db.CollectionMutationEvent
	nftMutationEvents        []db.NftMutationEvent

	txProcessor *TxProcessor
}
