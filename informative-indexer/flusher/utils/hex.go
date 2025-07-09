package utils

import (
	"encoding/hex"
	"fmt"
)

func DecodeHexToHash(hexString string) ([32]byte, error) {
	bytes, err := hex.DecodeString(hexString)
	if err != nil {
		return [32]byte{}, err
	}

	if len(bytes) != 32 {
		return [32]byte{}, fmt.Errorf("expected 32 bytes, got %d", len(bytes))
	}

	var hash [32]byte
	copy(hash[:], bytes)
	return hash, nil
}
