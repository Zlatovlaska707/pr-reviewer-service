package common

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"

	"pr-reviewer-service_Avito/internal/domain"
)

type APIError struct {
	Error APIErrorBody `json:"error"`
}

type APIErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RespondJSON отправляет JSON-ответ с указанным статус-кодом.
func RespondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// BadRequest отправляет JSON-ответ со статусом 400.
func BadRequest(w http.ResponseWriter, code, message string) {
	RespondJSON(w, http.StatusBadRequest, APIError{
		Error: APIErrorBody{Code: code, Message: message},
	})
}

// HTTPError описывает контролируемую HTTP-ошибку.
type HTTPError struct {
	status  int
	code    string
	message string
}

func (e *HTTPError) Error() string {
	return e.message
}

// NewHTTPError создаёт новую HTTP-ошибку.
func NewHTTPError(status int, code, message string) *HTTPError {
	return &HTTPError{
		status:  status,
		code:    code,
		message: message,
	}
}

// NewBadRequestError создаёт 400 ошибку.
func NewBadRequestError(code, message string) *HTTPError {
	return NewHTTPError(http.StatusBadRequest, code, message)
}

// WithErrorHandling оборачивает обработчик, централизуя выдачу ошибок.
// Преобразует доменные ошибки в HTTP-ответы с соответствующими статус-кодами.
func WithErrorHandling(fn func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			var httpErr *HTTPError
			// Если ошибка уже является HTTPError, используем её статус и код
			if errors.As(err, &httpErr) {
				RespondJSON(w, httpErr.status, APIError{
					Error: APIErrorBody{Code: httpErr.code, Message: httpErr.message},
				})
				return
			}
			// Иначе преобразуем доменную ошибку в HTTP-ответ
			WriteDomainError(w, r, err)
		}
	}
}

// WriteDomainError преобразует доменные ошибки в HTTP-ответы.
func WriteDomainError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()
	requestID := chimw.GetReqID(ctx)

	switch err {
	case domain.ErrTeamExists:
		slog.DebugContext(ctx, "team already exists", "request_id", requestID, "error", err)
		RespondJSON(w, http.StatusBadRequest, APIError{Error: APIErrorBody{Code: "TEAM_EXISTS", Message: err.Error()}})
	case domain.ErrTeamNotFound, domain.ErrUserNotFound, domain.ErrPRNotFound:
		slog.DebugContext(ctx, "resource not found", "request_id", requestID, "error", err)
		RespondJSON(w, http.StatusNotFound, APIError{Error: APIErrorBody{Code: "NOT_FOUND", Message: err.Error()}})
	case domain.ErrPRExists:
		slog.DebugContext(ctx, "PR already exists", "request_id", requestID, "error", err)
		RespondJSON(w, http.StatusConflict, APIError{Error: APIErrorBody{Code: "PR_EXISTS", Message: err.Error()}})
	case domain.ErrPRMerged:
		slog.DebugContext(ctx, "PR already merged", "request_id", requestID, "error", err)
		RespondJSON(w, http.StatusConflict, APIError{Error: APIErrorBody{Code: "PR_MERGED", Message: err.Error()}})
	case domain.ErrReviewerAbsent:
		slog.DebugContext(ctx, "reviewer not assigned", "request_id", requestID, "error", err)
		RespondJSON(w, http.StatusConflict, APIError{Error: APIErrorBody{Code: "NOT_ASSIGNED", Message: err.Error()}})
	case domain.ErrNoCandidate:
		slog.DebugContext(ctx, "no candidate for reassignment", "request_id", requestID, "error", err)
		RespondJSON(w, http.StatusConflict, APIError{Error: APIErrorBody{Code: "NO_CANDIDATE", Message: err.Error()}})
	default:
		slog.ErrorContext(ctx, "unhandled domain error", "request_id", requestID, "error", err)
		RespondJSON(w, http.StatusInternalServerError, APIError{Error: APIErrorBody{Code: "INTERNAL_ERROR", Message: "internal server error"}})
	}
}
