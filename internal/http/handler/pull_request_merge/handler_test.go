package pullrequestmerge

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
	prID string
}

func (s *stubUseCase) MergePullRequest(ctx context.Context, prID string) (domain.PullRequest, error) {
	s.prID = prID
	return domain.PullRequest{ID: prID}, nil
}

func TestHandler_ValidatesRequest(t *testing.T) {
	t.Parallel()

	handler := New(&stubUseCase{})
	router := chi.NewRouter()
	handler.Register(router)

	req := httptest.NewRequest(http.MethodPost, "/merge", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_PassesID(t *testing.T) {
	t.Parallel()

	useCase := &stubUseCase{}
	handler := New(useCase)
	router := chi.NewRouter()
	handler.Register(router)

	payload, err := json.Marshal(request{ID: "pr-1"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/merge", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "pr-1", useCase.prID)
}
