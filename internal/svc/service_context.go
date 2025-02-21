package svc

import (
	"im-service/config"
	"im-service/internal/data/kafka"
	"im-service/internal/data/mongodb"
	"im-service/internal/data/mysql"
	"im-service/internal/data/redis"
)

// ContextKey 用于存储 ServiceContext 的键
type ContextKey struct{}

// ServiceContext 定义服务上下文结构体
type ServiceContext struct {
	Config        config.Config
	KafkaProducer *kafka.KafkaProducer
	MongoClient   *mongodb.MongoClient
	MySQLClient   *mysql.MySQLClient
	RedisClient   *redis.RedisClient
	KafkaConsumer *kafka.KafkaConsumer
}

// NewServiceContext 创建服务上下文实例
func NewServiceContext(mysqlClient *mysql.MySQLClient, redisClient *redis.RedisClient, mongoClient *mongodb.MongoClient, kafkaProducer *kafka.KafkaProducer, kafkaConsumer *kafka.KafkaConsumer) *ServiceContext {
	return &ServiceContext{
		MySQLClient:   mysqlClient,
		RedisClient:   redisClient,
		MongoClient:   mongoClient,
		KafkaProducer: kafkaProducer,
		KafkaConsumer: kafkaConsumer,
	}
}
