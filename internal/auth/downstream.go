package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
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

func EncryptDownstreamToken(secret string, token string) (string, error) {
	block, err := aes.NewCipher(deriveDownstreamSecret(secret))
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("read nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(token), nil)
	payload := append(nonce, ciphertext...)
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func DecryptDownstreamToken(secret string, encrypted string) (string, error) {
	payload, err := base64.RawURLEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("decode encrypted token: %w", err)
	}
	block, err := aes.NewCipher(deriveDownstreamSecret(secret))
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	if len(payload) < gcm.NonceSize() {
		return "", fmt.Errorf("encrypted token payload too short")
	}
	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt token: %w", err)
	}
	return string(plaintext), nil
}

func deriveDownstreamSecret(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

func WithDownstreamKey(ctx context.Context, key types.DownstreamKey) context.Context {
	return context.WithValue(ctx, downstreamContextKey{}, key)
}

func DownstreamKeyFromContext(ctx context.Context) (types.DownstreamKey, bool) {
	key, ok := ctx.Value(downstreamContextKey{}).(types.DownstreamKey)
	return key, ok
}
