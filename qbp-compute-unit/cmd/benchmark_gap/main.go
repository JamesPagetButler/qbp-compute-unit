package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/gap"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
)

// Pos represents a 3D position.
type Pos struct {
	X, Y, Z float64
}

func main() {
	fmt.Println("========================================================================")
	fmt.Println("QBP BENCHMARK: GEOMETRIC ADJACENCY POINTERS (GAP)")
	fmt.Println("Eliminating the 'Relational Tax' in Headless Manifolds.")
	fmt.Println("========================================================================")

	numNodes := 2000 // Small enough for brute-force baseline to complete
	k := 8           // Neighbors per node

	fmt.Printf("Generating %d nodes with k=%d adjacency...\n", numNodes, k)

	// Mock node positions (float64 for search, but GAPs will use int8)
	positions := make([]Pos, numNodes)
	for i := range positions {
		positions[i] = Pos{rand.Float64() * 100, rand.Float64() * 100, rand.Float64() * 100}
	}

	nodes := make([]gap.GAPNode, numNodes)
	for i := range nodes {
		nodes[i] = gap.GAPNode{
			ID:    i,
			State: quat.Identity(), // Mock physical state
		}
	}

	// ─── 1. Baseline: Search-then-Compute ─────────────────────────────────
	// This simulates the "Relational Tax" where we don't have a mesh.
	fmt.Println("\nRunning Baseline (Search-then-Compute)...")
	startSearch := time.Now()
	for i := 0; i < numNodes; i++ {
		// MOCK KNN SEARCH: Find k closest nodes (Brute force O(N))
		_ = findKNN(i, positions, k)

		// Then compute (mock computation)
		_ = mockCompute(i, nodes)
	}
	elapsedBaseline := time.Since(startSearch)
	fmt.Printf("Baseline completed in: %v\n", elapsedBaseline)

	// ─── 2. GAP: Traversal-based Compute ──────────────────────────────────
	// Pre-bake the GAPs (one-time cost, or "Seam" update)
	fmt.Println("Pre-baking Geometric Adjacency Pointers...")
	for i := 0; i < numNodes; i++ {
		neighbors := findKNN(i, positions, k)
		for _, neighborIdx := range neighbors {
			// Calculate displacement
			dx := positions[neighborIdx].X - positions[i].X
			dy := positions[neighborIdx].Y - positions[i].Y
			dz := positions[neighborIdx].Z - positions[i].Z

			nodes[i].Adjacency = append(nodes[i].Adjacency, gap.AdjacencyPointer{
				TargetID: neighborIdx,
				RelPos:   quat.ToQuat8(quat.Pure(dx, dy, dz)),
				Weight:   quat.ToQuat8(quat.Scalar(rand.Float64())),
			})
		}
	}

	fmt.Println("Running GAP (Direct Traversal)...")
	startGAP := time.Now()
	for i := 0; i < numNodes; i++ {
		_ = nodes[i].CalculateGradient(nodes)
	}
	elapsedGAP := time.Since(startGAP)
	fmt.Printf("GAP completed in:      %v\n", elapsedGAP)

	fmt.Printf("\nSpeedup: %.1f×\n", float64(elapsedBaseline)/float64(elapsedGAP))
	fmt.Println("========================================================================")
}

func findKNN(targetIdx int, positions []Pos, k int) []int {
	// Simple brute-force KNN for demonstration
	type distPair struct {
		idx  int
		dist float64
	}
	dists := make([]distPair, len(positions))
	p1 := positions[targetIdx]
	for i, p2 := range positions {
		d := (p1.X-p2.X)*(p1.X-p2.X) + (p1.Y-p2.Y)*(p1.Y-p2.Y) + (p1.Z-p2.Z)*(p1.Z-p2.Z)
		dists[i] = distPair{i, d}
	}
	// (Sorting would go here, but let's just return first k for mock)
	res := make([]int, k)
	for i := 0; i < k; i++ {
		res[i] = dists[i].idx
	}
	return res
}

func mockCompute(i int, nodes []gap.GAPNode) quat.Quat {
	// Simulate the cost of computation without GAPs
	return quat.Scalar(float64(i))
}

// Add RandomUnit to quat package if it doesn't exist?
// Actually I'll just use Identity for mock.
