package general

import (
	"github.com/gorilla/websocket"
	"log"
	"time"
)

// Reconnect 重新连接函数
func Reconnect(oldConn *websocket.Conn) {
	// 关闭旧的连接
	oldConn.Close()

	maxRetries := 3
	for retryCount := 0; retryCount < maxRetries; retryCount++ {
		// 等待一段时间后再尝试重新连接
		time.Sleep(5 * time.Second)

		// 重新建立连接
		newConn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
		if err != nil {
			log.Printf("第 %d 次重新连接失败: %v", retryCount+1, err)
			continue
		}

		log.Println("重新连接成功")

		// 启动发送心跳包的协程
		go SendHeartBeat(newConn)

		// 启动监听消息的协程
		go ListenForMessages(newConn)

		return
	}

	log.Println("达到最大重试次数，无法重新连接")
}
