package ibc

import (
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

func (p *Processor) handleEvent(event abci.Event) error {
	switch event.Type {
	case sdk.EventTypeMessage:
		return p.handleMessageEvents(event)
	case ibcchanneltypes.EventTypeSendPacket:
		p.txProcessor.txData.IsIbc = true
		return nil
	default:
		return nil
	}
}

func (p *Processor) handleMessageEvents(event abci.Event) error {
	for _, attribute := range event.Attributes {
		if attribute.Key == sdk.AttributeKeyAction {
			if strings.HasPrefix(attribute.Value, "/ibc") {
				p.txProcessor.txData.IsIbc = true
			}
		}
	}
	return nil
}
