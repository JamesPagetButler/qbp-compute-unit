package mdcu

// qbp-kz-g2 round 3 (FROZEN-v1 §2 harness additions 1, 2, 5, 8; sha256 132f6a42…1a623):
// equilibrium Model-A sampling at fixed T. Per seed: Haar or aligned start → burn-in →
// sampling, recording (S, e) after every rec_every-th sampling step (no step-0 record);
// the energy density e = Energy()/(3 L³) read directly from the *KZPlaneField with the
// frozen Energy(); per-seed Sokal τ_int in record units, n_eff and the under-sampling flag.
//
// KZExecuteEqOnly is the orchestrator (pattern of KZExecuteStage0bOnly): guards → meta →
// per-step Check via kzTick (burn-in included) → one "eq" record → "end" / "abort". The raw
// record carries series and per-run τ only — no pooled mean, variance, Binder value,
// histogram or barrier (§7.1 blindness); those are analyzer-only (kzeqanalyze.go).
//
// The frozen round-1 functions (KZStage0bRun, KZStage0bRunWithSeries, KZExecuteStage0bOnly,
// KZRunRung, KZInterp, KZMaxJump, KZExecute, KZStage0aRun, KZStage0bAnalyze, Energy, Step)
// are called or left alone, never edited.

import (
	"fmt"
	"math"
	"runtime"
	"time"
)

// kzStreamEq is the noise stream of the equilibrium mode (FROZEN-v1 §2 row 8), beside Stage
// 0a's 100000 and Stage 0b's 100001: no equilibrium run shares a noise stream with any ramp
// run of the same seed.
const kzStreamEq = 100002

// kzG2R3Eta is the frozen step size (FROZEN-v1 §2 Dynamics: η = 0.1 reused, no descent test).
const kzG2R3Eta = 0.1

// KZEqParams are the step counts of one run. DefaultKZEqParams are the frozen ones.
type KZEqParams struct {
	BurnIn   int // plane-steps at T, discarded
	Sampling int // plane-steps at T, recorded every RecEvery
	RecEvery int // record spacing in steps
}

// DefaultKZEqParams: 5,000 burn-in, 50,000 sampling, a record every 10 → 5,000 records.
func DefaultKZEqParams() KZEqParams { return KZEqParams{BurnIn: 5000, Sampling: 50000, RecEvery: 10} }

// KZEqRecords is the number of (S, e) records a run writes: Sampling / RecEvery (records after
// sampling steps RecEvery, 2·RecEvery, …; none at step 0).
func KZEqRecords(p KZEqParams) int {
	if p.RecEvery <= 0 {
		return 0
	}
	return p.Sampling / p.RecEvery
}

// KZEqEnergyDensity is e = E / (3 L³): the frozen chordal bond energy per bond, e ∈ [0, 3],
// zero for the aligned field (FROZEN-v1 §2 row 1).
func KZEqEnergyDensity(f *KZPlaneField) float64 {
	return f.Energy() / float64(3*f.L*f.L*f.L)
}

// kzEqStartField builds the start state of a run: "haar" = i.i.d. Haar-G₂ planes from the
// seed (hot); "aligned" = one Haar plane drawn from the seed at every site (cold;
// NewKZPlaneFieldAligned, director's ruling r6). Anything else is a programming error.
func kzEqStartField(start string, L int, seed uint64) *KZPlaneField {
	switch start {
	case "haar":
		return NewKZPlaneFieldHaar(L, seed)
	case "aligned":
		return NewKZPlaneFieldAligned(L, seed)
	}
	panic(fmt.Sprintf("kzEqStartField: unknown start kind %q", start))
}

// kzEqValidStart reports whether s is one of the two start kinds.
func kzEqValidStart(s string) bool { return s == "haar" || s == "aligned" }

// KZEqRun runs one seed at fixed T: prm.BurnIn plane-steps, then prm.Sampling plane-steps
// with (S, e) recorded after every prm.RecEvery-th sampling step — exactly
// KZEqRecords(prm) records, no step-0 record. Every step (burn-in included) polls check
// through kzTick; a non-nil error aborts as a *KZAbortError panic (code 1 if the error is not
// one), a non-finite S or e aborts with code 6. The noise is KZNoiseRNG(seed, kzStreamEq).
func KZEqRun(f *KZPlaneField, seed uint64, eta, T float64, prm KZEqParams, check func() error) (S, E []float64) {
	tick := &kzTick{KZArm: &KZMainArm{F: f}, check: check}
	rng := KZNoiseRNG(seed, kzStreamEq)
	for k := 0; k < prm.BurnIn; k++ {
		tick.Step(eta, T, rng)
	}
	n := KZEqRecords(prm)
	S, E = make([]float64, 0, n), make([]float64, 0, n)
	for k := 0; k < prm.Sampling; k++ {
		tick.Step(eta, T, rng)
		if (k+1)%prm.RecEvery == 0 {
			s := tick.Order() // NaN-screened (code 6)
			e := KZEqEnergyDensity(f)
			if math.IsNaN(e) || math.IsInf(e, 0) {
				panic(&KZAbortError{Code: 6, Reason: "non-finite energy density"})
			}
			S, E = append(S, s), append(E, e)
		}
	}
	return S, E
}

// KZSokalTauInt is the integrated autocorrelation time of x in record units by the Sokal
// window: τ(W) = ½ + Σ_{k=1}^{W} ρ(k), W = the smallest W ≥ 1 with W ≥ c·τ(W) (c = 6 in
// FROZEN-v1 §2), ρ(k) = C(k)/C(0) with C(k) = (1/(n−k)) Σ (x_i − x̄)(x_{i+k} − x̄). A series
// with no variance (or fewer than two records) has τ = ½ by convention, W = 0. If no W
// satisfies the rule before n − 1 the sum is returned at W = n − 1.
func KZSokalTauInt(x []float64, c float64) (tau float64, W int) {
	n := len(x)
	if n < 2 {
		return 0.5, 0
	}
	m := 0.0
	for _, v := range x {
		m += v
	}
	m /= float64(n)
	d := make([]float64, n)
	c0 := 0.0
	for i, v := range x {
		d[i] = v - m
		c0 += d[i] * d[i]
	}
	c0 /= float64(n)
	if c0 == 0 || math.IsNaN(c0) {
		return 0.5, 0
	}
	tau = 0.5
	for W = 1; W < n; W++ {
		s := 0.0
		for i := 0; i+W < n; i++ {
			s += d[i] * d[i+W]
		}
		tau += s / float64(n-W) / c0
		if float64(W) >= c*tau {
			return tau, W
		}
	}
	return tau, n - 1
}

// KZEqUnderN is the n_eff floor of the under-sampling flag (FROZEN-v1 §2: n_eff < 20).
const KZEqUnderN = 20

// KZEqNEff is n_eff = n / (2·τ_rec) and the flag n_eff < 20. HARNESS: a τ_rec below ½ (an
// anticorrelated or degenerate series) is floored at ½, the white-noise value, so n_eff never
// exceeds the record count; the recorded tau_int_rec itself is left as computed.
func KZEqNEff(n int, tauRec float64) (neff float64, undersampled bool) {
	if tauRec < 0.5 || math.IsNaN(tauRec) {
		tauRec = 0.5
	}
	neff = float64(n) / (2 * tauRec)
	return neff, neff < KZEqUnderN
}

// KZEqSeedResult is one run's part of the "eq" record (FROZEN-v1 §2 row 5).
type KZEqSeedResult struct {
	Seed         uint64    `json:"seed"`
	Start        string    `json:"start"` // "haar" | "aligned"
	S            []float64 `json:"s_series"`
	E            []float64 `json:"e_series"`
	TauIntRec    float64   `json:"tau_int_rec"` // record units (τ_steps = RecEvery · tau_int_rec)
	SokalW       int       `json:"sokal_w"`
	NEff         float64   `json:"n_eff"` // records / (2 · tau_int_rec)
	Undersampled bool      `json:"undersampled"`
}

// KZEqOnlyConfig is everything one equilibrium process needs. Seeds are passed in and checked
// against (l, t) here as well as in the driver (FROZEN-v1 §5 seed guard).
type KZEqOnlyConfig struct {
	L      int
	LIndex int     // l ∈ {0: 8, 1: 12, 2: 16}; ignored in a smoke
	TIndex int     // t ∈ 0..9 into KZKzG2R3TTable
	T      float64 // must equal KZKzG2R3TTable[TIndex] in a run
	Eta    float64 // the frozen 0.1 in a run
	Rerun  bool    // the reserved re-run sub-block (dated amendment only)
	Smoke  bool    // a free-seed smoke of the mode: KZSeedFree seeds only, any lengths
	Seeds  []uint64
	Starts []string // per seed, "haar" or "aligned"; a run requires haar, haar, aligned
	Prm    KZEqParams
	Check  func() error // called at the start and at every step (burn-in included)
	Emit   func(rec map[string]any)
	Meta   map[string]any // driver-supplied meta fields (prereg_sha, binary sha, load …)
}

// KZEqRaw is what KZExecuteEqOnly returns besides emitting it.
type KZEqRaw struct {
	Runs      []KZEqSeedResult
	Steps     int
	WallS     float64
	MsPerStep float64
	PeakSysMB float64
}

// KZExecuteEqOnly runs one equilibrium process: meta, then the "eq" record (per-seed series
// and τ_int), then "end" with steps, wall, ms_per_step and the MemStats.Sys high-water. Any
// abort emits one "abort" record (with the steps completed, so the runner can tell a kill
// from a no-run — §5) and returns the *KZAbortError. The guards run before any arm is built
// and before any record is written (code 4).
func KZExecuteEqOnly(cfg KZEqOnlyConfig) (raw KZEqRaw, err error) {
	// 1. guards
	guard := func(format string, a ...any) (KZEqRaw, error) {
		return raw, &KZAbortError{Code: 4, Reason: fmt.Sprintf(format, a...)}
	}
	switch {
	case cfg.L < 2:
		return guard("bad L %d", cfg.L)
	case math.IsNaN(cfg.T) || math.IsInf(cfg.T, 0) || cfg.T <= 0:
		return guard("T %v must be finite and positive", cfg.T)
	case math.IsNaN(cfg.Eta) || math.IsInf(cfg.Eta, 0) || cfg.Eta <= 0:
		return guard("eta %v must be finite and positive", cfg.Eta)
	case cfg.Prm.RecEvery <= 0 || cfg.Prm.BurnIn < 0 || cfg.Prm.Sampling < cfg.Prm.RecEvery:
		return guard("step counts %+v: rec_every > 0, burn_in >= 0, sampling >= rec_every", cfg.Prm)
	case len(cfg.Seeds) == 0:
		return guard("no seeds")
	case len(cfg.Starts) != len(cfg.Seeds):
		return guard("%d start kinds for %d seeds", len(cfg.Starts), len(cfg.Seeds))
	}
	for _, s := range cfg.Starts {
		if !kzEqValidStart(s) {
			return guard("unknown start kind %q (haar | aligned)", s)
		}
	}
	mode := "eq"
	if cfg.Smoke {
		mode = "smoke-eq"
		if e := KZCheckSmokeSeeds(cfg.Seeds); e != nil {
			return guard("smoke-eq: %v", e)
		}
	} else {
		l, ok := KZKzG2R3LIndex(cfg.L)
		if !ok || l != cfg.LIndex {
			return guard("L %d (index %d) is not in the frozen set %v", cfg.L, cfg.LIndex, KZKzG2R3LSet)
		}
		if cfg.TIndex < 0 || cfg.TIndex >= len(KZKzG2R3TTable) {
			return guard("t index %d is not in 0..%d", cfg.TIndex, len(KZKzG2R3TTable)-1)
		}
		if !KZKzG2R3Runs(cfg.LIndex, cfg.TIndex) {
			return guard("L = %d does not run t = %d (FROZEN-v1 §2: L = 8 runs the seven coarse points only)", cfg.L, cfg.TIndex)
		}
		if cfg.T != KZKzG2R3TTable[cfg.TIndex] {
			return guard("T %v is not the frozen table's T[%d] = %v", cfg.T, cfg.TIndex, KZKzG2R3TTable[cfg.TIndex])
		}
		if cfg.Eta != kzG2R3Eta {
			return guard("eta %v is not the frozen %v", cfg.Eta, kzG2R3Eta)
		}
		if cfg.Prm != DefaultKZEqParams() {
			return guard("step counts %+v are not the frozen %+v", cfg.Prm, DefaultKZEqParams())
		}
		want := KZKzG2R3Starts()
		for j := range want {
			if j >= len(cfg.Starts) || cfg.Starts[j] != want[j] {
				return guard("start kinds %v are not the frozen %v", cfg.Starts, want)
			}
		}
		if e := KZCheckKzG2R3Seeds(cfg.Seeds, cfg.LIndex, cfg.TIndex, cfg.Rerun); e != nil {
			return guard("seed guard: %v", e)
		}
	}

	emit := cfg.Emit
	if emit == nil {
		emit = func(map[string]any) {}
	}
	start := time.Now()
	steps := 0
	peak := kzSysMB()
	sample := func() {
		if m := kzSysMB(); m > peak {
			peak = m
		}
	}
	defer func() {
		if r := recover(); r != nil {
			ae, ok := r.(*KZAbortError)
			if !ok {
				panic(r)
			}
			sample()
			emit(map[string]any{"kind": "abort", "code": ae.Code, "reason": ae.Reason, "steps": steps,
				"wall_s": time.Since(start).Seconds(), "peak_sys_mb": peak})
			err = ae
		}
	}()

	// 2. meta
	nrec := KZEqRecords(cfg.Prm)
	meta := map[string]any{"kind": "meta", "mode": mode, "L": cfg.L, "l_index": cfg.LIndex, "T": cfg.T, "t_index": cfg.TIndex,
		"eta": cfg.Eta, "seeds": cfg.Seeds, "starts": cfg.Starts, "rerun": cfg.Rerun,
		"burn_in": cfg.Prm.BurnIn, "sampling": cfg.Prm.Sampling, "rec_every": cfg.Prm.RecEvery, "records": nrec,
		"noise_stream": int(kzStreamEq), "started": start.Format(time.RFC3339)}
	for k, v := range cfg.Meta {
		meta[k] = v
	}
	emit(meta)

	// 3. start check (pause file / wall), then the runs; every step polls Check
	if cfg.Check != nil {
		if e := cfg.Check(); e != nil {
			if ae, ok := e.(*KZAbortError); ok {
				panic(ae)
			}
			panic(&KZAbortError{Code: 1, Reason: e.Error()})
		}
	}
	check := func() error {
		if cfg.Check != nil {
			if e := cfg.Check(); e != nil {
				return e
			}
		}
		steps++
		if steps%256 == 0 {
			sample()
		}
		return nil
	}
	raw.Runs = make([]KZEqSeedResult, 0, len(cfg.Seeds))
	for j, seed := range cfg.Seeds {
		f := kzEqStartField(cfg.Starts[j], cfg.L, seed)
		S, E := KZEqRun(f, seed, cfg.Eta, cfg.T, cfg.Prm, check)
		tau, W := KZSokalTauInt(E, 6)
		neff, under := KZEqNEff(len(E), tau)
		raw.Runs = append(raw.Runs, KZEqSeedResult{Seed: seed, Start: cfg.Starts[j], S: S, E: E,
			TauIntRec: tau, SokalW: W, NEff: neff, Undersampled: under})
		sample()
	}

	// 4. the records: series and per-run τ only (no pooled statistic — §7.1)
	emit(map[string]any{"kind": "eq", "L": cfg.L, "l_index": cfg.LIndex, "T": cfg.T, "t_index": cfg.TIndex, "eta": cfg.Eta,
		"seeds": cfg.Seeds, "starts": cfg.Starts, "burn_in": cfg.Prm.BurnIn, "sampling": cfg.Prm.Sampling,
		"rec_every": cfg.Prm.RecEvery, "records": nrec, "noise_stream": int(kzStreamEq), "runs": raw.Runs})
	sample()
	raw.Steps = steps
	raw.WallS = time.Since(start).Seconds()
	if steps > 0 {
		raw.MsPerStep = 1000 * raw.WallS / float64(steps)
	}
	raw.PeakSysMB = peak
	emit(map[string]any{"kind": "end", "verdict": mode + " complete", "seeds_run": len(cfg.Seeds), "steps": steps,
		"wall_s": raw.WallS, "ms_per_step": raw.MsPerStep, "peak_sys_mb": peak})
	return raw, nil
}

// kzSysMB is the Go runtime's Sys (bytes obtained from the OS) in MB.
func kzSysMB() float64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return float64(ms.Sys) / (1 << 20)
}
