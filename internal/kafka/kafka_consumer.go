package kafka

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
	"im-service/internal/websocket"
	"log"
	"time"
)

// KafkaConsumer 定义 Kafka 消费者结构体
type KafkaConsumer struct {
	reader *kafka.Reader
}

// NewKafkaConsumer 创建 Kafka 消费者实例
func NewKafkaConsumer(brokers []string, topic string) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: "message-group",
	})
	return &KafkaConsumer{
		reader: reader,
	}
}

// ConsumeMessages 消费 Kafka 消息
func (c *KafkaConsumer) ConsumeMessages() error {
	log.Printf("消费 Kafka 消息")
	maxRetries := 3
	for {
		msg, err := c.reader.ReadMessage(context.Background())
		if err != nil {
			fmt.Printf("Error reading message from Kafka: %v\n", err)
			if maxRetries > 0 {
				maxRetries--
				time.Sleep(2 * time.Second) // 等待 2 秒后重试
				continue
			} else {
				log.Fatalf("多次尝试读取 Kafka 消息失败，退出消费: %v", err)
				return err
			}
		}

		// 处理消息
		log.Printf("处理消息")
		err = HandleKafkaMessage(string(msg.Value))
		if err != nil {
			log.Printf("处理消息错误")
			return err
		}
	}
}

// Close 关闭 Kafka 消费者
func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}

// MyCustomError 自定义错误
type MyCustomError struct {
	ErrMsg string
}

func (e *MyCustomError) Error() string {
	return e.ErrMsg
}

// HandleKafkaMessage 处理 Kafka 消息
func HandleKafkaMessage(message string) error {
	// 解析消息
	parts := splitMessage(message)
	if len(parts) < 1 {
		err := MyCustomError{ErrMsg: "无效Kafka消息"}
		fmt.Printf("无效Kafka消息\n: %s\n", message)
		return &err
	}

	switch parts[0] {
	case "friend_accepted":
		// 好友关系建立，通知相关用户
		from := parts[1]
		to := parts[2]
		websocket.NotifyFriendAccepted(from, to)
	case "sendMessage":
		// 新消息，通知相关用户
		log.Printf("新消息，通知相关用户")
		from := parts[1]
		to := parts[2]
		content := parts[3]
		message := fmt.Sprintf("%s|%s|%s", from, to, content)
		websocket.NotifyNewMessage(message)
	default:
		err := MyCustomError{ErrMsg: "未知Kafka消息类型"}
		fmt.Printf("未知Kafka消息类型: %s\n", parts[0])
		return &err
	}
	return nil
}

// splitMessage 分割消息
func splitMessage(message string) []string {
	return split(message, "|")
}

// split 分割字符串
func split(s, sep string) []string {
	var parts []string
	for len(s) > 0 {
		i := index(s, sep)
		if i < 0 {
			parts = append(parts, s)
			break
		}
		parts = append(parts, s[:i])
		s = s[i+len(sep):]
	}
	return parts
}

// index 返回字符串中第一次出现子字符串的索引
func index(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
