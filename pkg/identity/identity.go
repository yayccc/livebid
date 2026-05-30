package identity

import (
	"context"
	"strconv"
)

type Kind string

const (
	KindShop Kind = "shop"
	KindUser Kind = "user"
)

type Principal struct {
	Kind Kind
	ID   int64
}

func NewPrincipal(kind Kind, subject string) (Principal, bool) {
	id, err := strconv.ParseInt(subject, 10, 64)
	if err != nil || id <= 0 {
		return Principal{}, false
	}
	principal := Principal{
		Kind: kind,
		ID:   id,
	}
	return principal, principal.Valid()
}

func (p Principal) Valid() bool {
	return p.Kind != "" && p.ID > 0
}

type contextKey struct{}

func NewContext(ctx context.Context, principal Principal) context.Context {
	if !principal.Valid() {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, principal)
}

func FromContext(ctx context.Context) (Principal, bool) {
	if ctx == nil {
		return Principal{}, false
	}
	principal, ok := ctx.Value(contextKey{}).(Principal)
	return principal, ok && principal.Valid()
}

func ShopID(ctx context.Context) (int64, bool) {
	principal, ok := FromContext(ctx)
	if !ok || principal.Kind != KindShop {
		return 0, false
	}
	return principal.ID, true
}

func UserID(ctx context.Context) (int64, bool) {
	principal, ok := FromContext(ctx)
	if !ok || principal.Kind != KindUser {
		return 0, false
	}
	return principal.ID, true
}
