package taskflow

import (
	"context"
	"strings"
)

type commentAuthorCtxKey struct{}

// WithCommentAuthor attaches an optional comment author for provider adapters
// that support authored review comments (Markdown "## Review Comments").
// TaskService.Patch uses this so manager/dashboard authors are preserved
// without widening TaskProvider.AddComment's signature.
func WithCommentAuthor(ctx context.Context, author string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	author = strings.TrimSpace(author)
	if author == "" {
		return ctx
	}
	return context.WithValue(ctx, commentAuthorCtxKey{}, author)
}

// CommentAuthorFrom returns the author attached by WithCommentAuthor, if any.
func CommentAuthorFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	author, _ := ctx.Value(commentAuthorCtxKey{}).(string)
	return strings.TrimSpace(author)
}
