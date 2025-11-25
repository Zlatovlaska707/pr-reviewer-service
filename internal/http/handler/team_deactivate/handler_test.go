package teamdeactivate

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"pr-reviewer-service_Avito/internal/service"
)

type stubUseCase struct {
	input service.MassDeactivateInput
}

func (s *stubUseCase) MassDeactivate(ctx context.Context, input service.MassDeactivateInput) (service.MassDeactivateResult, error) {
	s.input = input
	return service.MassDeactivateResult{}, nil
}

func TestHandler_ValidatesTeamName(t *testing.T) {
	t.Parallel()

	handler := New(&stubUseCase{})
	router := chi.NewRouter()
	handler.Register(router)

	req := httptest.NewRequest(http.MethodPost, "/deactivate", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_PassesPayloadToUsecase(t *testing.T) {
	t.Parallel()

	useCase := &stubUseCase{}
	handler := New(useCase)
	router := chi.NewRouter()
	handler.Register(router)

	payload, err := json.Marshal(request{
		TeamName: "backend",
		UserIDs:  []string{"u1"},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/deactivate", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "backend", useCase.input.TeamName)
	require.Equal(t, []string{"u1"}, useCase.input.UserIDs)
}
