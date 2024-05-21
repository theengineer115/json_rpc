package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(broker, topic string) *KafkaProducer {
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(broker),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *KafkaProducer) SendMessage(ctx context.Context, msg interface{}) error {
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{Value: jsonMsg})
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}

type KafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(broker, topic, groupID string) *KafkaConsumer {
	return &KafkaConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  []string{broker},
			Topic:    topic,
			GroupID:  groupID,
			MinBytes: 10e3,
			MaxBytes: 10e6,
		}),
	}
}

func (c *KafkaConsumer) ConsumeMessages(ctx context.Context, msgHandler func([]byte) error) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("error reading message: %w", err)
		}
		if err := msgHandler(msg.Value); err != nil {
			log.Printf("Message handling error: %v", err)
		}
	}
}

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}
