# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GoAnomalyDetect is a Go library for unsupervised anomaly detection in network traffic and logs, optimized for real-time cybersecurity in Kubernetes/Docker environments. It implements the Isolation Forest algorithm with streaming support.

## Common Commands

```bash
make build          # Build binary to bin/goanomaly
make test           # Run tests with race detection
make lint           # Run golangci-lint
make fmt            # Format code with gofmt
make bench          # Run benchmarks on detectors
make coverage       # Generate HTML coverage report
make security       # Run gosec security scan
make all            # Run lint, test, build
```

Run a single test:
```bash
go test -v -race ./pkg/detectors/iforest -run TestIsolationForest_Fit
```

## Architecture

**Core packages:**
- `pkg/detectors/` - Anomaly detection algorithms implementing the `Detector` interface
- `pkg/detectors/iforest/` - Isolation Forest implementation with streaming support
- `pkg/io/pcap/` - PCAP file and live network capture with feature extraction
- `pkg/io/csv/` - CSV data reader
- `cmd/goanomaly/` - CLI tool (Cobra-based)

**Key interfaces in `pkg/detectors/detector.go`:**
- `Detector` - Core interface: `Fit()`, `Predict()`, `PredictOne()`, `Save()`, `Load()`
- `StreamDetector` - Adds `PredictStream(ctx, input chan, output chan)` for real-time processing

**Design patterns:**
- Options pattern for configuration (e.g., `iforest.WithTrees(100)`, `iforest.WithContamination(0.1)`)
- Thread-safe with `sync.RWMutex` (Fit uses write lock, Predict uses read lock)
- Anomaly scores normalized to [0, 1] (higher = more anomalous)
- Model serialization via Go's gob encoding

## Code Style

- Standard Go conventions with golangci-lint (strict config in `.golangci.yml`)
- Table-driven tests with >90% coverage expected
- Conventional commits: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `perf:`

## Dependencies

Requires `libpcap-dev` for network packet capture functionality.
