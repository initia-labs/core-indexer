package mq

import (
	"context"

	"cloud.google.com/go/pubsub/v2"
)

type Message = pubsub.Message

type Consumer struct {
	client     *pubsub.Client
	subscriber *pubsub.Subscriber
}

func NewConsumer(ctx context.Context, projectID, subscription, endpoint string) (*Consumer, error) {
	client, err := newClient(ctx, projectID, endpoint)
	if err != nil {
		return nil, err
	}
	subscriber := client.Subscriber(subscription)
	// Indexers keep mutable per-block state and must process one block at a time.
	subscriber.ReceiveSettings.MaxOutstandingMessages = 1
	subscriber.ReceiveSettings.NumGoroutines = 1
	return &Consumer{client: client, subscriber: subscriber}, nil
}

// Receive acknowledges only completed work. Stop on failure so an ordered
// consumer cannot advance its database height past the failed block.
func (c *Consumer) Receive(ctx context.Context, process func(context.Context, *Message) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var processErr error
	err := c.subscriber.Receive(ctx, func(ctx context.Context, message *pubsub.Message) {
		if ctx.Err() != nil {
			message.Nack()
			return
		}
		if err := process(ctx, message); err != nil {
			processErr = err
			cancel()
			message.Nack()
			return
		}
		message.Ack()
	})
	if processErr != nil {
		return processErr
	}
	return err
}

func (c *Consumer) Close() error { return c.client.Close() }
