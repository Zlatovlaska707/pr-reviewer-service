package statsassignments

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"pr-reviewer-service_Avito/internal/http/handler/common"
)

// Handler реализует GET /stats/assignments.
type Handler struct {
	useCase UseCase
}

func New(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) Register(router chi.Router) {
	router.Get("/assignments", common.WithErrorHandling(h.handle))
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) error {
	stats, err := h.useCase.Stats(r.Context())
	if err != nil {
		return err
	}
	common.RespondJSON(w, http.StatusOK, stats)
	return nil
}
