package testdata

import "github.com/helpful-engineering/cth/model"

// ExpectedValues holds hand-verified compute results for fixtures.
type ExpectedValues struct {
	NetCompression float64
	Deficit        float64
	AxiomEntropy   float64
}

// MinimalValues are hand-calculated from minimal.json.
var MinimalValues = ExpectedValues{
	NetCompression: 0.28, // Example value
	Deficit:        6.64,
	AxiomEntropy:   2.0,
}

// QBPv32Values are regression values from Python engine.
var QBPv32Values = ExpectedValues{
	NetCompression: 0.765,
	Deficit:        63.1,
}

// QBPQuantumValues are the starting points for the Third River confluence.
var QBPQuantumValues = ExpectedValues{
	NetCompression: 0.388,
	Deficit:        16.6,
}
