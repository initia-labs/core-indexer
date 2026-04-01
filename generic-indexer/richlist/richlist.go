package richlist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

const ConsumerName = "rich-list"

const (
	MoveMetadataTypeTag           = "0x1::fungible_asset::Metadata"
	MoveDepositEventTypeTag       = "0x1::fungible_asset::DepositEvent"
	MoveDepositOwnerEventTypeTag  = "0x1::fungible_asset::DepositOwnerEvent"
	MoveWithdrawEventTypeTag      = "0x1::fungible_asset::WithdrawEvent"
	MoveWithdrawOwnerEventTypeTag = "0x1::fungible_asset::WithdrawOwnerEvent"
)

type MoveDepositEvent struct {
	StoreAddr    string `json:"store_addr"`
	MetadataAddr string `json:"metadata_addr"`
	Amount       string `json:"amount"`
}

type MoveDepositOwnerEvent struct {
	Owner string `json:"owner"`
}

type MoveWithdrawEvent struct {
	StoreAddr    string `json:"store_addr"`
	MetadataAddr string `json:"metadata_addr"`
	Amount       string `json:"amount"`
}

type MoveWithdrawOwnerEvent struct {
	Owner string `json:"owner"`
}

var moveDenomCache sync.Map

type moveFungibleAssetMetadata struct {
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

func (c *Consumer) getMoveDenomByMetadataAddr(ctx context.Context, metadataAddr string) (string, error) {
	key := strings.ToLower(metadataAddr)
	if cached, ok := moveDenomCache.Load(key); ok {
		return cached.(string), nil
	}

	resourceResponse, err := c.RPCClient.Resource(ctx, metadataAddr, MoveMetadataTypeTag, nil)
	if err != nil {
		return "", err
	}

	var metadata moveFungibleAssetMetadata
	if err := json.Unmarshal([]byte(resourceResponse.MoveResource), &metadata); err != nil {
		return "", err
	}

	var denom string
	if metadata.Decimals == 0 && metadata.Symbol != "" {
		denom = strings.ToLower(metadata.Symbol)
	} else {
		denom = strings.Replace(metadataAddr, "0x", "move/", 1)
	}

	moveDenomCache.Store(key, denom)
	return denom, nil
}

type BalanceChangeKey struct {
	Denom string
	Addr  string
}

// KafkaMessage is the message format produced by the sweeper for the rich list topic.
type KafkaMessage struct {
	Height int64         `json:"height"`
	Rows   []db.RichList `json:"rows"`
}

// Consumer consumes from a Kafka topic produced by the sweeper. When no rich_list_status row exists it inits from LCD at h-1 then applies Kafka for h; otherwise it applies Kafka for h.
type Consumer struct {
	Consumer          *mq.Consumer
	DB                *gorm.DB
	Logger            *zerolog.Logger
	RPCClient         cosmosrpc.CosmosJSONRPCHub
	RebalanceInterval int64
	Chain             string
	Topic             string
	ConsumerGroup     string
}

type IndexerConfig struct {
	// Worker ID. There supposed to be multiple workers running in parallel
	ID string

	RPCEndpoints string
	NumWorkers   int64

	// Chain ID to sweep
	Chain string

	DBConnectionString string

	// Kafka config
	KafkaBootstrapServer    string
	KafkaBlockTopic         string
	KafkaTxResponseTopic    string
	KafkaRichListTopic      string
	KafkaAPIKey             string
	KafkaAPISecret          string
	KafkaBlockConsumerGroup string

	// Claim check config
	ClaimCheckThresholdInMB       int64
	BlockClaimCheckBucket         string
	LCDTxResponseClaimCheckBucket string
	BlockResultsClaimCheckBucket  string

	// AWS
	AWSAccessKey string
	AWSSecretKey string

	// Functionality control
	DisableLCDTXResponse              bool
	DisableIndexingAccountTransaction bool

	Environment         string
	RebalanceInterval   int64
	RPCTimeOutInSeconds int64

	SentryDSN                string
	CommitSHA                string
	SentryProfilesSampleRate float64
	SentryTracesSampleRate   float64
}

// NewConsumer creates a rich list Kafka consumer. If LCDBaseURL is empty, init step will fail until LCD is configured.
func NewConsumer(config *IndexerConfig) (*Consumer, error) {
	if config.RPCEndpoints == "" {
		return nil, errors.New("RPC: no RPC endpoints provided")
	}

	topic := config.KafkaRichListTopic
	if topic == "" {
		topic = config.KafkaBlockTopic
	}
	consumerGroup := config.KafkaBlockConsumerGroup
	chain := config.Chain

	c, err := mq.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":    config.KafkaBootstrapServer,
		"group.id":             consumerGroup,
		"client.id":            consumerGroup + "-" + config.ID,
		"enable.auto.commit":   false,
		"auto.offset.reset":    "earliest",
		"security.protocol":    "SASL_SSL",
		"sasl.mechanisms":      "PLAIN",
		"sasl.username":        config.KafkaAPIKey,
		"sasl.password":        config.KafkaAPISecret,
		"max.poll.interval.ms": 600000,
	})
	if err != nil {
		return nil, fmt.Errorf("create kafka consumer: %w", err)
	}

	var rpcEndpoints mq.RPCEndpoints
	if err := json.Unmarshal([]byte(config.RPCEndpoints), &rpcEndpoints); err != nil {
		return nil, fmt.Errorf("RPC: error unmarshalling RPC endpoints: %w", err)
	}
	clientConfigs := make([]cosmosrpc.ClientConfig, 0)
	for _, rpc := range rpcEndpoints.RPCs {
		clientConfigs = append(clientConfigs, cosmosrpc.ClientConfig{
			URL:          rpc.URL,
			ClientOption: &cosmosrpc.ClientOption{CustomHeaders: rpc.Headers},
		})
	}

	logger := zerolog.Ctx(log.With().
		Str("component", ConsumerName).
		Str("chain", chain).
		Str("id", config.ID).
		Logger().
		WithContext(context.Background()),
	)

	rpcClient := cosmosrpc.NewHub(clientConfigs, logger, time.Duration(config.RPCTimeOutInSeconds)*time.Second)
	if err := rpcClient.Rebalance(context.Background()); err != nil {
		return nil, fmt.Errorf("RPC: error rebalancing RPC endpoints: %w", err)
	}

	dbClient, err := db.NewClient(config.DBConnectionString)
	if err != nil {
		return nil, fmt.Errorf("DB: error creating DB client: %w", err)
	}

	return &Consumer{
		Consumer:          c,
		DB:                dbClient,
		Logger:            logger,
		RPCClient:         rpcClient,
		RebalanceInterval: config.RebalanceInterval,
		Chain:             chain,
		Topic:             topic,
		ConsumerGroup:     consumerGroup,
	}, nil
}

// Name returns the extension name.
func (c *Consumer) Name() string {
	return ConsumerName
}

func (c *Consumer) parseBlockAndRebalanceRPCClient(parentCtx context.Context, blockBytes []byte) (mq.BlockResultMsg, error) {
	var blockMsg mq.BlockResultMsg
	err := json.Unmarshal(blockBytes, &blockMsg)
	if err != nil {
		c.Logger.Error().Err(err).Msg("unmarshal block message")
		return blockMsg, err
	}

	if blockMsg.Version != 0 {
		return blockMsg, fmt.Errorf("invalid version: %d", blockMsg.Version)
	}

	if c.RebalanceInterval != 0 && blockMsg.Height%c.RebalanceInterval == 0 {
		err := c.RPCClient.Rebalance(parentCtx)
		if err != nil {
			c.Logger.Error().Err(err).Msg("rebalance rpc clients")
			return blockMsg, err
		}
		actives := c.RPCClient.GetActiveClients()
		for _, active := range actives {
			c.Logger.Info().Str("client_url", active.Client.GetIdentifier()).Int64("latest_height", active.Height).Msg("active client")
		}
	}
	return blockMsg, err
}

func (c *Consumer) processClaimCheckMessage(key []byte, messageValue []byte) ([]byte, error) {
	if strings.HasPrefix(string(key), mq.NEW_BLOCK_RESULTS_CLAIM_CHECK_KAFKA_MESSAGE_KEY) {
		var claimCheckBlockResultsMsg mq.ClaimCheckMsg
		err := json.Unmarshal(messageValue, &claimCheckBlockResultsMsg)
		if err != nil {
			c.Logger.Error().Err(err).Msg("unmarshalling message")
			return nil, err
		}

		// richlist consumer does not support claim check; message value is expected
		// to already contain the full block result payload. If this path is hit,
		// just log and return an error so the message can be retried or handled
		// by the caller.
		c.Logger.Error().Msg("received claim check message in richlist consumer, which is not supported")
		return nil, fmt.Errorf("claim check messages are not supported in richlist consumer")
	}
	return messageValue, nil
}

// processMessage processes one Kafka message: if no rich_list_status row exists, inits from LCD at h-1 then applies message for h; else applies message for h.
func (c *Consumer) processMessage(ctx context.Context, msg *kafka.Message) error {
	messageValue, err := c.processClaimCheckMessage(msg.Key, msg.Value)
	if err != nil {
		c.Logger.Error().Err(err).Msg("process claim check message")
		return err
	}
	blockMsg, err := c.parseBlockAndRebalanceRPCClient(ctx, messageValue)
	if err != nil {
		c.Logger.Error().Err(err).Msg("processing block message")
		return err
	}

	var allEvents []abci.Event
	for _, tx := range blockMsg.Txs {
		if tx.ExecTxResults != nil {
			allEvents = append(allEvents, tx.ExecTxResults.Events...)
		}
	}
	allEvents = append(allEvents, blockMsg.FinalizeBlockEvents...)

	balanceMap := make(map[BalanceChangeKey]sdkmath.Int)
	c.processMoveTransferEvents(ctx, allEvents, balanceMap, nil)

	if len(balanceMap) == 0 {
		return db.UpdateRichListStatus(ctx, c.DB, blockMsg.Height)
	}

	return c.DB.Transaction(func(tx *gorm.DB) error {
		for key, amount := range balanceMap {
			if err := tx.WithContext(ctx).Exec(
				"INSERT INTO rich_list (denom, addr, amount) VALUES (?, ?, ?) ON CONFLICT (denom, addr) DO UPDATE SET amount = rich_list.amount + EXCLUDED.amount",
				key.Denom, key.Addr, amount.String(),
			).Error; err != nil {
				return fmt.Errorf("upsert rich list for %s/%s: %w", key.Denom, key.Addr, err)
			}
		}
		return db.UpdateRichListStatus(ctx, tx, blockMsg.Height)
	})
}

// NewBalanceChangeKey creates a BalanceChangeKey from asset and address.
func NewBalanceChangeKey(denom string, addr sdk.AccAddress) BalanceChangeKey {
	return BalanceChangeKey{
		Denom: denom,
		Addr:  addr.String(),
	}
}

// ApplyBalanceChange adds the amount to the balance map for the denom/address key.
func ApplyBalanceChange(balanceMap map[BalanceChangeKey]sdkmath.Int, denom string, addr sdk.AccAddress, amount sdkmath.Int) {
	key := NewBalanceChangeKey(denom, addr)
	if balance, ok := balanceMap[key]; !ok {
		balanceMap[key] = amount
	} else {
		balanceMap[key] = balance.Add(amount)
	}
}

// ContainsAddress checks whether the target address exists in the slice.
func ContainsAddress(addresses []sdk.AccAddress, target sdk.AccAddress) bool {
	for _, addr := range addresses {
		if addr.Equals(target) {
			return true
		}
	}
	return false
}

// processMoveTransferEvents processes Move VM transfer events (deposit/withdraw) and updates the balance map.
// It handles fungible asset transfers in the Move primary store by matching deposit/withdraw events with their owner events.
// Module accounts are excluded from balance tracking.
func (c *Consumer) processMoveTransferEvents(ctx context.Context, events []abci.Event, balanceMap map[BalanceChangeKey]sdkmath.Int, moduleAccounts []sdk.AccAddress) {
	for idx, event := range events {
		if idx == len(events)-1 || event.Type != "move" || len(event.Attributes) < 2 || event.Attributes[0].Key != "type_tag" || len(events[idx+1].Attributes) < 2 || events[idx+1].Attributes[0].Key != "type_tag" {
			continue
		}

		// Support only Fungible Asset in primary store (always following with an owner event)
		// - 0x1::fungible_asset::DepositEvent => 0x1::fungible_asset::DepositOwnerEvent
		// - 0x1::fungible_asset::WithdrawEvent => 0x1::fungible_asset::WithdrawOwnerEvent
		if event.Attributes[0].Value == MoveDepositEventTypeTag && events[idx+1].Attributes[0].Value == MoveDepositOwnerEventTypeTag {
			var depositEvent MoveDepositEvent
			err := json.Unmarshal([]byte(event.Attributes[1].Value), &depositEvent)
			if err != nil {
				c.Logger.Error().Err(err).Msg("failed to unmarshal deposit event")
				continue
			}

			var depositOwnerEvent MoveDepositOwnerEvent
			err = json.Unmarshal([]byte(events[idx+1].Attributes[1].Value), &depositOwnerEvent)
			if err != nil {
				c.Logger.Error().Err(err).Msg("failed to unmarshal deposit owner event")
				continue
			}

			recipient, err := parser.AccAddressFromString(depositOwnerEvent.Owner)
			if err != nil {
				c.Logger.Error().Err(err).Str("recipient", depositOwnerEvent.Owner).Msg("failed to parse recipient")
				continue
			}

			denom, err := c.getMoveDenomByMetadataAddr(ctx, depositEvent.MetadataAddr)
			if err != nil {
				c.Logger.Error().Err(err).Str("metadataAddr", depositEvent.MetadataAddr).Msg("failed to get move denom")
				continue
			}
			amount, ok := sdkmath.NewIntFromString(depositEvent.Amount)
			if !ok {
				c.Logger.Error().Err(err).Str("coin", depositEvent.Amount).Msg("failed to parse coin")
				continue
			}

			if !ContainsAddress(moduleAccounts, recipient) {
				ApplyBalanceChange(balanceMap, denom, recipient, amount)
			}
		}

		if event.Attributes[0].Value == MoveWithdrawEventTypeTag && events[idx+1].Attributes[0].Value == MoveWithdrawOwnerEventTypeTag {
			var withdrawEvent MoveWithdrawEvent
			err := json.Unmarshal([]byte(event.Attributes[1].Value), &withdrawEvent)
			if err != nil {
				c.Logger.Error().Err(err).Msg("failed to unmarshal withdraw event")
				continue
			}

			var withdrawOwnerEvent MoveWithdrawOwnerEvent
			err = json.Unmarshal([]byte(events[idx+1].Attributes[1].Value), &withdrawOwnerEvent)
			if err != nil {
				c.Logger.Error().Err(err).Msg("failed to unmarshal withdraw owner event")
				continue
			}

			sender, err := parser.AccAddressFromString(withdrawOwnerEvent.Owner)
			if err != nil {
				c.Logger.Error().Err(err).Str("sender", withdrawOwnerEvent.Owner).Msg("failed to parse sender")
				continue
			}
			denom, err := c.getMoveDenomByMetadataAddr(ctx, withdrawEvent.MetadataAddr)
			if err != nil {
				c.Logger.Error().Err(err).Str("metadataAddr", withdrawEvent.MetadataAddr).Msg("failed to get move denom")
				continue
			}
			amount, ok := sdkmath.NewIntFromString(withdrawEvent.Amount)
			if !ok {
				c.Logger.Error().Err(err).Str("coin", withdrawEvent.Amount).Msg("failed to parse coin")
				continue
			}

			if !ContainsAddress(moduleAccounts, sender) {
				ApplyBalanceChange(balanceMap, denom, sender, amount.Neg())
			}
		}
	}
}

func (c *Consumer) initializeRichList(ctx context.Context, height int64) error {
	err := c.DB.Transaction(func(tx *gorm.DB) error {
		err := db.InsertRichListStatus(ctx, tx, height)
		if err != nil {
			c.Logger.Error().Err(err).Int64("height", height).Msg("insert rich list status failed")
			return err
		}
		return nil
	})
	return err
}

// Run subscribes to the rich list topic and processes messages until ctx is done.
func (c *Consumer) Run(ctx context.Context) error {
	if err := c.Consumer.SubscribeTopics([]string{c.Topic}, nil); err != nil {
		return fmt.Errorf("subscribe to topic %s: %w", c.Topic, err)
	}
	c.Logger.Info().Str("topic", c.Topic).Msg("subscribed to rich list topic")

	isInitialized, err := db.IsRichListInitialized(ctx, c.DB)
	if err != nil {
		return fmt.Errorf("get latest rich list height: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.Consumer.ReadMessage(10 * time.Second)
			if err != nil {
				if err.(kafka.Error).IsTimeout() {
					continue
				}
				c.Logger.Error().Err(err).Msg("read rich list message")
				continue
			}

			var kafkaMessage KafkaMessage
			if err := json.Unmarshal(msg.Value, &kafkaMessage); err != nil {
				c.Logger.Error().Err(err).Msg("unmarshal rich list message")
				continue
			}

			if !isInitialized {
				if err := c.initializeRichList(ctx, kafkaMessage.Height-1); err != nil {
					c.Logger.Error().Err(err).Int64("height", kafkaMessage.Height-1).Msg("initialize rich list failed")
					return fmt.Errorf("initialize rich list: %w", err)
				}
				isInitialized = true
			}

			if err := c.processMessage(ctx, msg); err != nil {
				c.Logger.Error().Err(err).Int64("partition", int64(msg.TopicPartition.Partition)).Str("offset", msg.TopicPartition.Offset.String()).Msg("process rich list message failed")
				// Caller may commit or send to DLQ; here we do not commit so message can be retried.
				continue
			}
			if _, err := c.Consumer.CommitMessage(msg); err != nil {
				c.Logger.Error().Err(err).Msg("commit rich list message")
			}
		}
	}
}

// Close closes the Kafka consumer.
func (c *Consumer) Close() {
	c.Consumer.Close()
}
