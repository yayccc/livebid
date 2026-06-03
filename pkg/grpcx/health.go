package grpcx

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func RegisterHealth(server *grpc.Server, serviceNames ...string) *health.Server {
	healthServer := health.NewServer()
	SetServing(healthServer, serviceNames...)
	healthpb.RegisterHealthServer(server, healthServer)
	return healthServer
}

func SetServing(healthServer *health.Server, serviceNames ...string) {
	if healthServer == nil {
		return
	}
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	for _, serviceName := range serviceNames {
		healthServer.SetServingStatus(serviceName, healthpb.HealthCheckResponse_SERVING)
	}
}

func SetNotServing(healthServer *health.Server, serviceNames ...string) {
	if healthServer == nil {
		return
	}
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	for _, serviceName := range serviceNames {
		healthServer.SetServingStatus(serviceName, healthpb.HealthCheckResponse_NOT_SERVING)
	}
}
