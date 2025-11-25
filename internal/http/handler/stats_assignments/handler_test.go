package statsassignments

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
	called bool
}

func (s *stubUseCase) Stats(ctx context.Context) (domain.AssignmentStats, error) {
	s.called = true
	return domain.AssignmentStats{}, nil
}

func TestHandler_ReturnsStats(t *testing.T) {
	t.Parallel()

	useCase := &stubUseCase{}
	handler := New(useCase)
	router := chi.NewRouter()
	handler.Register(router)

	req := httptest.NewRequest(http.MethodGet, "/assignments", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, useCase.called)
}
