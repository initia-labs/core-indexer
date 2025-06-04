package utils

import (
	"encoding/hex"
)

func IsHex(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}
