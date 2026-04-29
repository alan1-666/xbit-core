package auth

import (
	"context"
	"net/http"
	"strings"
)

type claimsContextKey struct{}

func WithClaims(ctx context.Context, claims *AccessClaims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (*AccessClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(*AccessClaims)
	return claims, ok
}

func BearerMiddleware(tokens *TokenManager, onUnauthorized http.HandlerFunc) func(http.Handler) http.Handler {
	if onUnauthorized == nil {
		onUnauthorized = func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := BearerToken(r.Header.Get("Authorization"))
			if raw == "" {
				onUnauthorized(w, r)
				return
			}
			claims, err := tokens.ParseAccessToken(raw)
			if err != nil {
				onUnauthorized(w, r)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}

func BearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
