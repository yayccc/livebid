package router

import (
	goodsv1 "github.com/yayccc/livebid/gen/proto/goods/v1"
	"github.com/yayccc/livebid/services/goods-service/internal/handler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func RegisterGRPC(server *grpc.Server, goodsHandler *handler.GoodsGRPCHandler) {
	goodsv1.RegisterGoodsServiceServer(server, goodsHandler)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("livebid.goods.v1.GoodsService", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
}
