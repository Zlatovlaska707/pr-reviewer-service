package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"pr-reviewer-service_Avito/internal/config"
)

func TestSetupLoggerCreatesFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "app.log")
	cfg := config.Config{
		Logging: config.LoggingConfig{
			Output: path,
			Level:  "debug",
		},
	}

	cleanup := setupLogger(cfg)
	defer cleanup()

	slog.Info("test message")
	cleanup()

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))
}
