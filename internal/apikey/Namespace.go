package apikey

import "context"

type namespaceKey struct{}

func WithNamespace(ctx context.Context, namespace string) context.Context {
	return context.WithValue(ctx, namespaceKey{}, namespace)
}

// GetNamespace returns the namespace the authenticated caller is scoped to,
// or "" when the request did not pass through the Middleware.
func GetNamespace(ctx context.Context) string {
	namespace, _ := ctx.Value(namespaceKey{}).(string)
	return namespace
}
