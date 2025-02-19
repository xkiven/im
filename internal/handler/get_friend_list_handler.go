package handler

import (
	"context"
	"im-service/internal/rpc/friend"
)

func GetFriendListHandler(ctx context.Context, client friend.FriendServiceClient, req *friend.GetFriendListRequest) (*friend.GetFriendListResponse, error) {
	return client.GetFriendList(ctx, req)
}
