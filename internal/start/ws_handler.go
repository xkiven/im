package start

import (
	"context"
	"github.com/gorilla/websocket"
	"im-service/config"
	"im-service/internal/handler"
	"im-service/internal/loadmonitor"
	"im-service/internal/rpc/friend"
	"im-service/internal/rpc/message"
	"im-service/internal/rpc/user"
	websocket2 "im-service/internal/websocket"
	"log"
	"net/http"
	"strings"
	"time"
)

// UpGrader 定义升级器
var UpGrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WsHandler 处理 WebSocket 连接
func WsHandler(cfg config.Config, lm *loadmonitor.LoadMonitor, w http.ResponseWriter, r *http.Request) {
	// 创建一个带有超时的上下文，用于 gRPC 连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 建立与用户服务的 gRPC 连接
	userConn, err := handler.CreateGRPCConnection(cfg.UserRpc.Endpoints, lm)
	if err != nil {
		log.Fatalf("无法连接到用户服务: %v", err)
	}
	defer userConn.Close()
	userClient := user.NewUserServiceClient(userConn)

	// 建立与消息服务的 gRPC 连接
	messageConn, err := handler.CreateGRPCConnection(cfg.MessageRpc.Endpoints, lm)
	if err != nil {
		log.Fatalf("无法连接到消息服务: %v", err)
	}
	defer messageConn.Close()
	messageClient := message.NewMessageServiceClient(messageConn)

	//建立与好友服务的 gRPC 连接
	friendConn, err := handler.CreateGRPCConnection(cfg.FriendRpc.Endpoints, lm)
	if err != nil {
		log.Printf("无法连接到好友服务: %v", err)
	}
	defer friendConn.Close()
	friendClient := friend.NewFriendServiceClient(friendConn)

	conn, err := UpGrader.Upgrade(w, r, nil)
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
				resp, err := handler.HandleUserRegister(ctx, userClient, req)
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
				resp, err := handler.HandleUserLogin(ctx, userClient, req, conn)
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
				resp, err := handler.HandleSendMessage(ctx, messageClient, req, conn)
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
