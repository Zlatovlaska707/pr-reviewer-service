package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"pr-reviewer-service_Avito/internal/logging"
	"pr-reviewer-service_Avito/internal/metrics"
)

// LoggerMiddleware создаёт middleware для структурированного логирования HTTP запросов.
// Добавляет в контекст request ID, путь, метод и измеряет время выполнения запроса.
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Получаем шаблон пути из роутера (например, "/team/{id}") вместо конкретного пути
		var pathTemplate string
		if rctx := chi.RouteContext(ctx); rctx != nil {
			if pattern := rctx.RoutePattern(); pattern != "" {
				pathTemplate = pattern
			}
		}
		if pathTemplate == "" {
			pathTemplate = r.URL.Path
		}

		metrics.IncRestRequestsTotal(pathTemplate)

		// Генерируем уникальный ID для запроса
		requestID := uuid.New()
		slog.InfoContext(ctx, fmt.Sprintf("Start [%s] request processing", requestID.String()))
		start := time.Now()

		// Добавляем метаданные запроса в контекст для последующего логирования
		ctx = logging.WithLogRequestID(ctx, requestID.String())
		ctx = logging.WithLogRequestPath(ctx, r.URL.Path)
		ctx = logging.WithLogRequestMethod(ctx, r.Method)

		rw := &responseWriter{w, http.StatusOK}
		r = r.WithContext(ctx)

		next.ServeHTTP(rw, r)

		timeServe := time.Since(start)
		ctx = r.Context()
		ctx = logging.WithLogRequestStatus(ctx, rw.statusCode)
		ctx = logging.WithLogRequestDuration(ctx, timeServe.String())

		slog.InfoContext(ctx, fmt.Sprintf("Ended [%s] request processing", requestID.String()))

		metrics.IncRestResponsesDuration(pathTemplate, r.Method, timeServe)
		metrics.IncRestResponsesStatusesTotal(pathTemplate, rw.statusCode)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
