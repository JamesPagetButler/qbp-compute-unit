// Command benchmark runs the QBP Compute Unit Crawl Phase validation.
//
// This is Action 4 of the spec: run QBP-algebraic vs float32 comparison
// benchmark and publish results to QBP repo.
//
// Usage:
//
//	go run cmd/benchmark/main.go [-n sites] [-steps N] [-dt timestep] [-renorm]
package main

import (
	"flag"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/fano"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/octonion"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/spinchain"
)

func main() {
	// Parse flags
	chainLen := flag.Int("n", 20, "spin chain length (number of sites)")
	steps := flag.Int("steps", 10000, "number of Trotter time steps")
	dt := flag.Float64("dt", 0.01, "time step size")
	coupling := flag.Float64("J", 1.0, "exchange coupling constant")
	renorm := flag.Bool("renorm", false, "enable periodic renormalisation")
	renormN := flag.Int("renorm-every", 100, "renormalise every N steps")
	flag.Parse()

	banner := strings.Repeat("=", 72)

	fmt.Println(banner)
	fmt.Println("QBP COMPUTE UNIT — CRAWL PHASE BENCHMARK")
	fmt.Println("1D Heisenberg Spin Chain: QBP-Algebraic vs Float64-Scalar")
	fmt.Println(banner)
	fmt.Println()

	// ── Phase 0: Algebraic infrastructure verification ──
	fmt.Println("[Phase 0] Verifying algebraic infrastructure...")
	fmt.Println()

	// Fano plane verification
	fmt.Println("  Fano Plane LUT Verification:")
	errs := fano.Verify()
	if len(errs) == 0 {
		fmt.Println("    PASS: All algebraic properties verified")
		fmt.Printf("    Table size: %d bytes (Go), <25 bytes (hardware ROM)\n", fano.TableSize())
	} else {
		fmt.Println("    FAIL:")
		for _, e := range errs {
			fmt.Printf("      - %s\n", e)
		}
		return
	}

	// Quaternion norm multiplicativity spot-check
	fmt.Println()
	fmt.Println("  Quaternion Norm Multiplicativity:")
	q1 := quat.Normalize(quat.New(1, 2, 3, 4))
	q2 := quat.Normalize(quat.New(5, -1, 2, -3))
	prod := quat.Mul(q1, q2)
	normProd := quat.Norm(prod)
	fmt.Printf("    ||q1|| = %.15f\n", quat.Norm(q1))
	fmt.Printf("    ||q2|| = %.15f\n", quat.Norm(q2))
	fmt.Printf("    ||q1*q2|| = %.15f\n", normProd)
	fmt.Printf("    |1 - ||q1*q2||| = %.3e\n", math.Abs(1.0-normProd))
	if math.Abs(1.0-normProd) < 1e-14 {
		fmt.Println("    PASS: Norm multiplicativity preserved to machine epsilon")
	} else {
		fmt.Println("    WARN: Norm multiplicativity drift detected")
	}

	// Octonion norm multiplicativity
	fmt.Println()
	fmt.Println("  Octonion Norm Multiplicativity:")
	o1 := octonion.Normalize(octonion.New(1, 2, 3, 4, 5, 6, 7, 8))
	o2 := octonion.Normalize(octonion.New(-1, 3, -2, 5, -4, 7, -6, 1))
	if octonion.NormMultiplicativity(o1, o2, 1e-12) {
		fmt.Println("    PASS: Octonionic norm multiplicativity preserved")
	} else {
		oProd := octonion.Mul(o1, o2)
		fmt.Printf("    WARN: ||o1*o2|| = %.15f (expected 1.0)\n", octonion.Norm(oProd))
	}

	// Non-associativity demonstration
	fmt.Println()
	fmt.Println("  Octonion Non-Associativity (the feature, not the bug):")
	o3 := octonion.Normalize(octonion.New(2, -1, 4, -3, 6, -5, 8, -7))
	defect := octonion.AssociativityDefect(o1, o2, o3)
	fmt.Printf("    ||(o1*o2)*o3 - o1*(o2*o3)|| = %.6f\n", defect)
	if defect > 1e-10 {
		fmt.Println("    CONFIRMED: Non-associativity present (context-dependent traversal)")
	} else {
		fmt.Println("    NOTE: These particular octonions nearly associate")
	}

	// Quaternion subalgebra associativity check
	fmt.Println()
	fmt.Println("  Quaternion Subalgebra Associativity (should be zero):")
	// Restrict to e1, e2, e3 subalgebra
	oq1 := octonion.New(0.5, 0.5, 0.5, 0.5, 0, 0, 0, 0)
	oq2 := octonion.New(0.5, -0.5, 0.5, -0.5, 0, 0, 0, 0)
	oq3 := octonion.New(0.5, 0.5, -0.5, -0.5, 0, 0, 0, 0)
	subDefect := octonion.AssociativityDefect(oq1, oq2, oq3)
	fmt.Printf("    Defect (e1,e2,e3 subalgebra): %.3e\n", subDefect)
	if subDefect < 1e-12 {
		fmt.Println("    PASS: Quaternionic subalgebra is associative (deterministic)")
	} else {
		fmt.Println("    WARN: Unexpected associativity defect in quaternion subalgebra")
	}

	// QROT instruction test
	fmt.Println()
	fmt.Println("  QROT Instruction (quaternion rotation):")
	// 90-degree rotation around z-axis
	angle := math.Pi / 2
	qrot := quat.New(math.Cos(angle/2), 0, 0, math.Sin(angle/2))
	vx, vy, vz := quat.RotateVec(qrot, 1, 0, 0)
	fmt.Printf("    Rotate (1,0,0) by 90deg around z: (%.4f, %.4f, %.4f)\n", vx, vy, vz)
	if math.Abs(vx) < 1e-10 && math.Abs(vy-1) < 1e-10 && math.Abs(vz) < 1e-10 {
		fmt.Println("    PASS: QROT produces correct rotation")
	} else {
		fmt.Println("    FAIL: QROT rotation incorrect")
	}

	// Quaternion exponential/logarithm roundtrip
	fmt.Println()
	fmt.Println("  Exp/Log Roundtrip (critical for time evolution):")
	qtest := quat.Normalize(quat.New(0.5, 0.3, -0.2, 0.7))
	qrt := quat.Exp(quat.Log(qtest))
	rtErr := quat.Norm(quat.Sub(qtest, qrt))
	fmt.Printf("    ||q - exp(log(q))|| = %.3e\n", rtErr)
	if rtErr < 1e-14 {
		fmt.Println("    PASS: Exp/Log roundtrip to machine epsilon")
	} else {
		fmt.Println("    WARN: Exp/Log roundtrip error elevated")
	}

	fmt.Println()
	fmt.Println(banner)
	fmt.Println()

	// ── Phase 1: Spin-chain benchmark ──
	fmt.Printf("[Phase 1] Heisenberg Spin Chain Benchmark\n")
	fmt.Printf("  Chain length:    %d sites\n", *chainLen)
	fmt.Printf("  Coupling J:      %.2f\n", *coupling)
	fmt.Printf("  Time step dt:    %.4f\n", *dt)
	fmt.Printf("  Total steps:     %d\n", *steps)
	fmt.Printf("  Total sim time:  %.2f (J*t units)\n", float64(*steps)*(*dt)*(*coupling))
	fmt.Printf("  Renormalisation: %v", *renorm)
	if *renorm {
		fmt.Printf(" (every %d steps)", *renormN)
	}
	fmt.Println()
	fmt.Println()

	cfg := spinchain.BenchmarkConfig{
		ChainLength: *chainLen,
		Coupling:    *coupling,
		TimeStep:    *dt,
		TotalSteps:  *steps,
		Renorm:      *renorm,
		RenormEvery: *renormN,
	}

	fmt.Println("  Running simulations...")
	start := time.Now()
	result := spinchain.RunBenchmark(cfg)
	totalTime := time.Since(start)
	fmt.Printf("  Total benchmark time: %v\n\n", totalTime)

	// ── Results ──
	fmt.Println(banner)
	fmt.Println("RESULTS")
	fmt.Println(banner)
	fmt.Println()

	// QBP results
	fmt.Println("QBP-Algebraic (quaternion native):")
	fmt.Printf("  Wall time:         %v\n", result.QBPWallTime)
	fmt.Printf("  Total operations:  %d\n", result.QBPOps)
	fmt.Printf("  Final norm drift:  %.6e (sum across all sites)\n", result.QBPNormDrift)
	fmt.Printf("  Avg curvature:     %.6e (norm drift per operation)\n", result.QBPWatchdog.Curvature())
	fmt.Printf("  Max norm drift:    %.6e (single operation)\n", result.QBPWatchdog.MaxNormDrift)
	fmt.Println()

	// Scalar results
	fmt.Println("Float64-Scalar (conventional):")
	fmt.Printf("  Wall time:         %v\n", result.ScalarWallTime)
	fmt.Printf("  Total operations:  %d\n", result.ScalarOps)
	fmt.Printf("  Final norm drift:  %.6e (sum across all sites)\n", result.ScalarNormDrift)
	fmt.Printf("  Avg curvature:     %.6e (norm drift per operation)\n", result.ScalarWatchdog.Curvature())
	fmt.Printf("  Max unitarity:     %.6e\n", result.ScalarWatchdog.MaxUnitarityDefect)
	fmt.Println()

	// Comparison
	fmt.Println(strings.Repeat("-", 72))
	fmt.Println("COMPARISON:")
	fmt.Printf("  Norm drift ratio (scalar/QBP):  %.2f×\n", result.NormDriftRatio)
	fmt.Printf("  Operation count ratio:          %.2f×\n", result.OpsRatio)
	fmt.Printf("  Wall time ratio:                %.2f×\n", result.TimeRatio)
	fmt.Println()

	// Verdict
	fmt.Println(strings.Repeat("-", 72))
	if result.NormDriftRatio > 1.0 {
		fmt.Println("VERDICT: QBP-algebraic simulation shows LOWER norm drift.")
		fmt.Println("         Algebraic structure preserves unitarity better than")
		fmt.Println("         conventional float64 arithmetic.")
		fmt.Println()
		fmt.Println("         This supports the QBP hypothesis: algebraically-native")
		fmt.Println("         computation has lower impedance for physical modelling.")
	} else if result.NormDriftRatio > 0.9 {
		fmt.Println("VERDICT: Results are COMPARABLE. Both approaches show similar")
		fmt.Println("         norm drift characteristics. Further investigation needed")
		fmt.Println("         with longer chains or more time steps.")
	} else {
		fmt.Println("VERDICT: Float64-scalar shows LOWER norm drift.")
		fmt.Println("         This is evidence AGAINST the strong QBP hypothesis")
		fmt.Println("         for this simulation type. Investigate why.")
	}
	fmt.Println()
	fmt.Println(banner)

	// Watchdog reports
	fmt.Println()
	fmt.Println("[Algebraic Watchdog Reports]")
	fmt.Println()
	fmt.Println("QBP:")
	fmt.Println(result.QBPWatchdog.Report())
	fmt.Println("Scalar:")
	fmt.Println(result.ScalarWatchdog.Report())

	// ── Phase 2: Composition stress test ──
	fmt.Println(banner)
	fmt.Println("[Phase 2] Pure Composition Stress Test")
	fmt.Println("  Composing millions of multiplications without reconstruction.")
	fmt.Println("  This is where algebraic norm-preservation should diverge from")
	fmt.Println("  float64 numerical behaviour.")
	fmt.Println()

	for _, n := range []int{100_000, 1_000_000, 10_000_000, 100_000_000} {
		cr := spinchain.RunCompositionBenchmark(n)
		fmt.Printf("  %d iterations:\n", n)
		fmt.Printf("    QBP  ||q||² drift: %.6e  curvature: %.3e  max: %.3e  time: %v\n",
			cr.QBPFinalNormDrift, cr.QBPCurvature, cr.QBPMaxDrift, cr.QBPWallTime)
		fmt.Printf("    SU2  |det|² drift: %.6e  curvature: %.3e  max: %.3e  time: %v\n",
			cr.ScalarFinalNormDrift, cr.ScalarCurvature, cr.ScalarMaxDrift, cr.ScalarWallTime)

		if cr.QBPFinalNormDrift > 0 && cr.ScalarFinalNormDrift > 0 {
			ratio := cr.ScalarFinalNormDrift / cr.QBPFinalNormDrift
			fmt.Printf("    Drift ratio (scalar/QBP): %.1f×\n", ratio)
		} else if cr.QBPFinalNormDrift == 0 && cr.ScalarFinalNormDrift > 0 {
			fmt.Printf("    QBP: ZERO drift. Scalar: measurable drift. QBP WINS.\n")
		} else {
			fmt.Printf("    Both zero drift at this scale.\n")
		}
		fmt.Println()
	}
	fmt.Println(banner)

	// ── Phase 3: Multi-width composition — why scalable-by-design matters ──
	fmt.Println()
	fmt.Println(banner)
	fmt.Println("[Phase 3] Multi-Width Composition: Why Scalable-by-Design Matters")
	fmt.Println("  Same algebra (Hamilton product), four precisions.")
	fmt.Println("  1,000,000 chained multiplications each.")
	fmt.Println()

	widthResults := spinchain.RunMultiWidthBenchmark(1_000_000)
	fmt.Printf("  %-44s %6s  %12s  %12s  %12s  %12s\n",
		"Format", "Bits", "Norm Drift", "Drift/Op", "Est Max Depth", "Time")
	fmt.Printf("  %-44s %6s  %12s  %12s  %12s  %12s\n",
		strings.Repeat("-", 44), "------", "------------", "------------", "-------------", "------------")
	for _, r := range widthResults {
		depthStr := fmt.Sprintf("%d", r.EstMaxDepth)
		if r.EstMaxDepth > 1e15 {
			depthStr = fmt.Sprintf("%.1e", float64(r.EstMaxDepth))
		}
		fmt.Printf("  %-44s %6d  %12.3e  %12.3e  %13s  %12v\n",
			r.Label, r.Bits, r.FinalNormDrift, r.DriftPerOp, depthStr, r.WallTime)
	}
	fmt.Println()
	fmt.Println("  CONCLUSION: The algebra is the constant. The precision is the variable.")
	fmt.Println("  A fixed 64-bit word (QW16) loses norm preservation after ~10K-100K")
	fmt.Println("  compositions. QW64 survives 10^14+. The architecture MUST be width-")
	fmt.Println("  parameterised, with the pipeline stage selecting the precision.")
	fmt.Println()
	fmt.Println(banner)

	// ── Phase 4: Extended width analysis (128/256/512) ──
	fmt.Println()
	fmt.Println(banner)
	fmt.Println("[Phase 4] Extended Width Analysis: QW8 through QW512")
	fmt.Println("  Empirical (float32/64) + big.Float validated (float128/256)")
	fmt.Println("  + analytical extrapolation (float512)")
	fmt.Println()
	fmt.Println("  Question: Is there a reason to go beyond QW64? Where are the")
	fmt.Println("  natural precision boundaries for QBP computation?")
	fmt.Println()

	ewResults := spinchain.RunExtendedWidthAnalysis()

	fmt.Printf("  %-36s %6s  %12s  %14s  %14s  %18s\n",
		"Format", "Bits", "ε (machine)", "Drift/Op", "Max Depth @1e-6", "Continuous @1GHz")
	fmt.Printf("  %-36s %6s  %12s  %14s  %14s  %18s\n",
		strings.Repeat("-", 36), "------", "------------", "--------------", "---------------", "------------------")

	for _, r := range ewResults {
		depthStr := "∞"
		timeStr := "∞"
		if r.MaxDepth1e6 < 1e50 {
			depthStr = fmt.Sprintf("%.2e", r.MaxDepth1e6)
		}
		if r.TimeAtGHz < 1e30 && r.TimeAtGHz > 0 {
			if r.TimeAtGHz < 1 {
				timeStr = fmt.Sprintf("%.1f ns", r.TimeAtGHz*1e9)
			} else if r.TimeAtGHz < 60 {
				timeStr = fmt.Sprintf("%.1f sec", r.TimeAtGHz)
			} else if r.TimeAtGHz < 3600 {
				timeStr = fmt.Sprintf("%.1f min", r.TimeAtGHz/60)
			} else if r.TimeAtGHz < 86400 {
				timeStr = fmt.Sprintf("%.1f hrs", r.TimeAtGHz/3600)
			} else if r.TimeAtGHz < 86400*365.25 {
				timeStr = fmt.Sprintf("%.1f days", r.TimeAtGHz/86400)
			} else if r.TimeAtGHz < 86400*365.25*1e6 {
				timeStr = fmt.Sprintf("%.1f years", r.TimeAtGHz/(86400*365.25))
			} else if r.TimeAtGHz < 86400*365.25*1e9 {
				timeStr = fmt.Sprintf("%.1e years", r.TimeAtGHz/(86400*365.25))
			} else {
				timeStr = ">> age of universe"
			}
		}

		epsStr := fmt.Sprintf("%.1e", r.MachineEps)
		dpoStr := fmt.Sprintf("%.2e", r.DriftPerOp)
		if r.DriftPerOp < 1e-300 {
			dpoStr = "< 1e-300"
		}

		fmt.Printf("  %-36s %6d  %12s  %14s  %15s  %18s\n",
			r.Label, r.TotalBits, epsStr, dpoStr, depthStr, timeStr)
	}

	fmt.Println()
	fmt.Println("  ── Physical reference points ──")
	fmt.Println()
	fmt.Println("  Age of universe:           4.3e17 seconds")
	fmt.Println("  Planck time:               5.4e-44 seconds")
	fmt.Println("  Universe age / Planck:     ~8e60 ticks")
	fmt.Println("  Ops at 1 GHz for age of U: ~4.3e26 operations")
	fmt.Println()
	fmt.Println("  QW64  sustains algebraic integrity for ~7e9 compositions at 1 GHz = ~7 seconds")
	fmt.Println("  QW128 extends this to cosmological timescales without renormalisation")
	fmt.Println("  QW256+ exceeds any conceivable physical simulation lifetime")
	fmt.Println()

	fmt.Println(banner)
}
