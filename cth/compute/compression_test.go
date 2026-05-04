package compute

import (
	"math"
	"testing"

	"github.com/helpful-engineering/cth/store"
)

func TestCompression(t *testing.T) {
	inv, err := store.LoadInventory("../testdata/minimal.json")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	rho := GrossCompression(inv)
	// minimal: ι(MEAS-1)=3.32, η(AX-1)=2.0, η(IN-1)=6.64. denom=8.64. rho=3.32/8.64 ≈ 0.384
	expectedGross := 3.32 / (2.0 + 6.64)
	if math.Abs(rho-expectedGross) > 0.001 {
		t.Errorf("GrossCompression = %v; want %v", rho, expectedGross)
	}

	inputMap := BuildInputEntropy(inv)
	rhoNet, detail := NetCompression(inv, inputMap)

	// minimal: ι_net = 3.32 - 6.64 = -3.32. rho_net = -3.32 / 8.64 ≈ -0.384
	expectedNet := (3.32 - 6.64) / 8.64
	if math.Abs(rhoNet-expectedNet) > 0.001 {
		t.Errorf("NetCompression = %v; want %v", rhoNet, expectedNet)
	}
	if detail.TotalDenominator != 8.64 {
		t.Errorf("expected denom 8.64, got %v", detail.TotalDenominator)
	}
}

func TestNetCompressionCustomMap(t *testing.T) {
	inv, err := store.LoadInventory("../testdata/minimal.json")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Halve the input entropy — ρ_net should improve.
	halfMap := map[string]float64{"IN-1": 3.32}
	rhoHalf, _ := NetCompression(inv, halfMap)

	// Normal map for comparison.
	baseMap := BuildInputEntropy(inv)
	rhoBase, _ := NetCompression(inv, baseMap)

	if rhoHalf <= rhoBase {
		t.Errorf("halved input entropy should raise ρ_net: got %v <= %v", rhoHalf, rhoBase)
	}
}

func TestVelocity(t *testing.T) {
	// Δρ / Δn = (0.765 - 0.760) / (200 - 100) = 0.005 / 100 = 0.00005
	v := CompressionVelocity(
		VersionSnapshot{Rho: 0.760, NAnchor: 100},
		VersionSnapshot{Rho: 0.765, NAnchor: 200},
	)
	if math.Abs(v-0.00005) > 1e-10 {
		t.Errorf("CompressionVelocity = %v; want 0.00005", v)
	}

	// Zero / negative Δn returns 0.
	if v := CompressionVelocity(
		VersionSnapshot{Rho: 0.8, NAnchor: 200},
		VersionSnapshot{Rho: 0.9, NAnchor: 100},
	); v != 0 {
		t.Errorf("expected 0 for negative Δn, got %v", v)
	}
}
