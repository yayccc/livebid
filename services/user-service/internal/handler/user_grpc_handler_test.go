package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/yayccc/livebid/pkg/identity"
)

func TestCurrentUserIDReadsIdentityContext(t *testing.T) {
	ctx := identity.NewContext(context.Background(), identity.Principal{
		Kind: identity.KindUser,
		ID:   2001,
	})

	userID, err := currentUserID(ctx)
	if err != nil {
		t.Fatalf("current user id: %v", err)
	}
	if userID != 2001 {
		t.Fatalf("unexpected user id: %d", userID)
	}
}

func TestCurrentUserIDRejectsMissingIdentity(t *testing.T) {
	_, err := currentUserID(context.Background())
	if !errors.Is(err, errMissingSubject) {
		t.Fatalf("expected missing subject, got %v", err)
	}
}
