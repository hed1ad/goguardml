// Package iforest implements the Isolation Forest algorithm for anomaly detection.
package iforest

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"math"
	"math/rand"
	"sync"

	"github.com/hed1ad/goguardml/pkg/detectors"
)

// IsolationForest implements unsupervised anomaly detection using isolation trees.
type IsolationForest struct {
	mu sync.RWMutex

	// Configuration
	nTrees        int
	sampleSize    int
	contamination float64
	threshold     float64
	maxDepth      int
	rng           *rand.Rand

	// Trained model
	trees   []*iTree
	trained bool

	// Statistics from training
	avgPathLength float64
}

// iTree represents a single isolation tree.
type iTree struct {
	root *node
}

// node is a node in the isolation tree.
type node struct {
	// Split parameters (for internal nodes)
	splitFeature int
	splitValue   float64

	// Children
	left  *node
	right *node

	// Leaf information
	size int // number of samples that reached this leaf
}

// Option configures an IsolationForest.
type Option func(*IsolationForest)

// WithTrees sets the number of isolation trees.
func WithTrees(n int) Option {
	return func(f *IsolationForest) {
		f.nTrees = n
	}
}

// WithSampleSize sets the subsample size for each tree.
func WithSampleSize(n int) Option {
	return func(f *IsolationForest) {
		f.sampleSize = n
	}
}

// WithContamination sets the expected proportion of anomalies.
func WithContamination(c float64) Option {
	return func(f *IsolationForest) {
		f.contamination = c
	}
}

// WithSeed sets the random seed for reproducibility.
func WithSeed(seed int64) Option {
	return func(f *IsolationForest) {
		f.rng = rand.New(rand.NewSource(seed))
	}
}

// New creates a new IsolationForest with the given options.
func New(opts ...Option) *IsolationForest {
	f := &IsolationForest{
		nTrees:        100,
		sampleSize:    256,
		contamination: 0.1,
		threshold:     0.5,
		rng:           rand.New(rand.NewSource(42)),
	}

	for _, opt := range opts {
		opt(f)
	}

	// Max depth based on sample size
	f.maxDepth = int(math.Ceil(math.Log2(float64(f.sampleSize))))

	return f
}

// Fit trains the Isolation Forest on the provided data.
func (f *IsolationForest) Fit(data [][]float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(data) == 0 {
		return errors.New("empty training data")
	}

	nSamples := len(data)
	nFeatures := len(data[0])

	// Adjust sample size if needed
	sampleSize := f.sampleSize
	if sampleSize > nSamples {
		sampleSize = nSamples
	}

	// Build trees
	f.trees = make([]*iTree, f.nTrees)
	for i := 0; i < f.nTrees; i++ {
		// Sample without replacement
		indices := f.rng.Perm(nSamples)[:sampleSize]
		sample := make([][]float64, sampleSize)
		for j, idx := range indices {
			sample[j] = data[idx]
		}

		f.trees[i] = f.buildTree(sample, nFeatures, 0)
	}

	// Calculate average path length for normalization
	f.avgPathLength = averagePathLength(float64(sampleSize))
	f.trained = true

	// Set threshold based on contamination
	if f.contamination > 0 {
		scores, _ := f.predict(data)
		f.threshold = percentile(scores, 100*(1-f.contamination))
	}

	return nil
}

// buildTree recursively builds an isolation tree.
func (f *IsolationForest) buildTree(data [][]float64, nFeatures, depth int) *iTree {
	return &iTree{
		root: f.buildNode(data, nFeatures, depth),
	}
}

func (f *IsolationForest) buildNode(data [][]float64, nFeatures, depth int) *node {
	n := len(data)

	// Terminal conditions
	if depth >= f.maxDepth || n <= 1 {
		return &node{size: n}
	}

	// Random feature and split value
	feature := f.rng.Intn(nFeatures)

	// Find min/max for this feature
	minVal, maxVal := data[0][feature], data[0][feature]
	for _, row := range data[1:] {
		if row[feature] < minVal {
			minVal = row[feature]
		}
		if row[feature] > maxVal {
			maxVal = row[feature]
		}
	}

	// If all values are the same, return leaf
	if minVal == maxVal {
		return &node{size: n}
	}

	// Random split value
	splitValue := minVal + f.rng.Float64()*(maxVal-minVal)

	// Partition data
	var leftData, rightData [][]float64
	for _, row := range data {
		if row[feature] < splitValue {
			leftData = append(leftData, row)
		} else {
			rightData = append(rightData, row)
		}
	}

	return &node{
		splitFeature: feature,
		splitValue:   splitValue,
		left:         f.buildNode(leftData, nFeatures, depth+1),
		right:        f.buildNode(rightData, nFeatures, depth+1),
	}
}

// Predict returns anomaly scores for the given samples.
func (f *IsolationForest) Predict(data [][]float64) ([]float64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.trained {
		return nil, errors.New("model not trained")
	}

	return f.predict(data)
}

func (f *IsolationForest) predict(data [][]float64) ([]float64, error) {
	scores := make([]float64, len(data))

	for i, sample := range data {
		score, err := f.predictOne(sample)
		if err != nil {
			return nil, err
		}
		scores[i] = score
	}

	return scores, nil
}

// PredictOne returns the anomaly score for a single sample.
func (f *IsolationForest) PredictOne(sample []float64) (float64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.trained {
		return 0, errors.New("model not trained")
	}

	return f.predictOne(sample)
}

func (f *IsolationForest) predictOne(sample []float64) (float64, error) {
	// Average path length across all trees
	var totalPath float64
	for _, tree := range f.trees {
		totalPath += pathLength(sample, tree.root, 0)
	}
	avgPath := totalPath / float64(len(f.trees))

	// Anomaly score: 2^(-avgPath / c(n))
	// Higher score = more anomalous
	score := math.Pow(2, -avgPath/f.avgPathLength)

	return score, nil
}

// pathLength calculates the path length for a sample in a tree.
func pathLength(sample []float64, n *node, currentDepth int) float64 {
	if n.left == nil && n.right == nil {
		// Leaf node: add expected path length for remaining isolation
		return float64(currentDepth) + averagePathLength(float64(n.size))
	}

	if sample[n.splitFeature] < n.splitValue {
		return pathLength(sample, n.left, currentDepth+1)
	}
	return pathLength(sample, n.right, currentDepth+1)
}

// averagePathLength returns the average path length of unsuccessful search in BST.
func averagePathLength(n float64) float64 {
	if n <= 1 {
		return 0
	}
	// c(n) = 2*H(n-1) - 2*(n-1)/n, where H is harmonic number
	// Approximation: H(n) ≈ ln(n) + 0.5772156649 (Euler-Mascheroni constant)
	return 2*(math.Log(n-1)+0.5772156649) - 2*(n-1)/n
}

// PredictStream processes samples from a channel.
func (f *IsolationForest) PredictStream(ctx context.Context, input <-chan []float64, output chan<- detectors.Score) error {
	f.mu.RLock()
	if !f.trained {
		f.mu.RUnlock()
		return errors.New("model not trained")
	}
	f.mu.RUnlock()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sample, ok := <-input:
			if !ok {
				return nil
			}

			score, err := f.PredictOne(sample)
			if err != nil {
				continue
			}

			select {
			case output <- detectors.Score{
				Value:     score,
				IsAnomaly: score >= f.threshold,
				Features:  sample,
			}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// Save serializes the trained model.
func (f *IsolationForest) Save() ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.trained {
		return nil, errors.New("model not trained")
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(f.nTrees); err != nil {
		return nil, err
	}
	if err := enc.Encode(f.sampleSize); err != nil {
		return nil, err
	}
	if err := enc.Encode(f.contamination); err != nil {
		return nil, err
	}
	if err := enc.Encode(f.threshold); err != nil {
		return nil, err
	}
	if err := enc.Encode(f.avgPathLength); err != nil {
		return nil, err
	}
	if err := enc.Encode(f.trees); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Load deserializes a trained model.
func (f *IsolationForest) Load(data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)

	if err := dec.Decode(&f.nTrees); err != nil {
		return err
	}
	if err := dec.Decode(&f.sampleSize); err != nil {
		return err
	}
	if err := dec.Decode(&f.contamination); err != nil {
		return err
	}
	if err := dec.Decode(&f.threshold); err != nil {
		return err
	}
	if err := dec.Decode(&f.avgPathLength); err != nil {
		return err
	}
	if err := dec.Decode(&f.trees); err != nil {
		return err
	}

	f.maxDepth = int(math.Ceil(math.Log2(float64(f.sampleSize))))
	f.trained = true

	return nil
}

// Threshold returns the current anomaly threshold.
func (f *IsolationForest) Threshold() float64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.threshold
}

// SetThreshold updates the anomaly threshold.
func (f *IsolationForest) SetThreshold(t float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.threshold = t
}

// percentile calculates the p-th percentile of the data.
func percentile(data []float64, p float64) float64 {
	if len(data) == 0 {
		return 0
	}

	sorted := make([]float64, len(data))
	copy(sorted, data)

	// Simple insertion sort for small arrays
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	idx := int(float64(len(sorted)-1) * p / 100)
	return sorted[idx]
}
