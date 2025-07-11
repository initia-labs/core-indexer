package proposal

import (
	"time"

	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	vmapi "github.com/initia-labs/movevm/api"

	"github.com/initia-labs/core-indexer/pkg/db"
)

type TxProcessor struct {
	txID string
}

type Processor struct {
	height                     int64
	int32Height                int32
	validatorMap               map[string]mstakingtypes.Validator
	newProposals               map[int32]string
	proposalStatusChanges      map[int32]db.ProposalStatus
	proposalDeposits           []db.ProposalDeposit
	proposalVotes              []db.ProposalVote
	proposalExpeditedChanges   map[int32]bool
	proposalEmergencyNextTally map[int32]*time.Time
	modulePublishedEvents      []db.ModuleHistory
	moduleProposals            []db.ModuleProposal
	newModules                 map[vmapi.ModuleInfoResponse]bool

	txProcessor *TxProcessor
}
