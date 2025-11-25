package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestGetEndpointPrefersRoutePattern(t *testing.T) {
	rctx := chi.NewRouteContext()
	rctx.RoutePatterns = []string{"/team/{team_name}"}

	req := httptest.NewRequest(http.MethodGet, "/team/backend", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	require.Equal(t, "/team/{team_name}", getEndpoint(req))

	require.Equal(t, "/custom", getEndpoint(httptest.NewRequest(http.MethodGet, "/custom", nil)))
	require.Equal(t, "/", getEndpoint(nil))
}
