// Package detectors provides unsupervised anomaly detection algorithms.
package detectors

import "context"

// Detector is the common interface for all anomaly detection algorithms.
type Detector interface {
	// Fit trains the detector on historical data.
	// data is a 2D slice where each row is a sample and each column is a feature.
	Fit(data [][]float64) error

	// Predict returns anomaly scores for the given samples.
	// Scores are normalized to [0, 1] where higher values indicate anomalies.
	Predict(data [][]float64) ([]float64, error)

	// PredictOne returns the anomaly score for a single sample.
	PredictOne(sample []float64) (float64, error)

	// Save serializes the trained model to bytes.
	Save() ([]byte, error)

	// Load deserializes a trained model from bytes.
	Load(data []byte) error
}

// StreamDetector extends Detector with streaming capabilities.
type StreamDetector interface {
	Detector

	// PredictStream processes samples from a channel and outputs scores.
	PredictStream(ctx context.Context, input <-chan []float64, output chan<- Score) error
}

// Score represents an anomaly detection result.
type Score struct {
	// Value is the anomaly score in [0, 1].
	Value float64
	// IsAnomaly indicates if the score exceeds the threshold.
	IsAnomaly bool
	// Features contains the original input features.
	Features []float64
	// Metadata contains additional information.
	Metadata map[string]any
}

// Config holds common configuration for detectors.
type Config struct {
	// Contamination is the expected proportion of anomalies in training data.
	Contamination float64
	// Threshold is the score threshold for classifying anomalies.
	Threshold float64
	// RandomSeed for reproducibility.
	RandomSeed int64
}

// DefaultConfig returns sensible defaults for detector configuration.
func DefaultConfig() Config {
	return Config{
		Contamination: 0.1,
		Threshold:     0.5,
		RandomSeed:    42,
	}
}
