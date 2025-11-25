package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	pgxv5 "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	trm "github.com/avito-tech/go-transaction-manager/trm/v2"
	manager "github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"pr-reviewer-service_Avito/internal/config"
	"pr-reviewer-service_Avito/internal/http/router"
	"pr-reviewer-service_Avito/internal/infrastructure/nower"
	"pr-reviewer-service_Avito/internal/infrastructure/randomizer"
	"pr-reviewer-service_Avito/internal/repository"
	"pr-reviewer-service_Avito/internal/service"
)

// App отвечает за жизненный цикл сервиса.
type App struct {
	cfg    config.Config
	server *http.Server
	repo   *repository.Storage
	trMgr  trm.Manager
}

// New подготавливает все зависимости приложения: БД, репозитории, сервисы, HTTP-роутер.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	// Применяем миграции перед подключением к БД
	if err := runMigrations(cfg); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}

	// Подключаемся к БД с повторными попытками
	pool, err := connectWithRetry(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Инициализация transaction manager для управления транзакциями
	trMgr := manager.Must(pgxv5.NewDefaultFactory(pool))

	// Инициализация инфраструктурных зависимостей
	nowerImpl := nower.New()
	randomizerImpl := randomizer.New()

	repo := repository.New(pool, nowerImpl)
	svc := service.New(repo, cfg, trMgr, randomizerImpl)

	var swaggerSpec []byte
	if data, err := os.ReadFile(cfg.Swagger.SpecPath); err != nil {
		slog.Warn("failed to load swagger spec", "path", cfg.Swagger.SpecPath, "error", err)
	} else {
		swaggerSpec = data
	}
	handler := router.New(svc, swaggerSpec)

	srv := &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      handler.Router(),
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	return &App{
		cfg:    cfg,
		server: srv,
		repo:   repo,
		trMgr:  trMgr,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		slog.Info("HTTP server listening", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		// Graceful shutdown: даём серверу время завершить обработку текущих запросов
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.Timeouts.Shutdown)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		a.repo.Close()
		return nil
	case err := <-errCh:
		// Ошибка при запуске сервера
		a.repo.Close()
		return err
	}
}

func runMigrations(cfg config.Config) error {
	m, err := migrate.New("file://"+cfg.Database.MigrationsPath, cfg.Database.URL)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

// connectWithRetry подключается к БД с экспоненциальной задержкой между попытками.
func connectWithRetry(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	var lastErr error
	// Стратегия повторных попыток: 0s, 1s, 2s, 5s
	backoff := []time.Duration{0, time.Second, 2 * time.Second, 5 * time.Second}
	for attempt, delay := range backoff {
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
		if err != nil {
			lastErr = err
			slog.Warn("failed to parse connection string", "attempt", attempt+1, "error", err)
			continue
		}
		if cfg.Database.MaxConnections > 0 {
			poolCfg.MaxConns = cfg.Database.MaxConnections
		}
		if cfg.Database.MinConnections >= 0 {
			poolCfg.MinConns = cfg.Database.MinConnections
		}
		if cfg.Database.MaxConnIdleTime > 0 {
			poolCfg.MaxConnIdleTime = cfg.Database.MaxConnIdleTime
		}
		if cfg.Database.MaxConnLifetime > 0 {
			poolCfg.MaxConnLifetime = cfg.Database.MaxConnLifetime
		}
		pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
		if err == nil {
			return pool, nil
		}
		lastErr = err
		slog.Warn("failed to connect to database, retrying", "attempt", attempt+1, "error", err)
	}
	return nil, fmt.Errorf("connect db: %w", lastErr)
}
