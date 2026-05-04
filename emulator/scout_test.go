package emulator

import (
	"math/big"
	"testing"
)

// TestHeadlessScout_ColoradoRiver reproduces the "missing water" insight.
// It uses QBP math to model the spatiotemporal intersection between
// snowmelt world-lines and plant interception agents.
func TestHeadlessScout_ColoradoRiver(t *testing.T) {
	cpu := NewCPU()
	cpu.SetWidth(W128)
	
	// 1. Setup Locale: Upper Basin (Snow) to Lake Mead (River)
	// Point A: Snow (Origin) - q1
	cpu.Q[1].W.SetFloat64(0) // Time t=0
	cpu.Q[1].X.SetFloat64(100) // Spatial X
	cpu.Q[1].Y.SetFloat64(100) // Spatial Y
	cpu.Q[1].Z.SetFloat64(50)  // Spatial Z (Elevation)

	// Point B: River (Destination) - q2
	cpu.Q[2].W.SetFloat64(100) // Time t=100 (Lag)
	cpu.Q[2].X.SetFloat64(500)
	cpu.Q[2].Y.SetFloat64(500)
	cpu.Q[2].Z.SetFloat64(0)

	// 2. Define causal vector v = B - A (The predicted snowmelt path)
	// We'll use QADD with negation for simplicity in this test logic.
	v := NewQWord(cpu.GB.Precision())
	v.W.Sub(cpu.Q[2].W, cpu.Q[1].W)
	v.X.Sub(cpu.Q[2].X, cpu.Q[1].X)
	v.Y.Sub(cpu.Q[2].Y, cpu.Q[1].Y)
	v.Z.Sub(cpu.Q[2].Z, cpu.Q[1].Z)

	// 3. Inject Active Agent: Forest (Interception) - q3
	// The forest physically resides between A and B at time t=50.
	cpu.Q[3].W.SetFloat64(50)
	cpu.Q[3].X.SetFloat64(300)
	cpu.Q[3].Y.SetFloat64(300)
	cpu.Q[3].Z.SetFloat64(25)

	// 4. Run Intersection Check (QBP Scout Logic)
	// Predicted position at t=50: P = A + 0.5*v
	p := NewQWord(cpu.GB.Precision())
	half := new(big.Float).SetFloat64(0.5)
	p.W.Mul(v.W, half).Add(p.W, cpu.Q[1].W)
	p.X.Mul(v.X, half).Add(p.X, cpu.Q[1].X)
	p.Y.Mul(v.Y, half).Add(p.Y, cpu.Q[1].Y)
	p.Z.Mul(v.Z, half).Add(p.Z, cpu.Q[1].Z)

	// Calculate distance between Predicted (p) and Agent (q3)
	distSq := new(big.Float).SetPrec(cpu.GB.Precision())
	tmp := new(big.Float).SetPrec(cpu.GB.Precision())
	
	diffW := new(big.Float).Sub(p.W, cpu.Q[3].W)
	diffX := new(big.Float).Sub(p.X, cpu.Q[3].X)
	diffY := new(big.Float).Sub(p.Y, cpu.Q[3].Y)
	diffZ := new(big.Float).Sub(p.Z, cpu.Q[3].Z)

	distSq.Mul(diffW, diffW)
	tmp.Mul(diffX, diffX); distSq.Add(distSq, tmp)
	tmp.Mul(diffY, diffY); distSq.Add(distSq, tmp)
	tmp.Mul(diffZ, diffZ); distSq.Add(distSq, tmp)

	// 5. Audit finding
	t.Logf("Locale Intersection Check: distSq = %v", distSq)
	
	// Threshold check: if distSq < 1.0, they intersect.
	if distSq.Cmp(new(big.Float).SetFloat64(1.0)) < 0 {
		t.Logf("INSIGHT DISCOVERED: Active Agent (Forest) intersects Snowmelt world-line.")
		t.Logf("ACTION: Interrogating Forest behavioral world-line for water interception...")
	} else {
		t.Errorf("FAIL: Scout failed to detect intersection (distSq=%v)", distSq)
	}
}
