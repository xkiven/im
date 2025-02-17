package handler

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"im-service/internal/config"
	"im-service/internal/kafka"
	"im-service/internal/rpc/friend"
	"im-service/internal/rpc/message"
	"im-service/internal/rpc/user"
	websocket2 "im-service/internal/websocket"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// 定义升级器
var upGrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// createGRPCConnection 创建 gRPC 连接
func createGRPCConnection(ctx context.Context, target string) (*grpc.ClientConn, error) {
	// 创建拨号选项
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "tcp", addr)
		}),
	}

	// 使用 grpc.NewClient 创建连接
	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// handleUserRegister 处理用户注册请求
func handleUserRegister(ctx context.Context, client user.UserServiceClient, req *user.UserRegisterRequest) (*user.UserRegisterResponse, error) {
	return client.Register(ctx, req)
}

// handleUserLogin 处理用户登录请求
func handleUserLogin(ctx context.Context, client user.UserServiceClient, req *user.UserLoginRequest, conn *websocket.Conn) (*user.UserLoginResponse, error) {
	// 检查上下文是否已经超时
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	resp, err := client.Login(ctx, req)
	if err != nil {
		return nil, err
	}
	// 登录成功后调用 RegisterConnection 函数
	if resp.ErrorMsg == "" {
		websocket2.RegisterConnection(req.Username, conn)
		log.Printf("用户 %s 登录成功并注册连接", req.Username)
		// 发送欢迎消息给客户端
		welcomeMsg := fmt.Sprintf("欢迎你，%s！", req.Username)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(welcomeMsg)); err != nil {
			log.Printf("向用户 %s 发送欢迎消息失败: %v", req.Username, err)
		}
	}

	// 初始化 Kafka 消费者
	log.Printf("初始化 Kafka 消费者")
	consumer := kafka.NewKafkaConsumer([]string{"localhost:9092"}, "im-messages")
	go func() {
		err := consumer.ConsumeMessages()
		if err != nil {
			log.Printf("Kafka 消费者出现错误")
			log.Println(err)
		}
	}()

	return resp, nil
}

// handleSendMessage 处理发送消息请求
func handleSendMessage(ctx context.Context, client message.MessageServiceClient, req *message.SendMessageRequest, conn *websocket.Conn) (*message.SendMessageResponse, error) {
	// 发送消息前记录日志
	log.Printf("准备发送消息: 从 %s 到 %s，内容: %s", req.From, req.To, req.Content)

	resp, err := client.SendMessage(ctx, req)
	if err != nil {
		// 发送失败，记录错误日志并发送错误消息给客户端
		log.Printf("发送消息失败: %v", err)
		errMsg := fmt.Sprintf("发送消息失败: %v", err)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(errMsg)); err != nil {
			log.Printf("向客户端发送错误消息失败: %v", err)
		}
		return nil, err
	}

	// 发送成功，记录日志并发送成功消息给客户端
	log.Printf("发送消息成功: %v", resp)
	successMsg := fmt.Sprintf("发送消息成功: %v", resp)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(successMsg)); err != nil {
		log.Printf("向客户端发送成功消息失败: %v", err)
	}

	return resp, nil
}

// WsHandler 处理 WebSocket 连接
func WsHandler(cfg config.Config, w http.ResponseWriter, r *http.Request) {
	// 创建一个带有超时的上下文，用于 gRPC 连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 建立与用户服务的 gRPC 连接
	userConn, err := createGRPCConnection(ctx, cfg.UserRpc.Endpoints[0])
	if err != nil {
		log.Fatalf("无法连接到用户服务: %v", err)
	}
	defer userConn.Close()
	userClient := user.NewUserServiceClient(userConn)

	// 建立与消息服务的 gRPC 连接
	messageConn, err := createGRPCConnection(ctx, cfg.MessageRpc.Endpoints[0])
	if err != nil {
		log.Fatalf("无法连接到消息服务: %v", err)
	}
	defer messageConn.Close()
	messageClient := message.NewMessageServiceClient(messageConn)

	//建立与好友服务的 gRPC 连接
	friendConn, err := createGRPCConnection(ctx, cfg.FriendRpc.Endpoints[0])
	if err != nil {
		log.Printf("无法连接到好友服务: %v", err)
	}
	defer friendConn.Close()
	friendClient := friend.NewFriendServiceClient(friendConn)

	conn, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	for {
		_, messageData, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		// 解析消息，假设消息格式为 "command|param1|param2..."
		parts := strings.SplitN(string(messageData), "|", -1)
		if len(parts) == 0 {
			continue
		}
		command := parts[0]
		switch command {
		case "register":
			if len(parts) == 4 {
				username, password, nickname := parts[1], parts[2], parts[3]
				req := &user.UserRegisterRequest{
					Username: username,
					Password: password,
					Nickname: nickname,
				}
				resp, err := handleUserRegister(ctx, userClient, req)
				if err != nil {
					log.Printf("用户注册失败: %v", err)
				} else {
					log.Printf("注册结果: %v", resp)
					// 用户登录成功后注册消息监听器
					websocket2.RegisterMessageListener(username, func(from, to, message string) {
						log.Printf("用户 %s 收到来自 %s 的消息: %s", to, from, message)
					})
				}
			}
		case "login":
			if len(parts) == 3 {
				username, password := parts[1], parts[2]
				req := &user.UserLoginRequest{
					Username: username,
					Password: password,
				}
				resp, err := handleUserLogin(ctx, userClient, req, conn)
				if err != nil {
					log.Printf("用户登录失败: %v", err)
				} else {
					log.Printf("登录结果: %v", resp)
				}
			}
		case "sendMessage":
			if len(parts) == 4 {
				from, to, content := parts[1], parts[2], parts[3]
				req := &message.SendMessageRequest{
					From:    from,
					To:      to,
					Content: content,
				}
				resp, err := handleSendMessage(ctx, messageClient, req, conn)
				if err != nil {
					log.Printf("发送消息失败: %v", err)
				} else {
					log.Printf("发送消息结果\n: %v", resp)
				}
			}
		case "getFriendList":
			if len(parts) == 2 {
				userName := parts[1]
				req := &friend.GetFriendListRequest{
					Username: userName,
				}
				resp, err := friendClient.GetFriendList(ctx, req)
				if err != nil {
					log.Printf("获得好友列表失败: %v", err)
				} else {
					log.Printf("好友列表\n: %v", resp)
				}
			}
		default:
			log.Printf("未知命令: %s", command)
		}
	}
}
