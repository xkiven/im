package general

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

// SendHeartBeat 发送心跳包
func SendHeartBeat(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 发送心跳消息
			if err := conn.WriteMessage(websocket.TextMessage, []byte("heartbeat")); err != nil {
				log.Printf("发送心跳包失败: %v", err)
				return
			}
		}
	}

}

// ListenForMessages 监听心跳包
func ListenForMessages(conn *websocket.Conn) {
	// 心跳超时时间，例如 60 秒
	heartbeatTimeout := 60 * time.Second
	lastHeartbeatTime := time.Now()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("读取消息错误:", err)
			break
		}

		if string(message) == "heartbeat" {
			fmt.Println("收到心跳消息")
			lastHeartbeatTime = time.Now() // 更新最后一次收到心跳的时间
		} else {
			fmt.Println("收到消息:", string(message))
		}

		// 检查是否超时
		if time.Since(lastHeartbeatTime) > heartbeatTimeout {
			log.Println("连接超时，尝试重新连接")
			// 重新建立连接
			Reconnect(conn)
		}
	}
}
