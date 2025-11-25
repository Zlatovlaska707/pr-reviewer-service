package pullrequestcreate

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
	args struct {
		id     string
		name   string
		author string
	}
}

func (s *stubUseCase) CreatePullRequest(ctx context.Context, prID, name, authorID string) (domain.PullRequest, error) {
	s.args.id = prID
	s.args.name = name
	s.args.author = authorID
	return domain.PullRequest{ID: prID, Name: name, AuthorID: authorID}, nil
}

func TestHandler_ValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	handler := New(&stubUseCase{})
	router := chi.NewRouter()
	handler.Register(router)

	req := httptest.NewRequest(http.MethodPost, "/create", bytes.NewBufferString(`{}`))
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

	payload, err := json.Marshal(request{ID: "pr-1", Name: "Feature", Author: "u1"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/create", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "pr-1", useCase.args.id)
	require.Equal(t, "Feature", useCase.args.name)
	require.Equal(t, "u1", useCase.args.author)
}
