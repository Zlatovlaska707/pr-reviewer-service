package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	addteam "pr-reviewer-service_Avito/internal/http/handler/add_team"
	"pr-reviewer-service_Avito/internal/http/handler/common"
	getteam "pr-reviewer-service_Avito/internal/http/handler/get_team"
	pullrequestcreate "pr-reviewer-service_Avito/internal/http/handler/pull_request_create"
	pullrequestmerge "pr-reviewer-service_Avito/internal/http/handler/pull_request_merge"
	pullrequestreassign "pr-reviewer-service_Avito/internal/http/handler/pull_request_reassign"
	statsassignments "pr-reviewer-service_Avito/internal/http/handler/stats_assignments"
	teamdeactivate "pr-reviewer-service_Avito/internal/http/handler/team_deactivate"
	usergetreview "pr-reviewer-service_Avito/internal/http/handler/user_get_review"
	usersetactivity "pr-reviewer-service_Avito/internal/http/handler/user_set_activity"
	"pr-reviewer-service_Avito/internal/http/middleware"
	"pr-reviewer-service_Avito/internal/http/swagger"
	"pr-reviewer-service_Avito/internal/service"
)

// Handler агрегирует HTTP-эндпоинты.
type Handler struct {
	service     *service.Service
	swaggerSpec []byte
}

func New(service *service.Service, spec []byte) *Handler {
	return &Handler{service: service, swaggerSpec: spec}
}

// Router возвращает готовый chi.Router со всеми зарегистрированными маршрутами и middleware.
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	// Middleware применяются в порядке объявления
	r.Use(chimw.RequestID)              // Добавляет уникальный ID каждому запросу
	r.Use(chimw.RealIP)                 // Определяет реальный IP клиента
	r.Use(middleware.PanicMiddleware)   // Перехватывает паники
	r.Use(middleware.LoggerMiddleware)  // Логирует все запросы
	r.Use(middleware.MetricsMiddleware) // Собирает метрики Prometheus
	swagger.RegisterRoutes(r, h.swaggerSpec)

	// Health check эндпоинт для проверки доступности сервиса
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := h.service.HealthCheck(r.Context()); err != nil {
			slog.ErrorContext(r.Context(), "health check failed", "error", err)
			common.RespondJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "degraded",
				"error":  err.Error(),
			})
			return
		}
		common.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Prometheus metrics endpoint для сбора метрик
	r.Handle("/metrics", promhttp.Handler())

	h.registerTeamRoutes(r)
	h.registerUserRoutes(r)
	h.registerPullRequestRoutes(r)
	h.registerStatsRoutes(r)

	return r
}

func (h *Handler) registerTeamRoutes(r chi.Router) {
	r.Route("/team", func(router chi.Router) {
		addteam.New(h.service).Register(router)
		getteam.New(h.service).Register(router)
		teamdeactivate.New(h.service).Register(router)
	})
}

func (h *Handler) registerUserRoutes(r chi.Router) {
	r.Route("/users", func(router chi.Router) {
		usersetactivity.New(h.service).Register(router)
		usergetreview.New(h.service).Register(router)
	})
}

func (h *Handler) registerPullRequestRoutes(r chi.Router) {
	r.Route("/pullRequest", func(router chi.Router) {
		pullrequestcreate.New(h.service).Register(router)
		pullrequestmerge.New(h.service).Register(router)
		pullrequestreassign.New(h.service).Register(router)
	})
}

func (h *Handler) registerStatsRoutes(r chi.Router) {
	r.Route("/stats", func(router chi.Router) {
		statsassignments.New(h.service).Register(router)
	})
}
