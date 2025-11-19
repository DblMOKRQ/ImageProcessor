package messagebroker

import (
	"ImageProcessor/internal/models"
	"context"
	"fmt"
	kafkaGo "github.com/segmentio/kafka-go"
	"github.com/wb-go/wbf/kafka"
	"go.uber.org/zap"
)

type Consumer struct {
	consume *kafka.Consumer
	log     *zap.Logger
}

func NewConsumer(brokers []string, topic string, groupID string, log *zap.Logger) *Consumer {
	consume := kafka.NewConsumer(brokers, topic, groupID)

	return &Consumer{consume: consume, log: log.Named("consumer")}
}

func (c *Consumer) FetchMessage(ctx context.Context) (kafkaGo.Message, error) {
	c.log.Debug("Fetching message")
	msg, err := c.consume.FetchWithRetry(ctx, models.RetryStrategy)
	if err != nil {
		c.log.Error("Error fetching message", zap.Error(err))
		return kafkaGo.Message{}, fmt.Errorf("error fetching message: %w", err)
	}
	c.log.Debug("Fetched message", zap.ByteString("message", msg.Value))
	return msg, nil
}

func (c *Consumer) CommitMessage(ctx context.Context, msg kafkaGo.Message) error {
	c.log.Debug("Committing message", zap.ByteString("message", msg.Value))
	err := c.consume.Commit(ctx, msg)
	if err != nil {
		c.log.Error("Error committing message", zap.Error(err))
		return fmt.Errorf("error committing message: %w", err)
	}
	c.log.Debug("Committed message", zap.ByteString("message", msg.Value))

	return nil
}

func (c *Consumer) Close() error {
	return c.consume.Close()
}
