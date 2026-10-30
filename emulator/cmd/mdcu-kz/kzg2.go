package main

// qbp-kz-g2 round 1 (FROZEN-v1, sha256 785e0ee2…f51e28): the Stage-0b-only run mode, its
// free-seed smoke, and the -analyze-0b step. One tau_Q per process; the seed sub-block is
// derived from the tau index (FROZEN-v1 §2 Processes), never typed; L = 16 and eta = 0.1
// are the frozen round-1 values; T_c is typed (-tc, no default). The round-1 pipeline in
// main.go (smoke / probe / grid / calibration, KZExecute) is not touched.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/emulator/mdcu"
)

const (
	kzG2L     = 16  // FROZEN-v1 §2 Lattice: L = 16, no revision
	kzG2Eta   = 0.1 // FROZEN-v1 §2 Dynamics: eta = 0.1 reused, the descent test is not re-run
	kzG2Seeds = 10  // seeds per tau_Q
)

var hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

type opts0b struct {
	smoke               bool
	tau                 int
	tc                  float64
	preregSHA, rerunSHA string
	firstSeed           uint64
	firstTyped, lTyped  bool
	L                   int
	out                 string
	dry                 bool
	maxWall             time.Duration
	pause               string
	maxRSSMB            uint64
	maxOutMB            int64
}

// run0bMode validates the flags, derives the seeds, prints the plan, and (unless -dry-run)
// runs KZExecuteStage0bOnly with the pause/wall/memory/output check at every step. Exit
// codes: 0 ok · 2 wall · 3 pause · 4 seed/mode guard · 6 NaN · 8 missing input · 1 error.
func run0bMode(o opts0b) int {
	refuse := func(code int, format string, a ...any) int {
		fmt.Fprintf(os.Stderr, "refusing: "+format+"\n", a...)
		return code
	}
	// ---- guards, before anything else ----------------------------------------------
	if math.IsNaN(o.tc) {
		return refuse(8, "-tc is required and has no default (FROZEN-v1 §2: type the full-precision literal 1.0666262219986817)")
	}
	if math.IsInf(o.tc, 0) || o.tc <= 0 {
		return refuse(4, "-tc %v must be finite and positive", o.tc)
	}
	var seeds []uint64
	tauIdx := -1
	L := kzG2L
	mode := "0b"
	if o.smoke {
		mode = "smoke-0b"
		if o.tau < 10 {
			return refuse(4, "-smoke-0b needs -tau >= the record spacing 10")
		}
		if o.L < 4 {
			return refuse(4, "bad -L %d", o.L)
		}
		L = o.L
		seeds = make([]uint64, kzG2Seeds)
		for j := range seeds {
			seeds[j] = o.firstSeed + uint64(j)
		}
		if err := mdcu.KZCheckSmokeSeeds(seeds); err != nil {
			return refuse(4, "%v; the smoke of the 0b mode runs on free seeds (13000+) only", err)
		}
		if o.preregSHA != "" && !hex64.MatchString(o.preregSHA) {
			return refuse(8, "-prereg-sha must be 64 hex characters")
		}
	} else {
		if o.firstTyped {
			return refuse(4, "-run-0b derives its seeds from the tau index (FROZEN-v1 §2 Processes); -first-seed may not be typed")
		}
		if o.lTyped {
			return refuse(4, "-run-0b runs at the frozen L = %d; -L may not be typed", kzG2L)
		}
		var ok bool
		if tauIdx, ok = mdcu.KZKzG2R1TauIndex(o.tau); !ok {
			return refuse(4, "-tau %d is not in the frozen set %v", o.tau, mdcu.KZKzG2R1TauSet)
		}
		if !hex64.MatchString(o.preregSHA) {
			return refuse(8, "-run-0b needs -prereg-sha, the 64-hex sha256 of the frozen qbp-kz-g2 r1 pre-registration")
		}
		if o.rerunSHA != "" && !hex64.MatchString(o.rerunSHA) {
			return refuse(8, "-rerun-amend-sha must be the 64-hex sha256 of the dated amendment")
		}
		seeds = mdcu.KZKzG2R1Seeds(tauIdx, o.rerunSHA != "")
		if err := mdcu.KZCheckKzG2R1Seeds(seeds, tauIdx, o.rerunSHA != ""); err != nil {
			return refuse(4, "seed guard: %v", err)
		}
	}

	// ---- plan ------------------------------------------------------------------------
	block := "run block 5900-5939"
	if o.rerunSHA != "" {
		block = "RE-RUN block 5940-5979 under amendment " + o.rerunSHA[:8] + "…"
	}
	if o.smoke {
		block = "free seeds (smoke)"
	}
	fmt.Printf("mdcu-kz  mode=%s  qbp-kz-g2 r1 Stage-0b-only  prereg sha %s\n", strings.ToUpper(mode), shortSHA(o.preregSHA))
	fmt.Printf("  tau_Q=%d (index %d)  tc=%.17g  T_hi=%.6f  T_lo=%.6f  L=%d  eta=%g  hold=%d  record every 10  grid 41\n",
		o.tau, tauIdx, o.tc, 1.5*o.tc, 0.5*o.tc, L, kzG2Eta, mdcu.KZHoldSteps)
	fmt.Printf("  seeds %d..%d (%d, %s)  steps %d  out=%s  wall cap %s  pause file %s\n",
		seeds[0], seeds[len(seeds)-1], len(seeds), block, len(seeds)*(mdcu.KZHoldSteps+2*o.tau), o.out, o.maxWall, o.pause)
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
	cfg := mdcu.KZStage0bOnlyConfig{
		L: L, Eta: kzG2Eta, Tc: o.tc, Tau: o.tau, TauIndex: tauIdx, Rerun: o.rerunSHA != "", Seeds: seeds, Smoke: o.smoke,
		Stage0: mdcu.DefaultKZStage0Params(), // RecEvery 10, NGrid 41 (frozen); TauQ is replaced by Tau
		Mk:     func(seed uint64) mdcu.KZArm { return &mdcu.KZMainArm{F: mdcu.NewKZPlaneFieldHaar(L, seed)} },
		Check:  check,
		Meta:   meta,
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
	raw, err := mdcu.KZExecuteStage0bOnly(cfg)
	w.Flush()
	f.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "abort: %v\n", err)
		fmt.Println("raw sha256:", fileSHA(o.out))
		return abortCode(err)
	}
	fmt.Printf("tau_Q=%d done: %s  (wall %.1fs, load %s)\n", o.tau, mdcu.KZStage0bReason(raw), time.Since(start).Seconds(), loadavg())
	fmt.Println("raw sha256:", fileSHA(o.out))
	return 0
}

func shortSHA(s string) string {
	if len(s) >= 8 {
		return s[:8] + "…"
	}
	if s == "" {
		return "(none)"
	}
	return s
}

// codeSHA is the VCS revision stamped into the binary (go build in the repo), with
// "+dirty" when the tree was modified. Go does not stamp a build inside a git worktree
// (whose .git is a file), so it falls back to git itself in the working directory, which
// the wrapper runs from emulator/; "unknown" if neither is available.
func codeSHA() string {
	rev, dirty := "", false
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
			case "vcs.modified":
				dirty = s.Value == "true"
			}
		}
	}
	if rev == "" {
		out, err := exec.Command("git", "rev-parse", "HEAD").Output()
		if err != nil {
			return "unknown"
		}
		rev = strings.TrimSpace(string(out)) + " (git, cwd)"
		if st, err := exec.Command("git", "status", "--porcelain").Output(); err == nil && len(strings.TrimSpace(string(st))) > 0 {
			dirty = true
		}
	}
	if dirty {
		rev += "+dirty"
	}
	return rev
}

// analyze0bFiles loads the given Stage-0b-only raw files and applies the §3 rule. It
// refuses fewer or more than the four complete rungs (r5: no partial-set reading).
func analyze0bFiles(paths []string) int {
	if len(paths) != len(mdcu.KZKzG2R1TauSet) {
		fmt.Fprintf(os.Stderr, "refusing: %d file(s) given; -analyze-0b needs the four complete raw files (tau_Q %v), posted shas first — no partial-set reading (FROZEN-v1 §3, r5 harness-kill sub-clause)\n", len(paths), mdcu.KZKzG2R1TauSet)
		return 1
	}
	var rungs []mdcu.KZG2Rung
	fmt.Println("qbp-kz-g2 round 1 — raw files:")
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			fmt.Fprintln(os.Stderr, "refusing:", err)
			return 1
		}
		r, err := mdcu.KZG2LoadRung(f)
		f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "refusing: %s: %v\n", p, err)
			return 1
		}
		r.File = p
		state := "complete"
		if !r.Complete {
			state = fmt.Sprintf("INCOMPLETE (abort code %d)", r.AbortCode)
		}
		fmt.Printf("  %s  tau_Q=%d  mode=%s  %s  sha256 %s\n", p, r.Tau, r.Mode, state, fileSHA(p))
		rungs = append(rungs, r)
	}
	d, err := mdcu.KZG2Decide(rungs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println()
	fmt.Print(mdcu.KZG2FormatReport(d))
	return 0
}
