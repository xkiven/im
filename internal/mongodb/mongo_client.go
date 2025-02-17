package mongodb

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

// MongoClient 封装 MongoDB 客户端
type MongoClient struct {
	Client *mongo.Client
	DB     *mongo.Database
}

// NewMongoClient 创建 MongoDB 客户端实例
func NewMongoClient(uri, database string) (*MongoClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &MongoClient{
		Client: client,
		DB:     client.Database(database),
	}, nil
}

// Close 关闭 MongoDB 客户端连接
func (mc *MongoClient) Close(ctx context.Context) error {
	return mc.Client.Disconnect(ctx)
}
