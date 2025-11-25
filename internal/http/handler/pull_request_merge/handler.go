package pullrequestmerge

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"pr-reviewer-service_Avito/internal/domain"
	"pr-reviewer-service_Avito/internal/http/handler/common"
)

type request struct {
	ID string `json:"pull_request_id"`
}

// Handler реализует POST /pullRequest/merge.
type Handler struct {
	useCase UseCase
}

func New(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) Register(router chi.Router) {
	router.Post("/merge", common.WithErrorHandling(h.handle))
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) error {
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return common.NewBadRequestError("INVALID_BODY", "не удалось прочитать тело запроса")
	}
	if req.ID == "" {
		return common.NewBadRequestError("VALIDATION_ERROR", "pull_request_id обязателен")
	}
	pr, err := h.useCase.MergePullRequest(r.Context(), req.ID)
	if err != nil {
		return err
	}
	common.RespondJSON(w, http.StatusOK, map[string]domain.PullRequest{"pr": pr})
	return nil
}
