package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/zc12120/atomhub/internal/types"
)

const (
	DownstreamTokenPrefix      = "atom_"
	downstreamTokenRandomBytes = 24
	downstreamTokenPrefixBytes = 14
)

type downstreamContextKey struct{}

func GenerateDownstreamToken() (token string, tokenPrefix string, tokenHash string, err error) {
	randomBytes := make([]byte, downstreamTokenRandomBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", "", fmt.Errorf("read random token bytes: %w", err)
	}
	token = DownstreamTokenPrefix + base64.RawURLEncoding.EncodeToString(randomBytes)
	tokenPrefix = token
	if len(tokenPrefix) > downstreamTokenPrefixBytes {
		tokenPrefix = tokenPrefix[:downstreamTokenPrefixBytes]
	}
	tokenHash = HashDownstreamToken(token)
	return token, tokenPrefix, tokenHash, nil
}

func HashDownstreamToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func WithDownstreamKey(ctx context.Context, key types.DownstreamKey) context.Context {
	return context.WithValue(ctx, downstreamContextKey{}, key)
}

func DownstreamKeyFromContext(ctx context.Context) (types.DownstreamKey, bool) {
	key, ok := ctx.Value(downstreamContextKey{}).(types.DownstreamKey)
	return key, ok
}
