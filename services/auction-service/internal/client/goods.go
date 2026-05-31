package client

import (
	"context"

	goodsv1 "github.com/yayccc/livebid/gen/proto/goods/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GoodsClient interface {
	ValidateGoodsForShop(ctx context.Context, goodsID int64, shopID int64) error
	Close() error
}

type GRPCGoodsClient struct {
	conn   *grpc.ClientConn
	client goodsv1.GoodsServiceClient
}

func NewGRPCGoodsClient(addr string) (*GRPCGoodsClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &GRPCGoodsClient{
		conn:   conn,
		client: goodsv1.NewGoodsServiceClient(conn),
	}, nil
}

func (c *GRPCGoodsClient) ValidateGoodsForShop(ctx context.Context, goodsID int64, shopID int64) error {
	resp, err := c.client.GetGoods(ctx, &goodsv1.GetGoodsRequest{Id: goodsID})
	if err != nil {
		return err
	}
	// goods-service 当前没有按店铺查询单品的 RPC，这里取回后在本服务校验归属。
	if resp.GetGoods().GetShopId() != shopID {
		return ErrGoodsShopMismatch
	}
	return nil
}

func (c *GRPCGoodsClient) Close() error {
	return c.conn.Close()
}
