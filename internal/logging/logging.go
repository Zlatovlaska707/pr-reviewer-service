package logging

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"strings"
)

type keyType int

const key = keyType(0)

// logCtx содержит контекстную информацию для логирования.
type logCtx struct {
	RequestID        string
	Status           int
	RequestStart     string
	RequestDuration  string
	Method           string
	Path             string
	TeamName         string
	TeamMembersCount int
	UserID           string
	AuthorID         string
	PullRequestID    string
}

// LoggerImpl оборачивает slog.Handler для добавления контекстной информации.
type LoggerImpl struct {
	next slog.Handler
}

func NewLoggerImpl(next slog.Handler) *LoggerImpl {
	return &LoggerImpl{next: next}
}

// Enabled проверяет, включён ли указанный уровень логирования.
func (h *LoggerImpl) Enabled(ctx context.Context, rec slog.Level) bool {
	return h.next.Enabled(ctx, rec)
}

// Handle обрабатывает запись лога, добавляя контекстную информацию.
func (h *LoggerImpl) Handle(ctx context.Context, rec slog.Record) error {
	if c, ok := ctx.Value(key).(logCtx); ok {
		v := reflect.ValueOf(c)
		t := v.Type()

		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if !field.IsZero() {
				fieldName := strings.ToLower(t.Field(i).Name)
				rec.Add(fieldName, field.Interface())
			}
		}
	}

	if rec.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{rec.PC})
		f, _ := fs.Next()
		rec.Add("source", fmt.Sprintf("%s:%d", f.File, f.Line))
	}

	return h.next.Handle(ctx, rec)
}

// WithAttrs добавляет атрибуты к следующему обработчику.
func (h *LoggerImpl) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LoggerImpl{next: h.next.WithAttrs(attrs)}
}

// WithGroup добавляет группу к следующему обработчику.
func (h *LoggerImpl) WithGroup(name string) slog.Handler {
	return &LoggerImpl{next: h.next.WithGroup(name)}
}
