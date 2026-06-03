package grpcx

import (
	"github.com/yayccc/livebid/pkg/identity"
	_ "github.com/yayccc/livebid/pkg/nacosx"

	"google.golang.org/grpc"
	_ "google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials/insecure"
)

const roundRobinServiceConfig = `{"loadBalancingConfig":[{"round_robin":{}}]}`

func NewClient(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(roundRobinServiceConfig),
		grpc.WithUnaryInterceptor(identity.UnaryClientInterceptor()),
	}
	dialOptions = append(dialOptions, opts...)
	return grpc.NewClient(target, dialOptions...)
}
