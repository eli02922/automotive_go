package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/embuscado/automotive-catalog/pkg/config"
)

type HandlerFunc func(ctx context.Context, event Event) error

type Consumer struct {
	readers  map[string]*kafka.Reader
	handlers map[string][]HandlerFunc
	cfg      config.KafkaConfig
	log      *zap.Logger
}

func NewConsumer(cfg config.KafkaConfig, log *zap.Logger) *Consumer {
	newReader := func(topic string) *kafka.Reader {
		return kafka.NewReader(kafka.ReaderConfig{
			Brokers:        cfg.Brokers,
			Topic:          topic,
			GroupID:        cfg.ConsumerGroup,
			MinBytes:       1,
			MaxBytes:       10 << 20,
			CommitInterval: cfg.CommitInterval,
			StartOffset:    kafka.LastOffset,
			MaxWait:        3 * time.Second,
		})
	}

	return &Consumer{
		readers: map[string]*kafka.Reader{
			cfg.ProductTopic:   newReader(cfg.ProductTopic),
			cfg.InventoryTopic: newReader(cfg.InventoryTopic),
			cfg.FitmentTopic:   newReader(cfg.FitmentTopic),
		},
		handlers: make(map[string][]HandlerFunc),
		cfg:      cfg,
		log:      log,
	}
}

func (c *Consumer) Register(eventType string, handler HandlerFunc) {
	c.handlers[eventType] = append(c.handlers[eventType], handler)
}

// Start launches one goroutine per topic to consume messages concurrently.
func (c *Consumer) Start(ctx context.Context) {
	for topic, reader := range c.readers {
		go c.consumeTopic(ctx, topic, reader)
	}
}

func (c *Consumer) consumeTopic(ctx context.Context, topic string, reader *kafka.Reader) {
	c.log.Info("kafka consumer started", zap.String("topic", topic))

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.log.Error("kafka fetch error", zap.String("topic", topic), zap.Error(err))
			time.Sleep(2 * time.Second)
			continue
		}

		if err := c.processMessage(ctx, msg); err != nil {
			c.log.Error("kafka message processing failed",
				zap.String("topic", topic),
				zap.Int64("offset", msg.Offset),
				zap.Error(err),
			)
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			c.log.Error("kafka commit error", zap.Error(err))
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) error {
	var event Event
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("consumer: unmarshal event: %w", err)
	}

	handlers, ok := c.handlers[event.Type]
	if !ok {
		return nil
	}

	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			return fmt.Errorf("consumer: handler %s: %w", event.Type, err)
		}
	}
	return nil
}

func (c *Consumer) Close() {
	for topic, r := range c.readers {
		if err := r.Close(); err != nil {
			c.log.Error("kafka reader close error", zap.String("topic", topic), zap.Error(err))
		}
	}
}
