package middleware

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
)

// AuthMiddleware 是一个中间件函数，用于验证 token 并将用户名添加到上下文中
func AuthMiddleware(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// 从请求的元数据中获取 token
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "缺少元数据")
	}

	authHeader, ok := md["authorization"]
	if !ok || len(authHeader) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "缺少 Authorization 头")
	}

	tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")
	if tokenString == "" {
		return nil, status.Errorf(codes.Unauthenticated, "无效的 Authorization 头")
	}

	// 这里简单模拟 token 验证，实际中可使用 JWT 等库进行验证
	username, err := validateToken(tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "无效的 token: %v", err)
	}

	// 将用户名添加到上下文中
	newCtx := context.WithValue(ctx, "username", username)

	// 继续处理请求
	return handler(newCtx, req)
}

// validateToken 使用 JWT 验证 token 并返回用户名
func validateToken(tokenString string) (string, error) {
	// 解析 token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法，确保使用的是 HMAC 签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		// 这里的 "your_secret_key" 应替换为实际的密钥
		return []byte("your_secret_key"), nil
	})

	if err != nil {
		return "", fmt.Errorf("解析 token 失败: %v", err)
	}

	// 验证 token 的有效性
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// 获取用户名
		username, ok := claims["username"].(string)
		if !ok {
			return "", errors.New("token 中缺少用户名")
		}
		return username, nil
	}

	return "", errors.New("无效的 token")
}
