package identity

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestPrincipalContext(t *testing.T) {
	principal, ok := NewPrincipal(KindShop, "1001")
	if !ok {
		t.Fatal("expected valid principal")
	}

	ctx := NewContext(context.Background(), principal)
	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("expected principal from context")
	}
	if got.Kind != KindShop || got.ID != 1001 {
		t.Fatalf("unexpected principal: %#v", got)
	}
}

func TestNewPrincipalRejectsInvalidSubject(t *testing.T) {
	if _, ok := NewPrincipal(KindShop, "bad"); ok {
		t.Fatal("expected invalid subject to be rejected")
	}
	if _, ok := NewPrincipal(KindUser, "0"); ok {
		t.Fatal("expected non-positive subject to be rejected")
	}
}

func TestMetadataRoundTrip(t *testing.T) {
	principal, ok := NewPrincipal(KindUser, "2001")
	if !ok {
		t.Fatal("expected valid principal")
	}

	outgoing := NewOutgoingContext(context.Background(), principal)
	md, ok := metadata.FromOutgoingContext(outgoing)
	if !ok {
		t.Fatal("expected outgoing metadata")
	}

	got, ok := FromMetadata(md)
	if !ok {
		t.Fatal("expected principal from metadata")
	}
	if got.Kind != KindUser || got.ID != 2001 {
		t.Fatalf("unexpected principal: %#v", got)
	}
}

func TestServerInterceptorStoresIncomingPrincipal(t *testing.T) {
	md := metadata.Pairs(
		MetadataSubjectType, string(KindShop),
		MetadataSubjectID, "1001",
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := UnaryServerInterceptor()(ctx, nil, nil, func(ctx context.Context, req any) (any, error) {
		shopID, ok := ShopID(ctx)
		if !ok || shopID != 1001 {
			t.Fatalf("expected shop id from context, got %d ok=%v", shopID, ok)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
}

func TestClientInterceptorForwardsContextPrincipal(t *testing.T) {
	principal, ok := NewPrincipal(KindShop, "1001")
	if !ok {
		t.Fatal("expected valid principal")
	}
	ctx := NewContext(context.Background(), principal)

	err := UnaryClientInterceptor()(ctx, "/test.Service/Method", nil, nil, nil, func(ctx context.Context, method string, req any, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			t.Fatal("expected outgoing metadata")
		}
		got, ok := FromMetadata(md)
		if !ok || got.Kind != KindShop || got.ID != 1001 {
			t.Fatalf("unexpected principal from outgoing metadata: %#v ok=%v", got, ok)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
}
