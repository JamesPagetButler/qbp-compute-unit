package emulator

import (
	"testing"
)

// TestWord12_HolographicSearch demonstrates concept alignment at QW1024.
// We store a document as a "Holon" and search for specific semantic resonance.
func TestWord12_HolographicSearch(t *testing.T) {
	cpu := NewCPU()
	cpu.SetWidth(W1024) // Maximum bloat-proof precision

	// 1. Define base semantic vectors (unit quaternions)
	// Concept: "Colorado River" (q1)
	cpu.Q[1].W.SetFloat64(0.707)
	cpu.Q[1].X.SetFloat64(0.707)
	cpu.Q[1].Y.SetFloat64(0.0)
	cpu.Q[1].Z.SetFloat64(0.0)

	// Concept: "Plant Interception" (q2)
	cpu.Q[2].W.SetFloat64(0.707)
	cpu.Q[2].X.SetFloat64(0.0)
	cpu.Q[2].Y.SetFloat64(0.707)
	cpu.Q[2].Z.SetFloat64(0.0)

	// 2. Create the "Document Holon" (q3)
	// In Word 12, we 'layer' the concepts into one register.
	// q3 = Normalize(q1 + q2)
	cpu.Q[3].W.Add(cpu.Q[1].W, cpu.Q[2].W)
	cpu.Q[3].X.Add(cpu.Q[1].X, cpu.Q[2].X)
	cpu.Q[3].Y.Add(cpu.Q[1].Y, cpu.Q[2].Y)
	cpu.Q[3].Z.Add(cpu.Q[1].Z, cpu.Q[2].Z)
	// (Simplifying normalization for this test case)

	// 3. Perform a Search for "Interception" (q2) against the Document (q3)
	// Instruction: QMUL.1024 q4, q2, q3
	// The scalar result (q4.W) represents the semantic alignment.
	var word uint32 = OpcodeCustom0 | (4 << 7) | (7 << 12) | (2 << 15) | (3 << 20) | (Funct7QMUL << 25)
	cpu.Step(word)

	t.Logf("Word 12 Search Result (Alignment): %v", cpu.Q[4].W)

	// 4. Define an unrelated concept: "Tornado" (q5)
	// This vector is orthogonal (90 degrees away) from the document's concepts.
	cpu.Q[5].W.SetFloat64(0.0)
	cpu.Q[5].X.SetFloat64(0.0)
	cpu.Q[5].Y.SetFloat64(0.0)
	cpu.Q[5].Z.SetFloat64(1.0)

	// Search for "Tornado" in the "Colorado River" document
	// Instruction: QMUL.1024 q6, q5, q3
	var word2 uint32 = OpcodeCustom0 | (6 << 7) | (7 << 12) | (5 << 15) | (3 << 20) | (Funct7QMUL << 25)
	cpu.Step(word2)

	t.Logf("Word 12 Search Result (Unrelated): %v", cpu.Q[6].W)

	// 5. Verification: Alignment with document should be much higher than unrelated concept.
	if cpu.Q[4].W.Cmp(cpu.Q[6].W) <= 0 {
		t.Errorf("FAIL: Search failed to distinguish relevant concept. Alignment: %v, Unrelated: %v", cpu.Q[4].W, cpu.Q[6].W)
	} else {
		t.Logf("WORD 12 SUCCESS: Holographic resonance detected relevant concept at 1024-bit precision.")
	}
}
