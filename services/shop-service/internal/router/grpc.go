package router

import (
	shopv1 "github.com/yayccc/livebid/gen/proto/shop/v1"
	"github.com/yayccc/livebid/services/shop-service/internal/handler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func RegisterGRPC(server *grpc.Server, shopHandler *handler.ShopGRPCHandler) {
	shopv1.RegisterShopServiceServer(server, shopHandler)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("livebid.shop.v1.ShopService", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
}
