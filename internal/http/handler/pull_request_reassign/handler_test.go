package pullrequestreassign

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
	prID     string
	reviewer string
}

func (s *stubUseCase) ReassignReviewer(ctx context.Context, prID, oldReviewer string) (domain.PullRequest, string, error) {
	s.prID = prID
	s.reviewer = oldReviewer
	return domain.PullRequest{ID: prID}, "u2", nil
}

func TestHandler_ValidatesRequest(t *testing.T) {
	t.Parallel()

	handler := New(&stubUseCase{})
	router := chi.NewRouter()
	handler.Register(router)

	req := httptest.NewRequest(http.MethodPost, "/reassign", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_PassesArgs(t *testing.T) {
	t.Parallel()

	useCase := &stubUseCase{}
	handler := New(useCase)
	router := chi.NewRouter()
	handler.Register(router)

	payload, err := json.Marshal(request{PRID: "pr-1", OldUserID: "u1"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/reassign", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "pr-1", useCase.prID)
	require.Equal(t, "u1", useCase.reviewer)
}
