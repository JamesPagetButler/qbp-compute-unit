package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
)

// Vector represents a high-dimensional embedding as a slice of Quaternions.
type Vector []quat.Quat

// Vector8 represents a quantized embedding as a slice of Quat8.
type Vector8 []quat.Quat8

func NewVector(dim int) Vector {
	v := make(Vector, dim/4)
	for i := range v {
		v[i] = quat.New(rand.NormFloat64(), rand.NormFloat64(), rand.NormFloat64(), rand.NormFloat64())
	}
	return v
}

func (v Vector) Quantize() Vector8 {
	v8 := make(Vector8, len(v))
	for i, q := range v {
		// Map [-1.0, 1.0] to [-127, 127]
		v8[i] = quat.NewQuat8(
			clampInt8(q.W*127),
			clampInt8(q.X*127),
			clampInt8(q.Y*127),
			clampInt8(q.Z*127),
		)
	}
	return v8
}

func clampInt8(f float64) int8 {
	if f > 127 {
		return 127
	}
	if f < -127 {
		return -127
	}
	return int8(f)
}

func CosineSimilarity(a, b Vector) float64 {
	dot := 0.0
	normA := 0.0
	normB := 0.0
	for i := range a {
		dot += a[i].W*b[i].W + a[i].X*b[i].X + a[i].Y*b[i].Y + a[i].Z*b[i].Z
		normA += quat.NormSq(a[i])
		normB += quat.NormSq(b[i])
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func CosineSimilarity8(a, b Vector8) float64 {
	dot := 0.0
	normA := 0.0
	normB := 0.0
	for i := range a {
		// Use float64 for accumulation to simulate "accumulator" register
		aq := a[i].ToQuat()
		bq := b[i].ToQuat()
		dot += aq.W*bq.W + aq.X*bq.X + aq.Y*bq.Y + aq.Z*bq.Z
		normA += quat.NormSq(aq)
		normB += quat.NormSq(bq)
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

type Result struct {
	Index int
	Score float64
}

func RunExperiment(dim int, count int) {
	fmt.Printf("\n--- Dimension: %d | Vectors: %d ---\n", dim, count)

	vectors := make([]Vector, count)
	vectors8 := make([]Vector8, count)
	for i := 0; i < count; i++ {
		vectors[i] = NewVector(dim)
		vectors8[i] = vectors[i].Quantize()
	}

	queryIdx := 0
	query := vectors[queryIdx]
	query8 := vectors8[queryIdx]

	resultsFP := make([]Result, count)
	resultsQ8 := make([]Result, count)

	for i := 0; i < count; i++ {
		resultsFP[i] = Result{i, CosineSimilarity(query, vectors[i])}
		resultsQ8[i] = Result{i, CosineSimilarity8(query8, vectors8[i])}
	}

	// Sort by score descending
	sort.Slice(resultsFP, func(i, j int) bool { return resultsFP[i].Score > resultsFP[j].Score })
	sort.Slice(resultsQ8, func(i, j int) bool { return resultsQ8[i].Score > resultsQ8[j].Score })

	// Check Top-10 overlap
	topK := 10
	matches := 0
	overlapSet := make(map[int]bool)
	for i := 1; i <= topK; i++ { // Skip index 0 (self-match)
		overlapSet[resultsFP[i].Index] = true
	}
	for i := 1; i <= topK; i++ {
		if overlapSet[resultsQ8[i].Index] {
			matches++
		}
	}

	// Calculate Correlation (Spearman-like on scores)
	sumSqDiff := 0.0
	for i := 0; i < count; i++ {
		// Find rank of i in both lists
		rFP, rQ8 := 0, 0
		for j := range resultsFP {
			if resultsFP[j].Index == i {
				rFP = j
				break
			}
		}
		for j := range resultsQ8 {
			if resultsQ8[j].Index == i {
				rQ8 = j
				break
			}
		}
		diff := float64(rFP - rQ8)
		sumSqDiff += diff * diff
	}
	n := float64(count)
	rho := 1 - (6*sumSqDiff)/(n*(n*n-1))

	fmt.Printf("Top-%d Overlap:      %d/%d (%.1f%%)\n", topK, matches, topK, float64(matches)/float64(topK)*100)
	fmt.Printf("Rank Correlation (ρ): %.4f\n", rho)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("========================================================================")
	fmt.Println("EXPERIMENT A: QW8 PRECISION BOUNDARY FOR FATHOM")
	fmt.Println("========================================================================")

	RunExperiment(128, 1000)
	RunExperiment(256, 1000)
	RunExperiment(512, 1000)

	fmt.Println("========================================================================")
}
