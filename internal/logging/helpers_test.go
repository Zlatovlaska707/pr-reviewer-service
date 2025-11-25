package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithLogContextHelpersMutateSameStruct(t *testing.T) {
	ctx := context.Background()
	ctx = WithLogRequestID(ctx, "req")
	ctx = WithLogRequestPath(ctx, "/path")
	ctx = WithLogRequestMethod(ctx, "GET")
	ctx = WithLogRequestStatus(ctx, 200)
	ctx = WithLogRequestDuration(ctx, "5ms")
	ctx = WithLogTeamName(ctx, "backend")
	ctx = WithLogTeamMembersCount(ctx, 3)
	ctx = WithLogUserID(ctx, "user-1")
	ctx = WithLogAuthorID(ctx, "author-1")
	ctx = WithLogPullRequestID(ctx, "pr-1")

	value, ok := ctx.Value(key).(logCtx)
	require.True(t, ok)
	require.Equal(t, "req", value.RequestID)
	require.Equal(t, "/path", value.Path)
	require.Equal(t, "GET", value.Method)
	require.Equal(t, 200, value.Status)
	require.Equal(t, "5ms", value.RequestDuration)
	require.Equal(t, "backend", value.TeamName)
	require.Equal(t, 3, value.TeamMembersCount)
	require.Equal(t, "user-1", value.UserID)
	require.Equal(t, "author-1", value.AuthorID)
	require.Equal(t, "pr-1", value.PullRequestID)
}
