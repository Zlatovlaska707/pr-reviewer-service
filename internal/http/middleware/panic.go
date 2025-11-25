package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"pr-reviewer-service_Avito/internal/api"
)

// PanicMiddleware перехватывает паники и возвращает корректный HTTP ответ.
func PanicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			if err := recover(); err != nil {
				slog.Error(
					"Перехвачена паника",
					"method", r.Method,
					"url", r.URL.Path,
					"time", time.Since(start),
					"error", err,
					"stack_trace", string(debug.Stack()),
				)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				response := api.ErrorResponse{
					Error: struct {
						Code    api.ErrorResponseErrorCode `json:"code"`
						Message string                     `json:"message"`
					}{
						Code:    api.ErrorResponseErrorCode("UNKNOWN"),
						Message: "Internal server error",
					},
				}
				_ = json.NewEncoder(w).Encode(response)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
