package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

type Precision int
const (
	QW8 Precision = iota
	QW16
	QW32
)

func Quantize(f float64, p Precision) float64 {
	switch p {
	case QW8:
		// int8
		return float64(int8(f*127)) / 127.0
	case QW16:
		// int16
		return float64(int16(f*32767)) / 32767.0
	case QW32:
		// float32
		return float64(float32(f))
	}
	return f
}

func main() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("========================================================================")
	fmt.Println("EXPERIMENT F: PRECISION WIDTH FIELD FOR BRIDGE TASKPROFILE")
	fmt.Println("========================================================================")

	precisions := []Precision{QW8, QW16, QW32}
	pNames := map[Precision]string{QW8: "QW8 (int8)", QW16: "QW16 (int16)", QW32: "QW32 (float32)"}

	for _, p := range precisions {
		fmt.Printf("\n--- Testing %s ---\n", pNames[p])

		// 1. Cosine Similarity (100 pairs)
		simPass := 0
		for i := 0; i < 100; i++ {
			v1 := []float64{rand.NormFloat64(), rand.NormFloat64()}
			v2 := []float64{rand.NormFloat64(), rand.NormFloat64()}
			
			// Reference float64
			refDot := v1[0]*v2[0] + v1[1]*v2[1]
			refNorm := math.Sqrt(v1[0]*v1[0]+v1[1]*v1[1]) * math.Sqrt(v2[0]*v2[0]+v2[1]*v2[1])
			refSim := refDot / refNorm

			// Quantized
			qv1 := []float64{Quantize(v1[0], p), Quantize(v1[1], p)}
			qv2 := []float64{Quantize(v2[0], p), Quantize(v2[1], p)}
			qDot := qv1[0]*qv2[0] + qv1[1]*qv2[1]
			qNorm := math.Sqrt(qv1[0]*qv1[0]+qv1[1]*qv1[1]) * math.Sqrt(qv2[0]*qv2[0]+qv2[1]*qv2[1])
			qSim := qDot / qNorm

			if math.Abs(refSim-qSim) < 0.05 {
				simPass++
			}
		}
		fmt.Printf("  Cosine Similarity (>0.95 accuracy): %d/100\n", simPass)

		// 2. Salience Threshold (0.5)
		threshPass := 0
		threshold := 0.5
		for i := 0; i < 1000; i++ {
			val := rand.Float64()
			refDecision := val >= threshold
			qVal := Quantize(val, p)
			qDecision := qVal >= threshold

			if refDecision == qDecision {
				threshPass++
			}
		}
		fmt.Printf("  Salience Thresholding:              %d/1000\n", threshPass)

		// 3. Contradiction Detection (check if sign is preserved)
		signPass := 0
		for i := 0; i < 1000; i++ {
			val := (rand.Float64() - 0.5) * 2.0 // [-1, 1]
			if math.Abs(val) < 0.01 { continue } // Skip near zero
			refSign := val > 0
			qVal := Quantize(val, p)
			qSign := qVal > 0
			if refSign == qSign {
				signPass++
			}
		}
		fmt.Printf("  Sign/Contradiction Detection:       %d/1000 (filtered zero-ish)\n", signPass)
	}

	fmt.Println("\n========================================================================")
}
