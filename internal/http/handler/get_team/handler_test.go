package getteam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"pr-reviewer-service_Avito/internal/domain"
)

type stubUseCase struct {
	calledWith string
}

func (s *stubUseCase) GetTeam(ctx context.Context, name string) (domain.Team, error) {
	s.calledWith = name
	return domain.Team{Name: name}, nil
}

func TestHandler_ReturnsBadRequestWhenTeamNameMissing(t *testing.T) {
	t.Parallel()

	handler := New(&stubUseCase{})
	router := chi.NewRouter()
	handler.Register(router)

	req := httptest.NewRequest(http.MethodGet, "/get", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_PassesTeamNameToUsecase(t *testing.T) {
	t.Parallel()

	useCase := &stubUseCase{}
	handler := New(useCase)
	router := chi.NewRouter()
	handler.Register(router)

	req := httptest.NewRequest(http.MethodGet, "/get?team_name=backend", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "backend", useCase.calledWith)
}
