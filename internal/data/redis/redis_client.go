package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"time"
)

// RedisClient 定义 Redis 客户端结构体
type RedisClient struct {
	Client *redis.Client
}

// NewRedisClient 创建 Redis 客户端实例
func NewRedisClient(host, pass string) *RedisClient {
	client := redis.NewClient(&redis.Options{
		Addr:     host,
		Password: pass,
		DB:       0,
	})
	return &RedisClient{
		Client: client,
	}
}

// CheckAndSetIdempotency 检查并设置幂等性键
func (rc *RedisClient) CheckAndSetIdempotency(requestID string, expiration time.Duration) (bool, error) {
	// 使用 Redis 的 SetNX 命令（SET if not exists）
	set, err := rc.Client.SetNX(context.Background(), requestID, "processed", expiration).Result()
	if err != nil {
		return false, err
	}
	return set, nil
}
