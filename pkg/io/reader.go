// Package io provides input/output utilities for data ingestion.
package io

import "context"

// Reader is the interface for reading data from various sources.
type Reader interface {
	// Read returns the complete dataset.
	Read() ([][]float64, error)

	// Stream returns a channel of samples for real-time processing.
	Stream(ctx context.Context) (<-chan []float64, error)

	// Close releases resources.
	Close() error
}

// FeatureExtractor extracts numerical features from raw data.
type FeatureExtractor interface {
	// Extract converts raw input to feature vector.
	Extract(data any) ([]float64, error)

	// FeatureNames returns the names of extracted features.
	FeatureNames() []string
}

// Writer is the interface for writing detection results.
type Writer interface {
	// Write outputs a single result.
	Write(result Result) error

	// WriteAll outputs multiple results.
	WriteAll(results []Result) error

	// Close releases resources.
	Close() error
}

// Result represents an anomaly detection result.
type Result struct {
	Timestamp int64          `json:"timestamp"`
	Score     float64        `json:"score"`
	IsAnomaly bool           `json:"is_anomaly"`
	Features  []float64      `json:"features,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}
