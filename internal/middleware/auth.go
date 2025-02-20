package middleware

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"im-service/internal/svc"
	"log"
	"strings"
)

// 定义JWT密钥
const jwtSecret = "your_secret_key"

// AuthUnaryServerInterceptor 是一个 gRPC 一元拦截器，用于验证请求头中的 token
func AuthUnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// 从元数据中获取令牌
	log.Println("从元数据中获取令牌")
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("缺少元数据")
	}

	// 从元数据中获取 authorization 字段的值
	tokenValues := md.Get("authorization")
	if len(tokenValues) == 0 {
		return nil, errors.New("未提供令牌")
	}

	// 提取 token，去除 Bearer 前缀
	token := strings.TrimPrefix(tokenValues[0], "Bearer ")
	if token == tokenValues[0] {
		return nil, errors.New("无效的令牌格式")
	}

	// 验证 token 并提取用户名
	username, valid := VerifyToken(token)
	if !valid {
		return nil, errors.New("无效的令牌")
	}

	// 获取 ServiceContext 上下文
	serviceContext, ok := ctx.Value("serviceContext").(*svc.ServiceContext)
	if !ok {
		return nil, errors.New("无法获取 ServiceContext")
	}

	// 将用户名设置到 ServiceContext 中
	serviceContext.SetUsername(username)

	// 令牌验证通过，继续处理请求
	return handler(ctx, req)
}

// VerifyToken 验证 token 的函数
func VerifyToken(tokenString string) (string, bool) {
	log.Println("验证 token")
	// 解析 token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			log.Printf("意外的签名方法: %v", token.Header["alg"])
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		log.Printf("解析 token 失败: %v", err)
		return "", false
	}

	// 验证 token 是否有效
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Printf("无法将 claims 转换为 jwt.MapClaims")
		return "", false
	}
	// 提取用户名
	username, ok := claims["username"].(string)
	if !ok {
		log.Printf("无法提取用户名")
		return "", false
	}

	return username, true
}
