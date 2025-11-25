package main

import (
	"context"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"pr-reviewer-service_Avito/internal/app"
	"pr-reviewer-service_Avito/internal/config"
	"pr-reviewer-service_Avito/internal/logging"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := config.MustLoad()
	cleanup := setupLogger(cfg)
	defer cleanup()

	application, err := app.New(ctx, cfg)
	if err != nil {
		slog.Error("failed to init app", "error", err)
		os.Exit(1)
	}

	if err := application.Run(ctx); err != nil {
		slog.Error("application stopped with error", "error", err)
		os.Exit(1)
	}
}

// setupLogger настраивает структурированное логирование на основе конфигурации.
// Возвращает функцию для закрытия файлового дескриптора (если используется файл).
func setupLogger(cfg config.Config) func() {
	output := strings.ToLower(cfg.Logging.Output)
	var writer io.Writer
	var closer io.Closer

	// Определяем куда писать логи: stdout, stderr или файл
	switch output {
	case "stdout", "":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		// Если указан файл, создаём директорию и открываем файл для записи
		if err := os.MkdirAll(filepath.Dir(cfg.Logging.Output), 0o755); err != nil {
			log.Fatalf("failed to create log directory: %v", err)
		}
		f, err := os.OpenFile(cfg.Logging.Output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			log.Fatalf("failed to open log file: %v", err)
		}
		writer = f
		closer = f
	}

	var level slog.Level
	switch strings.ToLower(cfg.Logging.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.Handler(slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: level,
	}))
	handler = logging.NewLoggerImpl(handler)
	slog.SetDefault(slog.New(handler))

	return func() {
		if closer != nil {
			_ = closer.Close()
		}
	}
}
