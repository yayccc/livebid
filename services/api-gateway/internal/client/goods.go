package client

import (
	goodsv1 "github.com/yayccc/livebid/gen/proto/goods/v1"
	"github.com/yayccc/livebid/pkg/grpcx"
	"google.golang.org/grpc"
)

func NewGoodsServiceConn(target string) (*grpc.ClientConn, error) {
	return grpcx.NewClient(target)
}

func NewGoodsServiceClient(conn grpc.ClientConnInterface) goodsv1.GoodsServiceClient {
	return goodsv1.NewGoodsServiceClient(conn)
}
