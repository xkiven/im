package handler

import (
	"context"
	"github.com/gorilla/websocket"
	"im-service/internal/rpc/friend"
	"im-service/internal/rpc/message"
	"im-service/internal/rpc/user"
	websocket2 "im-service/internal/websocket"
	"log"
	"strings"
)

func ReadClientMessages(ctx context.Context, conn *websocket.Conn, userClient user.UserServiceClient, messageClient message.MessageServiceClient, friendClient friend.FriendServiceClient) {

	defer conn.Close()

	for {
		_, messageData, err := conn.ReadMessage()
		if err != nil {
			log.Printf("读取客户端消息失败: %v", err)
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
				resp, err := HandleUserRegister(ctx, userClient, req)
				if err != nil {
					log.Printf("用户注册失败: %v", err)
				} else {
					log.Printf("注册结果: %v", resp)
				}
			}
		case "login":
			if len(parts) == 3 {
				username, password := parts[1], parts[2]
				req := &user.UserLoginRequest{
					Username: username,
					Password: password,
				}
				resp, err := HandleUserLogin(ctx, userClient, req, conn)
				if err != nil {
					log.Printf("用户登录失败: %v", err)
				} else {
					log.Printf("登录结果: %v", resp)
					// 用户登录成功后注册消息监听器
					websocket2.RegisterMessageListener(username, func(from, to, message string) {
						log.Printf("用户 %s 收到来自 %s 的消息: %s", to, from, message)
					})
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
				resp, err := HandleSendMessage(ctx, messageClient, req, conn)
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
				resp, err := GetFriendListHandler(ctx, friendClient, req)
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
