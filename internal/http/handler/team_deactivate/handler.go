package teamdeactivate

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"pr-reviewer-service_Avito/internal/http/handler/common"
	"pr-reviewer-service_Avito/internal/service"
)

type request struct {
	TeamName string   `json:"team_name"`
	UserIDs  []string `json:"user_ids"`
}

// Handler реализует POST /team/deactivate.
type Handler struct {
	useCase UseCase
}

func New(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) Register(router chi.Router) {
	router.Post("/deactivate", common.WithErrorHandling(h.handle))
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) error {
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return common.NewBadRequestError("INVALID_BODY", "не удалось прочитать тело запроса")
	}
	if req.TeamName == "" {
		return common.NewBadRequestError("VALIDATION_ERROR", "team_name обязателен")
	}
	result, err := h.useCase.MassDeactivate(r.Context(), service.MassDeactivateInput{
		TeamName: req.TeamName,
		UserIDs:  req.UserIDs,
	})
	if err != nil {
		return err
	}
	common.RespondJSON(w, http.StatusOK, result)
	return nil
}
