package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
)

type Node struct {
	ID        int
	Embedding quat.Quat // Single quat embedding for simplicity
	Edges     []int
}

func main() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("========================================================================")
	fmt.Println("EXPERIMENT C: SPREADING ACTIVATION VS HNSW (LINEAR PROXY)")
	fmt.Println("========================================================================")

	numNodes := 100000
	numEdges := 500000
	topK := 10

	fmt.Printf("Building Graph: %d nodes, %d edges...\n", numNodes, numEdges)
	nodes := make([]Node, numNodes)
	for i := range nodes {
		nodes[i] = Node{
			ID:        i,
			Embedding: quat.New(rand.NormFloat64(), rand.NormFloat64(), rand.NormFloat64(), rand.NormFloat64()),
			Edges:     make([]int, 0),
		}
	}

	for i := 0; i < numEdges; i++ {
		from := rand.Intn(numNodes)
		to := rand.Intn(numNodes)
		nodes[from].Edges = append(nodes[from].Edges, to)
	}

	queryIdx := rand.Intn(numNodes)
	queryNode := nodes[queryIdx]

	// Method 1: Linear Search (HNSW Proxy)
	fmt.Println("\nRunning Linear Search (HNSW Proxy)...")
	startLS := time.Now()
	type LSResult struct {
		ID    int
		Score float64
	}
	lsResults := make([]LSResult, numNodes)
	for i := 0; i < numNodes; i++ {
		// Using dot product as similarity
		score := queryNode.Embedding.W*nodes[i].Embedding.W +
			queryNode.Embedding.X*nodes[i].Embedding.X +
			queryNode.Embedding.Y*nodes[i].Embedding.Y +
			queryNode.Embedding.Z*nodes[i].Embedding.Z
		lsResults[i] = LSResult{i, score}
	}
	sort.Slice(lsResults, func(i, j int) bool { return lsResults[i].Score > lsResults[j].Score })
	elapsedLS := time.Since(startLS)

	groundTruth := make(map[int]bool)
	for i := 1; i <= topK; i++ {
		groundTruth[lsResults[i].ID] = true
	}

	// Method 2: Spreading Activation
	fmt.Println("Running Spreading Activation (3 hops)...")
	startSA := time.Now()
	activation := make(map[int]float64)
	activation[queryIdx] = 1.0

	for hop := 0; hop < 3; hop++ {
		nextActivation := make(map[int]float64)
		for nodeID, energy := range activation {
			if energy < 0.01 {
				continue
			}
			share := energy / float64(len(nodes[nodeID].Edges)+1)
			for _, neighbor := range nodes[nodeID].Edges {
				nextActivation[neighbor] += share
			}
		}
		activation = nextActivation
	}

	type SAResult struct {
		ID    int
		Score float64
	}
	saResults := make([]SAResult, 0, len(activation))
	for id, score := range activation {
		saResults = append(saResults, SAResult{id, score})
	}
	sort.Slice(saResults, func(i, j int) bool { return saResults[i].Score > saResults[j].Score })
	elapsedSA := time.Since(startSA)

	// Evaluate
	matches := 0
	checkK := topK
	if len(saResults) < checkK {
		checkK = len(saResults)
	}
	for i := 0; i < checkK; i++ {
		if groundTruth[saResults[i].ID] {
			matches++
		}
	}

	fmt.Printf("\nLinear Search Time:     %v\n", elapsedLS)
	fmt.Printf("Spreading Activation Time: %v\n", elapsedSA)
	fmt.Printf("Recall@%d (SA vs LS):    %.1f%%\n", topK, float64(matches)/float64(topK)*100)

	fmt.Println("\nCONCLUSION:")
	if matches > 0 {
		fmt.Println("Spreading Activation captures some semantic neighbors from graph topology.")
	} else {
		fmt.Println("Spreading Activation (topology-only) did not overlap with embedding similarity.")
	}
	fmt.Println("========================================================================")
}
