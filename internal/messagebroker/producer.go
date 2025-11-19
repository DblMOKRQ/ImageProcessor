package messagebroker

import (
	"ImageProcessor/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"github.com/wb-go/wbf/kafka"
	"go.uber.org/zap"
)

type Producer struct {
	produce *kafka.Producer
	log     *zap.Logger
}

func NewProducer(brokers []string, topic string, log *zap.Logger) *Producer {
	produce := kafka.NewProducer(brokers, topic)
	return &Producer{
		produce: produce,
		log:     log.Named("producer"),
	}
}

func (p *Producer) Publish(ctx context.Context, task *models.ProcessingCommand) error {
	p.log.Debug("Sending task", zap.Any("task", task))
	msgBytes, err := json.Marshal(task)
	if err != nil {
		p.log.Error("Failed to marshal task", zap.Any("task", task), zap.Error(err))
		return fmt.Errorf("failed to marshal task: %w", err)
	}
	err = p.produce.SendWithRetry(ctx, models.RetryStrategy, []byte(task.ID), msgBytes)
	if err != nil {
		p.log.Error("Failed to send task", zap.Any("task", task), zap.Error(err))
		return fmt.Errorf("failed to send task: %w", err)
	}
	p.log.Debug("Successfully sent task", zap.Any("task", task))
	return nil
}
