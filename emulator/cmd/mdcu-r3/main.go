// Command mdcu-r3 runs the pre-registered round-3 R3-Q1 grid (frozen file
// doc/spikes/prereg/mdcu-round3-prereg-FROZEN-v1.md, sha256 070aeab8…c1848).
//
// DEFAULTS ARE A SMOKE RUN on out-of-grid seeds (9100…). The real grid (seeds
// 3000–3199) is refused unless -run-grid is given, and -run-grid additionally
// forces exactly seeds 3000..3199. Do not pass -run-grid until deming has
// cleared the box and the beekeeper has given the run go.
//
//	go run ./cmd/mdcu-r3 -dry-run                     print the plan, run nothing
//	nice -n 19 ./mdcu-r3                              smoke: 4 out-of-grid seeds
//	nice -n 19 ./mdcu-r3 -run-grid -out raw.jsonl     THE RUN (only after the go)
//
// Output is JSONL: one "meta" line, one "psi" line per seed, one "run" line per
// (cell, seed, arm), and a final "end" line (or "abort" with a reason).
//
// Exit codes: 0 ok · 1 error · 2 wall-clock cap · 3 pause file · 4 grid guard ·
// 5 memory/output cap · 6 NaN.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/emulator/mdcu"
)

const preregSHA = "070aeab8a54c6b749ad69faf5514c89eba48a1fbc3218de418b614c69d7c1848"

// Pre-registered cell order (§1.4): main first, then persistence, then steps.
func cells() []mdcu.R3Cell {
	return []mdcu.R3Cell{
		{Name: "main", N: 24, Steps: 250, Sigma: 0.1},
		{Name: "N12", N: 12, Steps: 250, Sigma: 0.1},
		{Name: "N48", N: 48, Steps: 250, Sigma: 0.1},
		{Name: "N48s1000", N: 48, Steps: 1000, Sigma: 0.1, Arms: []mdcu.R3Arm{mdcu.ArmFull, mdcu.ArmGram}},
	}
}

type psiRec struct {
	Type         string    `json:"type"`
	Seed         uint64    `json:"seed"`
	Stratum      string    `json:"stratum"`
	Sampler      string    `json:"sampler"`
	S1Draws      int       `json:"s1_draws"`
	S1Degenerate int       `json:"s1_degenerate"`
	S1Miss       bool      `json:"s1_miss"`
	S2S          *float64  `json:"s2_s,omitempty"`
	S2Dist       *float64  `json:"s2_dist,omitempty"`
	S2Cond       *float64  `json:"s2_cond,omitempty"`
	S2TraceS     *float64  `json:"s2_trace_s,omitempty"`
	PrimaryClass string    `json:"primary_class,omitempty"`
	PrimaryBEig  []float64 `json:"primary_b_eig,omitempty"`
	S1DefClass   string    `json:"s1def_class,omitempty"`
	Arms         []string  `json:"arms"`
}

type runRec struct {
	Type string `json:"type"`
	mdcu.R3Run
}

func loadavg() string {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(b))
}

func main() {
	dry := flag.Bool("dry-run", false, "print the plan and exit; run nothing")
	out := flag.String("out", "mdcu-r3-smoke.jsonl", "output JSONL")
	first := flag.Uint64("first-seed", 9100, "first seed (smoke default 9100; real grid forces 3000)")
	nseeds := flag.Int("seeds", 4, "number of consecutive seeds (smoke default 4; real grid forces 200)")
	runGrid := flag.Bool("run-grid", false, "REQUIRED to touch seeds 3000–3199; forces exactly 3000..3199")
	maxWall := flag.Duration("max-wall", time.Hour, "wall-clock cap (frozen: 1 h for R3-Q1)")
	pause := flag.String("pause-file", "mdcu-r3.PAUSE", "if this file exists the run stops cleanly (deming/federation pause)")
	maxRSSMB := flag.Uint64("max-rss-mb", 100, "abort if Go runtime Sys exceeds this (frozen: ≤ 100 MB)")
	maxOutMB := flag.Int64("max-out-mb", 50, "abort if output exceeds this (frozen: ≤ 50 MB)")
	flag.Parse()

	// ---- grid guard -------------------------------------------------------
	lo, hi := *first, *first+uint64(*nseeds)-1
	touches := false
	for s := lo; s <= hi; s++ {
		if mdcu.R3InGrid(s) {
			touches = true
			break
		}
	}
	if *runGrid {
		if *first != mdcu.R3GridLo || *nseeds != mdcu.R3GridHi-mdcu.R3GridLo+1 {
			if *first == 9100 && *nseeds == 4 { // flags left at smoke defaults: adopt the grid
				*first, *nseeds = mdcu.R3GridLo, mdcu.R3GridHi-mdcu.R3GridLo+1
			} else {
				fmt.Fprintln(os.Stderr, "refusing: -run-grid requires exactly seeds 3000..3199 (first-seed=3000, seeds=200)")
				os.Exit(4)
			}
		}
		if *out == "mdcu-r3-smoke.jsonl" {
			*out = "mdcu-r3-grid.jsonl"
		}
	} else if touches {
		fmt.Fprintf(os.Stderr, "refusing: seeds %d..%d touch the pre-registered R3 grid 3000–3199; use -run-grid (after deming's clearance and the beekeeper's go)\n", lo, hi)
		os.Exit(4)
	}
	seedList := make([]uint64, *nseeds)
	for i := range seedList {
		seedList[i] = *first + uint64(i)
	}

	// ---- plan -------------------------------------------------------------
	cs := cells()
	fmt.Printf("mdcu-r3  mode=%s  seeds %d..%d (%d)  prereg sha %s…\n", map[bool]string{true: "GRID", false: "smoke"}[*runGrid], seedList[0], seedList[len(seedList)-1], len(seedList), preregSHA[:8])
	for i, c := range cs {
		arms := "all applicable arms"
		if len(c.Arms) > 0 {
			arms = fmt.Sprint(c.Arms)
		}
		fmt.Printf("  %d. %-8s N=%-2d steps=%-4d sigma=%.1f  %s\n", i+1, c.Name, c.N, c.Steps, c.Sigma, arms)
	}
	if *dry {
		fmt.Println("dry-run: nothing executed")
		return
	}

	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	start := time.Now()
	var outBytes int64
	emit := func(v any) {
		if err := enc.Encode(v); err != nil {
			fmt.Fprintln(os.Stderr, "encode:", err)
			w.Flush()
			os.Exit(6) // NaN/Inf cannot be JSON-encoded
		}
	}
	finish := func(kind, why string, code int) {
		emit(map[string]any{"type": kind, "reason": why, "wall_s": time.Since(start).Seconds(), "loadavg": loadavg()})
		w.Flush()
		if st, err := f.Stat(); err == nil {
			outBytes = st.Size()
		}
		fmt.Printf("%s: %s  (wall %.1fs, out %d bytes)\n", kind, why, time.Since(start).Seconds(), outBytes)
		if code != 0 {
			os.Exit(code)
		}
	}
	// check returns a non-empty reason when the run must stop.
	check := func() (string, int) {
		if time.Since(start) > *maxWall {
			return "wall-clock cap exceeded", 2
		}
		if _, err := os.Stat(*pause); err == nil {
			return "pause file present: " + *pause, 3
		}
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		if ms.Sys>>20 > *maxRSSMB {
			return fmt.Sprintf("runtime Sys %d MB exceeds cap", ms.Sys>>20), 5
		}
		if err := w.Flush(); err == nil {
			if st, err := f.Stat(); err == nil && st.Size()>>20 > *maxOutMB {
				return "output size cap exceeded", 5
			}
		}
		return "", 0
	}

	emit(map[string]any{
		"type": "meta", "prereg_sha256": preregSHA, "mode": map[bool]string{true: "grid", false: "smoke"}[*runGrid],
		"first_seed": seedList[0], "seeds": len(seedList), "go": runtime.Version(), "gomaxprocs": runtime.GOMAXPROCS(0),
		"started": start.Format(time.RFC3339), "loadavg_start": loadavg(), "s1_cap": 20000,
	})

	// ---- ψ for every seed (cost independent of N; reused by every cell) -----
	psis := make([]*mdcu.R3Psi, len(seedList))
	for i, s := range seedList {
		if why, code := check(); why != "" {
			finish("abort", "during ψ sampling: "+why, code)
		}
		p := mdcu.BuildR3Psi(s)
		psis[i] = p
		rec := psiRec{Type: "psi", Seed: s, Stratum: p.Stratum, Sampler: p.Sampler,
			S1Draws: p.S1Draws, S1Degenerate: p.S1Degenerate, S1Miss: p.S1Miss}
		if p.S2 != nil {
			rec.S2S, rec.S2Dist, rec.S2Cond, rec.S2TraceS = &p.S2.S, &p.S2.Dist, &p.S2.Cond, &p.S2.TraceS
		}
		if p.Primary != nil {
			k, eig := mdcu.ClassifyPsi(p.Primary)
			rec.PrimaryClass, rec.PrimaryBEig = k, eig[:]
		}
		if p.S1Def != nil {
			rec.S1DefClass, _ = mdcu.ClassifyPsi(p.S1Def)
		}
		for _, a := range mdcu.R3ArmsFor(p) {
			rec.Arms = append(rec.Arms, string(a))
		}
		emit(rec)
	}
	fmt.Printf("ψ sampling done: %.1fs\n", time.Since(start).Seconds())

	// ---- cells ----------------------------------------------------------------
	nruns := 0
	for ci, c := range cs {
		t0 := time.Now()
		for _, p := range psis {
			if why, code := check(); why != "" {
				finish("abort", fmt.Sprintf("in cell %s (seed %d): %s; cells completed before this: %d", c.Name, p.Seed, why, ci), code)
			}
			for _, r := range mdcu.RunR3Seed(c, p) {
				if math.IsNaN(r.Stats.S3) || math.IsNaN(r.Stats.L3Share) {
					finish("abort", fmt.Sprintf("NaN in %s seed %d arm %s", c.Name, p.Seed, r.Arm), 6)
				}
				emit(runRec{Type: "run", R3Run: r})
				nruns++
			}
		}
		fmt.Printf("cell %-8s done in %.1fs (total %.1fs, %d runs)\n", c.Name, time.Since(t0).Seconds(), time.Since(start).Seconds(), nruns)
	}
	finish("end", fmt.Sprintf("complete: %d runs", nruns), 0)
}
