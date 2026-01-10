# GoAnomalyDetect

[![CI](https://github.com/hed1ad/goguardml/actions/workflows/ci.yml/badge.svg)](https://github.com/hed1ad/goguardml/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hed1ad/goguardml)](https://goreportcard.com/report/github.com/hed1ad/goguardml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Open-source Golang library for **unsupervised anomaly detection** in network traffic and logs. Optimized for real-time cybersecurity in Kubernetes/Docker environments.

## Features

- **Isolation Forest** - Fast, scalable anomaly detection
- **PCAP Support** - Direct network packet analysis
- **Streaming API** - Real-time detection with Go channels
- **Lightweight** - Minimal dependencies, pure Go (except libpcap)
- **Production Ready** - Docker, K8s native

## Installation

```bash
go get github.com/hed1ad/goguardml
```

### Requirements

- Go 1.23+
- libpcap-dev (for PCAP support)

```bash
# Debian/Ubuntu
sudo apt-get install libpcap-dev

# Arch Linux
sudo pacman -S libpcap

# macOS
brew install libpcap
```

## Quick Start

### As a Library

```go
package main

import (
    "fmt"
    "github.com/hed1ad/goguardml/pkg/detectors/iforest"
)

func main() {
    // Create detector
    detector := iforest.New(
        iforest.WithTrees(100),
        iforest.WithContamination(0.1),
    )

    // Training data (features: packet_size, interval, ...)
    data := [][]float64{
        {64, 0.001, 6, 443, 80},
        {128, 0.002, 6, 443, 80},
        // ... more samples
    }

    // Train
    detector.Fit(data)

    // Predict
    scores, _ := detector.Predict(data)
    for i, score := range scores {
        if score > 0.5 {
            fmt.Printf("Anomaly detected at sample %d (score: %.2f)\n", i, score)
        }
    }
}
```

### Streaming Detection

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

input := make(chan []float64)
output := make(chan detectors.Score)

go detector.PredictStream(ctx, input, output)

// Send samples
go func() {
    for _, sample := range newData {
        input <- sample
    }
    close(input)
}()

// Receive scores
for score := range output {
    if score.IsAnomaly {
        fmt.Printf("ALERT: Anomaly score %.2f\n", score.Value)
    }
}
```

### CLI Usage

```bash
# Build
make build

# Train on PCAP file
./bin/goanomaly fit --file traffic.pcap --model iforest --out model.gob

# Predict anomalies
./bin/goanomaly predict --file new_traffic.pcap --model model.gob --out scores.json

# Real-time detection
./bin/goanomaly stream --interface eth0 --model model.gob --threshold 0.7
```

### Docker

```bash
# Build image
docker build -t goanomalydetect:latest .

# Run
docker run --rm -v $(pwd)/data:/data goanomalydetect:latest \
    fit --file /data/traffic.pcap --out /data/model.gob
```

## Architecture

```
cmd/goanomaly/       # CLI application
pkg/
  detectors/         # Anomaly detection algorithms
    iforest/         # Isolation Forest implementation
    lstm/            # LSTM autoencoder (planned)
  io/                # Data ingestion
    pcap/            # PCAP reader
    csv/             # CSV reader
    prometheus/      # Prometheus metrics (planned)
  core/              # Matrix operations
  utils/             # Utilities
internal/            # Internal packages
examples/            # Usage examples
testdata/            # Test datasets
```

## Benchmarks

Target performance: <1ms/packet on 8-core CPU, 10k packets/sec streaming.

```bash
make bench
```

## Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Run linters
make lint

# Run security scan
make security

# Build
make build
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.
