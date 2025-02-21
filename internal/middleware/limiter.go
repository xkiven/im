package middleware

import (
	"context"
	"im-service/internal/data/redis"
	"net/http"

	"strconv"
	"time"
)

// RateLimiter 定义一个分布式限流器结构体
type RateLimiter struct {
	client *redis.RedisClient
	// 每秒允许的请求数
	rate int
	// 令牌桶的容量
	capacity int
}

// NewRateLimiter 创建一个新的分布式限流器
func NewRateLimiter(client *redis.RedisClient, rate, capacity int) *RateLimiter {
	return &RateLimiter{
		client:   client,
		rate:     rate,
		capacity: capacity,
	}
}

// RateLimitMiddleware 限流器中间件
func RateLimitMiddleware(limiter *RateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取客户端 IP 作为限流 key
		clientIP := r.RemoteAddr
		key := "rate_limit:ws_ip_" + clientIP

		// 调用 RateLimiter 的 Allow 方法判断是否允许请求通过
		allowed, err := limiter.Allow(r.Context(), key)
		if err != nil {
			http.Error(w, "限流判断出错", http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "请求过于频繁，请稍后再试", http.StatusTooManyRequests)
			return
		}

		// 允许请求通过，继续处理
		next(w, r)
	}
}

// getClientIP 从上下文中获取客户端 IP
func getClientIP(ctx context.Context) string {
	// 简单返回一个默认值，实际中需要从上下文或者请求中获取真实的客户端 IP
	return "127.0.0.1"
}

// Allow 判断是否允许请求通过
func (l *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	// 获取当前时间戳
	now := time.Now().Unix()

	// 定义 Redis 事务
	tx := l.client.Client.TxPipeline()
	defer tx.Close()

	// 从 Redis 中获取令牌桶的信息
	resp, err := tx.HGetAll(ctx, key).Result()
	if err != nil {
		return false, err
	}

	var tokens int
	var lastRefill int64

	// 如果令牌桶信息不存在，初始化令牌桶
	if len(resp) == 0 {
		tokens = l.capacity
		lastRefill = now
	} else {
		// 解析令牌桶信息
		tokens, _ = strconv.Atoi(resp["tokens"])
		lastRefill, _ = strconv.ParseInt(resp["last_refill"], 10, 64)

		// 计算需要补充的令牌数
		elapsed := now - lastRefill
		newTokens := int(elapsed) * l.rate
		tokens = tokens + newTokens
		if tokens > l.capacity {
			tokens = l.capacity
		}
	}

	// 判断是否有足够的令牌
	if tokens < 1 {
		return false, nil
	}

	// 消耗一个令牌
	tokens--

	// 更新令牌桶信息
	_, err = tx.HSet(ctx, key, "tokens", tokens).Result()
	if err != nil {
		tx.Discard()
		return false, err
	}
	_, err = tx.HSet(ctx, key, "last_refill", now).Result()
	if err != nil {
		tx.Discard()
		return false, err
	}

	// 执行 Redis 事务
	_, err = tx.Exec(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}
