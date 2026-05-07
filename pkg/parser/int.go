package parser

import (
	"fmt"
	"strconv"
)

func ParseInt32(str string) (int32, error) {
	value, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse int32: %w", err)
	}
	return int32(value), nil
}
