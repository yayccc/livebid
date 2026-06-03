package client

import (
	userv1 "github.com/yayccc/livebid/gen/proto/user/v1"
	"github.com/yayccc/livebid/pkg/grpcx"
	"google.golang.org/grpc"
)

func NewUserServiceConn(target string) (*grpc.ClientConn, error) {
	return grpcx.NewClient(target)
}

func NewUserServiceClient(conn grpc.ClientConnInterface) userv1.UserServiceClient {
	return userv1.NewUserServiceClient(conn)
}
