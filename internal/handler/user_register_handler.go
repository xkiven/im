package handler

import (
	"context"
	"im-service/internal/rpc/user"
)

// HandleUserRegister 处理用户注册请求
func HandleUserRegister(ctx context.Context, client user.UserServiceClient, req *user.UserRegisterRequest) (*user.UserRegisterResponse, error) {
	return client.Register(ctx, req)
}
