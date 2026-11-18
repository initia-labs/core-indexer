package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"
	"unicode/utf8"

	"cloud.google.com/go/pubsub/v2"
	"google.golang.org/api/option"

	"github.com/initia-labs/core-indexer/pkg/storage"
)

type Producer struct {
	client     *pubsub.Client
	publishers map[string]*pubsub.Publisher
}

func newClient(ctx context.Context, projectID, endpoint string) (*pubsub.Client, error) {
	var opts []option.ClientOption
	if endpoint != "" {
		opts = append(opts, option.WithEndpoint(endpoint))
	}
	return pubsub.NewClient(ctx, projectID, opts...)
}

func NewProducer(ctx context.Context, projectID, endpoint string) (*Producer, error) {
	client, err := newClient(ctx, projectID, endpoint)
	if err != nil {
		return nil, err
	}
	return &Producer{client: client, publishers: make(map[string]*pubsub.Publisher)}, nil
}

type PublishResult = pubsub.PublishResult

// Each service uses its producer from a single processing loop.
func (p *Producer) publisher(topic string) *pubsub.Publisher {
	publisher, ok := p.publishers[topic]
	if !ok {
		publisher = p.client.Publisher(topic)
		publisher.EnableMessageOrdering = true
		publisher.PublishSettings.DelayThreshold = time.Millisecond
		p.publishers[topic] = publisher
	}
	return publisher
}

// Publish waits for server confirmation before the caller advances or acks input.
func (p *Producer) Publish(ctx context.Context, topic string, message *Message) error {
	publisher := p.publisher(topic)
	_, err := publisher.Publish(ctx, message).Get(ctx)
	if err != nil && message.OrderingKey != "" {
		publisher.ResumePublish(message.OrderingKey)
	}
	return err
}

func (p *Producer) Close() error {
	for _, publisher := range p.publishers {
		publisher.Stop()
	}
	return p.client.Close()
}

type ProduceWithClaimCheckInput struct {
	Topic                   string
	Key                     []byte
	MessageInBytes          []byte
	OrderingKey             string
	ClaimCheckKey           []byte
	ClaimCheckThresholdInMB int64
	ClaimCheckBucket        string
	ClaimCheckObjectPath    string
	StorageClient           storage.Client
	Attributes              map[string]string
}

func (p *Producer) ProduceWithClaimCheck(ctx context.Context, input *ProduceWithClaimCheckInput) error {
	message, err := claimCheckMessage(input)
	if err != nil {
		return err
	}
	return p.Publish(ctx, input.Topic, message)
}

// ProduceWithClaimCheckAsync lets tx responses batch in the SDK. The caller must
// wait for every result before committing the block or acknowledging its input.
func (p *Producer) ProduceWithClaimCheckAsync(ctx context.Context, input *ProduceWithClaimCheckInput) (*PublishResult, error) {
	message, err := claimCheckMessage(input)
	if err != nil {
		return nil, err
	}
	return p.publisher(input.Topic).Publish(ctx, message), nil
}

func claimCheckMessage(input *ProduceWithClaimCheckInput) (*Message, error) {
	data, key := input.MessageInBytes, input.Key
	threshold := input.ClaimCheckThresholdInMB * 1024 * 1024
	// Leave room for attributes below Pub/Sub's 10 MB publish request limit.
	if threshold > 9*1024*1024 {
		threshold = 9 * 1024 * 1024
	}
	if int64(len(data)) > threshold {
		if err := input.StorageClient.UploadFile(input.ClaimCheckBucket, input.ClaimCheckObjectPath, data); err != nil {
			return nil, err
		}
		var err error
		data, err = json.Marshal(ClaimCheckMsg{ObjectPath: input.ClaimCheckObjectPath})
		if err != nil {
			return nil, err
		}
		key = input.ClaimCheckKey
	}
	attributes := make(map[string]string, len(input.Attributes)+1)
	maps.Copy(attributes, input.Attributes)
	attributes["key"] = string(key)
	return &Message{Data: data, Attributes: attributes, OrderingKey: input.OrderingKey}, nil
}

func (p *Producer) ProduceToDLQ(ctx context.Context, chain, component string, message *Message, cause error) error {
	attributes := make(map[string]string, len(message.Attributes)+2)
	maps.Copy(attributes, message.Attributes)
	errorText := strings.ToValidUTF8(cause.Error(), "�")
	// Pub/Sub attribute values are limited to 1024 bytes.
	if len(errorText) > 1024 {
		end := 1024
		for !utf8.RuneStart(errorText[end]) {
			end--
		}
		errorText = errorText[:end]
	}
	attributes["error"] = errorText
	attributes["timestamp"] = fmt.Sprint(time.Now().Unix())
	return p.Publish(ctx, fmt.Sprintf("dlq-%s-%s", chain, component), &Message{Data: message.Data, Attributes: attributes})
}
