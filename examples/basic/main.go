// Package main demonstrates basic usage of GoAnomalyDetect.
package main

import (
	"fmt"
	"math/rand"

	"github.com/hed1ad/goguardml/pkg/detectors/iforest"
)

func main() {
	// Generate synthetic training data
	// Normal traffic: small packets, regular intervals
	trainingData := generateNormalTraffic(1000)

	// Create Isolation Forest detector
	detector := iforest.New(
		iforest.WithTrees(100),
		iforest.WithSampleSize(256),
		iforest.WithContamination(0.1),
		iforest.WithSeed(42),
	)

	// Train the model
	fmt.Println("Training Isolation Forest...")
	if err := detector.Fit(trainingData); err != nil {
		panic(err)
	}
	fmt.Println("Training complete!")

	// Generate test data with some anomalies
	testData := generateMixedTraffic(100)

	// Predict anomaly scores
	scores, err := detector.Predict(testData)
	if err != nil {
		panic(err)
	}

	// Report results
	fmt.Println("\nAnomaly Detection Results:")
	fmt.Println("==========================")

	anomalyCount := 0
	threshold := detector.Threshold()

	for i, score := range scores {
		if score >= threshold {
			anomalyCount++
			fmt.Printf("Sample %3d: score=%.3f [ANOMALY] features=%v\n",
				i, score, testData[i])
		}
	}

	fmt.Printf("\nTotal anomalies detected: %d/%d (threshold: %.2f)\n",
		anomalyCount, len(testData), threshold)
}

// generateNormalTraffic creates synthetic normal network traffic.
// Features: [packet_size, interval, protocol, src_port, dst_port]
func generateNormalTraffic(n int) [][]float64 {
	data := make([][]float64, n)
	for i := 0; i < n; i++ {
		data[i] = []float64{
			64 + rand.Float64()*200,   // packet size: 64-264 bytes
			0.001 + rand.Float64()*0.1, // interval: 1-100ms
			6,                          // TCP
			float64(1024 + rand.Intn(64000)), // src port
			443,                        // dst port (HTTPS)
		}
	}
	return data
}

// generateMixedTraffic creates test data with some anomalies.
func generateMixedTraffic(n int) [][]float64 {
	data := make([][]float64, n)
	for i := 0; i < n; i++ {
		if rand.Float64() < 0.1 {
			// Anomaly: unusual pattern
			data[i] = []float64{
				1400 + rand.Float64()*100, // large packets
				0.0001,                     // very short interval (burst)
				17,                         // UDP
				float64(rand.Intn(1024)),  // privileged port
				float64(rand.Intn(1024)),  // privileged port
			}
		} else {
			// Normal traffic
			data[i] = []float64{
				64 + rand.Float64()*200,
				0.001 + rand.Float64()*0.1,
				6,
				float64(1024 + rand.Intn(64000)),
				443,
			}
		}
	}
	return data
}
