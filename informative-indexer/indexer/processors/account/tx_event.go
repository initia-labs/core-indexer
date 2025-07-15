package account

import (
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	movetypes "github.com/initia-labs/initia/x/move/types"

	"github.com/initia-labs/core-indexer/pkg/parser"
)

// handleEvent routes events to appropriate handlers based on event type
func (p *Processor) handleEvent(event abci.Event) error {
	for _, attr := range event.Attributes {
		var addrs []string

		switch {
		case event.Type == movetypes.EventTypeMove && attr.Key == movetypes.AttributeKeyData:
			addrs = append(addrs, parser.FindAllMoveHexAddress(attr.Value)...)

		default:
			for _, attrVal := range strings.Split(attr.Value, ",") {
				addrs = append(addrs, parser.FindAllBech32Address(attrVal)...)
				addrs = append(addrs, parser.FindAllHexAddress(attrVal)...)
			}
		}
		for _, addr := range addrs {
			accAddr, err := parser.AccAddressFromString(addr)
			if err != nil {
				continue // there might be invalid bech32 addresses so do not return error
			}
			p.txProcessor.relatedAccs = append(p.txProcessor.relatedAccs, accAddr)

			if event.Type == "message" {
				for _, attr := range event.Attributes {
					if attr.Key == sdk.AttributeKeySender {
						sender, err := sdk.AccAddressFromBech32(attr.Value)
						if err != nil {
							return err
						}
						p.txProcessor.sender = sender
					}
				}
			}
		}
	}
	return nil
}
