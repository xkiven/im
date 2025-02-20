package message

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"im-service/internal/data/kafka"
	"im-service/internal/data/mongodb"
	"im-service/internal/rpc/friend"
	"im-service/internal/svc"
	"log"
	"time"
)

// CustomMessageServiceServer 实现 MessageService 服务
type CustomMessageServiceServer struct {
	UnimplementedMessageServiceServer
	kafkaProducer *kafka.KafkaProducer
	mongoClient   *mongodb.MongoClient // 修改为新的类型
}

// NewCustomMessageServiceServer 创建消息服务端实例
func NewCustomMessageServiceServer(kafkaProducer *kafka.KafkaProducer, mongoClient *mongodb.MongoClient) *CustomMessageServiceServer {
	return &CustomMessageServiceServer{
		kafkaProducer: kafkaProducer,
		mongoClient:   mongoClient,
	}
}

// SendMessage 处理发送消息请求
func (s *CustomMessageServiceServer) SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error) {
	// 获取 ServiceContext
	serviceContext, ok := ctx.Value("serviceContext").(*svc.ServiceContext)
	if !ok {
		return nil, errors.New("无法获取 ServiceContext")
	}

	// 从 ServiceContext 中获取用户名
	username := serviceContext.GetUsername()
	if username != req.From {
		return &SendMessageResponse{
			Success:  false,
			ErrorMsg: "你不是此用户",
		}, errors.New("发送者不是此用户")
	}

	// 检查发送者和接收者是否为好友
	isFriend, err := friend.IsFriends(ctx, s.mongoClient, req.From, req.To)
	if err != nil {
		log.Printf("检查好友关系时出错: %v", err)
		return &SendMessageResponse{
			Success:  false,
			ErrorMsg: err.Error(),
		}, nil
	}

	if !isFriend {
		return &SendMessageResponse{
			Success:  false,
			ErrorMsg: "发送者和接收者不是好友，无法发送消息",
		}, nil
	}
	// 构建 "command|param1|param2..." 格式的消息
	command := "sendMessage"
	kafkaMessage := fmt.Sprintf("%s|%s|%s|%s", command, req.From, req.To, req.Content)
	// 发送消息到 Kafka
	err = s.kafkaProducer.SendMessage(kafkaMessage)
	if err != nil {
		log.Printf("发送消息到 Kafka 失败: %v", err)
		return &SendMessageResponse{
			Success:  false,
			ErrorMsg: err.Error(),
		}, nil
	}
	log.Printf("消息成功发送到 Kafka，内容: %s", kafkaMessage)
	// 插入消息到 MongoDB
	messagesCollection := s.mongoClient.DB.Collection("messages")
	message := bson.M{
		"from":      req.From,
		"to":        req.To,
		"content":   req.Content,
		"timestamp": time.Now(),
	}
	_, insertErr := messagesCollection.InsertOne(ctx, message)
	if insertErr != nil {
		return &SendMessageResponse{
			Success:  false,
			ErrorMsg: insertErr.Error(),
		}, nil
	}

	return &SendMessageResponse{
		Success:  true,
		ErrorMsg: "",
	}, nil
}

// GetMessageHistory 处理获取消息历史请求
func (s *CustomMessageServiceServer) GetMessageHistory(ctx context.Context, req *GetMessageHistoryRequest) (*GetMessageHistoryResponse, error) {

	// 获取 ServiceContext
	serviceContext, ok := ctx.Value("serviceContext").(*svc.ServiceContext)
	if !ok {
		return nil, errors.New("无法获取 ServiceContext")
	}

	// 从 ServiceContext 中获取用户名
	username := serviceContext.GetUsername()
	if username != req.From {
		return nil, errors.New("请求获取历史消息者不是此用户")
	}

	messagesCollection := s.mongoClient.DB.Collection("messages")
	filter := bson.M{
		"$or": []bson.M{
			{"from": req.From, "to": req.To},
			{"from": req.To, "to": req.From},
		},
	}
	opts := options.Find().SetSort(bson.M{"timestamp": -1}).SetLimit(int64(req.Limit))
	cursor, err := messagesCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []*MessageItem
	for cursor.Next(ctx) {
		var msg bson.M
		if err := cursor.Decode(&msg); err != nil {
			return nil, err
		}
		// 处理 timestamp 字段
		timestamp, ok := msg["timestamp"].(primitive.DateTime)
		if !ok {
			// 处理类型断言失败的情况
			return nil, fmt.Errorf("failed to convert timestamp to primitive.DateTime")
		}
		messages = append(messages, &MessageItem{
			From:      msg["from"].(string),
			To:        msg["to"].(string),
			Content:   msg["content"].(string),
			Timestamp: timestamp.Time().Format(time.RFC3339), // 将 primitive.DateTime 转换为 time.Time
		})
	}

	return &GetMessageHistoryResponse{
		Messages: messages,
		ErrorMsg: "",
	}, nil
}
