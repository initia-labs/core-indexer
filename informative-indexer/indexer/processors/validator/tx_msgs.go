package validator

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"

	"github.com/initia-labs/core-indexer/pkg/db"
)

func (p *Processor) handleMsgs(msg sdk.Msg) {
	switch msg := msg.(type) {
	case *slashingtypes.MsgUnjail:
		p.validators[msg.ValidatorAddr] = true
		p.slashEvents = append(p.slashEvents, db.ValidatorSlashEvent{
			ValidatorAddress: msg.ValidatorAddr,
			BlockHeight:      p.Height,
			Type:             string(db.Unjailed),
		})
	}
}
