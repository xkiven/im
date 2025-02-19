package notify

import (
	"fmt"
	"github.com/gorilla/websocket"
	websocket2 "im-service/internal/websocket"
)

// NotifyNewMessage 通知新消息
func NotifyNewMessage(content string) {
	// 查找相关用户的 WebSocket 连接并发送消息
	// 假设 content 格式为 "from|to|message"
	parts := websocket2.SplitMessage(content)
	if len(parts) != 3 {
		fmt.Printf("无效的消息格式: %s\n", content)
		return
	}

	from := parts[0]
	to := parts[1]
	message := parts[2]

	// 调用已注册的监听器
	if listener, ok := websocket2.RegisteredListeners[to]; ok {
		listener(from, to, message)
	}

	toConn, ok := websocket2.UserConnections[to]
	if ok {
		notification := fmt.Sprintf("你收到来自 %s 的新消息: %s", from, message)
		if err := toConn.Conn.WriteMessage(websocket.TextMessage, []byte(notification)); err != nil {
			fmt.Printf("向用户 %s 发送新消息通知失败: %v\n", to, err)
		}
	} else {
		fmt.Printf("用户 %s 的 WebSocket 连接未找到\n", to)
	}

	fromConn, ok := websocket2.UserConnections[from]
	if ok {
		notification := fmt.Sprintf("你向好友 %s 发出的消息: %s", to, message)
		if err := fromConn.Conn.WriteMessage(websocket.TextMessage, []byte(notification)); err != nil {
			fmt.Printf("向用户 %s 发送新消息通知失败: %v\n", from, err)
		}
	} else {
		fmt.Printf("用户 %s 的 WebSocket 连接未找到\n", from)
	}

}
