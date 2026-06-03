package router

import (
	shopv1 "github.com/yayccc/livebid/gen/proto/shop/v1"
	"github.com/yayccc/livebid/pkg/grpcx"
	"github.com/yayccc/livebid/services/shop-service/internal/handler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

const HealthServiceName = "livebid.shop.v1.ShopService"

func RegisterGRPC(server *grpc.Server, shopHandler *handler.ShopGRPCHandler) *health.Server {
	shopv1.RegisterShopServiceServer(server, shopHandler)
	return grpcx.RegisterHealth(server, HealthServiceName)
}
