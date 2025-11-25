package logging

import (
	"context"
	"errors"
)

// errorWithLogCtx оборачивает ошибку с контекстом логирования.
type errorWithLogCtx struct {
	next error
	ctx  logCtx
}

// Error возвращает строковое представление ошибки.
func (e *errorWithLogCtx) Error() string {
	return e.next.Error()
}

// Unwrap возвращает исходную ошибку.
func (e *errorWithLogCtx) Unwrap() error {
	return e.next
}

// WrapError оборачивает ошибку с контекстом из ctx.
func WrapError(ctx context.Context, err error) error {
	c := logCtx{}
	if x, ok := ctx.Value(key).(logCtx); ok {
		c = x
	}
	return &errorWithLogCtx{
		next: err,
		ctx:  c,
	}
}

// ErrorCtx извлекает контекст из ошибки и возвращает его в context.Context.
func ErrorCtx(ctx context.Context, err error) context.Context {
	var e *errorWithLogCtx
	if errors.As(err, &e) {
		return context.WithValue(ctx, key, e.ctx)
	}
	return ctx
}
