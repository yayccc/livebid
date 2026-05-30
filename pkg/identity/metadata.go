package identity

import (
	"context"
	"strconv"

	"google.golang.org/grpc/metadata"
)

const (
	MetadataSubjectType = "livebid-auth-subject-type"
	MetadataSubjectID   = "livebid-auth-subject-id"
)

func FromIncomingContext(ctx context.Context) (Principal, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return Principal{}, false
	}
	return FromMetadata(md)
}

func FromMetadata(md metadata.MD) (Principal, bool) {
	types := md.Get(MetadataSubjectType)
	ids := md.Get(MetadataSubjectID)
	if len(types) == 0 || len(ids) == 0 {
		return Principal{}, false
	}
	return NewPrincipal(Kind(types[0]), ids[0])
}

func NewOutgoingContext(ctx context.Context, principal Principal) context.Context {
	if !principal.Valid() {
		return ctx
	}
	return metadata.AppendToOutgoingContext(
		ctx,
		MetadataSubjectType, string(principal.Kind),
		MetadataSubjectID, strconv.FormatInt(principal.ID, 10),
	)
}

func ForwardOutgoingContext(ctx context.Context) context.Context {
	principal, ok := FromContext(ctx)
	if !ok {
		return ctx
	}
	return NewOutgoingContext(ctx, principal)
}
