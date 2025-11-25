package logging

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapErrorPreservesContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithLogRequestID(ctx, "req")

	err := WrapError(ctx, errors.New("boom"))
	require.EqualError(t, err, "boom")

	ctx2 := ErrorCtx(context.Background(), err)
	value, ok := ctx2.Value(key).(logCtx)
	require.True(t, ok)
	require.Equal(t, "req", value.RequestID)
}
