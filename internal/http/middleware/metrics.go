package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// httpRequestsTotal общее количество HTTP запросов
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// httpRequestDuration гистограмма длительности HTTP запросов
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status"},
	)

	// httpRequestSize размер тела запроса
	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 50000},
		},
		[]string{"method", "endpoint"},
	)
)

// MetricsMiddleware собирает метрики для всех HTTP запросов.
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Обёртка для ResponseWriter для получения статус-кода
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Обрабатываем запрос
		next.ServeHTTP(ww, r)

		// Получаем метрики
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(ww.Status())
		method := r.Method
		endpoint := getEndpoint(r)

		// Записываем метрики
		httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
		httpRequestDuration.WithLabelValues(method, endpoint, status).Observe(duration)

		// Размер запроса
		if r.ContentLength > 0 {
			httpRequestSize.WithLabelValues(method, endpoint).Observe(float64(r.ContentLength))
		}
	})
}

// getEndpoint нормализует путь для метрик, используя шаблон маршрута вместо конкретного пути.
// Это позволяет группировать метрики по эндпоинтам, а н по конкретным значениям параметров
func getEndpoint(r *http.Request) string {
	if r == nil {
		return "/"
	}
	// Пытаемся получить шаблон маршрута (например, "/team/{id}") из контекста
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	// Если шаблон недоступен, используем реальный путь
	if path := r.URL.Path; path != "" {
		return path
	}
	return "/"
}
