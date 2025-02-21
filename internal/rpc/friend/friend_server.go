package friend

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"im-service/internal/data/kafka"
	"im-service/internal/data/mongodb"
	"im-service/internal/data/redis"
	"log"
	"time"
)

// CustomFriendServiceServer 实现 FriendService 服务
type CustomFriendServiceServer struct {
	kafkaProducer *kafka.KafkaProducer
	mongoClient   *mongodb.MongoClient
	redisClient   *redis.RedisClient
	kafkaConsumer *kafka.KafkaConsumer
}

func (s *CustomFriendServiceServer) mustEmbedUnimplementedFriendServiceServer() {
	//TODO implement me
	panic("implement me")
}

// NewCustomFriendServiceServer 创建好友服务端实例
func NewCustomFriendServiceServer(kafkaProducer *kafka.KafkaProducer, mongoClient *mongodb.MongoClient, redisClient *redis.RedisClient, kafkaConsumer *kafka.KafkaConsumer) *CustomFriendServiceServer {
	return &CustomFriendServiceServer{
		kafkaProducer: kafkaProducer,
		mongoClient:   mongoClient,
		redisClient:   redisClient,
		kafkaConsumer: kafkaConsumer,
	}
}

// SendFriendRequest 处理发送好友请求
func (s *CustomFriendServiceServer) SendFriendRequest(ctx context.Context, req *FriendRequest) (*FriendRequestResponse, error) {
	// 生成请求 ID

	requestID := fmt.Sprintf(req.From, "_addFriend")
	// 检查并设置幂等性键
	exists, err := s.redisClient.CheckAndSetIdempotency(requestID, 10*time.Minute)
	if err != nil {
		return nil, err
	}
	if !exists {
		// 请求已经处理过，直接返回成功
		return &FriendRequestResponse{
			Success:  true,
			ErrorMsg: "",
		}, nil
	}

	//从上下文中获取用户名
	username := ctx.Value("username")
	log.Println(username)
	//验证用户
	if username != req.From {
		return &FriendRequestResponse{
			Success:  false,
			ErrorMsg: "你不是用户本人",
		}, errors.New("验证用户出错")
	}
	// 检查发送者和接收者是否为好友
	isFriend, err := IsFriends(ctx, s.mongoClient, req.From, req.To)

	if err != nil {
		log.Printf("检查好友关系时出错: %v", err)
		return &FriendRequestResponse{
			Success:  false,
			ErrorMsg: err.Error(),
		}, err
	}
	if isFriend {
		return &FriendRequestResponse{
			Success:  false,
			ErrorMsg: "发送者和接收者已经是好友，无法发送好友申请",
		}, err
	}
	// 发送消息到 Kafka
	content := "有新的好友请求"
	command := "sendMessage"
	kafkaMessage := fmt.Sprintf("%s|%s|%s|%s", command, req.From, req.To, content)
	err = s.kafkaProducer.SendMessage(kafkaMessage)
	if err != nil {
		log.Printf("发送消息到 Kafka 失败: %v", err)
		return &FriendRequestResponse{
			Success:  false,
			ErrorMsg: err.Error(),
		}, nil
	}
	log.Printf("消息成功发送到 Kafka，内容: %s", kafkaMessage)
	// 插入好友请求到 MongoDB
	friendRequestsCollection := s.mongoClient.DB.Collection("friend_requests")
	friendRequest := bson.M{
		"from":      req.From,
		"to":        req.To,
		"status":    "pending",
		"timestamp": time.Now(),
	}
	_, insertErr := friendRequestsCollection.InsertOne(ctx, friendRequest)
	if insertErr != nil {
		return &FriendRequestResponse{
			Success:  false,
			ErrorMsg: insertErr.Error(),
		}, nil
	}

	return &FriendRequestResponse{
		Success:  true,
		ErrorMsg: "",
	}, nil
}

// AcceptFriendRequest 处理同意好友请求
func (s *CustomFriendServiceServer) AcceptFriendRequest(ctx context.Context, req *FriendRequest) (*FriendRequestResponse, error) {
	// 生成请求 ID
	requestID := fmt.Sprintf(req.To, "_acceptFriend")
	// 检查并设置幂等性键
	exists, err := s.redisClient.CheckAndSetIdempotency(requestID, 10*time.Minute)
	if err != nil {
		return nil, err
	}
	if !exists {
		// 请求已经处理过，直接返回成功
		return &FriendRequestResponse{
			Success:  true,
			ErrorMsg: "",
		}, nil
	}
	//从上下文中获取用户名
	username := ctx.Value("username")
	//验证用户
	if username != req.To {
		return &FriendRequestResponse{
			Success:  false,
			ErrorMsg: "你不是用户本人",
		}, nil
	}
	// 检查发送者和接收者是否为好友
	isFriend, err := IsFriends(ctx, s.mongoClient, req.From, req.To)

	if err != nil {
		log.Printf("检查好友关系时出错: %v", err)
		return &FriendRequestResponse{
			Success:  false,
			ErrorMsg: err.Error(),
		}, err
	}
	if isFriend {
		return &FriendRequestResponse{
			Success:  false,
			ErrorMsg: "发送者和接收者已经是好友，无法处理好友申请",
		}, err
	}
	// 更新好友请求状态为已接受
	friendRequestsCollection := s.mongoClient.DB.Collection("friend_requests")
	filter := bson.M{
		"from":   req.From,
		"to":     req.To,
		"status": "pending",
	}
	update := bson.M{
		"$set": bson.M{
			"status":    "accepted",
			"timestamp": time.Now(),
		},
	}
	_, updateErr := friendRequestsCollection.UpdateOne(ctx, filter, update)
	if updateErr != nil {
		return &FriendRequestResponse{
			Success:  false,
			ErrorMsg: updateErr.Error(),
		}, nil
	}

	// 插入好友关系到 MongoDB
	friendsCollection := s.mongoClient.DB.Collection("friends")
	friend := bson.M{
		"user1":     req.From,
		"user2":     req.To,
		"timestamp": time.Now(),
	}
	_, insertErr := friendsCollection.InsertOne(ctx, friend)
	if insertErr != nil {
		return &FriendRequestResponse{
			Success:  false,
			ErrorMsg: insertErr.Error(),
		}, nil
	}

	// 发送好友关系建立通知到 Kafka
	err = s.kafkaProducer.SendFriendAcceptedNotification(req.From, req.To)
	if err != nil {
		return &FriendRequestResponse{
			Success:  false,
			ErrorMsg: err.Error(),
		}, nil
	}

	go func() {
		err := s.kafkaConsumer.ConsumeMessages()
		if err != nil {
			log.Printf("Kafka 消费者出现错误")
			log.Println(err)
		}
	}()

	return &FriendRequestResponse{
		Success:  true,
		ErrorMsg: "",
	}, nil
}

// IsFriends  检查两个用户是否为好友
func IsFriends(ctx context.Context, mongoClient *mongodb.MongoClient, user1, user2 string) (bool, error) {
	log.Printf("检查发送者和接收者是否为好友")
	// 获取好友关系集合
	friendshipsCollection := mongoClient.DB.Collection("friend_requests")

	// 构建查询条件，检查用户 1 和用户 2 之间是否存在已接受的好友关系
	filter := bson.M{
		"$or": []bson.M{
			{
				"from":   user1,
				"to":     user2,
				"status": "accepted",
			},
			{
				"from":   user2,
				"to":     user1,
				"status": "accepted",
			},
		},
	}

	// 执行查询
	count, err := friendshipsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	// 如果存在已接受的好友关系，则返回 true，否则返回 false
	return count > 0, nil
}

// GetFriendList 处理获取好友列表请求
func (s *CustomFriendServiceServer) GetFriendList(ctx context.Context, req *GetFriendListRequest) (*GetFriendListResponse, error) {
	//从上下文中获取用户名
	username := ctx.Value("username")
	//验证用户
	if username != req.Username {
		return &GetFriendListResponse{
			Success:  false,
			ErrorMsg: "你不是用户本人",
		}, errors.New("验证用户出错")
	}

	log.Printf("准备获取好友列表")
	// 获取好友关系集合
	friendsCollection := s.mongoClient.DB.Collection("friends")

	// 构建查询条件，查找用户的所有好友
	filter := bson.M{
		"$or": []bson.M{
			{
				"user1": req.Username,
			},
			{
				"user2": req.Username,
			},
		},
	}
	log.Printf("查询用户好友")
	// 执行查询
	cursor, err := friendsCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("查询好友列表时出错: %v", err)
		return &GetFriendListResponse{
			Success:  false,
			ErrorMsg: err.Error(),
		}, err
	}
	defer cursor.Close(ctx)

	var friends []bson.M
	if err := cursor.All(ctx, &friends); err != nil {
		log.Printf("获取好友列表时出错: %v", err)
		return &GetFriendListResponse{
			Success:  false,
			ErrorMsg: err.Error(),
		}, err
	}

	var friendUsernames []string
	for _, friend := range friends {
		if friend["user1"] == req.Username {
			friendUsernames = append(friendUsernames, friend["user2"].(string))
		} else {
			friendUsernames = append(friendUsernames, friend["user1"].(string))
		}
	}

	return &GetFriendListResponse{
		FriendUsernames: friendUsernames,
		Success:         true,
		ErrorMsg:        "",
	}, nil
}
