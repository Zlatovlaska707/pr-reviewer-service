package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestLoggerMiddlewareHandlesRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/team/add", nil)
	rctx := chi.NewRouteContext()
	rctx.RoutePatterns = []string{"/team/add"}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	LoggerMiddleware(next).ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestMetricsMiddlewareRecordsRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	body := strings.NewReader("{}")
	req := httptest.NewRequest(http.MethodPost, "/team/add", body)
	req.ContentLength = int64(body.Len())
	rctx := chi.NewRouteContext()
	rctx.RoutePatterns = []string{"/team/add"}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	MetricsMiddleware(next).ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
