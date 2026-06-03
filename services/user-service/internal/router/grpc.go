package router

import (
	userv1 "github.com/yayccc/livebid/gen/proto/user/v1"
	"github.com/yayccc/livebid/pkg/grpcx"
	"github.com/yayccc/livebid/services/user-service/internal/handler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

const HealthServiceName = "livebid.user.v1.UserService"

func RegisterGRPC(server *grpc.Server, userHandler *handler.UserGRPCHandler) *health.Server {
	userv1.RegisterUserServiceServer(server, userHandler)
	return grpcx.RegisterHealth(server, HealthServiceName)
}
