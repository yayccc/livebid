package router

import (
	goodsv1 "github.com/yayccc/livebid/gen/proto/goods/v1"
	"github.com/yayccc/livebid/pkg/grpcx"
	"github.com/yayccc/livebid/services/goods-service/internal/handler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

const HealthServiceName = "livebid.goods.v1.GoodsService"

func RegisterGRPC(server *grpc.Server, goodsHandler *handler.GoodsGRPCHandler) *health.Server {
	goodsv1.RegisterGoodsServiceServer(server, goodsHandler)
	return grpcx.RegisterHealth(server, HealthServiceName)
}
