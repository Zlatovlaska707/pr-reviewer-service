package metrics

import (
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestTechnicalCounters(t *testing.T) {
	path := "/team/{team_name}"
	method := http.MethodPost

	before := testutil.ToFloat64(RestRequestsTotal.WithLabelValues(path))
	IncRestRequestsTotal(path)
	require.Equal(t, before+1, testutil.ToFloat64(RestRequestsTotal.WithLabelValues(path)))

	beforeStatus := testutil.ToFloat64(RestEndpointsResponsesTotal.WithLabelValues(path, http.StatusText(http.StatusCreated)))
	IncRestResponsesStatusesTotal(path, http.StatusCreated)
	require.Equal(t, beforeStatus+1, testutil.ToFloat64(RestEndpointsResponsesTotal.WithLabelValues(path, http.StatusText(http.StatusCreated))))

	IncRestResponsesDuration(path, method, 25*time.Millisecond)
}
