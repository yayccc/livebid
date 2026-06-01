package client

import (
	auctionv1 "github.com/yayccc/livebid/gen/proto/auction/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewAuctionServiceConn(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(identity.UnaryClientInterceptor()),
	)
}

func NewAuctionServiceClient(conn grpc.ClientConnInterface) auctionv1.AuctionServiceClient {
	return auctionv1.NewAuctionServiceClient(conn)
}
