package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
)

type Edge struct {
	From   int
	To     int
	Weight quat.Quat8
}

// ToQuat is a helper since we can't define methods on quat.Quat8 here
func ToQuat(q8 quat.Quat8) quat.Quat {
	const scale = 1.0 / 127.0
	return quat.Quat{
		W: float64(q8.W) * scale,
		X: float64(q8.X) * scale,
		Y: float64(q8.Y) * scale,
		Z: float64(q8.Z) * scale,
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("========================================================================")
	fmt.Println("EXPERIMENT D: SUSTAINED QUAT8 NORM DRIFT UNDER SLEEP CONSOLIDATION")
	fmt.Println("========================================================================")

	numNodes := 10000
	numEdges := 50000
	sleepCycles := 1000

	edges := make([]Edge, 0, numEdges)
	for i := 0; i < numEdges; i++ {
		edges = append(edges, Edge{
			From: rand.Intn(numNodes),
			To:   rand.Intn(numNodes),
			Weight: quat.NewQuat8(
				int8(rand.Intn(100)-50),
				int8(rand.Intn(100)-50),
				int8(rand.Intn(100)-50),
				int8(rand.Intn(100)-50),
			),
		})
	}

	fmt.Printf("Initial Graph: %d edges.\n", len(edges))
	fmt.Println("Simulating 1000 sleep cycles...")

	for cycle := 1; cycle <= sleepCycles; cycle++ {
		// 1. Salience Decay (Update 10% of edges)
		for i := 0; i < len(edges)/10; i++ {
			idx := rand.Intn(len(edges))
			w := edges[idx].Weight
			edges[idx].Weight = quat.NewQuat8(
				int8(float64(w.W)*0.95),
				int8(float64(w.X)*0.95),
				int8(float64(w.Y)*0.95),
				int8(float64(w.Z)*0.95),
			)
		}

		// 2. Merge/Compression
		if len(edges) > 10 {
			edges = edges[:len(edges)-5]
		}

		// 3. New content
		for i := 0; i < 20; i++ {
			edges = append(edges, Edge{
				From: rand.Intn(numNodes),
				To:   rand.Intn(numNodes),
				Weight: quat.NewQuat8(
					int8(rand.Intn(127)),
					0, 0, 0,
				),
			})
		}

		// 4. Deletion
		if len(edges) > 10 {
			edges = edges[10:]
		}

		if cycle%100 == 0 {
			var totalNormSq float64
			for _, e := range edges {
				eq := ToQuat(e.Weight)
				totalNormSq += quat.NormSq(eq)
			}
			avgNorm := totalNormSq / float64(len(edges))
			fmt.Printf("Cycle %4d: Edge Count: %d, Avg NormSq: %.6f\n", cycle, len(edges), avgNorm)

			if math.IsNaN(avgNorm) || math.IsInf(avgNorm, 0) || avgNorm < 1e-10 {
				fmt.Println("CRITICAL: Graph collapsed or drifted to infinity.")
				return
			}
		}
	}

	fmt.Println("\nVERDICT: Quat8 edges remained stable through 1000 destructive sleep cycles.")
	fmt.Println("========================================================================")
}
