package parser

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

func FindAllBech32Address(attr string) []string {
	return regexInitBech.FindAllString(attr, -1)
}

func FindAllHexAddress(attr string) []string {
	return regexHex.FindAllString(attr, -1)
}

func FindAllMoveHexAddress(attr string) []string {
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
