package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const sessionCookieName = "atomhub_session"

// SessionManager manages signed cookie sessions.
type SessionManager struct {
	secret []byte
	ttl    time.Duration
}

// NewSessionManager creates a session manager using the provided secret and TTL.
func NewSessionManager(secret string, ttl time.Duration) *SessionManager {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &SessionManager{secret: []byte(secret), ttl: ttl}
}

// Set creates or refreshes a session cookie for username.
func (m *SessionManager) Set(w http.ResponseWriter, username string) error {
	if strings.TrimSpace(username) == "" {
		return errors.New("username is required")
	}

	expiresAt := time.Now().Add(m.ttl).Unix()
	payload := username + "|" + strconv.FormatInt(expiresAt, 10)
	signature := m.sign(payload)

	encodedPayload := base64.RawURLEncoding.EncodeToString([]byte(payload))
	encodedSig := base64.RawURLEncoding.EncodeToString(signature)
	token := encodedPayload + "." + encodedSig

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(m.ttl.Seconds()),
		Expires:  time.Now().Add(m.ttl),
	})
	return nil
}

// Get validates and returns the username from the session cookie.
func (m *SessionManager) Get(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}

	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return "", false
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}

	payload := string(payloadBytes)
	expectedSig := m.sign(payload)
	if subtle.ConstantTimeCompare(expectedSig, signature) != 1 {
		return "", false
	}

	claimParts := strings.Split(payload, "|")
	if len(claimParts) != 2 {
		return "", false
	}

	exp, err := strconv.ParseInt(claimParts[1], 10, 64)
	if err != nil {
		return "", false
	}
	if time.Now().Unix() > exp {
		return "", false
	}

	username := strings.TrimSpace(claimParts[0])
	if username == "" {
		return "", false
	}

	return username, true
}

// Clear invalidates the current session cookie.
func (m *SessionManager) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func (m *SessionManager) sign(payload string) []byte {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func (m *SessionManager) String() string {
	return fmt.Sprintf("SessionManager{ttl=%s}", m.ttl)
}
