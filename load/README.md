# Load Testing with Vegeta

[🇷🇺 Русский](README.ru.md) | [🇬🇧 English](README.md)

This project uses the [Vegeta](https://github.com/tsenart/vegeta) library for load testing the PR reviewer assignment service API. Testing is implemented as a Go program in `load/cli/`.

## Table of Contents

- [Quick Start](#quick-start)
- [Structure](#structure)
- [Parameters](#parameters)
  - [Available Flags](#available-flags)
  - [Usage Examples](#usage-examples)
- [Makefile Commands](#makefile-commands)
- [Testing Scenario](#testing-scenario)
- [Results Analysis](#results-analysis)
  - [Text Report](#text-report)
  - [HTML Graph](#html-graph)
  - [Detailed Statistics in JSON](#detailed-statistics-in-json)
- [SLA Requirements](#sla-requirements)

## Quick Start

1. Start the service:
   ```bash
   docker compose up --build
   ```

2. Run load testing:
   ```bash
   make load-test
   ```

   Or directly:
   ```bash
   go run ./load/cli
   ```

## Structure

- `load/cli/` — Go application for load testing
  - `main.go` — main code (setup, request generation, Vegeta launch)
  - `main_test.go` — unit tests
- `load/scripts/` — helper shell scripts (deprecated, used for reference)
  - `setup.sh` — test environment preparation
  - `load_test.sh` — load test execution
  - `generate_targets.sh` — Vegeta targets generation
- `load/artifacts/` — test results and artifacts
  - `results.bin` — binary file with test results
  - `plot.html` — HTML graph of results (generated separately)
  - `vegeta-plot.png` — results visualization

## Parameters

Can be configured via command-line flags:

```bash
go run ./load/cli -url=http://localhost:8080 -rate=5 -duration=60s -team=load-team
```

### Available Flags

| Flag | Description | Default Value |
|------|----------|----------------------|
| `-url` | Base URL of the service | `http://localhost:8080` |
| `-rate` | Requests per second | `5` |
| `-duration` | Test duration | `60s` |
| `-team` | Test team name | `load-team` |
| `-setup-only` | Only prepare environment (create team) | `false` |
| `-report` | Show report from saved results | `false` |
| `-plot` | Show instructions for graph generation | `false` |

### Usage Examples

```bash
# Custom parameters
go run ./load/cli -rate=10 -duration=30s

# Only prepare environment
go run ./load/cli -setup-only

# Show report from saved results
go run ./load/cli -report

# Show instructions for graph generation
go run ./load/cli -plot
```

## Makefile Commands

| Command | Description |
|---------|----------|
| `make load-test` | Run full testing cycle (setup + load) |
| `make load-test-setup` | Only prepare environment (create team) |
| `make load-test-report` | Show report from saved results |
| `make load-test-plot` | Show instructions for graph generation |

## Testing Scenario

1. **Preparation**: automatically creates `load-team` with three active users:
   - `lu1` — Load Alice
   - `lu2` — Load Bob
   - `lu3` — Load Carol

2. **Load**: Vegeta sends POST requests to `/pullRequest/create` with unique IDs (generated based on `time.Now().UnixNano()` to avoid conflicts)

3. **Verification**: reviewers are automatically assigned from the author's team (up to 2)

## Results Analysis

After the test, results are saved in `load/artifacts/results.bin`. For analysis:

### Text Report

```bash
# Via Go program
go run ./load/cli -report

# Or via vegeta CLI utility (if installed)
vegeta report load/artifacts/results.bin
```

### HTML Graph

```bash
# Install vegeta CLI utility
go install github.com/tsenart/vegeta/v12@latest

# Generate graph
vegeta plot load/artifacts/results.bin > load/artifacts/plot.html

# Open in browser
open load/artifacts/plot.html  # macOS
xdg-open load/artifacts/plot.html  # Linux
```

### Detailed Statistics in JSON

```bash
vegeta report -type=json load/artifacts/results.bin | jq
```

## SLA Requirements

According to the requirements:
- **RPS**: 5 requests per second
- **Response time SLI**: 300 ms (95th percentile)
- **Success SLI**: 99.9%

Vegeta automatically checks these metrics and outputs a report. Detailed results see in [load testing report](load-test-report.md).
