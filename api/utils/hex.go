package utils

import (
	"encoding/hex"
)

func IsHex(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}

func BytesToHex(b string) string {
	return "\\x" + hex.EncodeToString([]byte(b))
}
