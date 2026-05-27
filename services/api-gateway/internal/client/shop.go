package client

import (
	shopv1 "github.com/yayccc/livebid/gen/proto/shop/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewShopServiceConn(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

func NewShopServiceClient(conn grpc.ClientConnInterface) shopv1.ShopServiceClient {
	return shopv1.NewShopServiceClient(conn)
}
