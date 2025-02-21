package main

import (
	"google.golang.org/grpc"
	"im-service/config"
	"im-service/internal/data/kafka"
	"im-service/internal/data/mongodb"
	"im-service/internal/data/mysql"
	"im-service/internal/data/redis"
	"im-service/internal/loadmonitor"
	"im-service/internal/middleware"
	"im-service/internal/rpc/friend"
	"im-service/internal/rpc/message"
	"im-service/internal/rpc/user"
	"im-service/internal/start"
	"im-service/internal/svc"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

func main() {
	//加载配置文件
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

	// 初始化负载监控系统
	lm := loadmonitor.NewLoadMonitor("http://localhost:8081/report_load")
	// 实际的服务实例端点
	endpoints := []string{"127.0.0.1:9000", "127.0.0.1:9001", "127.0.0.1:9002"}
	// 上报间隔
	interval := 5 * time.Second

	lm.Start(endpoints, interval)

	// 启动 WebSocket 服务
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		start.WsHandler(cfg, lm, w, r)
	})
	log.Printf("启动WebSocket服务器，在端口 %d 上", cfg.Port)
	if err := http.ListenAndServe(":"+strconv.Itoa(cfg.Port), nil); err != nil {
		log.Fatalf("启动WebSocket服务器失败: %v", err)
	}

	// 启动 HTTP 服务器
	//log.Fatal(http.ListenAndServe(":8080", nil))

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
	//创建 gRPC 服务器并注册拦截器
	s := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.AuthMiddleware),
	)
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
	// 创建 gRPC 服务器并注册拦截器
	s := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.AuthMiddleware),
	)
	friendServer := friend.NewCustomFriendServiceServer(sc.KafkaProducer, sc.MongoClient, sc.RedisClient)
	friend.RegisterFriendServiceServer(s, friendServer)
	log.Printf("正在启动好友服务 %s", endpoint)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("无法为好友服务提供服务: %v", err)
	}

}
