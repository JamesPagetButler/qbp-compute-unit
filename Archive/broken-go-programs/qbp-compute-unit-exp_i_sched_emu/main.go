//go:build never

// Command exp_i_sched_emu is the scheduler → emulator integration test.
//
// It exercises the full pipeline that the width-aware execution work closed:
//
//	mesh.Task  →  Scheduler.Submit  →  Task.Context()
//	          →  emu.Engine.Execute (quantized at ctx.Width)
//	          →  watchdog.Stats (drift tracking)
//	          →  mesh.CheckReallocation  →  Engine.PromoteWidth
//
// Three independent PASS/FAIL checks:
//
//  1. PIPELINE EXECUTION (Step 2): instructions execute at ctx.Width with
//     correct cycle counts and non-zero drift. No execution errors.
//
//  2. REALLOCATION ROUND-TRIP (Step 3): CheckReallocation advises an upgrade
//     to a strictly wider width; PromoteWidth charges correct stall cycles;
//     post-upgrade drift rate is lower than pre-upgrade.
//
//  3. WIDTH DISCRIMINATION (Step 4): drift increases monotonically as width
//     narrows. Validates quantize-on-entry is wired to instruction width.
//
// Step 5 reports a model calibration gap as an OBSERVATION — the scheduler's
// drift model (ε²) is ~5 orders of magnitude too optimistic vs empirical data.
// This is a known finding; recalibrating MaxCompositionDepth is a separate task.
//
// Usage:
//
//	go run cmd/exp_i_sched_emu/main.go [-cells N] [-native-width N]
package main

import (
	"flag"
	"fmt"
	"math"
	"strings"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/emu"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/mesh"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/qword"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/watchdog"
)

const calibIters = 500 // ops per width in the empirical calibration sweep

func main() {
	cells := flag.Int("cells", 2, "number of Fano cells in the scheduler")
	nativeWidth := flag.Int("native-width", 64, "native node width in bits (8/16/32/64/128)")
	flag.Parse()

	native := bitsToWidth(*nativeWidth)

	banner := strings.Repeat("=", 72)
	sep := strings.Repeat("─", 72)

	fmt.Println(banner)
	fmt.Println("QBP: SCHEDULER → EMULATOR INTEGRATION TEST")
	fmt.Println("Full pipeline: mesh.Task → context → emu.Engine → watchdog → realloc")
	fmt.Println(banner)
	fmt.Printf("\nScheduler: %d Fano cell(s), native %s\n\n", *cells, native)

	engine := emu.NewEngine(emu.DefaultPipelineConfig())
	rotation := quat.Normalize(quat.New(math.Cos(math.Pi/4), 0, 0, math.Sin(math.Pi/4)))
	engine.RF.LoadQuat(4, rotation) // rs2: rotation applied at every QMUL step

	// Measure empirical drift rates before scheduling.
	// These are ground truth for the model calibration report (Step 5).
	empirical := measureEmpiricalDrift(engine, rotation)

	// ── Step 1: Schedule tasks ───────────────────────────────────────
	fmt.Println(sep)
	fmt.Println("[Step 1] Schedule tasks — scheduler computes RequiredWidth per task")
	fmt.Println()

	sched := mesh.NewScheduler(*cells, native)

	// Task parameters chosen so the MODEL selects a specific width.
	// Note: empirical drift rates are higher than model predictions (see Step 5).
	// Step 2 checks pipeline connectivity; Step 5 reports the calibration gap.
	tasks := []*mesh.Task{
		// Model selects W8:  MaxDepth(W8, 0.5) = 0.5/6.2e-5 ≈ 8000 >= 10 ✓
		{ID: "surface", CompositionDepth: 10, DriftTolerance: 0.5},
		// Model selects W16: MaxDepth(W16, 1e-6) = 1e-6/9.3e-10 ≈ 1075 >= 1000 ✓
		// Also used in the reallocation test (Step 3) — depth chosen so upgrade
		// to W32 is triggered: MaxDepth(W16, 5e-7) = 537 < 1000 → W32 required.
		{ID: "realloc-demo", CompositionDepth: 1000, DriftTolerance: 1e-6},
		// Model selects W32: MaxDepth(W32, 1e-11) = 1e-11/1.42e-14 ≈ 704 < 1000;
		//                    MaxDepth(W64, 1e-11) ≈ 2e20 >= 1000 ✓
		// Wait — corrected: MaxDepth(W32, 1e-9) ≈ 70K >= 5000 → W32 selected.
		{ID: "physics", CompositionDepth: 5000, DriftTolerance: 1e-9},
		// Model selects W64: MaxDepth(W64, 1e-9) >> 100K ✓
		{ID: "precision", CompositionDepth: 100000, DriftTolerance: 1e-9},
	}

	fmt.Printf("  %-14s  %10s  %10s  %14s  %6s  %s\n",
		"Task", "Depth", "Tolerance", "RequiredWidth", "Nodes", "Status")
	fmt.Printf("  %-14s  %10s  %10s  %14s  %6s  %s\n",
		strings.Repeat("─", 14), strings.Repeat("─", 10), strings.Repeat("─", 10),
		strings.Repeat("─", 14), strings.Repeat("─", 6), strings.Repeat("─", 6))

	for _, t := range tasks {
		err := sched.Submit(t)
		status := "OK"
		if err != nil {
			status = "FULL"
		}
		fmt.Printf("  %-14s  %10d  %10.0e  %14s  %6d  %s\n",
			t.ID, t.CompositionDepth, t.DriftTolerance,
			t.RequiredWidth, t.NodesNeeded, status)
	}
	fmt.Println()
	fmt.Println(" ", sched.Status())
	fmt.Println()

	// ── Step 2: Pipeline execution check ────────────────────────────
	fmt.Println(sep)
	fmt.Println("[Step 2] Pipeline execution — instructions run at ctx.Width without error")
	fmt.Println("  Pass criteria: no Execute errors, correct cycle count, non-zero drift.")
	fmt.Println("  Drift vs tolerance is INFORMATIONAL here (model calibration — see Step 5).")
	fmt.Println()

	pipelinePass := true
	for _, t := range tasks {
		if !t.Allocated {
			fmt.Printf("  [SKIP] %s — not allocated\n\n", t.ID)
			continue
		}

		ctx := t.Context()
		wc := emu.WidthToCode(ctx.Width)
		inst := emu.Instruction{Op: emu.OpQMUL, Width: wc, Rd: 0, Rs1: 0, Rs2: 4}

		runIters := int64(500)
		if t.CompositionDepth < runIters {
			runIters = t.CompositionDepth
		}

		engine.RF.LoadQuat(0, quat.Normalize(quat.New(1, 2, 3, 4)))
		wd := watchdog.New()
		engine.Cycles, engine.Ops = 0, 0

		execOK := true
		for range runIters {
			if _, err := engine.Execute(inst); err != nil {
				fmt.Printf("  [FAIL] %s: Execute error: %v\n\n", t.ID, err)
				execOK = false
				pipelinePass = false
				break
			}
			wd.ObserveMul(engine.RF.ReadQuat(0))
		}
		if !execOK {
			continue
		}

		// Pipeline validity checks: cycles and non-zero drift.
		expectedCycles := runIters * int64(engine.Config.QMULCycles[ctx.Width])
		cyclesOK := engine.Cycles == expectedCycles
		driftOK := wd.CumulativeNormDrift > 0

		if !cyclesOK || !driftOK {
			pipelinePass = false
		}

		driftRate := wd.CumulativeNormDrift / float64(runIters)
		projDrift := driftRate * float64(t.CompositionDepth)
		modelPasses := projDrift <= t.DriftTolerance

		fmt.Printf("  %-14s  ctx.Width=%-14s  iters=%d\n",
			t.ID, ctx.Width, runIters)
		fmt.Printf("    Cycles:   got %d, expected %d  → %s\n",
			engine.Cycles, expectedCycles, passStr(cyclesOK))
		fmt.Printf("    Drift>0:  %.3e  → %s\n", wd.CumulativeNormDrift, passStr(driftOK))
		fmt.Printf("    Drift/op: %.3e  projected@depth: %.3e  (tol %.3e)  [model: %s]\n\n",
			driftRate, projDrift, t.DriftTolerance, passStr(modelPasses))

		// Feed runtime data into the task for reallocation use in Step 3.
		t.ActualDepth = runIters
		t.ActualDrift = wd.CumulativeNormDrift
	}

	// ── Step 3: Reallocation round-trip ─────────────────────────────
	fmt.Println(sep)
	fmt.Println("[Step 3] Reallocation — watchdog drift → CheckReallocation → PromoteWidth")
	fmt.Println("  Uses 'realloc-demo' (W16, depth=1000, tol=1e-6).")
	fmt.Println("  Model: MaxDepth(W16, 5e-7)=537 < 1000 → RequiredWidthFor recommends W32.")
	fmt.Println()

	reallocPass := true
	demo := tasks[1] // "realloc-demo"
	if !demo.Allocated {
		fmt.Println("  [SKIP] realloc-demo not allocated\n")
		reallocPass = false
	} else {
		baseRate := demo.ActualDrift / float64(demo.ActualDepth)
		fmt.Printf("  Task: %s  scheduled=%s  measured drift/op=%.3e\n\n",
			demo.ID, demo.RequiredWidth, baseRate)

		// The actual W16 drift rate is ~2.8e-5/op. After 500 ops (ActualDepth),
		// ActualDrift ≈ 500 × 2.8e-5 = 1.4e-2. That already >> 1e-6 tolerance.
		// No injection needed — the real drift naturally triggers upgrade.
		advice := mesh.CheckReallocation(demo)
		fmt.Printf("  CheckReallocation: %s\n\n", advice)

		if advice.Action != "upgrade" {
			fmt.Printf("  FAIL: expected 'upgrade', got %q\n\n", advice.Action)
			reallocPass = false
		} else {
			widerThan := advice.AdvisedWidth > advice.CurrentWidth
			fmt.Printf("  Advised: %s → %s  (wider: %s)\n",
				advice.CurrentWidth, advice.AdvisedWidth, passStr(widerThan))
			if !widerThan {
				fmt.Println("  FAIL: advised width is not wider than current\n")
				reallocPass = false
			} else {
				// Execute PromoteWidth and check stall cycles.
				stallsBefore := engine.Cycles
				err := engine.PromoteWidth(0, advice.CurrentWidth, advice.AdvisedWidth)
				if err != nil {
					fmt.Printf("  FAIL: PromoteWidth: %v\n\n", err)
					reallocPass = false
				} else {
					stalls := engine.Cycles - stallsBefore
					expectedStall := int64(int(advice.AdvisedWidth)/int(advice.CurrentWidth) - 1)
					stallOK := stalls == expectedStall
					fmt.Printf("  PromoteWidth stall: got %d cycle(s), expected %d  → %s\n",
						stalls, expectedStall, passStr(stallOK))
					if !stallOK {
						reallocPass = false
					}

					// Continue execution at the upgraded width.
					upgWC := emu.WidthToCode(advice.AdvisedWidth)
					upgInst := emu.Instruction{Op: emu.OpQMUL, Width: upgWC, Rd: 0, Rs1: 0, Rs2: 4}
					engine.RF.LoadQuat(0, quat.Normalize(quat.New(1, 2, 3, 4)))
					wdUpg := watchdog.New()
					engine.Cycles = 0
					for range int64(500) {
						engine.Execute(upgInst) //nolint:errcheck
						wdUpg.ObserveMul(engine.RF.ReadQuat(0))
					}
					upgRate := wdUpg.CumulativeNormDrift / 500.0
					driftDecreased := upgRate < baseRate
					fmt.Printf("  Post-upgrade drift/op: %.3e  (pre %.3e)  → %s\n\n",
						upgRate, baseRate, passStr(driftDecreased))
					if !driftDecreased {
						reallocPass = false
					}
				}
			}
		}
	}

	// ── Step 4: Width discrimination ────────────────────────────────
	fmt.Println(sep)
	fmt.Println("[Step 4] Width discrimination — emulator must honour ctx.Width")
	fmt.Println("  Drift must increase strictly: W64 < W32 < W16 < W8.")
	fmt.Println()

	fmt.Printf("  %-8s  %14s  %14s  %12s\n",
		"Width", "Drift/op", "MaxDepth@1e-9", "vs W64")
	fmt.Printf("  %-8s  %14s  %14s  %12s\n",
		strings.Repeat("─", 8), strings.Repeat("─", 14),
		strings.Repeat("─", 14), strings.Repeat("─", 12))

	discrimPass := true
	var prevRate float64
	prevName := ""
	driftW64 := empirical[qword.W64]

	for _, entry := range []struct {
		wc   emu.WidthCode
		name string
		w    qword.Width
	}{
		{emu.WC64, "W64", qword.W64},
		{emu.WC32, "W32", qword.W32},
		{emu.WC16, "W16", qword.W16},
		{emu.WC8, "W8", qword.W8},
	} {
		rate := empirical[entry.w]
		maxDepth := qword.MaxCompositionDepth(entry.w, 1e-9)
		ratioStr := "baseline"
		if entry.wc != emu.WC64 && driftW64 > 0 {
			ratioStr = fmt.Sprintf("%.0f×", rate/driftW64)
		}
		fmt.Printf("  %-8s  %14.3e  %14d  %12s\n",
			entry.name, rate, maxDepth, ratioStr)

		if prevRate > 0 && rate <= prevRate {
			fmt.Printf("  FAIL: %s drift (%.3e) ≤ %s drift (%.3e)\n",
				entry.name, rate, prevName, prevRate)
			discrimPass = false
		}
		prevRate = rate
		prevName = entry.name
	}
	fmt.Printf("\n  Width discrimination: %s\n\n", passStr(discrimPass))

	// ── Step 5: Model calibration observation ───────────────────────
	fmt.Println(sep)
	fmt.Println("[Step 5] Model calibration — OBSERVATION, not a pass/fail check")
	fmt.Println("  qword.MaxCompositionDepth uses drift ∝ ε².")
	fmt.Println("  Empirical data shows drift ∝ k·ε  (k ≈ 10–30 for chained rotations).")
	fmt.Println("  Ratio M/E = model_rate / empirical_rate; 1.0 = perfectly calibrated.")
	fmt.Println()
	fmt.Printf("  %-8s  %14s  %14s  %10s  %s\n",
		"Width", "Empirical/op", "Model ε²/op", "M/E ratio", "Finding")
	fmt.Printf("  %-8s  %14s  %14s  %10s  %s\n",
		strings.Repeat("─", 8), strings.Repeat("─", 14), strings.Repeat("─", 14),
		strings.Repeat("─", 10), strings.Repeat("─", 20))

	calibWidths := []struct {
		w   qword.Width
		eps float64
	}{
		{qword.W8, 1.0 / 127.0},
		{qword.W16, 1.0 / 32767.0},
		{qword.W32, 1.19e-7},
		{qword.W64, 2.22e-16},
		{qword.W128, 9.63e-35},
	}
	for _, cw := range calibWidths {
		emp := empirical[cw.w]
		modelRate := cw.eps * cw.eps
		ratioStr := "—"
		finding := "calibrated"
		if emp > 0 {
			r := modelRate / emp
			ratioStr = fmt.Sprintf("%.2e", r)
			if r < 0.01 {
				finding = "model too optimistic"
			}
		}
		fmt.Printf("  %-8s  %14.3e  %14.3e  %10s  %s\n",
			cw.w, emp, modelRate, ratioStr, finding)
	}
	fmt.Println()
	fmt.Println("  Action: recalibrate MaxCompositionDepth to use drift ∝ k·ε.")
	fmt.Println("  Use the Empirical/op column as the new per-op drift estimate.")
	fmt.Println()

	// ── Verdict ──────────────────────────────────────────────────────
	fmt.Println(banner)
	overallPass := pipelinePass && reallocPass && discrimPass
	if overallPass {
		fmt.Println("VERDICT: PASS — pipeline integration validated.")
	} else {
		fmt.Println("VERDICT: FAIL — one or more pipeline checks failed (see above).")
	}
	fmt.Println()
	fmt.Printf("  Pipeline execution  (Step 2): %s\n", passStr(pipelinePass))
	fmt.Printf("  Reallocation path   (Step 3): %s\n", passStr(reallocPass))
	fmt.Printf("  Width discrimination(Step 4): %s\n", passStr(discrimPass))
	fmt.Println()
	fmt.Println("  Model calibration   (Step 5): OBSERVATION — recalibration needed.")
	fmt.Println(banner)
}

// measureEmpiricalDrift runs calibIters QMULs at each width and returns
// the average drift-per-op for each. Used for the calibration report and
// to set task tolerances that are achievable at the measured drift rates.
func measureEmpiricalDrift(engine *emu.Engine, rotation quat.Quat) map[qword.Width]float64 {
	result := make(map[qword.Width]float64)
	q0 := quat.Normalize(quat.New(1, 2, 3, 4))

	engine.RF.LoadQuat(4, rotation)
	for _, entry := range []struct {
		wc emu.WidthCode
		w  qword.Width
	}{
		{emu.WC8, qword.W8}, {emu.WC16, qword.W16}, {emu.WC32, qword.W32},
		{emu.WC64, qword.W64}, {emu.WC128, qword.W128},
	} {
		inst := emu.Instruction{Op: emu.OpQMUL, Width: entry.wc, Rd: 0, Rs1: 0, Rs2: 4}
		engine.RF.LoadQuat(0, q0)
		wd := watchdog.New()
		for range calibIters {
			engine.Execute(inst) //nolint:errcheck
			wd.ObserveMul(engine.RF.ReadQuat(0))
		}
		result[entry.w] = wd.CumulativeNormDrift / float64(calibIters)
	}
	return result
}

func passStr(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

func bitsToWidth(bits int) qword.Width {
	switch bits {
	case 8:
		return qword.W8
	case 16:
		return qword.W16
	case 32:
		return qword.W32
	case 128:
		return qword.W128
	default:
		return qword.W64
	}
}
