package services

import (
	"context"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

const (
	MaxWorkers    = 10
	MaxRetries    = 3
	RetryDelay    = time.Second
	MaxRetryDelay = 10 * time.Second
	CacheSize     = 10000
)

// Task represents a work item for the worker pool
type Task struct {
	Index int
	Hash  string
}

// TaskResult represents the result of processing a task
type TaskResult struct {
	Index int
	Tx    *dto.TxByHashResponse
	Err   error
}

type GCSManager struct {
	MaxWorkers    int
	MaxRetries    int
	RetryDelay    time.Duration
	MaxRetryDelay time.Duration
	repo          repositories.TxRepositoryI
	cache         *lru.Cache[string, *dto.TxByHashResponse]
}

func NewGCSManager(repo repositories.TxRepositoryI) GCSManager {
	cache, _ := lru.New[string, *dto.TxByHashResponse](CacheSize)
	return GCSManager{
		repo:          repo,
		MaxWorkers:    MaxWorkers,
		MaxRetries:    MaxRetries,
		RetryDelay:    RetryDelay,
		MaxRetryDelay: MaxRetryDelay,
		cache:         cache,
	}
}

// retryWithBackoff executes a function with exponential backoff retry logic
func (g *GCSManager) retryWithBackoff(ctx context.Context, fn func() error) error {
	var lastErr error
	delay := g.RetryDelay

	for attempt := 0; attempt <= g.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		if attempt < g.MaxRetries {
			delay = min(time.Duration(float64(delay)*1.5), g.MaxRetryDelay)
		}
	}

	return lastErr
}

func (g *GCSManager) QueryTxs(hash []string) ([]*dto.TxByHashResponse, error) {
	if len(hash) == 0 {
		return []*dto.TxByHashResponse{}, nil
	}

	txs := make([]*dto.TxByHashResponse, len(hash))
	resultChan := make(chan TaskResult, len(hash))
	ctx := context.Background()

	// Check cache first and collect uncached hashes
	uncachedHashes := make([]int, 0)
	for idx, h := range hash {
		if cached, exists := g.cache.Get(h); exists {
			txs[idx] = cached
		} else {
			uncachedHashes = append(uncachedHashes, idx)
		}
	}

	// Launch goroutines only for uncached hashes
	for _, idx := range uncachedHashes {
		h := hash[idx]
		go func(index int, hash string) {
			var tx *dto.TxByHashResponse
			var err error

			err = g.retryWithBackoff(ctx, func() error {
				tx, err = g.repo.GetTxByHash(hash)
				return err
			})

			// Cache successful results
			if err == nil && tx != nil {
				g.cache.Add(hash, tx)
			}

			resultChan <- TaskResult{
				Index: index,
				Tx:    tx,
				Err:   err,
			}
		}(idx, h)
	}

	// Collect results for uncached hashes
	for i := 0; i < len(uncachedHashes); i++ {
		result := <-resultChan
		if result.Err != nil {
			return nil, result.Err
		}
		txs[result.Index] = result.Tx
	}

	return txs, nil
}
