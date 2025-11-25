package usersetactivity

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"pr-reviewer-service_Avito/internal/domain"
	"pr-reviewer-service_Avito/internal/http/handler/common"
)

type request struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

// Handler реализует POST /users/setIsActive.
type Handler struct {
	useCase UseCase
}

func New(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) Register(router chi.Router) {
	router.Post("/setIsActive", common.WithErrorHandling(h.handle))
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) error {
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return common.NewBadRequestError("INVALID_BODY", "не удалось прочитать тело запроса")
	}
	if req.UserID == "" {
		return common.NewBadRequestError("VALIDATION_ERROR", "user_id обязателен")
	}
	user, err := h.useCase.SetUserActivity(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		return err
	}
	common.RespondJSON(w, http.StatusOK, map[string]domain.User{"user": user})
	return nil
}
