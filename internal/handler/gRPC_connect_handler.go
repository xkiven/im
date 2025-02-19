package handler

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"im-service/internal/handler/general"
	"im-service/internal/loadmonitor"
	"net"
)

// CreateGRPCConnection  创建 gRPC 连接
func CreateGRPCConnection(endpoints []string, lm *loadmonitor.LoadMonitor) (*grpc.ClientConn, error) {
	// 使用P2C算法选择服务实例

	target, err := general.PickServerWithP2C(endpoints, lm)
	if err != nil {
		return nil, err
	}

	// 创建拨号选项
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "tcp", addr)
		}),
	}

	// 创建连接
	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
