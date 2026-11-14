package middleware

import (
	"context"
	"maps"
	"net/http"
	"strings"

	"client-services/internal/domain"
)

type userIDContextKey struct{}

type Authenticator struct{ tokens map[string]int64 }

func NewAuthenticator(tokens map[string]int64) *Authenticator {
	return &Authenticator{tokens: maps.Clone(tokens)}
}

func (a *Authenticator) Authenticate(ctx context.Context, header string) (context.Context, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, domain.ErrUnauthenticated
	}
	id, ok := a.tokens[parts[1]]
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	if current, err := RetrieveUserID(ctx); err == nil && current != id {
		return nil, domain.ErrUnauthenticated
	}
	return WithUserID(ctx, id), nil
}

func Auth(tokens map[string]int64) func(http.Handler) http.Handler {
	return NewAuthenticator(tokens).Middleware
}

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
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
		ctx, err := a.Authenticate(r.Context(), headers[0])
		if err != nil {
			unauthorized(w)
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
