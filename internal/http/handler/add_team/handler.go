package addteam

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"pr-reviewer-service_Avito/internal/api"
	"pr-reviewer-service_Avito/internal/http/handler/common"
)

// Handler отвечает за HTTP-слой создания команды.
type Handler struct {
	useCase UseCase
}

// New создаёт новый feature-handler.
func New(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

// Register вешает эндпоинт POST /team/add.
func (h *Handler) Register(router chi.Router) {
	router.Post("/add", common.WithErrorHandling(h.handle))
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) error {
	var req api.Team
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return common.NewBadRequestError("INVALID_BODY", "не удалось прочитать тело запроса")
	}
	if req.TeamName == "" {
		return common.NewBadRequestError("VALIDATION_ERROR", "team_name обязателен")
	}
	team, err := h.useCase.CreateTeam(r.Context(), common.ToDomainTeam(req))
	if err != nil {
		return err
	}
	common.RespondJSON(w, http.StatusCreated, map[string]api.Team{"team": common.FromDomainTeam(team)})
	return nil
}
