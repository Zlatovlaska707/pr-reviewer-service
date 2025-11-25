package getteam

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"pr-reviewer-service_Avito/internal/http/handler/common"
)

// Handler реализует HTTP-эндпоинт получения команды.
type Handler struct {
	useCase UseCase
}

func New(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) Register(router chi.Router) {
	router.Get("/get", common.WithErrorHandling(h.handle))
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) error {
	name := r.URL.Query().Get("team_name")
	if name == "" {
		return common.NewBadRequestError("VALIDATION_ERROR", "team_name обязателен")
	}
	team, err := h.useCase.GetTeam(r.Context(), name)
	if err != nil {
		return err
	}
	common.RespondJSON(w, http.StatusOK, common.FromDomainTeam(team))
	return nil
}
