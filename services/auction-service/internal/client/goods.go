package client

import (
	"context"

	goodsv1 "github.com/yayccc/livebid/gen/proto/goods/v1"
	"github.com/yayccc/livebid/pkg/grpcx"
	"google.golang.org/grpc"
)

type GoodsClient interface {
	ValidateGoodsForShop(ctx context.Context, goodsID int64, shopID int64) error
	ListGoodsForShop(ctx context.Context, shopID int64, keyword string) ([]*goodsv1.Goods, error)
	BatchGetGoods(ctx context.Context, ids []int64) ([]*goodsv1.Goods, error)
	Close() error
}

type GRPCGoodsClient struct {
	conn   *grpc.ClientConn
	client goodsv1.GoodsServiceClient
}

func NewGRPCGoodsClient(target string) (*GRPCGoodsClient, error) {
	conn, err := grpcx.NewClient(target)
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

func (c *GRPCGoodsClient) ListGoodsForShop(ctx context.Context, shopID int64, keyword string) ([]*goodsv1.Goods, error) {
	const pageSize = 100
	var all []*goodsv1.Goods
	for page := int32(1); ; page++ {
		resp, err := c.client.ListShopGoods(ctx, &goodsv1.ListShopGoodsRequest{
			ShopId:   shopID,
			Page:     page,
			PageSize: pageSize,
			Keyword:  keyword,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, resp.GetList()...)
		if int64(len(all)) >= resp.GetTotal() || len(resp.GetList()) == 0 {
			return all, nil
		}
	}
}

func (c *GRPCGoodsClient) BatchGetGoods(ctx context.Context, ids []int64) ([]*goodsv1.Goods, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, err := c.client.BatchGetGoods(ctx, &goodsv1.BatchGetGoodsRequest{Ids: ids})
	if err != nil {
		return nil, err
	}
	return resp.GetList(), nil
}

func (c *GRPCGoodsClient) Close() error {
	return c.conn.Close()
}
