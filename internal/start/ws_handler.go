package start

import (
	"context"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc/metadata"
	"im-service/config"
	general2 "im-service/internal/general"
	"im-service/internal/handler"
	"im-service/internal/loadmonitor"
	"im-service/internal/rpc/friend"
	"im-service/internal/rpc/message"
	"im-service/internal/rpc/user"
	"log"
	"net/http"
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
	// 从请求头中获取 token
	token := r.Header.Get("Authorization")
	if token == "" {
		http.Error(w, "缺少 Authorization 头", http.StatusUnauthorized)
		return
	}

	// 将 token 添加到 gRPC 上下文中
	md := metadata.New(map[string]string{"authorization": token})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	//// 创建一个无超时的上下文
	//ctx := context.Background()

	// 建立与用户服务的 gRPC 连接
	userConn, err := general2.CreateGRPCConnection(cfg.UserRpc.Endpoints, lm)
	if err != nil {
		log.Fatalf("无法连接到用户服务: %v", err)
	}
	defer userConn.Close()
	userClient := user.NewUserServiceClient(userConn)

	// 建立与消息服务的 gRPC 连接
	messageConn, err := general2.CreateGRPCConnection(cfg.MessageRpc.Endpoints, lm)
	if err != nil {
		log.Fatalf("无法连接到消息服务: %v", err)
	}
	defer messageConn.Close()
	messageClient := message.NewMessageServiceClient(messageConn)

	//建立与好友服务的 gRPC 连接
	friendConn, err := general2.CreateGRPCConnection(cfg.FriendRpc.Endpoints, lm)
	if err != nil {
		log.Printf("无法连接到好友服务: %v", err)
	}
	defer friendConn.Close()
	friendClient := friend.NewFriendServiceClient(friendConn)

	//WebSocket 连接升级
	conn, err := UpGrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	//连接成功建立后，启动心跳机制
	go general2.SendHeartBeat(conn)

	//监听心跳机制
	go general2.ListenForMessages(conn)

	// 启动读取客户端消息的循环
	handler.ReadClientMessages(ctx, conn, userClient, messageClient, friendClient)

}
