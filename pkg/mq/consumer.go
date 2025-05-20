package mq

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Consumer struct {
	*kafka.Consumer
}

func NewConsumer(config *kafka.ConfigMap) (*Consumer, error) {
	c, err := kafka.NewConsumer(config)
	if err != nil {
		return nil, err
	}

	return &Consumer{c}, nil
}
