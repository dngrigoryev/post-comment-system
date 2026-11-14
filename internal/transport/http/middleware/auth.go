package middleware

import (
	"context"
	"maps"
	"net/http"
	"strings"

	"client-services/internal/domain"
)

type userIDContextKey struct{}

func Auth(tokens map[string]int64) func(http.Handler) http.Handler {
	tokens = maps.Clone(tokens)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headers := r.Header.Values("Authorization")
			if len(headers) == 0 {
				next.ServeHTTP(w, r)
				return
			}
			if len(headers) != 1 {
				unauthorized(w)
				return
			}
			parts := strings.Fields(headers[0])
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				unauthorized(w)
				return
			}
			userID, ok := tokens[parts[1]]
			if !ok {
				unauthorized(w)
				return
			}
			ctx := WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

func RetrieveUserID(ctx context.Context) (int64, error) {
	id, ok := ctx.Value(userIDContextKey{}).(int64)
	if !ok || id <= 0 {
		return 0, domain.ErrUnauthenticated
	}
	return id, nil
}
