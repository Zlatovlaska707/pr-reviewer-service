package logging

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type stubHandler struct {
	records []slog.Record
	enabled bool
}

func (h *stubHandler) Enabled(context.Context, slog.Level) bool {
	return h.enabled
}

func (h *stubHandler) Handle(_ context.Context, rec slog.Record) error {
	copy := rec.Clone()
	h.records = append(h.records, copy)
	return nil
}

func (h *stubHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *stubHandler) WithGroup(string) slog.Handler {
	return h
}

func TestLoggerImplHandleAddsContextFields(t *testing.T) {
	base := &stubHandler{enabled: true}
	logger := NewLoggerImpl(base)

	ctx := context.Background()
	ctx = WithLogRequestID(ctx, "req")
	ctx = WithLogRequestPath(ctx, "/path")

	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	require.NoError(t, logger.Handle(ctx, rec))

	require.Len(t, base.records, 1)
	attrs := map[string]any{}
	base.records[0].Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr.Value.Any()
		return true
	})

	require.Equal(t, "req", attrs["requestid"])
	require.Equal(t, "/path", attrs["path"])
}

func TestLoggerImplEnabledDelegates(t *testing.T) {
	base := &stubHandler{enabled: true}
	logger := NewLoggerImpl(base)
	require.True(t, logger.Enabled(context.Background(), slog.LevelInfo))
	base.enabled = false
	require.False(t, logger.Enabled(context.Background(), slog.LevelInfo))
}

func TestLoggerImplWithersWrapHandler(t *testing.T) {
	base := &stubHandler{}
	logger := NewLoggerImpl(base)
	require.IsType(t, &LoggerImpl{}, logger.WithAttrs(nil))
	require.IsType(t, &LoggerImpl{}, logger.WithGroup("grp"))
}
