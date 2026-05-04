package gap

import (
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
)

// Relevance bits for stance-based gating.
const (
	RelAtmospheric uint64 = 1 << iota
	RelHydrological
	RelBiological
	RelChemical
	RelAnthropogenic
)

// GatedNode extends GAPNode with a relevance mask for lossless dismissal.
type GatedNode struct {
	ID            int
	State         quat.Quat
	Adjacency     []AdjacencyPointer
	RelevanceMask uint64
}

// CalculateGatedGradient implements Stance-Based Gating (Spec 1.4).
// If a neighbor's mask does not match the active stance, it is dismissed 
// (treated as Identity) during the traversal.
func (n *GatedNode) CalculateGatedGradient(nodes []GatedNode, activeStanceMask uint64) quat.Quat {
	grad := quat.Scalar(0)
	
	for _, ptr := range n.Adjacency {
		neighbor := nodes[ptr.TargetID]
		
		// THE GATE: If no overlap in relevance bits, dismiss neighbor.
		if (neighbor.RelevanceMask & activeStanceMask) == 0 {
			continue 
		}
		
		// standard traversal for relevant nodes
		diff := quat.Sub(neighbor.State, n.State)
		contribution := quat.Mul(diff, ptr.Weight.ToQuat())
		grad = quat.Add(grad, contribution)
	}
	
	return grad
}
