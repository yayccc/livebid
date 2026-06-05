package client

import (
	"context"

	livev1 "github.com/yayccc/livebid/gen/proto/live/v1"
	"github.com/yayccc/livebid/pkg/grpcx"
	"github.com/yayccc/livebid/pkg/identity"
	"google.golang.org/grpc"
)

type LiveClient interface {
	ValidateLiveRoomForAuction(ctx context.Context, roomID int64, shopID int64, requireLiving bool) error
	Close() error
}

type GRPCLiveClient struct {
	conn   *grpc.ClientConn
	client livev1.LiveServiceClient
}

func NewGRPCLiveClient(target string) (*GRPCLiveClient, error) {
	conn, err := grpcx.NewClient(target)
	if err != nil {
		return nil, err
	}
	return &GRPCLiveClient{
		conn:   conn,
		client: livev1.NewLiveServiceClient(conn),
	}, nil
}

func (c *GRPCLiveClient) ValidateLiveRoomForAuction(ctx context.Context, roomID int64, shopID int64, requireLiving bool) error {
	ctx = identity.NewContext(ctx, identity.Principal{Kind: identity.KindShop, ID: shopID})
	resp, err := c.client.ValidateLiveRoomForAuction(ctx, &livev1.ValidateLiveRoomForAuctionRequest{
		Id:            roomID,
		ShopId:        shopID,
		RequireLiving: requireLiving,
	})
	if err != nil {
		return err
	}
	if resp.GetLiveRoom().GetShopId() != shopID {
		return ErrLiveRoomShopMismatch
	}
	return nil
}

func (c *GRPCLiveClient) Close() error {
	return c.conn.Close()
}
