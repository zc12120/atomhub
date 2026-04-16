package auth

import (
	"context"
	"testing"

	"github.com/zc12120/atomhub/internal/types"
)

func TestGenerateDownstreamTokenAndHash(t *testing.T) {
	token, tokenPrefix, tokenHash, err := GenerateDownstreamToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if len(token) <= len(DownstreamTokenPrefix) || token[:len(DownstreamTokenPrefix)] != DownstreamTokenPrefix {
		t.Fatalf("expected token to start with %q, got %q", DownstreamTokenPrefix, token)
	}
	if tokenPrefix == "" || tokenPrefix == token || len(tokenPrefix) >= len(token) {
		t.Fatalf("expected short token prefix, got %q for token %q", tokenPrefix, token)
	}
	if tokenHash == "" {
		t.Fatalf("expected token hash to be populated")
	}
	computedHash := HashDownstreamToken(token)
	if computedHash != tokenHash {
		t.Fatalf("hash mismatch: got %q want %q", computedHash, tokenHash)
	}
}

func TestEncryptAndDecryptDownstreamToken(t *testing.T) {
	encrypted, err := EncryptDownstreamToken("secret-value", "atom_plaintext_token")
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}
	if encrypted == "" || encrypted == "atom_plaintext_token" {
		t.Fatalf("expected encrypted payload, got %q", encrypted)
	}
	decrypted, err := DecryptDownstreamToken("secret-value", encrypted)
	if err != nil {
		t.Fatalf("decrypt token: %v", err)
	}
	if decrypted != "atom_plaintext_token" {
		t.Fatalf("unexpected decrypted token: %q", decrypted)
	}
}

func TestDownstreamKeyContextRoundTrip(t *testing.T) {
	key := types.DownstreamKey{ID: 7, Name: "client-a", TokenPrefix: "atom_test"}
	ctx := WithDownstreamKey(context.Background(), key)
	roundTrip, ok := DownstreamKeyFromContext(ctx)
	if !ok {
		t.Fatalf("expected downstream key in context")
	}
	if roundTrip.ID != key.ID || roundTrip.Name != key.Name || roundTrip.TokenPrefix != key.TokenPrefix {
		t.Fatalf("unexpected downstream key: %+v", roundTrip)
	}
}
