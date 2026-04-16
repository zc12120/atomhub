package auth

import (
	"context"
	"encoding/json"
	"net/http"
)

type contextKey string

const adminUsernameContextKey contextKey = "admin_username"

// RequireAdmin ensures requests carry a valid admin session.
func RequireAdmin(manager *SessionManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if manager == nil {
			writeUnauthorized(w)
			return
		}

		username, ok := manager.Get(r)
		if !ok {
			writeUnauthorized(w)
			return
		}

		ctx := context.WithValue(r.Context(), adminUsernameContextKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UsernameFromContext extracts the authenticated admin username.
func UsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(adminUsernameContextKey).(string)
	return username, ok
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
