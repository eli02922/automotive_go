package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/embuscado/automotive-catalog/internal/catalog/model"
	"github.com/embuscado/automotive-catalog/pkg/config"
)

const (
	EventProductCreated  = "product.created"
	EventProductUpdated  = "product.updated"
	EventProductDeleted  = "product.deleted"
	EventInventorySync   = "inventory.sync"
	EventFitmentUpserted = "fitment.upserted"
)

type Event struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

type Producer struct {
	writers map[string]*kafka.Writer
	cfg     config.KafkaConfig
	log     *zap.Logger
}

func NewProducer(cfg config.KafkaConfig, log *zap.Logger) *Producer {
	newWriter := func(topic string) *kafka.Writer {
		return &kafka.Writer{
			Addr:                   kafka.TCP(cfg.Brokers...),
			Topic:                  topic,
			Balancer:               &kafka.Hash{},
			RequiredAcks:           kafka.RequireAll,
			Async:                  false,
			BatchSize:              100,
			BatchTimeout:           10 * time.Millisecond,
			WriteTimeout:           10 * time.Second,
			AllowAutoTopicCreation: true,
		}
	}

	return &Producer{
		writers: map[string]*kafka.Writer{
			cfg.ProductTopic:   newWriter(cfg.ProductTopic),
			cfg.InventoryTopic: newWriter(cfg.InventoryTopic),
			cfg.FitmentTopic:   newWriter(cfg.FitmentTopic),
		},
		cfg: cfg,
		log: log,
	}
}

func (p *Producer) publish(ctx context.Context, topic, key, eventType string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("kafka producer: marshal payload: %w", err)
	}

	event := Event{Type: eventType, Timestamp: time.Now().UTC(), Payload: payloadBytes}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka producer: marshal event: %w", err)
	}

	w, ok := p.writers[topic]
	if !ok {
		return fmt.Errorf("kafka producer: unknown topic %s", topic)
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: eventBytes,
		Headers: []kafka.Header{
			{Key: "event-type", Value: []byte(eventType)},
			{Key: "content-type", Value: []byte("application/json")},
		},
	}

	if err := w.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka producer: write to %s: %w", topic, err)
	}

	p.log.Debug("kafka event published", zap.String("topic", topic), zap.String("event", eventType), zap.String("key", key))
	return nil
}

func (p *Producer) PublishProductEvent(ctx context.Context, eventType string, product *model.Product) error {
	return p.publish(ctx, p.cfg.ProductTopic, product.ID, eventType, product)
}

func (p *Producer) PublishFitmentEvent(ctx context.Context, eventType string, fitment *model.Fitment) error {
	return p.publish(ctx, p.cfg.FitmentTopic, fitment.ProductID, eventType, fitment)
}

func (p *Producer) PublishInventorySync(ctx context.Context, inventory *model.Inventory) error {
	return p.publish(ctx, p.cfg.InventoryTopic, inventory.ProductID, EventInventorySync, inventory)
}

func (p *Producer) Close() {
	for topic, w := range p.writers {
		if err := w.Close(); err != nil {
			p.log.Error("kafka producer close error", zap.String("topic", topic), zap.Error(err))
		}
	}
}
