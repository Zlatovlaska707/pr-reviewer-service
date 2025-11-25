package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

func TestSetupTeamSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/team/add", r.URL.Path)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	require.NoError(t, setupTeam(srv.URL, "test-team"))
}

func TestRunLoadTestCreatesResultsFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	tmpFile := filepath.Join(t.TempDir(), "results.bin")
	prev := resultsFile
	resultsFile = tmpFile
	defer func() { resultsFile = prev }()

	require.NoError(t, runLoadTest(srv.URL, 1, 20*time.Millisecond, "team"))

	info, err := os.Stat(tmpFile)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))
}

func TestRenderReportReadsFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "results.bin")
	file, err := os.Create(tmpFile)
	require.NoError(t, err)
	enc := vegeta.NewEncoder(file)
	now := time.Now()
	require.NoError(t, enc.Encode(&vegeta.Result{
		Code:      http.StatusOK,
		Timestamp: now,
		Latency:   time.Millisecond,
		BytesIn:   10,
		BytesOut:  5,
	}))
	require.NoError(t, enc.Encode(&vegeta.Result{
		Code:      http.StatusBadRequest,
		Timestamp: now.Add(time.Millisecond),
		Latency:   2 * time.Millisecond,
	}))
	require.NoError(t, file.Close())

	var buf bytes.Buffer
	require.NoError(t, renderReport(&buf, tmpFile))
	require.Contains(t, buf.String(), "Requests      [total")
}

func TestWritePlotInstructions(t *testing.T) {
	var buf bytes.Buffer
	prev := resultsFile
	resultsFile = "custom.bin"
	defer func() { resultsFile = prev }()

	writePlotInstructions(&buf)
	output := buf.String()
	require.Contains(t, output, "vegeta plot custom.bin")
	require.Contains(t, output, "go install github.com/tsenart/vegeta/v12@latest")
}
