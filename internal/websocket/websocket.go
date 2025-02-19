package websocket

import (
	"github.com/gorilla/websocket"
	"strings"
)

// WebSocketConnection 定义 WebSocket 连接结构体
type WebSocketConnection struct {
	Conn *websocket.Conn
}

// UserConnections 用户名 到 WebSocket 连接的映射
var UserConnections = make(map[string]*WebSocketConnection)

// MessageListener 定义消息监听器函数类型
type MessageListener func(from, to, message string)

// RegisteredListeners 存储已注册的监听器
var RegisteredListeners = make(map[string]MessageListener)

// RegisterConnection 注册用户的 WebSocket 连接
func RegisterConnection(userName string, conn *websocket.Conn) {
	UserConnections[userName] = &WebSocketConnection{Conn: conn}
}

// RegisterMessageListener 注册消息监听器，根据用户名筛选消息
func RegisterMessageListener(userName string, listener MessageListener) {
	RegisteredListeners[userName] = listener
}

// SplitMessage 辅助函数，用于分割消息
func SplitMessage(message string) []string {
	return strings.Split(message, "|")
}
