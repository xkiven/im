package handler

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"im-service/internal/data/kafka"
	"im-service/internal/rpc/user"
	websocket2 "im-service/internal/websocket"
	"log"
)

// HandleUserLogin 处理用户登录请求
func HandleUserLogin(ctx context.Context, client user.UserServiceClient, req *user.UserLoginRequest, conn *websocket.Conn) (*user.UserLoginResponse, error) {
	// 检查上下文是否已经超时
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	resp, err := client.Login(ctx, req)
	if err != nil {
		log.Printf("登录失败")
		return nil, err
	}
	// 登录成功后调用 RegisterConnection 函数
	if resp.ErrorMsg == "" {
		websocket2.RegisterConnection(req.Username, conn)
		log.Printf("用户 %s 登录成功并注册连接", req.Username)
		// 发送欢迎消息给客户端
		welcomeMsg := fmt.Sprintf("欢迎你，%s！", req.Username)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(welcomeMsg)); err != nil {
			log.Printf("向用户 %s 发送欢迎消息失败: %v", req.Username, err)
		}
	}

	// 初始化 Kafka 消费者
	log.Printf("初始化 Kafka 消费者")
	consumer := kafka.NewKafkaConsumer([]string{"localhost:9092"}, "im-messages")
	go func() {
		err := consumer.ConsumeMessages()
		if err != nil {
			log.Printf("Kafka 消费者出现错误")
			log.Println(err)
		}
	}()

	return resp, nil
}
