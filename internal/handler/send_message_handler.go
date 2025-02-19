package handler

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"im-service/internal/rpc/message"
	"log"
)

// HandleSendMessage  处理发送消息请求
func HandleSendMessage(ctx context.Context, client message.MessageServiceClient, req *message.SendMessageRequest, conn *websocket.Conn) (*message.SendMessageResponse, error) {
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
