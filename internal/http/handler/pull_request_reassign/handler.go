package pullrequestreassign

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"pr-reviewer-service_Avito/internal/http/handler/common"
)

type request struct {
	PRID      string `json:"pull_request_id"`
	OldUserID string `json:"old_user_id"`
}

// Handler реализует POST /pullRequest/reassign.
type Handler struct {
	useCase UseCase
}

func New(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) Register(router chi.Router) {
	router.Post("/reassign", common.WithErrorHandling(h.handle))
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) error {
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return common.NewBadRequestError("INVALID_BODY", "не удалось прочитать тело запроса")
	}
	if req.PRID == "" || req.OldUserID == "" {
		return common.NewBadRequestError("VALIDATION_ERROR", "pull_request_id и old_user_id обязательны")
	}
	pr, replacedBy, err := h.useCase.ReassignReviewer(r.Context(), req.PRID, req.OldUserID)
	if err != nil {
		return err
	}
	common.RespondJSON(w, http.StatusOK, map[string]any{
		"pr":          pr,
		"replaced_by": replacedBy,
	})
	return nil
}
