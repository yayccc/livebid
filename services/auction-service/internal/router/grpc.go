package router

import (
	auctionv1 "github.com/yayccc/livebid/gen/proto/auction/v1"
	"github.com/yayccc/livebid/pkg/grpcx"
	"github.com/yayccc/livebid/services/auction-service/internal/handler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

const HealthServiceName = "livebid.auction.v1.AuctionService"

func RegisterGRPC(server *grpc.Server, auctionHandler *handler.AuctionGRPCHandler) *health.Server {
	auctionv1.RegisterAuctionServiceServer(server, auctionHandler)
	return grpcx.RegisterHealth(server, HealthServiceName)
}
