package client

import (
	livev1 "github.com/yayccc/livebid/gen/proto/live/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewLiveServiceConn(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(identity.UnaryClientInterceptor()),
	)
}

func NewLiveServiceClient(conn grpc.ClientConnInterface) livev1.LiveServiceClient {
	return livev1.NewLiveServiceClient(conn)
}
