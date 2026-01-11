package iforest

import (
	"context"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hed1ad/goguardml/pkg/detectors"
)

func TestNewIsolationForest(t *testing.T) {
	tests := []struct {
		name       string
		opts       []Option
		wantNTrees int
	}{
		{
			name:       "default configuration",
			opts:       nil,
			wantNTrees: 100,
		},
		{
			name:       "custom trees",
			opts:       []Option{WithTrees(50)},
			wantNTrees: 50,
		},
		{
			name:       "multiple options",
			opts:       []Option{WithTrees(200), WithContamination(0.05), WithSeed(123)},
			wantNTrees: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New(tt.opts...)
			assert.Equal(t, tt.wantNTrees, f.nTrees)
		})
	}
}

func TestFit(t *testing.T) {
	tests := []struct {
		name    string
		data    [][]float64
		wantErr bool
	}{
		{
			name:    "empty data",
			data:    [][]float64{},
			wantErr: true,
		},
		{
			name:    "single sample",
			data:    [][]float64{{1.0, 2.0, 3.0}},
			wantErr: false,
		},
		{
			name:    "normal data",
			data:    generateTestData(100, 5),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New(WithTrees(10), WithSeed(42))
			err := f.Fit(tt.data)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, f.trained)
				assert.Len(t, f.trees, f.nTrees)
			}
		})
	}
}

func TestPredict(t *testing.T) {
	// Train on normal data
	trainData := generateTestData(500, 5)
	f := New(WithTrees(50), WithSampleSize(100), WithSeed(42))
	require.NoError(t, f.Fit(trainData))

	t.Run("predict on normal data", func(t *testing.T) {
		testData := generateTestData(100, 5)
		scores, err := f.Predict(testData)

		require.NoError(t, err)
		assert.Len(t, scores, len(testData))

		// All scores should be in [0, 1]
		for _, score := range scores {
			assert.GreaterOrEqual(t, score, 0.0)
			assert.LessOrEqual(t, score, 1.0)
		}
	})

	t.Run("predict on anomalies", func(t *testing.T) {
		// Anomalous data: very different from training
		anomalies := [][]float64{
			{1000, 1000, 1000, 1000, 1000},
			{-500, -500, -500, -500, -500},
		}
		scores, err := f.Predict(anomalies)

		require.NoError(t, err)
		// Anomalies should have higher scores
		for _, score := range scores {
			assert.Greater(t, score, 0.4, "anomalies should have high scores")
		}
	})

	t.Run("predict before fit", func(t *testing.T) {
		untrained := New()
		_, err := untrained.Predict(trainData)
		assert.Error(t, err)
	})
}

func TestPredictOne(t *testing.T) {
	trainData := generateTestData(200, 3)
	f := New(WithTrees(20), WithSeed(42))
	require.NoError(t, f.Fit(trainData))

	score, err := f.PredictOne([]float64{0.5, 0.5, 0.5})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)
}

func TestPredictStream(t *testing.T) {
	trainData := generateTestData(200, 3)
	f := New(WithTrees(20), WithSeed(42))
	require.NoError(t, f.Fit(trainData))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	input := make(chan []float64, 10)
	output := make(chan detectors.Score, 10)

	go func() {
		err := f.PredictStream(ctx, input, output)
		assert.NoError(t, err)
	}()

	// Send test samples
	testSamples := [][]float64{
		{0.5, 0.5, 0.5},
		{100, 100, 100}, // anomaly
		{0.3, 0.3, 0.3},
	}

	go func() {
		for _, sample := range testSamples {
			input <- sample
		}
		close(input)
	}()

	// Receive results
	results := make([]detectors.Score, 0, len(testSamples))
	for score := range output {
		results = append(results, score)
	}

	assert.Len(t, results, len(testSamples))
}

func TestSaveLoad(t *testing.T) {
	trainData := generateTestData(200, 4)
	original := New(WithTrees(30), WithContamination(0.15), WithSeed(42))
	require.NoError(t, original.Fit(trainData))

	// Get predictions before save
	testData := generateTestData(50, 4)
	originalScores, err := original.Predict(testData)
	require.NoError(t, err)

	// Save
	data, err := original.Save()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Load into new instance
	loaded := New()
	err = loaded.Load(data)
	require.NoError(t, err)

	// Predictions should match
	loadedScores, err := loaded.Predict(testData)
	require.NoError(t, err)

	assert.Equal(t, originalScores, loadedScores)
}

func TestThreshold(t *testing.T) {
	f := New()
	f.trained = true

	// Test getter
	assert.Equal(t, 0.5, f.Threshold())

	// Test setter
	f.SetThreshold(0.7)
	assert.Equal(t, 0.7, f.Threshold())
}

func BenchmarkFit(b *testing.B) {
	data := generateTestData(10000, 10)
	f := New(WithTrees(100), WithSampleSize(256))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Fit(data)
	}
}

func BenchmarkPredict(b *testing.B) {
	trainData := generateTestData(5000, 10)
	testData := generateTestData(1000, 10)

	f := New(WithTrees(100), WithSampleSize(256))
	f.Fit(trainData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Predict(testData)
	}
}

func BenchmarkPredictOne(b *testing.B) {
	trainData := generateTestData(5000, 10)
	sample := make([]float64, 10)
	for i := range sample {
		sample[i] = rand.Float64()
	}

	f := New(WithTrees(100), WithSampleSize(256))
	f.Fit(trainData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.PredictOne(sample)
	}
}

func generateTestData(n, features int) [][]float64 {
	data := make([][]float64, n)
	for i := 0; i < n; i++ {
		data[i] = make([]float64, features)
		for j := 0; j < features; j++ {
			data[i][j] = rand.NormFloat64()
		}
	}
	return data
}
