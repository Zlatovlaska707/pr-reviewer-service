package nower

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNowReturnsRecentTime(t *testing.T) {
	n := New()
	now := n.Now()
	require.WithinDuration(t, time.Now(), now, 50*time.Millisecond)
}
