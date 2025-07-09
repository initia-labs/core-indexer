package utils

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/initia-labs/core-indexer/pkg/parser"
)

// FindAttribute finds the first attribute with the given key and returns its value
func FindAttribute(attributes []abci.EventAttribute, key string) (string, bool) {
	for _, attr := range attributes {
		if attr.Key == key {
			return attr.Value, true
		}
	}
	return "", false
}

func FindAttributeWithValue(attributes []abci.EventAttribute, key, value string) bool {
	for _, attr := range attributes {
		if attr.Key == key && attr.Value == value {
			return true
		}
	}
	return false
}

// HandleEventWithKey is a generic handler for events that need to decode data from a specific attribute
func HandleEventWithKey[T any](event abci.Event, key string, flag *bool, store func(T)) error {
	if value, found := FindAttribute(event.Attributes, key); found {
		e, err := parser.DecodeEvent[T](value)
		if err != nil {
			return fmt.Errorf("failed to decode event data: %w", err)
		}
		if flag != nil {
			*flag = true
		}
		store(e)
	}
	return nil
}
