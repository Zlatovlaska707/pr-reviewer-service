package usergetreview

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"pr-reviewer-service_Avito/internal/http/handler/common"
)

// Handler реализует GET /users/getReview.
type Handler struct {
	useCase UseCase
}

func New(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) Register(router chi.Router) {
	router.Get("/getReview", common.WithErrorHandling(h.handle))
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) error {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		return common.NewBadRequestError("VALIDATION_ERROR", "user_id обязателен")
	}
	assignments, err := h.useCase.ListReviewAssignments(r.Context(), userID)
	if err != nil {
		return err
	}
	common.RespondJSON(w, http.StatusOK, map[string]any{
		"user_id":       userID,
		"pull_requests": assignments,
	})
	return nil
}
