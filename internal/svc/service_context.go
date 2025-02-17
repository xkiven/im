package svc

import (
	"im-service/internal/config"
	"im-service/internal/kafka"
	"im-service/internal/mongodb"
	"im-service/internal/mysql"
	"im-service/internal/redis"
)

// ServiceContext 定义服务上下文结构体
type ServiceContext struct {
	Config        config.Config
	KafkaProducer *kafka.KafkaProducer
	MongoClient   *mongodb.MongoClient
	MySQLClient   *mysql.MySQLClient
	RedisClient   *redis.RedisClient
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
