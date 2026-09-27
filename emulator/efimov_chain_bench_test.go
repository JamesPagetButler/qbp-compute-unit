// efimov_chain_bench_test.go — chained emulator-vs-native complex-multiply
// overhead, to arch's Efimov round-2 spec (live-test seq=1758).
//
// This is the CHAINED-PATH companion to the PRIMITIVE numbers in
// baseline_bench_test.go (BenchmarkNativeX86_ComplexMul64 / the Gearbox CMul64
// path). The primitive number times one op; this times an O(N) dependent chain,
// which is what a generated-code amplitude pipeline actually looks like.
//
// Anti-measurement-artifact discipline (the trap that made the prior native
// timing flat across N — arch seq=1758):
//   1. testing.B / ns/op — never wall-clock of a single run.
//   2. Node values from a RUNTIME slice, not compile-time constants (no folding).
//   3. One input perturbed per iteration by the loop index (tiny delta, |q|≈1),
//      so the result is not loop-invariant (no hoisting).
//   4. Final result assigned to a package-level sink each iteration (no dead-code
//      elimination of the whole chain).
// Both sides get identical treatment so the ratio is honest.
//
// Caveat that MUST travel with any banked number: this box is AVX+FMA, NO AVX2 —
// the Gearbox abstraction loses to naive scalar here, so overhead > 1 is a
// HARDWARE artifact, not an emulator defect. Re-measure on AVX2 hardware before
// reading any factor as an architecture ceiling.
package emulator

import "testing"

// Package-level sinks — defeat dead-code elimination of the chain result.
var (
	SinkEmu  [2]float64
	SinkNat  complex128
)

// makeChainNodes builds N unit-ish complex nodes from a runtime seed so the
// values are not compile-time constants.
func makeChainNodes(n int) [][2]float64 {
	nodes := make([][2]float64, n)
	for i := range nodes {
		// spread around the unit circle; |q| ~ 1
		a := 0.6 + float64(i%7)*0.01
		b := 0.8 - float64(i%5)*0.01
		nodes[i] = [2]float64{a, b}
	}
	return nodes
}

func benchEmuChain(b *testing.B, n int) {
	g := NewGearbox()
	nodes := makeChainNodes(n)
	b.ResetTimer()
	b.ReportAllocs()
	for iter := 0; iter < b.N; iter++ {
		// perturb one input per iteration so the chain is not loop-invariant
		delta := float64(iter%1000) * 1e-9
		acc := [2]float64{nodes[0][0] + delta, nodes[0][1]}
		for k := 1; k < n; k++ {
			acc = g.CMul64(acc, nodes[k])
		}
		SinkEmu = acc
	}
}

func benchNatChain(b *testing.B, n int) {
	nodes := makeChainNodes(n)
	// native uses the same node data as complex128
	cn := make([]complex128, n)
	for i, q := range nodes {
		cn[i] = complex(q[0], q[1])
	}
	b.ResetTimer()
	b.ReportAllocs()
	for iter := 0; iter < b.N; iter++ {
		delta := float64(iter%1000) * 1e-9
		acc := complex(real(cn[0])+delta, imag(cn[0]))
		for k := 1; k < n; k++ {
			acc = acc * cn[k]
		}
		SinkNat = acc
	}
}

func BenchmarkEfimovChainEmu_N100(b *testing.B)    { benchEmuChain(b, 100) }
func BenchmarkEfimovChainNat_N100(b *testing.B)    { benchNatChain(b, 100) }
func BenchmarkEfimovChainEmu_N1000(b *testing.B)   { benchEmuChain(b, 1000) }
func BenchmarkEfimovChainNat_N1000(b *testing.B)   { benchNatChain(b, 1000) }
func BenchmarkEfimovChainEmu_N10000(b *testing.B)  { benchEmuChain(b, 10000) }
func BenchmarkEfimovChainNat_N10000(b *testing.B)  { benchNatChain(b, 10000) }
func BenchmarkEfimovChainEmu_N100000(b *testing.B) { benchEmuChain(b, 100000) }
func BenchmarkEfimovChainNat_N100000(b *testing.B) { benchNatChain(b, 100000) }
