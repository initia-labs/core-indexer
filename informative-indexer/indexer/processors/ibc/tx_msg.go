package ibc

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (p *Processor) handleMsg(msg sdk.Msg) {
	if strings.HasPrefix(sdk.MsgTypeURL(msg), "/ibc") {
		p.txProcessor.txData.IsIbc = true
	}
}
