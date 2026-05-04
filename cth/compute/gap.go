package compute

import (
	"math"
	"sort"

	"github.com/helpful-engineering/cth/model"
)

// StepDifficulty returns the estimated information cost to close an unproven step (Definition 16).
func StepDifficulty(stepCategory string) float64 {
	switch stepCategory {
	case "routine_lean":
		return 0.1
	case "novel_lean":
		return 0.5
	case "open_derivation":
		return 1.0
	case "open_problem":
		return 5.0
	case "irreducible":
		return math.Inf(1)
	default:
		return 1.0 // Default to open derivation
	}
}

// EddyRanking ranks irreducible inputs by their derivation proximity.
type EddyRanking struct {
	AnchorID      string
	Proximity     float64
	WeightedGap   float64
	NearestProven string
}

// WeightedGap computes the sum of difficulties along the shortest path 
// from an input to the nearest proven ancestor (Definition 17).
func WeightedGap(inputID string, inv model.Inventory) (float64, string) {
	// 1. Build an adjacency map of the hypergraph (walking backwards)
	// target -> sources
	adj := make(map[string][]string)
	for _, c := range inv.Chains {
		for _, src := range c.SourceIDs {
			adj[c.TargetID] = append(adj[c.TargetID], src)
		}
	}

	// 2. Identify all proven anchors (Tier 1)
	proven := make(map[string]bool)
	for _, a := range inv.DerivedPrinciples {
		if a.Tier == model.Proof {
			proven[a.ID] = true
		}
	}
	for _, a := range inv.Axioms {
		proven[a.ID] = true // Axioms are "proven" for the purpose of gaps
	}

	// 3. Dijkstra-like weighted traversal to find nearest proven anchor
	// We want the shortest path from inputID to any ID in 'proven'
	type node struct {
		id   string
		dist float64
	}
	
	dist := make(map[string]float64)
	dist[inputID] = 0
	
	queue := []node{{id: inputID, dist: 0}}
	
	minDist := math.Inf(1)
	nearest := ""

	for len(queue) > 0 {
		// Sort queue by distance (simple priority queue)
		sort.Slice(queue, func(i, j int) bool { return queue[i].dist < queue[j].dist })
		curr := queue[0]
		queue = queue[1:]

		if proven[curr.id] {
			if curr.dist < minDist {
				minDist = curr.dist
				nearest = curr.id
			}
			continue // Found a path to proven, but there might be shorter ones
		}

		if curr.dist >= minDist {
			continue
		}

		// In the Crawl phase, we assume each "step" in the chain 
		// has a difficulty of 'open_derivation' (1.0) unless specified.
		difficulty := 1.0 
		
		for _, neighbor := range adj[curr.id] {
			newDist := curr.dist + difficulty
			if d, ok := dist[neighbor]; !ok || newDist < d {
				dist[neighbor] = newDist
				queue = append(queue, node{id: neighbor, dist: newDist})
			}
		}
	}

	return minDist, nearest
}

// EddyProximity computes η/g_w (Definition 17).
func EddyProximity(inputID string, inv model.Inventory) float64 {
	gap, _ := WeightedGap(inputID, inv)
	if gap == 0 || math.IsInf(gap, 1) {
		return 0
	}
	
	// Find the anchor's entropy
	var entropy float64
	for _, a := range inv.Inputs {
		if a.ID == inputID {
			entropy = ResidualEntropy(a)
			break
		}
	}
	
	return entropy / gap
}

// RankEddies sorts all programme inputs by weighted eddy proximity.
func RankEddies(inv model.Inventory) []EddyRanking {
	var rankings []EddyRanking
	for _, input := range inv.Inputs {
		gap, nearest := WeightedGap(input.ID, inv)
		proximity := 0.0
		if gap > 0 && !math.IsInf(gap, 1) {
			proximity = ResidualEntropy(input) / gap
		}
		
		rankings = append(rankings, EddyRanking{
			AnchorID:      input.ID,
			Proximity:     proximity,
			WeightedGap:   gap,
			NearestProven: nearest,
		})
	}
	
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].Proximity > rankings[j].Proximity
	})
	
	return rankings
}
