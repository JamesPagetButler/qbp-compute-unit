// Command mdcu-r4 runs the pre-registered round-4 experiment (frozen file
// doc/spikes/prereg/mdcu-round4-prereg-FROZEN-v1.md, sha256 699a8a21…587a5d).
//
// DEFAULTS ARE A SMOKE RUN on out-of-range seeds (9970…). Seeds in the frozen R4
// ranges (grid 4000–4199, S-D 4200–4399, timing probe 9960–9969) are refused unless
// the matching mode flag is given, and each mode forces exactly its own range. Do
// not pass -run-probe / -run-grid until deming has cleared the box and the run go
// has been given.
//
//	go run ./cmd/mdcu-r4 -dry-run                                print the plan, run nothing
//	GOMAXPROCS=1 nice -n 19 ./mdcu-r4                            smoke: 4 grid + 4 S-D seeds, out of range
//	GOMAXPROCS=1 nice -n 19 ./mdcu-r4 -run-probe -out probe.jsonl      the pre-grid timing probe (outcome-blind)
//	GOMAXPROCS=1 nice -n 19 ./mdcu-r4 -run-grid -probe-file probe.jsonl -out raw.jsonl   THE RUN
//
// The grid run first re-runs the bit-identity regression (seeds 3000–3009 against the
// R3 raw; frozen §1.3) and refuses to start the grid if it differs. It reads the Tier C
// decision from the probe file and refuses a probe file that is missing, was not run on
// seeds 9960–9969, or whose decision does not follow from its own timings.
//
// Output is JSONL: "meta", "plan", "regression", one "seed" line per seed (native
// comass, rescale factors, two-set convergence difference, s and d), one "run" line per
// (cell, seed, arm), "progress" lines carrying the loadavg, and a final "end" (or
// "abort" with a reason).
//
// Exit codes: 0 ok · 1 error · 2 wall-clock cap · 3 pause file · 4 seed guard ·
// 5 memory/output cap · 6 NaN · 7 regression differs from R3 · 8 bad probe file.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/emulator/mdcu"
)

const preregSHA = "699a8a21692d4f612c038425d23c2ef5ec21d142309940a453d23ae1c5587a5d"

const (
	smokeFirst   = 9970
	smokeGrid    = 4
	smokeSD      = 4
	defaultR3Raw = "../doc/spikes/raw/mdcu-r3-2026-10-26.jsonl"
)

func loadavg() string {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(b))
}

type seedRec struct {
	Type string `json:"type"`
	*mdcu.R4Seed
}

type runRec struct {
	Type string `json:"type"`
	mdcu.R3Run
}

// readProbe validates a probe file and returns the Tier C seed count it decided.
func readProbe(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var p struct {
			Type       string    `json:"type"`
			Seeds      []uint64  `json:"seeds"`
			Steps      int       `json:"steps"`
			Arm        string    `json:"arm"`
			PerRun     []float64 `json:"per_run_s"`
			MeanS      float64   `json:"mean_s"`
			TierCSeeds int       `json:"tierc_seeds"`
		}
		if json.Unmarshal(sc.Bytes(), &p) != nil || p.Type != "probe" {
			continue
		}
		want := mdcu.R4ProbeSeeds()
		if len(p.Seeds) != len(want) {
			return 0, fmt.Errorf("probe ran on %d seeds, want %d", len(p.Seeds), len(want))
		}
		for i := range want {
			if p.Seeds[i] != want[i] {
				return 0, fmt.Errorf("probe seeds differ from 9960–9969")
			}
		}
		if p.Steps != mdcu.R4StepsB || p.Arm != string(mdcu.ArmPhi) || len(p.PerRun) != len(want) {
			return 0, fmt.Errorf("probe is not K-phi at %d steps on %d seeds", mdcu.R4StepsB, len(want))
		}
		var tot float64
		for _, v := range p.PerRun {
			tot += v
		}
		mean := tot / float64(len(p.PerRun))
		if math.Abs(mean-p.MeanS) > 1e-9 {
			return 0, fmt.Errorf("probe mean %g does not match its per-run times (%g)", p.MeanS, mean)
		}
		n, _ := mdcu.R4TierCSeeds(mean)
		if n != p.TierCSeeds {
			return 0, fmt.Errorf("probe decision %d does not follow from its timings (rule gives %d)", p.TierCSeeds, n)
		}
		return n, nil
	}
	return 0, fmt.Errorf("no probe record in %s", path)
}

func main() {
	dry := flag.Bool("dry-run", false, "print the plan and exit; run nothing")
	out := flag.String("out", "", "output JSONL (default depends on the mode)")
	runProbe := flag.Bool("run-probe", false, "REQUIRED to touch seeds 9960–9969: the pre-grid timing probe")
	runGrid := flag.Bool("run-grid", false, "REQUIRED to touch seeds 4000–4399: THE RUN (forces both ranges)")
	probeFile := flag.String("probe-file", "", "probe JSONL whose Tier C decision the grid run uses (required with -run-grid)")
	r3raw := flag.String("r3-raw", defaultR3Raw, "R3 raw file for the bit-identity regression")
	regress := flag.Bool("regress", false, "run the bit-identity regression (always done with -run-grid) and exit")
	first := flag.Uint64("first-seed", smokeFirst, "smoke: first grid seed (out of every frozen range)")
	nseeds := flag.Int("seeds", smokeGrid, "smoke: number of grid seeds")
	nsd := flag.Int("sd-seeds", smokeSD, "smoke: number of S-D seeds")
	tiercSmoke := flag.Int("tierc-seeds", 0, "smoke only: Tier C on the first N seeds (0 = all)")
	maxWall := flag.Duration("max-wall", time.Hour, "wall-clock cap (frozen: 1 h)")
	pauseDefault := filepath.Join(os.Getenv("HOME"), ".federation-watcher", "box-pause", "mdcu")
	pause := flag.String("pause-file", pauseDefault, "if this file exists the run stops cleanly (deming/federation pause)")
	maxRSSMB := flag.Uint64("max-rss-mb", 100, "abort if Go runtime Sys exceeds this (frozen: ≤ 100 MB)")
	maxOutMB := flag.Int64("max-out-mb", 50, "abort if output exceeds this (frozen: ≤ 50 MB)")
	flag.Parse()

	mode := "smoke"
	switch {
	case *runGrid && *runProbe:
		fmt.Fprintln(os.Stderr, "refusing: -run-grid and -run-probe are separate steps; the probe is posted before the grid starts")
		os.Exit(4)
	case *runGrid:
		mode = "grid"
	case *runProbe:
		mode = "probe"
	}

	// ---- seed lists and guards ----------------------------------------------
	var gridSeeds, sdSeeds []uint64
	switch mode {
	case "grid":
		if *probeFile == "" {
			fmt.Fprintln(os.Stderr, "refusing: -run-grid needs -probe-file (the Tier C decision is made before the grid, from the probe)")
			os.Exit(8)
		}
		for s := uint64(mdcu.R4GridLo); s <= mdcu.R4GridHi; s++ {
			gridSeeds = append(gridSeeds, s)
		}
		for s := uint64(mdcu.R4SDLo); s <= mdcu.R4SDHi; s++ {
			sdSeeds = append(sdSeeds, s)
		}
	case "probe":
		// the probe seeds are fixed
	default:
		for i := 0; i < *nseeds; i++ {
			gridSeeds = append(gridSeeds, *first+uint64(i))
		}
		for i := 0; i < *nsd; i++ {
			sdSeeds = append(sdSeeds, *first+uint64(*nseeds)+uint64(i))
		}
		for _, s := range append(append([]uint64{}, gridSeeds...), sdSeeds...) {
			if mdcu.R4Reserved(s) {
				fmt.Fprintf(os.Stderr, "refusing: smoke seed %d lies in a frozen R4 range (grid 4000–4199, S-D 4200–4399, probe 9960–9969); use -run-grid / -run-probe after the go\n", s)
				os.Exit(4)
			}
		}
	}
	if *out == "" {
		*out = map[string]string{"grid": "mdcu-r4-grid.jsonl", "probe": "mdcu-r4-probe.jsonl", "smoke": "mdcu-r4-smoke.jsonl"}[mode]
	}

	tierCN := len(gridSeeds)
	if mode == "grid" {
		n, err := readProbe(*probeFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "refusing: bad probe file:", err)
			os.Exit(8)
		}
		tierCN = n
	} else if mode == "smoke" && *tiercSmoke > 0 && *tiercSmoke < tierCN {
		tierCN = *tiercSmoke
	}

	// ---- plan ----------------------------------------------------------------
	fmt.Printf("mdcu-r4  mode=%s  prereg sha %s…\n", strings.ToUpper(mode), preregSHA[:8])
	switch mode {
	case "probe":
		fmt.Printf("  timing probe: K-phi at %d steps on seeds %d..%d (K=%d); guard: %g × mean > %g s ⇒ Tier C on %d seeds, else %d\n",
			mdcu.R4StepsB, mdcu.R4ProbeLo, mdcu.R4ProbeHi, len(mdcu.R4ProbeSeeds()), mdcu.R4TierCFactor, mdcu.R4TierCCap, mdcu.R4TierCFallback, mdcu.R4TierCFull)
	default:
		fmt.Printf("  grid seeds %d (%d..%d) · S-D seeds %d · Tier C on the first %d grid seeds\n", len(gridSeeds), first0(gridSeeds), last0(gridSeeds), len(sdSeeds), tierCN)
		fmt.Printf("  Tier A %d steps: %v\n  Tier B %d steps: %v\n  Tier C %d steps: %v\n  S-D: K-psi@1, K-phi at %d and %d steps; scales %v × %d seeds\n",
			mdcu.R4StepsA, mdcu.R4TierAArms, mdcu.R4StepsB, mdcu.R4TierBArms, mdcu.R4StepsC, mdcu.R4TierCArms, mdcu.R4StepsA, mdcu.R4StepsB, mdcu.R4SDScales, mdcu.R4SDPerScale)
	}
	if *dry {
		fmt.Println("dry-run: nothing executed")
		return
	}
	if *regress && mode == "smoke" {
		n, err := mdcu.R4Regression(*r3raw)
		if err != nil {
			fmt.Fprintln(os.Stderr, "REGRESSION DIFFERS FROM R3:", err)
			os.Exit(7)
		}
		fmt.Printf("regression: %d records bit-identical to the R3 raw\n", n)
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
		var sz int64
		if st, err := f.Stat(); err == nil {
			sz = st.Size()
		}
		fmt.Printf("%s: %s  (wall %.1fs, out %d bytes)\n", kind, why, time.Since(start).Seconds(), sz)
		if code != 0 {
			os.Exit(code)
		}
	}
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
		"type": "meta", "prereg_sha256": preregSHA, "mode": mode, "go": runtime.Version(), "gomaxprocs": runtime.GOMAXPROCS(0),
		"started": start.Format(time.RFC3339), "loadavg_start": loadavg(), "grid_seeds": len(gridSeeds), "sd_seeds": len(sdSeeds),
		"tierc_seeds": tierCN, "kappa": mdcu.R4Kappa,
	})
	if why, code := check(); why != "" {
		finish("abort", "before start: "+why, code)
	}

	// ---- probe mode -----------------------------------------------------------
	if mode == "probe" {
		seeds := mdcu.R4ProbeSeeds()
		var per []float64
		var tot float64
		for _, s := range seeds {
			if why, code := check(); why != "" {
				finish("abort", fmt.Sprintf("during the probe (seed %d): %s", s, why), code)
			}
			m, p := mdcu.R4TimingProbe([]uint64{s})
			per = append(per, p[0])
			tot += m
		}
		mean := tot / float64(len(seeds))
		n, fb := mdcu.R4TierCSeeds(mean)
		emit(map[string]any{
			"type": "probe", "arm": string(mdcu.ArmPhi), "steps": mdcu.R4StepsB, "seeds": seeds, "per_run_s": per, "mean_s": mean,
			"factor": mdcu.R4TierCFactor, "cap_s": mdcu.R4TierCCap, "projected_tier_c_s": mdcu.R4TierCFactor * mean,
			"tierc_seeds": n, "fallback": fb, "loadavg": loadavg(),
		})
		fmt.Printf("probe: mean %.4f s per K-phi run at %d steps; projected Tier C per-run %.3f s vs cap %.1f s ⇒ Tier C on %d seeds (fallback=%v)\n",
			mean, mdcu.R4StepsB, mdcu.R4TierCFactor*mean, mdcu.R4TierCCap, n, fb)
		finish("end", fmt.Sprintf("probe complete: Tier C on %d seeds", n), 0)
		return
	}

	// ---- bit-identity regression (grid mode: mandatory before any grid seed) ---
	if mode == "grid" || *regress {
		n, err := mdcu.R4Regression(*r3raw)
		if err != nil {
			finish("abort", "bit-identity regression differs from the R3 raw: "+err.Error(), 7)
		}
		emit(map[string]any{"type": "regression", "records": n, "r3_raw": *r3raw, "ok": true})
		fmt.Printf("regression: %d records bit-identical to the R3 raw\n", n)
	}

	// ---- ψ, comass and levels for every seed ---------------------------------
	nruns := 0
	grid := make([]*mdcu.R4Seed, len(gridSeeds))
	for i, s := range gridSeeds {
		if why, code := check(); why != "" {
			finish("abort", "during ψ/comass: "+why, code)
		}
		grid[i] = mdcu.BuildR4Seed(s)
		emit(seedRec{Type: "seed", R4Seed: grid[i]})
	}
	sd := make([]*mdcu.R4Seed, len(sdSeeds))
	for i, s := range sdSeeds {
		if why, code := check(); why != "" {
			finish("abort", "during S-D ψ/comass: "+why, code)
		}
		if mode == "grid" {
			sd[i] = mdcu.BuildR4SD(s)
		} else { // smoke: out-of-range seeds cycle through the ladder
			sd[i] = mdcu.BuildR4SDAt(s, mdcu.R4SDScales[i%len(mdcu.R4SDScales)])
		}
		emit(seedRec{Type: "seed", R4Seed: sd[i]})
	}
	fmt.Printf("ψ/comass done: %.1fs\n", time.Since(start).Seconds())

	record := func(runs []mdcu.R3Run) {
		for _, r := range runs {
			if math.IsNaN(r.Stats.S3) || math.IsNaN(r.Stats.L3Share) {
				finish("abort", fmt.Sprintf("NaN in %s seed %d arm %s", r.Cell, r.Seed, r.Arm), 6)
			}
			emit(runRec{Type: "run", R3Run: r})
			nruns++
		}
	}

	// ---- grid: Tiers A/B/C, chained snapshots, seed by seed ---------------------
	for i, sdd := range grid {
		if why, code := check(); why != "" {
			finish("abort", fmt.Sprintf("in the grid (seed %d, %d seeds done): %s", sdd.Seed, i, why), code)
		}
		record(mdcu.RunR4Seed(sdd, i < tierCN))
		if (i+1)%25 == 0 {
			emit(map[string]any{"type": "progress", "phase": "grid", "seeds_done": i + 1, "wall_s": time.Since(start).Seconds(), "loadavg": loadavg()})
			fmt.Printf("grid %d/%d  %.1fs  load %s\n", i+1, len(grid), time.Since(start).Seconds(), loadavg())
		}
	}
	// ---- S-D -----------------------------------------------------------------
	for i, sdd := range sd {
		if why, code := check(); why != "" {
			finish("abort", fmt.Sprintf("in S-D (seed %d, %d seeds done): %s", sdd.Seed, i, why), code)
		}
		record(mdcu.RunR4SD(sdd))
		if (i+1)%50 == 0 {
			emit(map[string]any{"type": "progress", "phase": "sd", "seeds_done": i + 1, "wall_s": time.Since(start).Seconds(), "loadavg": loadavg()})
		}
	}
	finish("end", fmt.Sprintf("complete: %d runs", nruns), 0)
}

func first0(s []uint64) uint64 {
	if len(s) == 0 {
		return 0
	}
	return s[0]
}

func last0(s []uint64) uint64 {
	if len(s) == 0 {
		return 0
	}
	return s[len(s)-1]
}
