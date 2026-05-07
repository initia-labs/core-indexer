package cosmosrpc

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const MAX_RETRY_COUNT = 3

// createTimeoutContext checks if the provided context already has a deadline or timeout.
// If it does, it returns the existing context and a no-op cancel function.
// If not, it creates a new context with the specified timeout.
func createTimeoutContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		// If context already has a deadline, return the existing context and a no-op cancel function.
		return ctx, func() {}
	}
	// Create a new context with the specified timeout.
	return context.WithTimeout(ctx, timeout)
}

func handleTimeoutError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("RPC: request timed out")
	}
	return err
}

func handleQuery[T any, C ActiveClient](ctx context.Context, timeout time.Duration, clients []C, queryFn func(context.Context, C) (*T, error)) (*T, error) {
	var lastError error
	for range MAX_RETRY_COUNT {
		for _, client := range clients {
			ctx, cancel := createTimeoutContext(ctx, timeout)
			defer cancel()
			result, err := queryFn(ctx, client)
			if err != nil {
				lastError = handleTimeoutError(err)
				continue
			}
			return result, nil
		}
	}
	return nil, fmt.Errorf("RPC: All RPC Clients failed to query. Last error: %v", lastError)
}
