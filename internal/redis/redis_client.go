package redis

import "github.com/go-redis/redis/v8"

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
