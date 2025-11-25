package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestBusinessCounters(t *testing.T) {
	beforeTeams := testutil.ToFloat64(teamsCreated)
	IncTeamsCreated()
	require.Equal(t, beforeTeams+1, testutil.ToFloat64(teamsCreated))

	beforePR := testutil.ToFloat64(prCreated)
	IncPullRequestsCreated()
	require.Equal(t, beforePR+1, testutil.ToFloat64(prCreated))

	beforeReassign := testutil.ToFloat64(reassignments)
	IncReassignments()
	require.Equal(t, beforeReassign+1, testutil.ToFloat64(reassignments))
}

func TestAddUsersProcessedIgnoresNonPositive(t *testing.T) {
	before := testutil.ToFloat64(usersProcessed)
	AddUsersProcessed(-1)
	require.Equal(t, before, testutil.ToFloat64(usersProcessed))
	AddUsersProcessed(2)
	require.Equal(t, before+2, testutil.ToFloat64(usersProcessed))
}
