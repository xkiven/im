package svc

import (
	"im-service/config"
	"im-service/internal/data/kafka"
	"im-service/internal/data/mongodb"
	"im-service/internal/data/mysql"
	"im-service/internal/data/redis"
)

// ServiceContext 定义服务上下文结构体
type ServiceContext struct {
	Config        config.Config
	KafkaProducer *kafka.KafkaProducer
	MongoClient   *mongodb.MongoClient
	MySQLClient   *mysql.MySQLClient
	RedisClient   *redis.RedisClient
	// 新增字段，用于存储用户名
	Username string
}

// NewServiceContext 创建服务上下文实例
func NewServiceContext(mysqlClient *mysql.MySQLClient, redisClient *redis.RedisClient, mongoClient *mongodb.MongoClient, kafkaProducer *kafka.KafkaProducer) *ServiceContext {
	return &ServiceContext{
		MySQLClient:   mysqlClient,
		RedisClient:   redisClient,
		MongoClient:   mongoClient,
		KafkaProducer: kafkaProducer,
	}
}

// SetUsername 设置上下文中的用户名
func (sc *ServiceContext) SetUsername(username string) {
	sc.Username = username
}

// GetUsername 获取上下文中的用户名
func (sc *ServiceContext) GetUsername() string {
	return sc.Username
}
