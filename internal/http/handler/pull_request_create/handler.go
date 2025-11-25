package pullrequestcreate

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"pr-reviewer-service_Avito/internal/domain"
	"pr-reviewer-service_Avito/internal/http/handler/common"
)

type request struct {
	ID     string `json:"pull_request_id"`
	Name   string `json:"pull_request_name"`
	Author string `json:"author_id"`
}

// Handler реализует POST /pullRequest/create.
type Handler struct {
	useCase UseCase
}

func New(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) Register(router chi.Router) {
	router.Post("/create", common.WithErrorHandling(h.handle))
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) error {
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return common.NewBadRequestError("INVALID_BODY", "не удалось прочитать тело запроса")
	}
	if req.ID == "" || req.Name == "" || req.Author == "" {
		return common.NewBadRequestError("VALIDATION_ERROR", "все поля обязательны")
	}
	pr, err := h.useCase.CreatePullRequest(r.Context(), req.ID, req.Name, req.Author)
	if err != nil {
		return err
	}
	common.RespondJSON(w, http.StatusCreated, map[string]domain.PullRequest{"pr": pr})
	return nil
}
