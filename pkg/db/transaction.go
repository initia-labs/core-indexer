package db

import (
	"fmt"
	"strings"
)

func GetTxID(hash string, blockHeight int64) string {
	return fmt.Sprintf("%s/%d", strings.ToUpper(hash), blockHeight)
}
