package mq

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog"

	"github.com/alleslabs/initia-mono/generic-indexer/common"
	"github.com/alleslabs/initia-mono/generic-indexer/storage"
)

type Producer struct {
	*kafka.Producer
}

func NewProducer(config *kafka.ConfigMap) (*Producer, error) {
	p, err := kafka.NewProducer(config)
	if err != nil {
		return nil, err
	}

	return &Producer{p}, nil
}

func (p *Producer) ListenToKafkaProduceEvents(logger *zerolog.Logger) {
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					logger.Error().Msgf("Failed to deliver message: %v\n", ev.TopicPartition)
				}
			}
		}
	}()
}

func (p *Producer) RetryableProduce(message kafka.Message, logger *zerolog.Logger) {
	for {
		err := p.Produce(&message, nil)
		if err != nil {
			if err.(kafka.Error).Code() == kafka.ErrQueueFull {
				// Producer queue is full, wait 1s for messages
				// to be delivered then try again.
				time.Sleep(time.Second)
				continue
			}

			logger.Error().Msgf("Failed to produce message: %v\n", err)
			continue
		}

		break
	}
}

type ProduceWithClaimCheckInput struct {
	Topic          string
	Key            []byte
	MessageInBytes []byte

	ClaimCheckKey           []byte
	ClaimCheckThresholdInMB int64
	ClaimCheckBucket        string
	ClaimCheckObjectPath    string

	StorageClient storage.StorageClient

	Headers []kafka.Header
}

func uploadToStorage(storageClient storage.StorageClient, bucket string, objectPath string, msg []byte, logger *zerolog.Logger) {
	var err error
	for {
		if err = storageClient.UploadFile(bucket, objectPath, msg); err != nil {
			logger.Error().Msgf("cannot upload to Storage: %v", err)
			time.Sleep(time.Second)
			continue
		}

		break
	}
}

func (p *Producer) ProduceWithClaimCheck(input *ProduceWithClaimCheckInput, logger *zerolog.Logger) {
	claimCheckThreshold := int(input.ClaimCheckThresholdInMB * 1024 * 1024)
	var kafkaMessage kafka.Message
	if len(input.MessageInBytes) > claimCheckThreshold {
		uploadToStorage(input.StorageClient, input.ClaimCheckBucket, input.ClaimCheckObjectPath, input.MessageInBytes, logger)

		blockClaimCheckMsgBytes, _ := json.Marshal(common.ClaimCheckBlockMsg{
			ObjectPath: input.ClaimCheckObjectPath,
		})

		kafkaMessage = kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &input.Topic, Partition: int32(kafka.PartitionAny)},
			Key:            input.ClaimCheckKey,
			Value:          blockClaimCheckMsgBytes,
			Headers:        input.Headers,
		}
	} else {
		kafkaMessage = kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &input.Topic, Partition: int32(kafka.PartitionAny)},
			Key:            input.Key,
			Value:          input.MessageInBytes,
			Headers:        input.Headers,
		}
	}

	p.RetryableProduce(kafkaMessage, logger)
}

func (p *Producer) ProduceToDLQ(message *kafka.Message, err error, logger *zerolog.Logger) {
	DLQTopic := "dlq-" + *message.TopicPartition.Topic
	p.RetryableProduce(kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &DLQTopic, Partition: int32(kafka.PartitionAny)},
		Key:            message.Key,
		Value:          message.Value,
		Headers:        append(message.Headers, kafka.Header{Key: "error", Value: []byte(err.Error())}, kafka.Header{Key: "timestamp", Value: []byte(fmt.Sprint(time.Now().Unix()))}),
	}, logger)
}
