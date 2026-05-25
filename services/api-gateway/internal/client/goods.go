package client

import (
	goodsv1 "github.com/yayccc/livebid/gen/proto/goods/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewGoodsServiceConn(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

func NewGoodsServiceClient(conn grpc.ClientConnInterface) goodsv1.GoodsServiceClient {
	return goodsv1.NewGoodsServiceClient(conn)
}
