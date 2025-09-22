package parser

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
)

const (
	InitBech32Regex = "^init1(?:[a-z0-9]{38}|[a-z0-9]{58})$"
	InitHexRegex    = "^0x(?:[a-fA-F0-9]{1,64})$"
	MoveHexRegex    = "0x(?:[a-fA-F0-9]{1,64})"
)

var (
	regexInitBech = regexp.MustCompile(InitBech32Regex)
	regexHex      = regexp.MustCompile(InitHexRegex)
	regexMoveHex  = regexp.MustCompile(MoveHexRegex)
)

func findAllBech32Address(attr string) []string {
	return regexInitBech.FindAllString(attr, -1)
}

func findAllHexAddress(attr string) []string {
	return regexHex.FindAllString(attr, -1)
}

func findAllMoveHexAddress(attr string) []string {
	return regexMoveHex.FindAllString(attr, -1)
}

func AccAddressFromString(addrStr string) (sdk.AccAddress, error) {
	if !strings.HasPrefix(addrStr, "0x") {
		return sdk.AccAddressFromBech32(addrStr)
	}

	hexStr := strings.ToLower(strings.TrimLeft(strings.TrimPrefix(addrStr, "0x"), "0"))

	if len(hexStr) <= 40 {
		hexStr = strings.Repeat("0", 40-len(hexStr)) + hexStr
	} else if len(hexStr) <= 64 {
		hexStr = strings.Repeat("0", 64-len(hexStr)) + hexStr
	} else {
		return nil, fmt.Errorf("invalid address string: %s", addrStr)
	}

	return hex.DecodeString(hexStr)
}

func BytesToHex(b []byte) string {
	return hex.EncodeToString(b)
}

func BytesToHexWithPrefix(b []byte) string {
	return "0x" + strings.TrimLeft(hex.EncodeToString(b), "0")
}

func GrepAddressesFromEvents(events []abci.Event) (grepped []sdk.AccAddress, err error) {
	for _, event := range events {
		for _, attr := range event.Attributes {
			var addrs []string

			switch {
			case event.Type == movetypes.EventTypeMove && attr.Key == movetypes.AttributeKeyData:
				addrs = append(addrs, findAllMoveHexAddress(attr.Value)...)

			default:
				for _, attrVal := range strings.Split(attr.Value, ",") {
					addrs = append(addrs, findAllBech32Address(attrVal)...)
					addrs = append(addrs, findAllHexAddress(attrVal)...)
				}
			}

			for _, addr := range addrs {
				accAddr, err := AccAddressFromString(addr)
				if err != nil {
					continue // there might be invalid bech32 addresses so do not return error
				}
				grepped = append(grepped, accAddr)
			}
		}
	}

	return
}

func GrepSenderFromEvents(events []abci.Event) (sdk.AccAddress, error) {
	for _, event := range events {
		if event.Type == "message" {
			for _, attr := range event.Attributes {
				if attr.Key == sdk.AttributeKeySender {
					return sdk.AccAddressFromBech32(attr.Value)
				}
			}
		}
	}

	return nil, fmt.Errorf("sender not found")
}
