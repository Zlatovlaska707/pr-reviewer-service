package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	pgxv5 "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	manager "github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"pr-reviewer-service_Avito/internal/config"
	"pr-reviewer-service_Avito/internal/http/router"
	"pr-reviewer-service_Avito/internal/infrastructure/nower"
	"pr-reviewer-service_Avito/internal/infrastructure/randomizer"
	"pr-reviewer-service_Avito/internal/repository"
	"pr-reviewer-service_Avito/internal/service"
)

func TestHappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("пропуск интеграционного теста в режиме -short")
	}
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("testcontainers не поддерживается в среде Windows CI")
	}

	ctx := context.Background()
	pgContainer, dsn := setupPostgres(t, ctx)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	runMigrations(t, dsn)

	// Подключаемся с повторными попытками для стабильности в CI
	var pool *pgxpool.Pool
	var err error
	for i := 0; i < 5; i++ {
		pool, err = pgxpool.New(ctx, dsn)
		if err == nil {
			err = pool.Ping(ctx)
			if err == nil {
				break
			}
			pool.Close()
		}
		if i < 4 {
			time.Sleep(time.Second)
		}
	}
	require.NoError(t, err, "failed to connect to database after retries")
	defer pool.Close()

	repo := repository.New(pool, nower.New())
	trMgr := manager.Must(pgxv5.NewDefaultFactory(pool))
	rnd := randomizer.New()
	svc := service.New(repo, config.Config{
		Timeouts: config.TimeoutConfig{
			Operation:     time.Second,
			LongOperation: 2 * time.Second,
		},
	}, trMgr, rnd)
	// Определяем путь к openapi.yml относительно корня проекта
	cwd, err := os.Getwd()
	require.NoError(t, err)
	root := filepath.Clean(filepath.Join(cwd, "..", ".."))
	openapiPath := filepath.Join(root, "openapi.yml")
	spec, err := os.ReadFile(openapiPath)
	require.NoError(t, err)
	h := router.New(svc, spec)
	server := httptest.NewServer(h.Router())
	defer server.Close()

	teamBody := map[string]any{
		"team_name": "backend",
		"members": []map[string]any{
			{"user_id": "u1", "username": "Alice", "is_active": true},
			{"user_id": "u2", "username": "Bob", "is_active": true},
			{"user_id": "u3", "username": "Charlie", "is_active": true},
		},
	}
	resp := doRequest(t, server, http.MethodPost, "/team/add", teamBody)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	prBody := map[string]string{
		"pull_request_id":   "pr-1",
		"pull_request_name": "Add feature",
		"author_id":         "u1",
	}
	resp = doRequest(t, server, http.MethodPost, "/pullRequest/create", prBody)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var prResp struct {
		PR struct {
			Assigned []string `json:"assigned_reviewers"`
		} `json:"pr"`
	}
	decode(t, resp, &prResp)
	require.NotEmpty(t, prResp.PR.Assigned)

	reassignBody := map[string]string{
		"pull_request_id": "pr-1",
		"old_user_id":     prResp.PR.Assigned[0],
	}
	resp = doRequest(t, server, http.MethodPost, "/pullRequest/reassign", reassignBody)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, server, http.MethodPost, "/pullRequest/merge", map[string]string{"pull_request_id": "pr-1"})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, server, http.MethodPost, "/pullRequest/reassign", reassignBody)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, server, http.MethodPost, "/team/deactivate", map[string]any{
		"team_name": "backend",
		"user_ids":  []string{"u2"},
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func setupPostgres(t *testing.T, ctx context.Context) (*tcpostgres.PostgresContainer, string) {
	t.Helper()
	// Увеличиваем таймаут для CI окружения
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("pr_reviewer"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
	)
	require.NoError(t, err)

	// Ждём готовности контейнера с повторными попытками
	var connStr string
	var lastErr error
	for i := 0; i < 10; i++ {
		connStr, err = container.ConnectionString(ctx, "sslmode=disable")
		if err == nil {
			// Проверяем подключение
			pool, testErr := pgxpool.New(ctx, connStr)
			if testErr == nil {
				testErr = pool.Ping(ctx)
				pool.Close()
				if testErr == nil {
					return container, connStr
				}
				lastErr = testErr
			} else {
				lastErr = testErr
			}
		} else {
			lastErr = err
		}
		if i < 9 {
			time.Sleep(time.Second)
		}
	}
	require.NoError(t, lastErr, "failed to connect to postgres container after retries")
	return container, connStr
}

func runMigrations(t *testing.T, dsn string) {
	t.Helper()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	root := filepath.Clean(filepath.Join(cwd, "..", ".."))
	migrationsPath := filepath.Join(root, "migrations")
	m, err := migrate.New("file://"+migrationsPath, dsn)
	require.NoError(t, err)
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		require.NoError(t, err)
	}
}

func doRequest(t *testing.T, server *httptest.Server, method, path string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req, err := http.NewRequest(method, server.URL+path, &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func decode(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(v))
}
