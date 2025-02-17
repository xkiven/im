package websocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"strings"
)

// WebSocketConnection 定义 WebSocket 连接结构体
type WebSocketConnection struct {
	conn *websocket.Conn
}

// 用户名 到 WebSocket 连接的映射
var userConnections = make(map[string]*WebSocketConnection)

// MessageListener 定义消息监听器函数类型
type MessageListener func(from, to, message string)

// registeredListeners 存储已注册的监听器
var registeredListeners = make(map[string]MessageListener)

// RegisterConnection 注册用户的 WebSocket 连接
func RegisterConnection(userName string, conn *websocket.Conn) {
	userConnections[userName] = &WebSocketConnection{conn: conn}
}

// RegisterMessageListener 注册消息监听器，根据用户名筛选消息
func RegisterMessageListener(userName string, listener MessageListener) {
	registeredListeners[userName] = listener
}

// NotifyFriendAccepted 通知好友关系建立
func NotifyFriendAccepted(from, to string) {
	// 查找相关用户的 WebSocket 连接并发送通知
	log.Printf("通知好友关系建立")
	fromConn, ok := userConnections[from]
	if ok {
		notification := fmt.Sprintf("你与 %s 的好友请求已被接受", to)
		if err := fromConn.conn.WriteMessage(websocket.TextMessage, []byte(notification)); err != nil {
			fmt.Printf("向用户 %s 发送好友接受通知失败: %v\n", from, err)
		}
	} else {
		fmt.Printf("用户 %s 的 WebSocket 连接未找到\n", from)
	}

	toConn, ok := userConnections[to]
	if ok {
		notification := fmt.Sprintf("你接受了 %s 的好友请求", from)
		if err := toConn.conn.WriteMessage(websocket.TextMessage, []byte(notification)); err != nil {
			fmt.Printf("向用户 %s 发送好友接受通知失败: %v\n", to, err)
		}
	} else {
		fmt.Printf("用户 %s 的 WebSocket 连接未找到\n", to)
	}
}

// NotifyNewMessage 通知新消息
func NotifyNewMessage(content string) {
	// 查找相关用户的 WebSocket 连接并发送消息
	// 假设 content 格式为 "from|to|message"
	parts := splitMessage(content)
	if len(parts) != 3 {
		fmt.Printf("无效的消息格式: %s\n", content)
		return
	}

	from := parts[0]
	to := parts[1]
	message := parts[2]

	// 调用已注册的监听器
	if listener, ok := registeredListeners[to]; ok {
		listener(from, to, message)
	}

	toConn, ok := userConnections[to]
	if ok {
		notification := fmt.Sprintf("你收到来自 %s 的新消息: %s", from, message)
		if err := toConn.conn.WriteMessage(websocket.TextMessage, []byte(notification)); err != nil {
			fmt.Printf("向用户 %s 发送新消息通知失败: %v\n", to, err)
		}
	} else {
		fmt.Printf("用户 %s 的 WebSocket 连接未找到\n", to)
	}

	fromConn, ok := userConnections[from]
	if ok {
		notification := fmt.Sprintf("你向好友 %s 发出的消息: %s", to, message)
		if err := fromConn.conn.WriteMessage(websocket.TextMessage, []byte(notification)); err != nil {
			fmt.Printf("向用户 %s 发送新消息通知失败: %v\n", from, err)
		}
	} else {
		fmt.Printf("用户 %s 的 WebSocket 连接未找到\n", from)
	}

}

// splitMessage 辅助函数，用于分割消息
func splitMessage(message string) []string {
	return strings.Split(message, "|")
}
