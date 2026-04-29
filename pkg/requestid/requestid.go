package requestid

import (
	"context"

	"github.com/google/uuid"
)

type contextKey struct{}

func New() string {
	return uuid.NewString()
}

func WithContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

func FromContext(ctx context.Context) string {
	value, _ := ctx.Value(contextKey{}).(string)
	return value
}
