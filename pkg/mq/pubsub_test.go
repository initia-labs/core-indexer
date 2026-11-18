package mq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"cloud.google.com/go/pubsub/v2/pstest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func setupPubSub(t *testing.T) (*pstest.Server, *Producer) {
	t.Helper()
	server := pstest.NewServer()
	t.Cleanup(func() { server.Close() })
	t.Setenv("PUBSUB_EMULATOR_HOST", server.Addr)
	producer, err := NewProducer(context.Background(), "test-project", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { producer.Close() })
	return server, producer
}

func createTopic(t *testing.T, producer *Producer, topic string) {
	t.Helper()
	_, err := producer.client.TopicAdminClient.CreateTopic(context.Background(), &pubsubpb.Topic{Name: "projects/test-project/topics/" + topic})
	if err != nil {
		t.Fatal(err)
	}
}

func createConsumer(t *testing.T, producer *Producer, topic, subscription string) *Consumer {
	t.Helper()
	_, err := producer.client.SubscriptionAdminClient.CreateSubscription(context.Background(), &pubsubpb.Subscription{
		Name:                  "projects/test-project/subscriptions/" + subscription,
		Topic:                 "projects/test-project/topics/" + topic,
		EnableMessageOrdering: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := NewConsumer(context.Background(), "test-project", subscription, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { consumer.Close() })
	return consumer
}

func TestPubSubOrderedFanout(t *testing.T) {
	_, producer := setupPubSub(t)
	createTopic(t, producer, "blocks")
	consumers := []*Consumer{
		createConsumer(t, producer, "blocks", "informative"),
		createConsumer(t, producer, "blocks", "events"),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for height := 1; height <= 3; height++ {
		err := producer.ProduceWithClaimCheck(ctx, &ProduceWithClaimCheckInput{
			Topic: "blocks", Key: []byte(fmt.Sprintf("NEW_BLOCK_RESULTS_%d", height)),
			MessageInBytes: []byte(fmt.Sprint(height)), OrderingKey: "chain-1",
			ClaimCheckThresholdInMB: 1, Attributes: map[string]string{"height": fmt.Sprint(height)},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, consumer := range consumers {
		receiveCtx, stop := context.WithCancel(ctx)
		count := 0
		err := consumer.Receive(receiveCtx, func(_ context.Context, msg *Message) error {
			count++
			if string(msg.Data) != fmt.Sprint(count) || msg.OrderingKey != "chain-1" || msg.Attributes["key"] != fmt.Sprintf("NEW_BLOCK_RESULTS_%d", count) || msg.Attributes["height"] != fmt.Sprint(count) {
				return fmt.Errorf("unexpected block %d: %+v", count, msg)
			}
			if count == 3 {
				stop()
			}
			return nil
		})
		stop()
		if err != nil {
			t.Fatal(err)
		}
		if count != 3 {
			t.Fatalf("received %d blocks, want 3", count)
		}
	}
}

type claimStorage struct {
	data         []byte
	bucket, path string
	err          error
}

func (s *claimStorage) UploadFile(bucket, path string, data []byte) error {
	s.bucket, s.path, s.data = bucket, path, bytes.Clone(data)
	return s.err
}
func (s *claimStorage) ReadFile(string, string) ([]byte, error) { return s.data, s.err }

func TestPubSubClaimCheckAndDLQ(t *testing.T) {
	server, producer := setupPubSub(t)
	createTopic(t, producer, "blocks")
	createTopic(t, producer, "dlq-chain-events")
	storage := &claimStorage{}
	data := bytes.Repeat([]byte("x"), 10*1024*1024)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	input := &ProduceWithClaimCheckInput{
		Topic: "blocks", Key: []byte("NEW_BLOCK_RESULTS_7"), MessageInBytes: data, OrderingKey: "chain",
		ClaimCheckKey: []byte("NEW_BLOCK_RESULTS_CLAIM_CHECK_7"), ClaimCheckThresholdInMB: 20,
		ClaimCheckBucket: "large-blocks", ClaimCheckObjectPath: "7", StorageClient: storage,
		Attributes: map[string]string{"height": "7"},
	}
	if err := producer.ProduceWithClaimCheck(ctx, input); err != nil {
		t.Fatal(err)
	}
	messages := server.Messages()
	if len(messages) != 1 {
		t.Fatalf("published %d messages", len(messages))
	}
	msg := messages[0]
	var claim ClaimCheckMsg
	if err := json.Unmarshal(msg.Data, &claim); err != nil {
		t.Fatal(err)
	}
	if claim.ObjectPath != "7" || storage.bucket != "large-blocks" || storage.path != "7" || !bytes.Equal(storage.data, data) || msg.Attributes["key"] != string(input.ClaimCheckKey) || msg.OrderingKey != "chain" {
		t.Fatal("claim check did not preserve the payload, key, or ordering")
	}
	original := &Message{Data: msg.Data, Attributes: msg.Attributes}
	if err := producer.ProduceToDLQ(ctx, "chain", "events", original, errors.New("invalid block")); err != nil {
		t.Fatal(err)
	}
	if _, changed := original.Attributes["error"]; changed {
		t.Fatal("DLQ mutated input attributes")
	}
	found := false
	for _, m := range server.Messages() {
		if m.Topic == "projects/test-project/topics/dlq-chain-events" {
			found = true
			if !bytes.Equal(m.Data, msg.Data) || m.Attributes["key"] != string(input.ClaimCheckKey) || m.Attributes["error"] != "invalid block" || m.Attributes["height"] != "7" {
				t.Fatal("DLQ lost message data or attributes")
			}
		}
	}
	if !found {
		t.Fatal("DLQ message was not delivered")
	}
	storage.err = errors.New("storage unavailable")
	if err := producer.ProduceWithClaimCheck(ctx, input); !errors.Is(err, storage.err) {
		t.Fatalf("upload failure: %v", err)
	}
	if len(server.Messages()) != 2 {
		t.Fatal("published a claim check after failed upload")
	}
}

func TestPubSubFailureDoesNotAck(t *testing.T) {
	server, producer := setupPubSub(t)
	createTopic(t, producer, "blocks")
	consumer := createConsumer(t, producer, "blocks", "indexer")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := producer.Publish(ctx, "blocks", &Message{Data: []byte("block"), OrderingKey: "chain"}); err != nil {
		t.Fatal(err)
	}
	messageID := server.Messages()[0].ID
	if err := producer.Publish(ctx, "blocks", &Message{Data: []byte("next block"), OrderingKey: "chain"}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	err := consumer.Receive(ctx, func(ctx context.Context, msg *Message) error {
		calls++
		// A failed DLQ publish must leave the source message available for redelivery.
		return producer.ProduceToDLQ(ctx, "chain", "missing", msg, errors.New("bad block"))
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("want DLQ publish error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("processed %d messages after failure, want 1", calls)
	}
	message := server.Message(messageID)
	if message.Acks != 0 {
		t.Fatal("acked a message after failed DLQ publish")
	}
	receiveCtx, stop := context.WithCancel(ctx)
	defer stop()
	received := false
	err = consumer.Receive(receiveCtx, func(_ context.Context, msg *Message) error {
		received = msg.ID == message.ID
		stop()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !received {
		t.Fatal("failed message was not redelivered")
	}
	if server.Message(message.ID).Acks != 1 {
		t.Fatal("successful processing was not acknowledged")
	}
}

func TestPubSubOrderedPublishResumesAfterFailure(t *testing.T) {
	_, producer := setupPubSub(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	message := &Message{Data: []byte("block"), OrderingKey: "chain"}
	if err := producer.Publish(ctx, "blocks", message); status.Code(err) != codes.NotFound {
		t.Fatalf("want missing topic error, got %v", err)
	}
	createTopic(t, producer, "blocks")
	if err := producer.Publish(ctx, "blocks", message); err != nil {
		t.Fatalf("ordering key stayed paused after a failed publish: %v", err)
	}
}

func TestAsyncResponsesBatchBeforeWaiting(t *testing.T) {
	server, producer := setupPubSub(t)
	createTopic(t, producer, "responses")
	publisher := producer.publisher("responses")
	publisher.PublishSettings.DelayThreshold = time.Hour
	publisher.PublishSettings.CountThreshold = 2
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var results []*PublishResult
	for _, payload := range []string{"first", "second"} {
		result, err := producer.ProduceWithClaimCheckAsync(ctx, &ProduceWithClaimCheckInput{
			Topic: "responses", MessageInBytes: []byte(payload), ClaimCheckThresholdInMB: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		results = append(results, result)
	}
	for _, result := range results {
		if _, err := result.Get(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if len(server.Messages()) != 2 {
		t.Fatal("responses were not published together")
	}
}

func TestDLQLongErrorFitsAttributeLimit(t *testing.T) {
	server, producer := setupPubSub(t)
	createTopic(t, producer, "dlq-chain-events")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, detail := range []string{strings.Repeat("x", 1025), strings.Repeat("x", 1023) + "ไทย"} {
		if err := producer.ProduceToDLQ(ctx, "chain", "events", &Message{Data: []byte("block")}, errors.New(detail)); err != nil {
			t.Fatal(err)
		}
	}
	for _, message := range server.Messages() {
		detail := message.Attributes["error"]
		if len(detail) > 1024 || !utf8.ValidString(detail) || detail == "" {
			t.Fatalf("invalid error attribute: %q", detail)
		}
	}
}
