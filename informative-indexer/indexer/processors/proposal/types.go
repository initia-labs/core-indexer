package proposal

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	vmapi "github.com/initia-labs/movevm/api"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/processors"
	"github.com/initia-labs/core-indexer/pkg/db"
)

var _ processors.Processor = &Processor{}

type TxProcessor struct {
	txData *db.Transaction
}

type Processor struct {
	processors.BaseProcessor
	newProposals               map[int32]string
	proposalStatusChanges      map[int32]db.ProposalStatus
	proposalDeposits           []db.ProposalDeposit
	totalDepositChanges        map[int32][]sdk.Coin
	proposalVotes              []db.ProposalVote
	proposalEmergencyNextTally map[int32]*time.Time
	modulePublishedEvents      []db.ModuleHistory
	moduleProposals            []db.ModuleProposal
	newModules                 map[vmapi.ModuleInfoResponse]bool

	txProcessor *TxProcessor
}
