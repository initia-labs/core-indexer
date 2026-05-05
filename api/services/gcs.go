package services

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

const (
	MaxWorkers    = 10
	MaxRetries    = 3
	RetryDelay    = time.Second
	MaxRetryDelay = 10 * time.Second
	CacheBytes    = 64 * 1024 * 1024
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

type txCacheEntry struct {
	hash string
	tx   *dto.TxByHashResponse
	size int64
}

type txResponseCache struct {
	mu       sync.Mutex
	maxBytes int64
	used     int64
	items    map[string]*list.Element
	order    *list.List
}

func newTxResponseCache(maxBytes int64) *txResponseCache {
	if maxBytes <= 0 {
		return nil
	}

	return &txResponseCache{
		maxBytes: maxBytes,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

func (c *txResponseCache) Get(hash string) (*dto.TxByHashResponse, bool) {
	if c == nil {
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	element, exists := c.items[hash]
	if !exists {
		return nil, false
	}

	c.order.MoveToFront(element)
	entry := element.Value.(*txCacheEntry)
	return entry.tx, true
}

func (c *txResponseCache) Add(hash string, tx *dto.TxByHashResponse) {
	if c == nil || tx == nil {
		return
	}

	size := tx.CacheSizeBytes
	if size <= 0 || size > c.maxBytes {
		c.Remove(hash)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.items[hash]; exists {
		entry := element.Value.(*txCacheEntry)
		c.used += size - entry.size
		entry.tx = tx
		entry.size = size
		c.order.MoveToFront(element)
	} else {
		entry := &txCacheEntry{hash: hash, tx: tx, size: size}
		c.items[hash] = c.order.PushFront(entry)
		c.used += size
	}

	for c.used > c.maxBytes {
		c.removeOldest()
	}
}

func (c *txResponseCache) Remove(hash string) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.items[hash]; exists {
		c.removeElement(element)
	}
}

func (c *txResponseCache) removeOldest() {
	element := c.order.Back()
	if element != nil {
		c.removeElement(element)
	}
}

func (c *txResponseCache) removeElement(element *list.Element) {
	entry := element.Value.(*txCacheEntry)
	delete(c.items, entry.hash)
	c.used -= entry.size
	c.order.Remove(element)
}

type GCSManager struct {
	MaxWorkers    int
	MaxRetries    int
	RetryDelay    time.Duration
	MaxRetryDelay time.Duration
	repo          repositories.TxRepositoryI
	cache         *txResponseCache
}

func NewGCSManager(repo repositories.TxRepositoryI) GCSManager {
	return NewGCSManagerWithConfig(repo, CacheBytes, MaxWorkers)
}

func NewGCSManagerWithConfig(repo repositories.TxRepositoryI, cacheBytes int64, maxWorkers int) GCSManager {
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	return GCSManager{
		repo:          repo,
		MaxWorkers:    maxWorkers,
		MaxRetries:    MaxRetries,
		RetryDelay:    RetryDelay,
		MaxRetryDelay: MaxRetryDelay,
		cache:         newTxResponseCache(cacheBytes),
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

func (g *GCSManager) QueryTxs(ctx context.Context, hashes []string) ([]*dto.TxByHashResponse, error) {
	if len(hashes) == 0 {
		return []*dto.TxByHashResponse{}, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	txs := make([]*dto.TxByHashResponse, len(hashes))
	resultChan := make(chan TaskResult, len(hashes))

	// Check cache first and collect uncached hashes
	uncachedHashes := make([]int, 0)
	for idx, h := range hashes {
		if cached, exists := g.cache.Get(h); exists {
			txs[idx] = cached
		} else {
			uncachedHashes = append(uncachedHashes, idx)
		}
	}

	workerCount := min(g.MaxWorkers, len(uncachedHashes))
	taskChan := make(chan int)

	for worker := 0; worker < workerCount; worker++ {
		go func() {
			for index := range taskChan {
				hash := hashes[index]

				var tx *dto.TxByHashResponse
				var err error

				err = g.retryWithBackoff(ctx, func() error {
					tx, err = g.repo.GetTxByHash(ctx, hash)
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
			}
		}()
	}

	go func() {
		defer close(taskChan)
		for _, idx := range uncachedHashes {
			select {
			case <-ctx.Done():
				return
			case taskChan <- idx:
			}
		}
	}()

	for i := 0; i < len(uncachedHashes); i++ {
		result := <-resultChan
		if result.Err != nil {
			cancel()
			return nil, result.Err
		}
		txs[result.Index] = result.Tx
	}

	return txs, nil
}
