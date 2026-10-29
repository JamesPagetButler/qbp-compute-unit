// Command mdcu-kz runs the pre-registered Kibble-Zurek defect-scaling probe (frozen file
// doc/spikes/prereg/mdcu-kz-prereg-FROZEN-v1.md, sha256 176bdce2…a5d, as amended by the
// signed amend-1). Branch B, spike/napkin grade.
//
// DEFAULTS ARE A SMOKE RUN on free seeds (13000…). Seeds in any frozen kz range (main
// 5000–5199, null 5200–5399, reserved 5400–5599, Stage 0 5600–5629, saturation 5630–5649,
// probe 9930–9949) or in a range an earlier round used are refused unless the matching
// mode flag is given, and each mode forces exactly its own range. Do not pass -run-probe
// or -run-grid until the amendment is signed, the harness is reviewed and pushed, deming
// has cleared the box and the run go has been given.
//
//	mdcu-kz -dry-run                                          print the plan, run nothing
//	GOMAXPROCS=1 nice -n 19 mdcu-kz                           smoke: reduced, free seeds
//	GOMAXPROCS=1 nice -n 19 mdcu-kz -run-probe -out probe.jsonl                         probe
//	GOMAXPROCS=1 nice -n 19 mdcu-kz -run-grid -probe-file probe.jsonl -amend-sha <sha> -out raw.jsonl   THE RUN
//	mdcu-kz -analyze raw.jsonl                                the analysis, after the raw sha is posted
//
// The grid run reads (L, R, eta) from the probe file and refuses a probe file that is
// missing, was not run on the frozen probe seeds, or whose decision does not follow from
// its own timings. The probe is outcome-blind: timings and the descent test, no counts.
//
// Output is JSONL: "meta", "saturation", "stage0a"/"stage0b" per arm, one "run" line per
// (rep, arm, tau_Q), "progress" lines with the loadavg, and a final "end" or "abort".
//
// Exit codes: 0 ok · 1 error · 2 wall-clock cap · 3 pause file · 4 seed/mode guard ·
// 5 memory/output cap · 6 NaN · 7 parity violation · 8 bad probe file / missing input ·
// 9 saturation gate.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/emulator/mdcu"
)

const preregSHA = "176bdce2bd4544f0f0bac8523ecc04f5a6eb582d603bb2714713fa72fa676a5f"

const (
	smokeFirst = 13000
	smokeL     = 8
)

func loadavg() string {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(b))
}

func die(code int, format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(code)
}

func main() {
	dry := flag.Bool("dry-run", false, "print the plan and exit; run nothing")
	out := flag.String("out", "", "output JSONL (default depends on the mode)")
	runProbe := flag.Bool("run-probe", false, "REQUIRED to touch seeds 9930–9949: the outcome-blind probe")
	runGrid := flag.Bool("run-grid", false, "REQUIRED to touch the frozen main/null/Stage 0/saturation/tilt seeds: THE RUN")
	runCal := flag.Bool("run-calibration", false, "REQUIRED to touch the sigma* calibration seeds 5850–5889: the pre-run calibration (no dynamics)")
	calEns := flag.String("cal-ensemble", "", "calibration only: field ensemble: smooth (the ruled E3), or aligned/haar (non-ruled controls); no default")
	smokeSigma := flag.Float64("sigma", 0.20, "smoke: the null's tilt sigma")
	probeFile := flag.String("probe-file", "", "probe JSONL whose (L, R, eta) the grid run uses (required with -run-grid)")
	amendSHA := flag.String("amend-sha", "", "sha256 of the signed amend-1 file (required with -run-grid; recorded in meta)")
	analyze := flag.String("analyze", "", "analyze a finished raw JSONL and exit; touches no seed")
	first := flag.Uint64("first-seed", smokeFirst, "smoke: first seed of the seed block (must be free)")
	smokeLsz := flag.Int("L", smokeL, "smoke: lattice size")
	smokeR := flag.Int("R", 2, "smoke: replicates")
	smokeRungs := flag.Int("rungs", 2, "smoke: number of ladder rungs (first N)")
	maxWall := flag.Duration("max-wall", 2*time.Hour, "wall-clock cap (frozen: 2 h)")
	pauseDefault := filepath.Join(os.Getenv("HOME"), ".federation-watcher", "box-pause", "mdcu")
	pause := flag.String("pause-file", pauseDefault, "if this file exists the run stops cleanly (deming/federation pause)")
	maxRSSMB := flag.Uint64("max-rss-mb", 100, "abort if Go runtime Sys exceeds this (frozen: ≤ 100 MB)")
	maxOutMB := flag.Int64("max-out-mb", 50, "abort if output exceeds this (frozen: ≤ 50 MB)")
	flag.Parse()

	if *analyze != "" {
		if *runGrid || *runProbe || *runCal {
			die(4, "refusing: -analyze is its own step and touches no seed")
		}
		os.Exit(analyzeFile(*analyze))
	}

	mode := "smoke"
	nmodes := 0
	for _, m := range []struct {
		on   bool
		name string
	}{{*runGrid, "grid"}, {*runProbe, "probe"}, {*runCal, "calibration"}} {
		if m.on {
			mode = m.name
			nmodes++
		}
	}
	if nmodes > 1 {
		die(4, "refusing: -run-grid, -run-probe and -run-calibration are separate steps")
	}
	if *out == "" {
		*out = map[string]string{"grid": "mdcu-kz-grid.jsonl", "probe": "mdcu-kz-probe.jsonl", "smoke": "mdcu-kz-smoke.jsonl", "calibration": "mdcu-kz-calibration.jsonl"}[mode]
	}
	if mode == "calibration" {
		if *calEns != "smooth" && *calEns != "aligned" && *calEns != "haar" {
			die(8, "refusing: -run-calibration needs -cal-ensemble smooth|aligned|haar")
		}
		os.Exit(runCalibration(*calEns, *out, *pause))
	}

	// ---- configuration and guards ---------------------------------------------------
	var cfg mdcu.KZRunConfig
	var probeRep mdcu.KZProbeReport
	var decision mdcu.KZProbeDecision
	switch mode {
	case "grid":
		if *probeFile == "" {
			die(8, "refusing: -run-grid needs -probe-file (L and R are decided before the grid, from the probe)")
		}
		if len(*amendSHA) != 64 {
			die(8, "refusing: -run-grid needs -amend-sha, the sha256 of the signed amend-1 file")
		}
		var err error
		if probeRep, err = readProbe(*probeFile); err != nil {
			die(8, "refusing: bad probe file: %v", err)
		}
		if decision, err = mdcu.KZValidateProbe(probeRep); err != nil {
			die(8, "refusing: bad probe file: %v", err)
		}
		sigma, pinned := mdcu.KZPinnedSigma(decision.L)
		if !pinned {
			die(8, "refusing: no sigma* is pinned for L=%d (the calibration must be run, its sha pinned and amend-1 signed first)", decision.L)
		}
		cfg = mdcu.KZFrozenRunConfig(decision.L, decision.R, probeRep.Eta, sigma)
	case "probe":
		// the probe seeds are fixed
	default:
		if *smokeR < 1 || *smokeRungs < 1 || *smokeRungs > len(mdcu.KZLadder) || *smokeLsz < 4 {
			die(1, "bad smoke parameters")
		}
		var etaSeeds []uint64
		cfg, etaSeeds = smokeConfig(*first, *smokeLsz, *smokeR, *smokeRungs, *smokeSigma)
		if err := mdcu.KZCheckSmokeSeeds(append(smokeSeeds(cfg), etaSeeds...)); err != nil {
			die(4, "refusing: %v; use -run-grid / -run-probe after the go", err)
		}
		if !*dry {
			if e, ok := mdcu.KZChooseEta(cfg.L, etaSeeds); ok {
				cfg.Eta = e
			} else {
				cfg.Eta = mdcu.KZEtaLadder[len(mdcu.KZEtaLadder)-1]
			}
		}
	}

	// ---- plan -----------------------------------------------------------------------
	fmt.Printf("mdcu-kz  mode=%s  prereg sha %s…\n", strings.ToUpper(mode), preregSHA[:8])
	switch mode {
	case "probe":
		sp := mdcu.KZFrozenProbeSpec()
		fmt.Printf("  probe: eta descent at L=%d on seeds %v; step timing %d steps/arm at L=%d (%v) and L=%d (%v); detector timing on %v\n",
			sp.L1, sp.Eta, sp.Steps, sp.L1, sp.Step16, sp.L2, sp.Step24, sp.Det)
	default:
		fmt.Printf("  L=%d  R=%d  eta=%g  ladder=%v  main seeds %d..%d  null seeds %d..%d\n", cfg.L, cfg.R, cfg.Eta, cfg.Ladder,
			cfg.MainSeed0, cfg.MainSeed0+uint64(cfg.R)-1, cfg.NullSeed0, cfg.NullSeed0+uint64(cfg.R)-1)
		fmt.Printf("  stage 0: main 0a %v 0b %v; null 0a %v 0b %v; saturation %d seeds from %d\n",
			cfg.MainStage0a, cfg.MainStage0b, cfg.NullStage0a, cfg.NullStage0b, cfg.SatN, cfg.SatSeed0)
		if mode == "grid" && decision.LowPower {
			fmt.Printf("  LOW POWER: R=%d\n", decision.R)
		}
	}
	if *dry {
		fmt.Println("dry-run: nothing executed")
		return
	}

	// ---- output and guards -------------------------------------------------------------
	f, err := os.Create(*out)
	if err != nil {
		die(1, "%v", err)
	}
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	start := time.Now()
	emit := func(v any) {
		if err := enc.Encode(v); err != nil {
			w.Flush()
			die(6, "encode: %v", err) // NaN/Inf cannot be JSON-encoded
		}
	}
	var calls uint64
	check := func() error {
		if time.Since(start) > *maxWall {
			return &mdcu.KZAbortError{Code: 2, Reason: "wall-clock cap exceeded"}
		}
		if _, err := os.Stat(*pause); err == nil {
			return &mdcu.KZAbortError{Code: 3, Reason: "pause file present: " + *pause}
		}
		calls++
		if calls%256 == 1 {
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			if ms.Sys>>20 > *maxRSSMB {
				return &mdcu.KZAbortError{Code: 5, Reason: fmt.Sprintf("runtime Sys %d MB exceeds cap", ms.Sys>>20)}
			}
			if err := w.Flush(); err == nil {
				if st, err := f.Stat(); err == nil && st.Size()>>20 > *maxOutMB {
					return &mdcu.KZAbortError{Code: 5, Reason: "output size cap exceeded"}
				}
			}
		}
		return nil
	}
	finishFile := func() string {
		w.Flush()
		f.Close()
		return fileSHA(*out)
	}

	meta := map[string]any{"kind": "meta", "prereg_sha256": preregSHA, "amend_sha256": *amendSHA, "mode": mode,
		"go": runtime.Version(), "gomaxprocs": runtime.GOMAXPROCS(0), "started": start.Format(time.RFC3339),
		"loadavg_start": loadavg()}
	if mode != "probe" {
		meta["L"], meta["R"], meta["eta"], meta["ladder"] = cfg.L, cfg.R, cfg.Eta, cfg.Ladder
		meta["z2_rule"] = "amend-1: pi + edge pi/2"
		if n, ok := cfg.MkNull(0).(*mdcu.KZNullArm); ok {
			meta["null_sigma"] = n.Sigma
		}
	}
	emit(meta)
	if err := check(); err != nil {
		emit(map[string]any{"kind": "abort", "reason": "before start: " + err.Error()})
		finishFile()
		die(abortCode(err), "abort: %v", err)
	}

	// ---- probe ------------------------------------------------------------------------
	if mode == "probe" {
		rep, err := mdcu.KZRunProbe(mdcu.KZFrozenProbeSpec(), check)
		if err != nil {
			emit(map[string]any{"kind": "abort", "reason": err.Error()})
			finishFile()
			die(abortCode(err), "abort: %v", err)
		}
		emit(rep)
		d := rep.Decision
		fmt.Printf("probe: eta=%g (ok=%v)  t_s16=%.3gs t_d16=%.3gs t_s24=%.3gs t_d24=%.3gs\n", rep.Eta, rep.EtaOK,
			rep.Timing.TS16, rep.Timing.TD16, rep.Timing.TS24, rep.Timing.TD24)
		fmt.Printf("probe decision: run=%v L=%d R=%d projected wall %.0fs lowpower=%v\n", d.Run, d.L, d.R, d.WallSeconds, d.LowPower)
		emit(map[string]any{"kind": "end", "verdict": "probe complete", "wall_s": time.Since(start).Seconds(), "loadavg": loadavg()})
		fmt.Println("raw sha256:", finishFile())
		return
	}

	// ---- smoke and grid ----------------------------------------------------------------
	cfg.Check = check
	cfg.Emit = func(r map[string]any) {
		emit(r)
		if r["kind"] == "run" && r["arm"] == "null" && r["tau"] == cfg.Ladder[len(cfg.Ladder)-1] {
			emit(map[string]any{"kind": "progress", "rep": r["rep"], "reps": cfg.R, "wall_s": time.Since(start).Seconds(), "loadavg": loadavg()})
			fmt.Printf("replicate %d/%d  %.1fs  load %s\n", r["rep"].(int)+1, cfg.R, time.Since(start).Seconds(), loadavg())
		}
	}
	res, err := mdcu.KZExecute(cfg)
	if err != nil {
		finishFile()
		die(abortCode(err), "abort: %v", err)
	}
	emit(map[string]any{"kind": "wall", "wall_s": time.Since(start).Seconds(), "loadavg": loadavg()})
	fmt.Printf("verdict: %s  (%d runs, wall %.1fs)\n", res.Verdict, res.Runs, time.Since(start).Seconds())
	if res.MainStage0Reason != "" || res.NullStage0Reason != "" {
		fmt.Println("stage 0:", res.MainStage0Reason, res.NullStage0Reason)
	}
	fmt.Println("raw sha256:", finishFile())
}

func abortCode(err error) int {
	var ae *mdcu.KZAbortError
	if errors.As(err, &ae) {
		return ae.Code
	}
	return 1
}

func fileSHA(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "unreadable: " + err.Error()
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// readProbe returns the probe record of a probe JSONL file.
func readProbe(path string) (mdcu.KZProbeReport, error) {
	var rep mdcu.KZProbeReport
	f, err := os.Open(path)
	if err != nil {
		return rep, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var probe mdcu.KZProbeReport
		if json.Unmarshal(sc.Bytes(), &probe) == nil && probe.Kind == "probe" {
			return probe, nil
		}
	}
	return rep, fmt.Errorf("no probe record in %s", path)
}

// smokeConfig is a reduced, out-of-range configuration: free seeds from first, a short
// ladder, and shortened Stage 0. It exercises the code path only; its verdicts mean nothing.
func smokeConfig(first uint64, L, R, rungs int, sigma float64) (mdcu.KZRunConfig, []uint64) {
	s := first
	take := func(n int) []uint64 {
		out := make([]uint64, n)
		for i := range out {
			out[i] = s + uint64(i)
		}
		s += uint64(n)
		return out
	}
	cfg := mdcu.KZFrozenRunConfig(L, R, 0, sigma)
	// smoke tilt seeds are the null seeds themselves (free range), never the pinned 5650+ block
	cfg.MkNull = func(seed uint64) mdcu.KZArm {
		return &mdcu.KZNullArm{F: mdcu.NewKZS3FieldHaar(L, seed), Sigma: sigma, TiltSeed: seed}
	}
	cfg.Ladder = mdcu.KZLadder[:rungs]
	cfg.Stage0 = mdcu.KZStage0Params{NT: 40, Tmin: 0.05, Tmax: 8, Equil: 30, Meas: 30, TauQ: 200, RecEvery: 10, NGrid: 41}
	cfg.MainSeed0 = take(R)[0]
	cfg.NullSeed0 = take(R)[0]
	cfg.MainStage0a, cfg.MainStage0b = take(3), take(10)
	cfg.NullStage0a, cfg.NullStage0b = take(3), take(10)
	sat := take(20)
	cfg.SatSeed0, cfg.SatN = sat[0], len(sat)
	return cfg, take(2) // the second value is the eta-test seeds; the caller checks them
}

// smokeSeeds lists every seed a smoke configuration would touch.
func smokeSeeds(c mdcu.KZRunConfig) []uint64 {
	var all []uint64
	for j := 0; j < c.R; j++ {
		all = append(all, c.MainSeed0+uint64(j), c.NullSeed0+uint64(j))
	}
	all = append(all, c.MainStage0a...)
	all = append(all, c.MainStage0b...)
	all = append(all, c.NullStage0a...)
	all = append(all, c.NullStage0b...)
	for j := 0; j < c.SatN; j++ {
		all = append(all, c.SatSeed0+uint64(j))
	}
	return all
}

// analyzeFile runs the frozen analysis on a finished raw file. It refuses partial data.
func analyzeFile(path string) int {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer f.Close()
	var L, R int
	var ladder []int
	var mode string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var m struct {
			Kind   string `json:"kind"`
			Mode   string `json:"mode"`
			L, R   int
			Ladder []int `json:"ladder"`
		}
		if json.Unmarshal(sc.Bytes(), &m) == nil && m.Kind == "meta" {
			L, R, ladder, mode = m.L, m.R, m.Ladder, m.Mode
			break
		}
	}
	if R == 0 || len(ladder) == 0 {
		fmt.Fprintln(os.Stderr, "refusing: no meta record with L, R and the ladder")
		return 1
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	mainD, nullD, err := mdcu.KZLoadRunRecords(f, ladder, R)
	if err != nil {
		fmt.Fprintln(os.Stderr, "refusing:", err)
		return 1
	}
	fmt.Printf("raw file %s  (mode %s)\n", path, mode)
	fmt.Print(mdcu.KZFormatReport(mdcu.KZAnalyze(mainD, nullD), L, R, "complete"))
	return 0
}
