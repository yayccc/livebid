package auth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestJWTManagerSignAndVerify(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	manager, err := NewJWTManager("test-secret", WithIssuer("livebid"), WithNow(func() time.Time {
		return now
	}))
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}

	token, err := manager.Sign(Claims{
		Subject:   "10001",
		ID:        "token-1",
		ExpiresAt: now.Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if len(strings.Split(token, ".")) != 3 {
		t.Fatalf("expected jwt with 3 parts, got %q", token)
	}

	claims, err := manager.Verify(token)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if claims.Issuer != "livebid" || claims.Subject != "10001" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
	if claims.ID != "token-1" {
		t.Fatalf("unexpected token id: %#v", claims)
	}
}

func TestJWTManagerVerifyExpiredToken(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	signer, err := NewJWTManager("test-secret", WithNow(func() time.Time {
		return now
	}))
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}
	token, err := signer.Sign(Claims{
		Subject:   "10001",
		ExpiresAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	verifier, err := NewJWTManager("test-secret", WithNow(func() time.Time {
		return now.Add(2 * time.Minute)
	}))
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}
	_, err = verifier.Verify(token)
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestJWTManagerVerifyRejectsWrongSecret(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	signer, err := NewJWTManager("test-secret", WithNow(func() time.Time {
		return now
	}))
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}
	token, err := signer.Sign(Claims{
		Subject:   "10001",
		ExpiresAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	verifier, err := NewJWTManager("wrong-secret", WithNow(func() time.Time {
		return now
	}))
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}
	_, err = verifier.Verify(token)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestJWTManagerRejectsInvalidInput(t *testing.T) {
	if _, err := NewJWTManager(""); !errors.Is(err, ErrMissingSecret) {
		t.Fatalf("expected ErrMissingSecret, got %v", err)
	}

	manager, err := NewJWTManager("test-secret")
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}
	if _, err := manager.Sign(Claims{Subject: "10001"}); !errors.Is(err, ErrInvalidClaims) {
		t.Fatalf("expected ErrInvalidClaims, got %v", err)
	}
	if _, err := manager.Verify("not-a-jwt"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestJWTManagerVerifyRejectsUnexpectedIssuer(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	signer, err := NewJWTManager("test-secret", WithIssuer("livebid-dev"), WithNow(func() time.Time {
		return now
	}))
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}
	token, err := signer.Sign(Claims{
		Subject:   "10001",
		ExpiresAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	verifier, err := NewJWTManager("test-secret", WithIssuer("livebid"), WithNow(func() time.Time {
		return now
	}))
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}
	_, err = verifier.Verify(token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
