package client

import (
	auctionv1 "github.com/yayccc/livebid/gen/proto/auction/v1"
	"github.com/yayccc/livebid/pkg/grpcx"
	"google.golang.org/grpc"
)

func NewAuctionServiceConn(target string) (*grpc.ClientConn, error) {
	return grpcx.NewClient(target)
}

func NewAuctionServiceClient(conn grpc.ClientConnInterface) auctionv1.AuctionServiceClient {
	return auctionv1.NewAuctionServiceClient(conn)
}
