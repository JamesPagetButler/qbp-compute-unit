// Package report generates human-readable output from CTH compute results.
package report

import (
	"github.com/helpful-engineering/cth/compute"
	"github.com/helpful-engineering/cth/model"
)

// FullAnalysis aggregates every compute result for a single inventory.
// Build it with Analyse, then pass to Dashboard or MarkdownReport.
type FullAnalysis struct {
	Rho         float64
	RhoDetail   compute.NetCompressionDetail
	Sensitivity [3]float64 // [halfH, baseH, doubleH]
	SensRatio   float64

	Eddies       []compute.EddyRanking
	BridgeNodes  []compute.BridgeNode
	Sediment     compute.SedimentReport
	AnchorDepths map[string]int
	ChainDepths  map[string]int
	AbInitio     []compute.AbInitioResult
}

// Analyse runs all Crawl-phase compute functions and returns a FullAnalysis.
// inputEntropy maps each input anchor ID to its entropy cost; pass
// compute.BuildInputEntropy(inv) for the default (no scaling).
func Analyse(inv model.Inventory, inputEntropy map[string]float64) FullAnalysis {
	rho, detail := compute.NetCompression(inv, inputEntropy)
	halfH, baseH, doubleH := compute.SensitivityBracket(inv, inputEntropy)

	return FullAnalysis{
		Rho:          rho,
		RhoDetail:    detail,
		Sensitivity:  [3]float64{halfH, baseH, doubleH},
		SensRatio:    compute.SensitivityRatio(halfH, doubleH),
		Eddies:       compute.RankEddies(inv),
		BridgeNodes:  compute.BridgeCentrality(inv, false),
		Sediment:     compute.DetectSedimentPartitions(inv),
		AnchorDepths: compute.AnchorConfluenceDepth(inv),
		ChainDepths:  compute.ChainConfluenceDepth(inv),
		AbInitio:     compute.AbInitioScore(inv),
	}
}
