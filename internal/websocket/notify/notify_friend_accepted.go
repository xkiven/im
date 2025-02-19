package notify

import (
	"fmt"
	"github.com/gorilla/websocket"
	websocket2 "im-service/internal/websocket"
	"log"
)

// NotifyFriendAccepted 通知好友关系建立
func NotifyFriendAccepted(from, to string) {
	// 查找相关用户的 WebSocket 连接并发送通知
	log.Printf("通知好友关系建立")
	fromConn, ok := websocket2.UserConnections[from]
	if ok {
		notification := fmt.Sprintf("你与 %s 的好友请求已被接受", to)
		if err := fromConn.Conn.WriteMessage(websocket.TextMessage, []byte(notification)); err != nil {
			fmt.Printf("向用户 %s 发送好友接受通知失败: %v\n", from, err)
		}
	} else {
		fmt.Printf("用户 %s 的 WebSocket 连接未找到\n", from)
	}

	toConn, ok := websocket2.UserConnections[to]
	if ok {
		notification := fmt.Sprintf("你接受了 %s 的好友请求", from)
		if err := toConn.Conn.WriteMessage(websocket.TextMessage, []byte(notification)); err != nil {
			fmt.Printf("向用户 %s 发送好友接受通知失败: %v\n", to, err)
		}
	} else {
		fmt.Printf("用户 %s 的 WebSocket 连接未找到\n", to)
	}
}
