package types

import "github.com/initia-labs/core-indexer/pkg/db"

type StateUpdateProposal struct {
	ID             int32
	Status         db.ProposalStatus
	ResolvedHeight *int32
}
