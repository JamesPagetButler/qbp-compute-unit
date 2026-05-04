package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
)

// Node represents a node in the BMA hypergraph.
type Node struct {
	ID    int
	Value quat.Quat
}

// Edge represents a directed hyperedge with a Quat8 weight.
type Edge struct {
	From   int
	To     int
	Weight quat.Quat8
}

func main() {
	fmt.Println("========================================================================")
	fmt.Println("QBP EVALUATION: TERNARY & BMA INTEGRATION (PHASE 4)")
	fmt.Println("Testing Large-Scale Hypergraph Traversal using Quat8 (int8).")
	fmt.Println("========================================================================")

	numNodes := 100000
	numEdges := 500000

	fmt.Printf("Generating Graph: %d nodes, %d edges...\n", numNodes, numEdges)

	nodes := make([]Node, numNodes)
	for i := range nodes {
		nodes[i] = Node{ID: i, Value: quat.Identity()}
	}

	edges := make([]Edge, numEdges)
	for i := range edges {
		edges[i] = Edge{
			From: rand.Intn(numNodes),
			To:   rand.Intn(numNodes),
			Weight: quat.NewQuat8(
				int8(rand.Intn(255)-127),
				int8(rand.Intn(255)-127),
				int8(rand.Intn(255)-127),
				int8(rand.Intn(255)-127),
			),
		}
	}

	fmt.Println("Simulating BMA Traversal (Int8 Weight Propagation)...")

	// Benchmark the traversal
	start := time.Now()
	for _, edge := range edges {
		// Propagation logic: target = target + (source * weight)
		// We promote weight to Quat for the calculation.
		sourceVal := nodes[edge.From].Value
		weight := edge.Weight.ToQuat()
		
		prod := quat.Mul(sourceVal, weight)
		nodes[edge.To].Value = quat.MulAccum(nodes[edge.To].Value, prod, quat.Scalar(1.0))
	}
	elapsed := time.Since(start)

	fmt.Printf("Traversal completed in: %v\n", elapsed)
	fmt.Printf("Average time per edge:  %v\n", elapsed/time.Duration(numEdges))
	
	// Memory footprint check
	q8Size := 4 // 4 bytes for int8 w,x,y,z
	q64Size := 32 // 32 bytes for float64 w,x,y,z
	
	fmt.Println("\nMemory Analysis:")
	fmt.Printf("  Quat8 Storage (Total):  %.2f MB\n", float64(numEdges*q8Size)/1024/1024)
	fmt.Printf("  Quat64 Storage (Total): %.2f MB\n", float64(numEdges*q64Size)/1024/1024)
	fmt.Printf("  Memory Savings:         %.1f×\n", float64(q64Size)/float64(q8Size))

	fmt.Println("\nVERDICT: BMA Integration (Quat8) provides significant memory density gains.")
	fmt.Println("========================================================================")
}
