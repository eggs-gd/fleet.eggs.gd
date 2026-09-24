package taskflow

import "context"

type statusOverrideCtxKey struct{}

// WithStatusOverride marks a provider Update as a system status override that
// may skip DOMAIN_MODEL transition checks at the adapter Mutate layer.
// TaskService.Patch sets this when PatchInput.AllowStatusOverride is true.
func WithStatusOverride(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, statusOverrideCtxKey{}, true)
}

// StatusOverrideAllowed reports whether WithStatusOverride was attached.
func StatusOverrideAllowed(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	ok, _ := ctx.Value(statusOverrideCtxKey{}).(bool)
	return ok
}
