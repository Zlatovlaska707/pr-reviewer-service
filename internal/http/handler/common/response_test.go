package common

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/require"

	"pr-reviewer-service_Avito/internal/domain"
)

func TestRespondJSONWritesBodyAndStatus(t *testing.T) {
	rec := httptest.NewRecorder()

	RespondJSON(rec, http.StatusAccepted, map[string]string{"ok": "true"})

	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var body map[string]string
	require.NoError(t, json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&body))
	require.Equal(t, "true", body["ok"])
}

func TestWithErrorHandlingReturnsHTTPError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler := WithErrorHandling(func(http.ResponseWriter, *http.Request) error {
		return NewHTTPError(http.StatusTeapot, "CUSTOM", "boom")
	})
	handler(rec, req)

	require.Equal(t, http.StatusTeapot, rec.Code)
	var apiErr APIError
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &apiErr))
	require.Equal(t, "CUSTOM", apiErr.Error.Code)
}

func TestWithErrorHandlingFallsBackToDomainErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), chimw.RequestIDKey, "req-1")
	req = req.WithContext(ctx)

	handler := WithErrorHandling(func(http.ResponseWriter, *http.Request) error {
		return domain.ErrTeamExists
	})
	handler(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var apiErr APIError
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &apiErr))
	require.Equal(t, "TEAM_EXISTS", apiErr.Error.Code)
}

func TestWriteDomainErrorUnknownError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	WriteDomainError(rec, req, errors.New("unexpected"))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	var apiErr APIError
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &apiErr))
	require.Equal(t, "INTERNAL_ERROR", apiErr.Error.Code)
}
