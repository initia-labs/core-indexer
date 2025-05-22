package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetColumns(t *testing.T) {
	columns := getColumns[TransactionEvent]()
	assert.Equal(t, []string{
		"transaction_hash",
		"block_height",
		"event_key",
		"event_value",
		"event_index",
	}, columns)
}
