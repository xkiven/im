package user

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
	"im-service/internal/data/mysql"
	"im-service/internal/data/redis"
	"im-service/internal/general"
	"log"
	"time"
)

// CustomUserServiceServer 实现 UserService 服务
type CustomUserServiceServer struct {
	UnimplementedUserServiceServer
	mysqlClient *mysql.MySQLClient
	redisClient *redis.RedisClient
}

// NewCustomUserServiceServer 创建用户服务端实例
func NewCustomUserServiceServer(mysqlClient *mysql.MySQLClient, redisClient *redis.RedisClient) *CustomUserServiceServer {
	return &CustomUserServiceServer{
		mysqlClient: mysqlClient,
		redisClient: redisClient,
	}
}

// Register 处理用户注册请求
func (s *CustomUserServiceServer) Register(ctx context.Context, req *UserRegisterRequest) (*UserRegisterResponse, error) {
	// 生成请求 ID
	requestID := fmt.Sprint(req.Username, "_Register")
	// 检查并设置幂等性键
	exists, err := s.redisClient.CheckAndSetIdempotency(requestID, 10*time.Minute)
	if err != nil {
		return nil, err
	}
	if !exists {
		// 请求已经处理过，直接返回成功
		return &UserRegisterResponse{
			Success:  true,
			ErrorMsg: "",
		}, nil
	}

	var user mysql.User
	//查看用户名是否存在
	result := s.mysqlClient.DB.Where("username = ?", req.Username).First(&user)
	if result.Error == nil {
		return &UserRegisterResponse{
			Success:  false,
			ErrorMsg: "用户名已存在",
		}, nil
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}
	//哈希加密
	req.Password = general.HashEncryption(req.Password)

	newUser := mysql.User{
		Username: req.Username,
		Password: req.Password,
		Nickname: req.Nickname,
	}
	//插入数据库
	result = s.mysqlClient.DB.Create(&newUser)
	if result.Error != nil {
		return nil, result.Error
	}

	// 将用户信息写入 Redis
	err = s.redisClient.Client.Set(ctx, req.Username, newUser.Password, 0).Err()
	if err != nil {
		// 记录日志
		log.Printf("Redis 写入失败: %v", err)
		// 返回错误信息给用户
		return &UserRegisterResponse{
			Success:  false,
			ErrorMsg: "注册成功，但写入缓存失败，请稍后重试",
		}, nil
	}
	log.Printf("Redis 写入成功 ")

	return &UserRegisterResponse{
		Success:  true,
		ErrorMsg: "",
	}, nil
}

// Login 处理用户登录请求
func (s *CustomUserServiceServer) Login(ctx context.Context, req *UserLoginRequest) (*UserLoginResponse, error) {
	//哈希加密
	req.Password = general.HashEncryption(req.Password)
	// 生成请求 ID
	requestID := fmt.Sprint(req.Username, "_Login")
	// 检查并设置幂等性键
	exists, err := s.redisClient.CheckAndSetIdempotency(requestID, 10*time.Minute)
	if err != nil {
		return nil, err
	}
	if !exists {
		// 请求已经处理过，直接返回之前的结果
		token, err := s.redisClient.Client.Get(ctx, requestID+"_token").Result()
		if err != nil {
			return nil, err
		}
		return &UserLoginResponse{
			Token:    token,
			ErrorMsg: "",
		}, nil
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	log.Printf("验证的用户名: %s", req.Username)
	// 尝试从 Redis 中获取用户名及密码进行验证
	cachedPassword, err := s.redisClient.Client.Get(ctx, req.Username).Result()
	if err != nil {
		// 记录 Redis 获取失败的错误信息
		log.Printf("Redis 获取失败: %v", err)
	}
	log.Printf("Redis 获取成功")
	if err == nil && cachedPassword == req.Password {
		log.Printf("生成JWT")
		// 生成JWT
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": req.Username,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})
		tokenString, err := token.SignedString([]byte("your_secret_key"))
		if err != nil {
			return nil, err
		}

		// 将生成的 token 存储到 Redis 中，用于幂等性检查
		err = s.redisClient.Client.Set(ctx, requestID+"_token", tokenString, 10*time.Minute).Err()
		if err != nil {
			return nil, err
		}

		return &UserLoginResponse{
			Token:    tokenString,
			ErrorMsg: "",
		}, nil
	}

	var user mysql.User
	// 若 Redis 验证未通过，继续从数据库验证密码
	result := s.mysqlClient.DB.Where("username = ? AND password = ?", req.Username, req.Password).First(&user)
	log.Printf("读取MySQL数据")

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &UserLoginResponse{
				Token:    "",
				ErrorMsg: "用户名或密码错误",
			}, nil
		}
		return nil, result.Error
	}
	// 将用户名和密码存入 Redis，方便下次验证
	err = s.redisClient.Client.Set(ctx, req.Username, req.Password, 0).Err()
	log.Printf("将用户名和密码存入 Redis，方便下次验证")
	if err != nil {
		// 记录日志但不影响登录流程
		log.Printf("将用户名和密码存入 Redis 时出错: %v", err)
	}
	log.Printf("Redis 写入成功 ")
	//生成JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": req.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte("your_secret_key"))
	if err != nil {
		return nil, err
	}

	// 将生成的 token 存储到 Redis 中，用于幂等性检查
	err = s.redisClient.Client.Set(ctx, requestID+"_token", tokenString, 10*time.Minute).Err()
	if err != nil {
		return nil, err
	}

	return &UserLoginResponse{
		Token:    tokenString,
		ErrorMsg: "",
	}, nil
}
