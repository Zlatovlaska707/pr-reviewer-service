package addteam

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"pr-reviewer-service_Avito/internal/api"
	"pr-reviewer-service_Avito/internal/domain"
)

type stubUseCase struct {
	called bool
	result domain.Team
	err    error
}

func (s *stubUseCase) CreateTeam(ctx context.Context, team domain.Team) (domain.Team, error) {
	s.called = true
	if s.result.Name == "" {
		s.result = domain.Team{Name: "backend"}
	}
	return s.result, s.err
}

func TestHandler_Success(t *testing.T) {
	t.Parallel()

	useCase := &stubUseCase{
		result: domain.Team{
			Name: "backend",
			Members: []domain.User{
				{ID: "u1", Username: "Alice"},
			},
		},
	}
	handler := New(useCase)
	router := chi.NewRouter()
	handler.Register(router)

	body := api.Team{
		TeamName: "backend",
		Members: []api.TeamMember{
			{UserId: "u1", Username: "Alice"},
		},
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/add", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.True(t, useCase.called)
}
