package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"pr-reviewer-service_Avito/internal/api"
)

func TestPanicMiddlewareConvertsPanicToErrorResponse(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})

	handler := PanicMiddleware(next)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	var resp api.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "UNKNOWN", string(resp.Error.Code))
}
