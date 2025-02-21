package middleware

import (
	"context"
	"errors"
	redis2 "github.com/go-redis/redis/v8"
	"im-service/internal/data/redis"
	"log"
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
	//log.Printf("初始化令牌桶")
	limiter := &RateLimiter{
		client:   client,
		rate:     rate,
		capacity: capacity,
	}
	// 初始化令牌桶信息
	ctx := context.Background()
	key := "global_rate_limit"
	now := time.Now().Unix()
	err := client.Client.HSet(ctx, key, "tokens", capacity).Err()
	if err != nil {
		log.Printf("初始化令牌桶信息失败: %v", err)
	}
	err = client.Client.HSet(ctx, key, "last_refill", now).Err()
	if err != nil {
		log.Printf("初始化令牌桶信息失败: %v", err)
	}

	return limiter
}

// RateLimitMiddleware 限流器中间件
func RateLimitMiddleware(limiter *RateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 固定的限流key，表示全局令牌桶
		key := "global_rate_limit"
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

func (l *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if ctx.Err() != nil {
		log.Printf("上下文已经取消或超时: %v", ctx.Err())
		return false, ctx.Err()
	}
	// 获取当前时间戳
	now := time.Now().Unix()

	// 定义 Redis 事务
	tx := l.client.Client.TxPipeline()
	defer tx.Close()

	// 从 Redis 中获取令牌桶的信息
	//log.Printf("开始获取令牌桶信息，key: %s", key)
	resp, err := l.client.Client.HGetAll(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis2.Nil) {
			log.Printf("令牌桶信息不存在，初始化令牌桶")
			// 令牌桶信息不存在，初始化令牌桶
			tokens := l.capacity
			lastRefill := now
			// 保存初始化信息到 Redis
			err := l.client.Client.HSet(ctx, key, "tokens", tokens, "last_refill", lastRefill).Err()
			if err != nil {
				log.Printf("保存令牌桶信息失败: %v", err)
				return false, err
			}
			resp = map[string]string{
				"tokens":      strconv.Itoa(tokens),
				"last_refill": strconv.FormatInt(lastRefill, 10),
			}
		} else {
			log.Printf("获取令牌桶信息失败: %v", err)
			return false, err
		}
	}
	var tokens int
	var lastRefill int64

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

	// 判断是否有足够的令牌
	if tokens < 1 {
		return false, nil
	}

	// 消耗一个令牌
	tokens--

	// 开始事务操作
	tx.HSet(ctx, key, "tokens", tokens)
	tx.HSet(ctx, key, "last_refill", now)

	// 执行 Redis 事务
	_, err = tx.Exec(ctx)
	if err != nil {
		log.Printf("事务执行失败: %v", err)
		return false, err
	}

	return true, nil
}
