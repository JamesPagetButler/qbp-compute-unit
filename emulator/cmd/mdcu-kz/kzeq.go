package main

// qbp-kz-g2 round 3 (FROZEN-v1, sha256 132f6a42…1a623): the equilibrium run mode -run-eq, its
// free-seed smoke -smoke-eq, and the -analyze-eq step. One (L, t) per process; the 3-seed
// sub-block is derived from (l, t) (FROZEN-v1 §2 Seeds), never typed; T is the frozen table's
// T[t]; L = 8 refuses t ∈ {3, 5, 7}; -tc is not an eq flag. The round-1 pipeline (main.go) and
// the qbp-kz-g2 round-1 mode (kzg2.go) are not touched.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/emulator/mdcu"
)

// kzG2R3FrozenSHA is FROZEN-v1's sha256, for the plan's consistency note only: the driver
// records whatever -prereg-sha it is given (the wrapper hashes the frozen file) and refuses
// only a malformed one, as round 1 did.
const kzG2R3FrozenSHA = "132f6a4251ab3ff9ecb8f9b56c607ba1a1dd7e7ed4256fd4fb6f952c0fe1a623"

// smoke lengths (FROZEN-v1 §5 smoke ruling): 500 burn-in + 2,000 sampling, one hot + one cold seed
var kzEqSmokePrm = mdcu.KZEqParams{BurnIn: 500, Sampling: 2000, RecEvery: 10}

const kzEqSmokeT = 5 // smoke default T index (T = 1.15, the window centre); -t may pick another

type optsEq struct {
	smoke               bool
	L                   int
	lTyped              bool
	t                   int
	tTyped              bool
	tcTyped             bool
	preregSHA, rerunSHA string
	firstSeed           uint64
	firstTyped          bool
	out                 string
	dry                 bool
	maxWall             time.Duration
	pause               string
	maxRSSMB            uint64
	maxOutMB            int64
}

// runEqMode validates the flags, derives the seeds, prints the plan, and (unless -dry-run)
// runs KZExecuteEqOnly with the pause/wall/memory/output check at every step. Exit codes:
// 0 ok · 2 wall · 3 pause · 4 seed/mode guard · 5 memory/output · 6 NaN · 8 missing input · 1 error.
// The end line prints wall, steps, peak Sys, exit code and the raw sha only (§7.1).
func runEqMode(o optsEq) int {
	refuse := func(code int, format string, a ...any) int {
		fmt.Fprintf(os.Stderr, "refusing: "+format+"\n", a...)
		return code
	}
	// ---- guards, before anything else ----------------------------------------------
	if o.tcTyped {
		return refuse(4, "-tc is not an eq-mode flag (T comes from the frozen table via -t; T_c is not re-derived)")
	}
	if o.rerunSHA != "" && !hex64.MatchString(o.rerunSHA) {
		return refuse(8, "-rerun-amend-sha must be the 64-hex sha256 of the dated amendment")
	}
	var seeds []uint64
	var starts []string
	lIdx := -1
	mode := "eq"
	prm := mdcu.DefaultKZEqParams()
	tIdx := o.t
	if o.smoke {
		mode = "smoke-eq"
		prm = kzEqSmokePrm
		if o.L < 4 {
			return refuse(4, "bad -L %d", o.L)
		}
		if !o.tTyped {
			tIdx = kzEqSmokeT
		}
		if tIdx < 0 || tIdx >= len(mdcu.KZKzG2R3TTable) {
			return refuse(4, "-t %d is not in 0..%d", tIdx, len(mdcu.KZKzG2R3TTable)-1)
		}
		seeds = []uint64{o.firstSeed, o.firstSeed + 1}
		starts = []string{"haar", "aligned"}
		if err := mdcu.KZCheckSmokeSeeds(seeds); err != nil {
			return refuse(4, "%v; the smoke of the eq mode runs on free seeds (13000+) only", err)
		}
		if o.preregSHA != "" && !hex64.MatchString(o.preregSHA) {
			return refuse(8, "-prereg-sha must be 64 hex characters")
		}
	} else {
		if o.firstTyped {
			return refuse(4, "-run-eq derives its seeds from (l, t) (FROZEN-v1 §2 Seeds); -first-seed may not be typed")
		}
		if !o.lTyped {
			return refuse(4, "-run-eq needs -L <8|12|16> (typed, no default)")
		}
		if !o.tTyped {
			return refuse(4, "-run-eq needs -t <0..9> (typed, no default)")
		}
		var ok bool
		if lIdx, ok = mdcu.KZKzG2R3LIndex(o.L); !ok {
			return refuse(4, "-L %d is not in the frozen set %v", o.L, mdcu.KZKzG2R3LSet)
		}
		if tIdx < 0 || tIdx >= len(mdcu.KZKzG2R3TTable) {
			return refuse(4, "-t %d is not in 0..%d", tIdx, len(mdcu.KZKzG2R3TTable)-1)
		}
		if !mdcu.KZKzG2R3Runs(lIdx, tIdx) {
			return refuse(4, "L = %d does not run t = %d (FROZEN-v1 §2: L = 8 runs the seven coarse points t ∈ {0,1,2,4,6,8,9} only)", o.L, tIdx)
		}
		if !hex64.MatchString(o.preregSHA) {
			return refuse(8, "-run-eq needs -prereg-sha, the 64-hex sha256 of the frozen qbp-kz-g2 r3 pre-registration")
		}
		seeds = mdcu.KZKzG2R3Seeds(lIdx, tIdx, o.rerunSHA != "")
		starts = mdcu.KZKzG2R3Starts()
		if err := mdcu.KZCheckKzG2R3Seeds(seeds, lIdx, tIdx, o.rerunSHA != ""); err != nil {
			return refuse(4, "seed guard: %v", err)
		}
	}
	T := mdcu.KZKzG2R3TTable[tIdx]
	if o.out == "" {
		o.out = "mdcu-kz-smoke-eq.jsonl"
		if !o.smoke {
			o.out = fmt.Sprintf("qbp-kz-g2-r3-eq-L%d-t%02d.jsonl", o.L, tIdx)
		}
	}

	// ---- plan ------------------------------------------------------------------------
	block := "run block 6000-6119"
	if o.rerunSHA != "" {
		block = "RE-RUN block 6120-6239 under amendment " + o.rerunSHA[:8] + "…"
	}
	if o.smoke {
		block = "free seeds (smoke)"
	}
	seedList := make([]string, len(seeds))
	for i, s := range seeds {
		seedList[i] = fmt.Sprint(s)
	}
	steps := len(seeds) * (prm.BurnIn + prm.Sampling)
	fmt.Printf("mdcu-kz  mode=%s  qbp-kz-g2 r3 equilibrium  prereg sha %s\n", strings.ToUpper(mode), shortSHA(o.preregSHA))
	fmt.Printf("  L=%d (l=%d)  t=%d  T=%g  eta=%g  burn-in %d  sampling %d  record every %d (%d records/seed)  noise stream %d\n",
		o.L, lIdx, tIdx, T, 0.1, prm.BurnIn, prm.Sampling, prm.RecEvery, mdcu.KZEqRecords(prm), 100002)
	fmt.Printf("  seeds %s (%d, %s)  starts %s  steps %d  out=%s  wall cap %s  pause file %s  caps Sys %d MB out %d MB\n",
		strings.Join(seedList, ","), len(seeds), block, strings.Join(starts, ","), steps, o.out, o.maxWall, o.pause, o.maxRSSMB, o.maxOutMB)
	if !o.smoke && o.preregSHA != kzG2R3FrozenSHA {
		fmt.Printf("  note: -prereg-sha is not the FROZEN-v1 sha this binary knows (%s…); it is recorded as given\n", kzG2R3FrozenSHA[:8])
	}
	if o.dry {
		fmt.Println("dry-run: nothing executed")
		return 0
	}

	// ---- output and the step check -------------------------------------------------------
	f, err := os.Create(o.out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	start := time.Now()
	exit := 0
	emit := func(v any) {
		if err := enc.Encode(v); err != nil {
			w.Flush()
			fmt.Fprintln(os.Stderr, "encode:", err) // NaN/Inf cannot be JSON-encoded
			os.Exit(6)
		}
	}
	var calls uint64
	check := func() error {
		if time.Since(start) > o.maxWall {
			return &mdcu.KZAbortError{Code: 2, Reason: "wall-clock cap exceeded"}
		}
		if _, err := os.Stat(o.pause); err == nil {
			return &mdcu.KZAbortError{Code: 3, Reason: "pause file present: " + o.pause}
		}
		calls++
		if calls%256 == 1 {
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			if ms.Sys>>20 > o.maxRSSMB {
				return &mdcu.KZAbortError{Code: 5, Reason: fmt.Sprintf("runtime Sys %d MB exceeds cap", ms.Sys>>20)}
			}
			if err := w.Flush(); err == nil {
				if st, err := f.Stat(); err == nil && st.Size()>>20 > o.maxOutMB {
					return &mdcu.KZAbortError{Code: 5, Reason: "output size cap exceeded"}
				}
			}
		}
		return nil
	}
	_, pauseAtStart := os.Stat(o.pause)
	exe, _ := os.Executable()
	meta := map[string]any{
		"prereg_sha": o.preregSHA, "rerun_amend_sha": o.rerunSHA, "round1_prereg_sha256": preregSHA,
		"binary_sha256": fileSHA(exe), "code_sha": codeSHA(),
		"go": runtime.Version(), "gomaxprocs": runtime.GOMAXPROCS(0),
		"loadavg_start": loadavg(), "pause_file": o.pause, "pause_present_at_start": pauseAtStart == nil,
		"max_wall_s": o.maxWall.Seconds(), "max_rss_mb": o.maxRSSMB, "max_out_mb": o.maxOutMB,
	}
	cfg := mdcu.KZEqOnlyConfig{
		L: o.L, LIndex: lIdx, TIndex: tIdx, T: T, Eta: 0.1, Rerun: o.rerunSHA != "", Smoke: o.smoke,
		Seeds: seeds, Starts: starts, Prm: prm, Check: check, Meta: meta,
		Emit: func(r map[string]any) {
			if r["kind"] == "end" || r["kind"] == "abort" {
				r["loadavg_end"] = loadavg()
				if r["kind"] == "abort" {
					if c, ok := r["code"].(int); ok {
						exit = c
					}
				}
				r["exit_code"] = exit
			}
			emit(r)
		},
	}
	raw, err := mdcu.KZExecuteEqOnly(cfg)
	w.Flush()
	f.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "abort: %v\n", err)
		fmt.Printf("eq L=%d t=%d ABORT: wall %.1fs  exit %d\n", o.L, tIdx, time.Since(start).Seconds(), abortCode(err))
		fmt.Println("raw sha256:", fileSHA(o.out))
		return abortCode(err)
	}
	// §7.1 blindness: no mean, variance, Binder value or barrier on the terminal
	fmt.Printf("eq L=%d t=%d done: wall %.1fs  steps %d  ms/step %.3f  peak Sys %.1f MB  exit 0\n",
		o.L, tIdx, raw.WallS, raw.Steps, raw.MsPerStep, raw.PeakSysMB)
	fmt.Println("raw sha256:", fileSHA(o.out))
	return 0
}

// analyzeEqFiles loads the given equilibrium raw files and applies the §3 rule. It refuses
// anything but the complete set of 27 (no partial-set reading); smoke-eq files are refused by
// the decision. With plotOut non-empty the plot data JSON is written there.
func analyzeEqFiles(paths []string, plotOut string) int {
	if len(paths) != 27 {
		fmt.Fprintf(os.Stderr, "refusing: %d file(s) given; -analyze-eq needs the complete set of 27 raw files (one per (L, t) of FROZEN-v1 §2), all 27 shas posted first — no partial-set reading (§3 harness-kill row)\n", len(paths))
		return 1
	}
	var pts []mdcu.KZEqPoint
	fmt.Println("qbp-kz-g2 round 3 — raw files:")
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			fmt.Fprintln(os.Stderr, "refusing:", err)
			return 1
		}
		pt, err := mdcu.KZEqLoad(f)
		f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "refusing: %s: %v\n", p, err)
			return 1
		}
		pt.File = p
		state := "complete"
		if !pt.Complete {
			state = fmt.Sprintf("INCOMPLETE (abort code %d, exit %d, end %v)", pt.AbortCode, pt.ExitCode, pt.HaveEnd)
		}
		fmt.Printf("  %s  L=%d t=%d T=%g mode=%s  %s  sha256 %s\n", p, pt.L, pt.TIndex, pt.T, pt.Mode, state, fileSHA(p))
		pts = append(pts, pt)
	}
	d, err := mdcu.KZEqDecide(pts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println()
	fmt.Print(mdcu.KZEqFormatReport(d))
	if plotOut != "" {
		js, err := mdcu.KZEqPlotJSON(d)
		if err != nil {
			fmt.Fprintln(os.Stderr, "plot json:", err)
			return 1
		}
		if err := os.WriteFile(plotOut, js, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "plot json:", err)
			return 1
		}
		fmt.Printf("\nplot data written to %s\n", plotOut)
	}
	return 0
}
