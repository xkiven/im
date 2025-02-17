package kafka

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log"
)

// KafkaProducer 定义 Kafka 生产者结构体
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer 创建 Kafka 生产者实例
func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	return &KafkaProducer{
		writer: writer,
	}
}

// SendMessage 发送消息到 Kafka
func (p *KafkaProducer) SendMessage(content string) error {
	log.Printf("发送消息到 Kafka")
	err := p.writer.WriteMessages(context.Background(),
		kafka.Message{
			Value: []byte(content),
		},
	)
	return err
}

// SendFriendAcceptedNotification 发送好友关系建立通知到 Kafka
func (p *KafkaProducer) SendFriendAcceptedNotification(from, to string) error {
	log.Printf("发送好友关系建立通知到 Kafka")
	notification := fmt.Sprintf("friend_accepted|%s|%s", from, to)
	err := p.writer.WriteMessages(context.Background(),
		kafka.Message{
			Value: []byte(notification),
		},
	)
	return err
}

// Close 关闭 Kafka 生产者
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
