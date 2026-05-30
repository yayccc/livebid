package identity

import (
	"context"

	"google.golang.org/grpc"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if principal, ok := FromIncomingContext(ctx); ok {
			ctx = NewContext(ctx, principal)
		}
		return handler(ctx, req)
	}
}

func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req any, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = ForwardOutgoingContext(ctx)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
