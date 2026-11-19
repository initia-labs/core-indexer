package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
	coredb "github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RPCConfig struct {
	Chain        string
	RPCEndpoints string
	Timeout      time.Duration
	StartHeight  int64
}

func RunRPC(ctx context.Context, c RPCConfig) error {
	logger := log.With().Str("component", "transfer-alerts-rpc").Str("chain", c.Chain).Str("id", os.Getenv("ID")).Logger()
	ctx = logger.WithContext(ctx)
	if c.Chain == "" || os.Getenv("DB_CONNECTION_STRING") == "" {
		return fmt.Errorf("CHAIN and DB_CONNECTION_STRING are required")
	}
	if c.StartHeight < 0 || c.Timeout <= 0 {
		return fmt.Errorf("start height must be nonnegative and RPC timeout must be positive")
	}
	var endpoints mq.RPCEndpoints
	if err := json.Unmarshal([]byte(c.RPCEndpoints), &endpoints); err != nil {
		return fmt.Errorf("decode RPC_ENDPOINTS: %w", err)
	}
	if len(endpoints.RPCs) == 0 {
		return fmt.Errorf("RPC_ENDPOINTS must contain at least one RPC")
	}
	clients := make([]cosmosrpc.ClientConfig, 0, len(endpoints.RPCs))
	for _, endpoint := range endpoints.RPCs {
		clients = append(clients, cosmosrpc.ClientConfig{URL: endpoint.URL, ClientOption: &cosmosrpc.ClientOption{CustomHeaders: endpoint.Headers}})
	}
	hub := cosmosrpc.NewHub(clients, &logger, c.Timeout)
	logger.Info().Int64("start_height", c.StartHeight).Msg("Starting transfer alert RPC reader")
	db, err := coredb.NewClient(os.Getenv("DB_CONNECTION_STRING"))
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	return runRPC(ctx, c, db, hub)
}

func runRPC(ctx context.Context, c RPCConfig, db *gorm.DB, hub *cosmosrpc.Hub) error {
	logger := log.Ctx(ctx)
	var height int64
	initialized := false
	lastStatus := time.Now()
	for ctx.Err() == nil {
		if err := hub.Rebalance(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logger.Warn().Err(err).Msg("Transfer alert RPC unavailable; retrying")
			if err := wait(ctx, 2*time.Second); err != nil {
				return err
			}
			continue
		}
		head := hub.GetActiveClients()[0].Height
		if !initialized {
			var err error
			height, err = initRPCCursor(ctx, db, c.Chain, c.StartHeight, head)
			if err != nil {
				return err
			}
			initialized = true
			logger.Info().Int64("checkpoint_height", height).Int64("next_height", height+1).Int64("rpc_height", head).Msg("Transfer alert RPC cursor ready")
		}
		for height < head && ctx.Err() == nil {
			if err := ingestRPCBlock(ctx, db, hub, c.Chain, height+1); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				logger.Error().Err(err).Int64("height", height+1).Msg("Failed to ingest transfer alert block; retrying")
				break
			}
			height++
			logger.Info().Int64("height", height).Int64("rpc_height", head).Msg("Ingested transfer alert RPC block")
		}
		if time.Since(lastStatus) >= 30*time.Second {
			logger.Info().Int64("checkpoint_height", height).Int64("rpc_height", head).Msg("Transfer alert RPC polling active")
			lastStatus = time.Now()
		}
		if err := wait(ctx, 2*time.Second); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func initRPCCursor(ctx context.Context, db *gorm.DB, chain string, start, head int64) (int64, error) {
	if start == 0 {
		start = head
	}
	cursor := coredb.TransferAlertRPCCursor{ChainID: chain, Height: start - 1}
	if err := db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&cursor).Error; err != nil {
		return 0, err
	}
	if err := db.WithContext(ctx).Where("chain_id = ?", chain).First(&cursor).Error; err != nil {
		return 0, err
	}
	return cursor.Height, nil
}

func ingestRPCBlock(ctx context.Context, db *gorm.DB, hub *cosmosrpc.Hub, chain string, height int64) error {
	var message mq.BlockResultMsg
	if err := hub.WithBlockResults(ctx, height, func(block *coretypes.ResultBlock, results *coretypes.ResultBlockResults) error {
		if block.Block.ChainID != chain {
			return fmt.Errorf("RPC block belongs to chain %q, configured chain is %q", block.Block.ChainID, chain)
		}
		if len(block.Block.Txs) != len(results.TxsResults) {
			return fmt.Errorf("RPC transaction results are incomplete: block has %d transactions, results have %d", len(block.Block.Txs), len(results.TxsResults))
		}
		data, err := mq.NewBlockResultMsgBytes(block, results)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, &message)
	}); err != nil {
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := coredb.InsertTransferAlertBatches(ctx, tx, chain, &message); err != nil {
			return err
		}
		return tx.Exec(`UPDATE transfer_alert_rpc_cursors SET height = GREATEST(height, ?), updated_at = now()
			WHERE chain_id = ?`, height, chain).Error
	})
}
