package utils

import (
	"encoding/hex"
	"regexp"
	"strings"
)

func IsHex(s string) bool {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
	if len(s)%2 != 0 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func IsTxHash(txhash string) bool {
	hexRegex := regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
	return hexRegex.MatchString(txhash)
}
