package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"time"

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

// Publish waits for server confirmation before the caller advances or acks input.
// Each service uses its producer from a single processing loop.
func (p *Producer) Publish(ctx context.Context, topic string, message *Message) error {
	publisher, ok := p.publishers[topic]
	if !ok {
		publisher = p.client.Publisher(topic)
		publisher.EnableMessageOrdering = true
		publisher.PublishSettings.DelayThreshold = time.Millisecond
		p.publishers[topic] = publisher
	}
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
	data, key := input.MessageInBytes, input.Key
	threshold := input.ClaimCheckThresholdInMB * 1024 * 1024
	// Leave room for attributes below Pub/Sub's 10 MB publish request limit.
	if threshold > 9*1024*1024 {
		threshold = 9 * 1024 * 1024
	}
	if int64(len(data)) > threshold {
		if err := input.StorageClient.UploadFile(input.ClaimCheckBucket, input.ClaimCheckObjectPath, data); err != nil {
			return err
		}
		var err error
		data, err = json.Marshal(ClaimCheckMsg{ObjectPath: input.ClaimCheckObjectPath})
		if err != nil {
			return err
		}
		key = input.ClaimCheckKey
	}
	attributes := make(map[string]string, len(input.Attributes)+1)
	maps.Copy(attributes, input.Attributes)
	attributes["key"] = string(key)
	return p.Publish(ctx, input.Topic, &Message{Data: data, Attributes: attributes, OrderingKey: input.OrderingKey})
}

func (p *Producer) ProduceToDLQ(ctx context.Context, chain, component string, message *Message, cause error) error {
	attributes := make(map[string]string, len(message.Attributes)+2)
	maps.Copy(attributes, message.Attributes)
	attributes["error"] = cause.Error()
	attributes["timestamp"] = fmt.Sprint(time.Now().Unix())
	return p.Publish(ctx, fmt.Sprintf("dlq-%s-%s", chain, component), &Message{Data: message.Data, Attributes: attributes})
}
