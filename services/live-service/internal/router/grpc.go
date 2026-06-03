package router

import (
	livev1 "github.com/yayccc/livebid/gen/proto/live/v1"
	"github.com/yayccc/livebid/pkg/grpcx"
	"github.com/yayccc/livebid/services/live-service/internal/handler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

const HealthServiceName = "livebid.live.v1.LiveService"

func RegisterGRPC(server *grpc.Server, liveHandler *handler.LiveGRPCHandler) *health.Server {
	livev1.RegisterLiveServiceServer(server, liveHandler)
	return grpcx.RegisterHealth(server, HealthServiceName)
}
