package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestLoadReadsYamlAndEnvOverrides(t *testing.T) {
	path := writeTempConfig(t, `
http:
  port: "8081"
database:
  url: "postgres://localhost:5432/db"
timeouts:
  operation: 10s
`)
	t.Setenv("CONFIG_PATH", path)
	t.Setenv("HTTP_PORT", "9000")
	t.Setenv("SHUTDOWN_TIMEOUT", "20s")

	cfg, err := Load()
	require.NoError(t, err)

	require.Equal(t, "9000", cfg.HTTP.Port)
	require.Equal(t, 10*time.Second, cfg.Timeouts.Operation)
	require.Equal(t, 20*time.Second, cfg.Timeouts.Shutdown)
	require.Equal(t, "postgres://localhost:5432/db", cfg.Database.URL)
}

func TestLoadMissingFileReturnsError(t *testing.T) {
	t.Setenv("CONFIG_PATH", filepath.Join(t.TempDir(), "missing.yaml"))

	_, err := Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestMustLoadPanicsOnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")
	t.Setenv("CONFIG_PATH", path)

	require.PanicsWithError(t, "config file "+path+" not found", func() {
		MustLoad()
	})
}
