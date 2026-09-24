package apikey

import (
	"context"

	"github.com/kduong-dev/goutil/fatal"
)

type contextNamespace struct{}

func WithNamespace(ctx context.Context, namespace string) context.Context {
	return context.WithValue(ctx, contextNamespace{}, namespace)
}

func GetNamespace(ctx context.Context) string {
	v := ctx.Value(contextNamespace{})
	namespace, ok := v.(string)
	fatal.Unless(ok, "namespace not found in context")
	return namespace
}
