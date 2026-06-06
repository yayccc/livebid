package client

import (
	livev1 "github.com/yayccc/livebid/gen/proto/live/v1"
	"github.com/yayccc/livebid/pkg/grpcx"
	"google.golang.org/grpc"
)

func NewLiveServiceConn(target string) (*grpc.ClientConn, error) {
	return grpcx.NewClient(target)
}

func NewLiveServiceClient(conn grpc.ClientConnInterface) livev1.LiveServiceClient {
	return livev1.NewLiveServiceClient(conn)
}
