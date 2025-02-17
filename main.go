package main

import (
	"google.golang.org/grpc"
	"im-service/internal/config"
	"im-service/internal/handler"
	"im-service/internal/kafka"
	"im-service/internal/mongodb"
	"im-service/internal/mysql"
	"im-service/internal/redis"
	"im-service/internal/rpc/friend"
	"im-service/internal/rpc/message"
	"im-service/internal/rpc/user"
	"im-service/internal/svc"
	"log"
	"net"
	"net/http"
	"strconv"
)

func main() {
	var cfg config.Config
	err := config.LoadConfig("etc/im.yaml", &cfg)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 初始化数据库和消息队列客户端
	mysqlClient, err := mysql.NewMySQLClient(cfg.MySQL.DataSource)
	if err != nil {
		log.Fatalf("初始化 MySQL 失败: %v", err)
	}
	redisClient := redis.NewRedisClient(cfg.Redis.Host, cfg.Redis.Pass)
	mongoClient, err := mongodb.NewMongoClient(cfg.MongoDB.URI, cfg.MongoDB.Database)
	if err != nil {
		log.Fatalf("初始化 MongoDB 失败: %v", err)
	}
	kafkaProducer := kafka.NewKafkaProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)

	// 创建服务上下文
	sc := svc.NewServiceContext(mysqlClient, redisClient, mongoClient, kafkaProducer)

	// 启动用户服务 gRPC 服务器
	if len(cfg.UserRpc.Endpoints) > 0 {
		go startUserService(cfg.UserRpc.Endpoints[0], sc)
	} else {
		log.Println("UserRpc端点列表为空。跳过用户服务启动")
	}

	// 启动消息服务 gRPC 服务器
	if len(cfg.MessageRpc.Endpoints) > 0 {
		go startMessageService(cfg.MessageRpc.Endpoints[0], sc)
	} else {
		log.Println("MessageRpc端点列表为空。跳过消息服务启动")
	}

	//启动好友服务 gRPC 服务器
	if len(cfg.FriendRpc.Endpoints) > 0 {
		go startFriendService(cfg.FriendRpc.Endpoints[0], sc)
	} else {
		log.Println("FriendRpc端点列表为空。跳过好友服务启动")
	}

	// 启动 WebSocket 服务
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handler.WsHandler(cfg, w, r)
	})
	log.Printf("启动WebSocket服务器，在端口 %d 上", cfg.Port)
	if err := http.ListenAndServe(":"+strconv.Itoa(cfg.Port), nil); err != nil {
		log.Fatalf("启动WebSocket服务器失败: %v", err)
	}

}

// startUserService 启动用户服务 gRPC 服务器
func startUserService(endpoint string, sc *svc.ServiceContext) {
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.Fatalf("收听失败: %v", err)
	}
	s := grpc.NewServer()
	userServer := user.NewCustomUserServiceServer(sc.MySQLClient, sc.RedisClient)
	user.RegisterUserServiceServer(s, userServer)
	log.Printf("正在启动用户服务 %s", endpoint)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("无法为用户服务提供服务: %v", err)
	}
}

// startMessageService 启动消息服务 gRPC 服务器
func startMessageService(endpoint string, sc *svc.ServiceContext) {
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.Fatalf("收听失败: %v", err)
	}
	s := grpc.NewServer()
	messageServer := message.NewCustomMessageServiceServer(sc.KafkaProducer, sc.MongoClient)
	message.RegisterMessageServiceServer(s, messageServer)
	log.Printf("正在启动消息服务 %s", endpoint)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("无法为消息服务提供服务: %v", err)
	}
}

// startFriendService 启动好友服务 gRPC 服务器
func startFriendService(endpoint string, sc *svc.ServiceContext) {
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.Fatalf("收听失败: %v", err)
	}
	s := grpc.NewServer()
	friendServer := friend.NewCustomFriendServiceServer(sc.KafkaProducer, sc.MongoClient)
	friend.RegisterFriendServiceServer(s, friendServer)
	log.Printf("正在启动好友服务 %s", endpoint)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("无法为好友服务提供服务: %v", err)
	}

}
