package usersetactivity

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"pr-reviewer-service_Avito/internal/domain"
)

type stubUseCase struct {
	input struct {
		id     string
		active bool
	}
}

func (s *stubUseCase) SetUserActivity(ctx context.Context, userID string, active bool) (domain.User, error) {
	s.input.id = userID
	s.input.active = active
	return domain.User{ID: userID, IsActive: active}, nil
}

func TestHandler_ValidatesUserID(t *testing.T) {
	t.Parallel()

	handler := New(&stubUseCase{})
	router := chi.NewRouter()
	handler.Register(router)

	req := httptest.NewRequest(http.MethodPost, "/setIsActive", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_PassesPayload(t *testing.T) {
	t.Parallel()

	useCase := &stubUseCase{}
	handler := New(useCase)
	router := chi.NewRouter()
	handler.Register(router)

	payload, err := json.Marshal(request{UserID: "u1", IsActive: true})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/setIsActive", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "u1", useCase.input.id)
	require.True(t, useCase.input.active)
}
