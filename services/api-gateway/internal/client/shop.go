package client

import (
	shopv1 "github.com/yayccc/livebid/gen/proto/shop/v1"
	"github.com/yayccc/livebid/pkg/grpcx"
	"google.golang.org/grpc"
)

func NewShopServiceConn(target string) (*grpc.ClientConn, error) {
	return grpcx.NewClient(target)
}

func NewShopServiceClient(conn grpc.ClientConnInterface) shopv1.ShopServiceClient {
	return shopv1.NewShopServiceClient(conn)
}
