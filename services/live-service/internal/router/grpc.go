package router

import (
	livev1 "github.com/yayccc/livebid/gen/proto/live/v1"
	"github.com/yayccc/livebid/services/live-service/internal/handler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func RegisterGRPC(server *grpc.Server, liveHandler *handler.LiveGRPCHandler) {
	livev1.RegisterLiveServiceServer(server, liveHandler)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("livebid.live.v1.LiveService", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
}
