# Load Testing Report

[🇷🇺 Русский](load-test-report.ru.md) | [🇬🇧 English](load-test-report.md)

- **Service version:** current master branch
- **Tool:** Vegeta v12
- **Goal:** verify stability of automatic reviewer assignment at RPS ≈ 5 (requirement)

## Table of Contents

- [Scenario](#scenario)
- [Environment](#environment)
- [Vegeta Installation](#vegeta-installation)
- [Results](#results)
- [Conclusions](#conclusions)
- [Detailed Analysis](#detailed-analysis)
  - [Performance](#performance)
  - [Reliability](#reliability)
  - [Scalability](#scalability)
- [Repeating the Test](#repeating-the-test)

## Scenario

1. Automatically creates `load-team` with three active members (lu1, lu2, lu3).
2. Vegeta sends requests to create PR (`POST /pullRequest/create`) at a given frequency.
3. Each request uses a unique `pull_request_id` (generated based on `time.Now().UnixNano()`).
4. Duration — 60 seconds, rate — 5 req/s. Total ≈ 300 requests.

## Environment

- Huawei matebook D15, 16 GB RAM.
- Service and PostgreSQL started via `docker compose up`.
- Vegeta run locally via `make load-test` (see `load/cli/main.go`).

## Vegeta Installation

```bash
# Installation via go install
go install github.com/tsenart/vegeta/v12@latest

# Or use via go run (built into project)
go run ./load/cli
```

## Results

| Metric                | Value |
|------------------------|----------|
| Total requests         | 300      |
| Average RPS            | 5.02     |
| Throughput             | 5.02 req/s |
| Duration (total)   | 59.807 s |
| Duration (attack)  | 59.799 s |
| Wait time  | 7.563 ms |
| http_req_failed        | 0.0%     |
| http_req_duration min  | 6.026 ms |
| http_req_duration mean | 8.514 ms |
| http_req_duration p(50)| 8.089 ms |
| http_req_duration p(90)| 9.965 ms |
| http_req_duration p(95)| 10.825 ms |
| http_req_duration p(99)| 13.444 ms |
| Max response time       | 64.201 ms |
| Success rate           | 100.00%  |
| Status Codes           | 201:300  |
| Bytes In (total/mean)  | 70,124 / 233.75 |
| Bytes Out (total/mean) | 39,860 / 132.87 |

## Conclusions

- ✅ Service meets the target SLA of 300 ms at the 95th percentile with margin.
- ✅ No application or database level errors recorded.
- ✅ All requests successfully processed (100% success rate, 0% errors).
- ✅ Average latency of 8.514 ms shows excellent performance.
- ✅ Maximum latency of 64.201 ms is within SLA with large margin.

## Detailed Analysis

### Performance

- **Minimum latency**: 6.026 ms
- **Average latency**: 8.514 ms
- **Median latency (p50)**: 8.089 ms
- **90th percentile**: 9.965 ms
- **95th percentile**: 10.825 ms (~28 times better than SLA 300 ms)
- **99th percentile**: 13.444 ms (~22 times better than SLA 300 ms)
- **Maximum latency**: 64.201 ms (~4.7 times better than SLA)

**Latency over time:**

![Latency Graph](artifacts/vegeta-plot.png)

The graph shows excellent stability after the initial startup spike. Latency quickly stabilizes below 10ms and remains consistently low throughout the test, with occasional minor spikes up to ~15ms, well within the SLA requirements.

### Reliability

- **Request success rate**: 100%
- **Errors**: 0
- **Conflicts**: 0 (thanks to unique PR IDs)

### Scalability

Current results show that the service can handle significantly more load:
- All requests processed in 59.8 seconds at 5 req/s load
- Average response wait time: 7.563 ms
- Throughput matches the specified load (5.02 req/s)
- Stable performance throughout the test

## Repeating the Test

To repeat the test, run:

```bash
# Run full cycle
make load-test

# View report
make load-test-report

# Generate graph (requires vegeta CLI installation)
go install github.com/tsenart/vegeta/v12@latest
vegeta plot load/artifacts/results.bin > load/artifacts/plot.html
```

For more details on load testing, see `load/README.md`.
