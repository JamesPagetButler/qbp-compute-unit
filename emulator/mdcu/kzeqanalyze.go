package mdcu

// qbp-kz-g2 round 3 analyzer (FROZEN-v1 §3, sha256 132f6a42…1a623): per L the common 60-bin
// histogram grid over the pooled e, per (L, T) the 3-bin-smoothed histogram, the resolved
// two-peak test (3σ Poisson floor on smoothed counts), the barrier ΔF_L(T) = ln(h_lower /
// h_valley) with ΔF*(L) and T*(L) as the primary, the leave-one-seed-out jackknife σ_ΔF, the
// hot/cold start-dependence D_e / D_S (blocked SE at 250 records, hot blocks never straddling
// the two seeds), per-run τ_int / n_eff / undersampled, the rule (a) → (b) → (c) with the
// symmetric under-sampling gates, the start-dependence requirement for (b), the sub-labels
// (sampling > metastability > plain), and the non-gating secondaries (V_L(T), V_min, V*,
// equal-weight coexistence T, Δe, per-seed max step, phase-locking count, S histograms).
// It reads the raw files written by KZExecuteEqOnly and refuses anything but the complete
// 27-file set (§3 "Analyzer precision", r6). The round-1 analyzers are untouched.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
)

// Frozen §3 numbers.
const (
	KZEqBins       = 60  // equal-width bins over the pooled min–max of e at each L
	KZEqSigmaFloor = 3.0 // resolved iff n_lower − n_valley > 3·√(n_lower + n_valley) (smoothed counts)
	KZEqEpsFloor   = 0.2 // ε(L1, L2) = max(0.2, 2·√(σ*(L1)² + σ*(L2)²))
	KZEqBlock      = 250 // records per block for the start-dependence SEs
	KZEqDThreshold = 3.0 // start-dependent iff max(D_e, D_S) > 3
)

// KZEqDeltaF0 is the (a) threshold ΔF₀ = ln 2 (valley at most half the lower peak).
var KZEqDeltaF0 = math.Log(2)

// ---- histogram mechanics -------------------------------------------------------------------

// KZEqBinning is the common grid at one L: KZEqBins equal bins over [Lo, Hi].
type KZEqBinning struct {
	Lo, Hi float64
	N      int
}

// KZEqBinningOf sets 60 equal-width bins over the pooled min–max of all given series. A
// degenerate pool (all values equal, or empty) gets a unit-width grid around the value.
func KZEqBinningOf(series ...[]float64) KZEqBinning {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, s := range series {
		for _, v := range s {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
	}
	if math.IsInf(lo, 0) || math.IsInf(hi, 0) {
		lo, hi = 0, 1
	}
	if hi <= lo {
		hi = lo + 1
	}
	return KZEqBinning{Lo: lo, Hi: hi, N: KZEqBins}
}

// Width is the bin width.
func (b KZEqBinning) Width() float64 { return (b.Hi - b.Lo) / float64(b.N) }

// Index is the bin of x, clamped to 0..N−1 (x = Hi falls in the last bin).
func (b KZEqBinning) Index(x float64) int {
	i := int(math.Floor((x - b.Lo) / b.Width()))
	if i < 0 {
		return 0
	}
	if i >= b.N {
		return b.N - 1
	}
	return i
}

// Centre is the e value at the centre of bin i.
func (b KZEqBinning) Centre(i int) float64 { return b.Lo + (float64(i)+0.5)*b.Width() }

// KZEqHistogram counts the pooled series on the grid.
func KZEqHistogram(b KZEqBinning, series ...[]float64) []float64 {
	h := make([]float64, b.N)
	for _, s := range series {
		for _, v := range s {
			h[b.Index(v)]++
		}
	}
	return h
}

// KZEqSmooth3 is the 3-bin moving average; the two edge bins average the available
// neighbours (truncated window, director's ruling r6).
func KZEqSmooth3(h []float64) []float64 {
	n := len(h)
	s := make([]float64, n)
	for i := range h {
		sum, cnt := h[i], 1.0
		if i > 0 {
			sum += h[i-1]
			cnt++
		}
		if i+1 < n {
			sum += h[i+1]
			cnt++
		}
		s[i] = sum / cnt
	}
	return s
}

// KZEqBarrier is the two-peak reading of one smoothed histogram.
type KZEqBarrier struct {
	Peaks     []int   // every strict local maximum (a bin strictly above both neighbours)
	Candidate bool    // some pair of maxima ≥ 1 bin apart exists
	Resolved  bool    // the chosen pair passes the 3σ floor; only then is the histogram bimodal
	LowerPeak int     // the chosen pair by position: left (lower e) …
	UpperPeak int     // … and right (higher e) bin
	Valley    int     // the minimum between them (first on a tie)
	HLower    float64 // the smaller of the two peak heights (smoothed counts)
	HValley   float64 // the valley height (0 → 0.5 before the log, as recorded here)
	Depth     float64 // (n_lower − n_valley) / √(n_lower + n_valley), the floor statistic
	DeltaF    float64 // ln(HLower / HValley) if Resolved, else 0 by definition
}

// KZEqBarrierOf applies §3 to a smoothed histogram: among all pairs of strict local maxima
// with at least one bin between them, the pair with the largest resolved barrier (director's
// ruling r6); unresolved → ΔF = 0 (the closest-to-resolving pair is still reported).
func KZEqBarrierOf(hs []float64) KZEqBarrier {
	var b KZEqBarrier
	n := len(hs)
	for i := 1; i+1 < n; i++ {
		if hs[i] > hs[i-1] && hs[i] > hs[i+1] {
			b.Peaks = append(b.Peaks, i)
		}
	}
	bestDepth := math.Inf(-1)
	for a := 0; a < len(b.Peaks); a++ {
		for c := a + 1; c < len(b.Peaks); c++ {
			p, q := b.Peaks[a], b.Peaks[c]
			if q-p < 2 {
				continue
			}
			b.Candidate = true
			v := p + 1
			for i := p + 1; i < q; i++ {
				if hs[i] < hs[v] {
					v = i
				}
			}
			hl, hv := math.Min(hs[p], hs[q]), hs[v]
			depth := 0.0
			if hl+hv > 0 {
				depth = (hl - hv) / math.Sqrt(hl+hv)
			}
			resolved := hl-hv > KZEqSigmaFloor*math.Sqrt(hl+hv)
			dF := 0.0
			hvLog := hv
			if resolved {
				if hvLog == 0 {
					hvLog = 0.5
				}
				dF = math.Log(hl / hvLog)
			}
			better := false
			switch {
			case resolved && !b.Resolved:
				better = true
			case resolved && b.Resolved:
				better = dF > b.DeltaF
			case !resolved && !b.Resolved:
				better = depth > bestDepth
			}
			if better {
				bestDepth = depth
				b.Resolved, b.DeltaF = resolved, dF
				b.LowerPeak, b.UpperPeak, b.Valley = p, q, v
				b.HLower, b.HValley, b.Depth = hl, hvLog, depth
			}
		}
	}
	return b
}

// ---- small statistics ----------------------------------------------------------------------

// KZEqJackknifeSE is the jackknife standard error from the n leave-one-out estimates:
// √((n−1)/n · Σ (θ_i − θ̄)²). Fewer than two estimates → 0.
func KZEqJackknifeSE(theta []float64) float64 {
	n := float64(len(theta))
	if n < 2 {
		return 0
	}
	m := 0.0
	for _, v := range theta {
		m += v
	}
	m /= n
	s := 0.0
	for _, v := range theta {
		s += (v - m) * (v - m)
	}
	return math.Sqrt((n - 1) / n * s)
}

// KZEqEpsilon is the L-trend tolerance ε(L1, L2) = max(0.2, 2·√(σ1² + σ2²)).
func KZEqEpsilon(sigma1, sigma2 float64) float64 {
	return math.Max(KZEqEpsFloor, 2*math.Sqrt(sigma1*sigma1+sigma2*sigma2))
}

// KZEqBlockedSE blocks each series separately into consecutive blocks of `block` records
// (a trailing partial block is dropped; blocks never straddle two series), and returns the
// mean of the block means, their standard error sd(n−1)/√nb, and the block count. One block
// or fewer → SE 0.
func KZEqBlockedSE(series [][]float64, block int) (mean, se float64, nb int) {
	if block <= 0 {
		return 0, 0, 0
	}
	var means []float64
	for _, s := range series {
		for i := 0; i+block <= len(s); i += block {
			sum := 0.0
			for _, v := range s[i : i+block] {
				sum += v
			}
			means = append(means, sum/float64(block))
		}
	}
	nb = len(means)
	if nb == 0 {
		return 0, 0, 0
	}
	for _, m := range means {
		mean += m
	}
	mean /= float64(nb)
	if nb < 2 {
		return mean, 0, nb
	}
	v := 0.0
	for _, m := range means {
		v += (m - mean) * (m - mean)
	}
	return mean, math.Sqrt(v/float64(nb-1)) / math.Sqrt(float64(nb)), nb
}

// KZEqStartDep is the hot/cold comparison of one observable.
type KZEqStartDep struct {
	MeanHot, MeanCold float64
	SEHot, SECold     float64
	NHot, NCold       int     // blocks
	D                 float64 // |mean_cold − mean_hot| / √(SE_cold² + SE_hot²)
}

// kzEqDCap stands in for an infinite D (a non-zero difference with zero SEs) so the value
// stays JSON-encodable; it is far above the threshold.
const kzEqDCap = 1e6

// KZEqStartDependence compares the pooled hot series with the cold series at block size
// `block` (§3 start-dependence clause). A zero difference is D = 0 whatever the SEs.
func KZEqStartDependence(hot, cold [][]float64, block int) KZEqStartDep {
	var d KZEqStartDep
	d.MeanHot, d.SEHot, d.NHot = KZEqBlockedSE(hot, block)
	d.MeanCold, d.SECold, d.NCold = KZEqBlockedSE(cold, block)
	diff := math.Abs(d.MeanCold - d.MeanHot)
	se := math.Sqrt(d.SEHot*d.SEHot + d.SECold*d.SECold)
	switch {
	case diff == 0:
		d.D = 0
	case se == 0:
		d.D = kzEqDCap
	default:
		d.D = math.Min(diff/se, kzEqDCap)
	}
	return d
}

// KZEqBinder is the energy Binder cumulant V = 1 − ⟨e⁴⟩ / (3 ⟨e²⟩²) over the pooled samples.
func KZEqBinder(series ...[]float64) float64 {
	m2, m4, n := 0.0, 0.0, 0.0
	for _, s := range series {
		for _, v := range s {
			m2 += v * v
			m4 += v * v * v * v
			n++
		}
	}
	if n == 0 || m2 == 0 {
		return 0
	}
	m2 /= n
	m4 /= n
	return 1 - m4/(3*m2*m2)
}

// KZEqMaxStep is the largest single-record change |x[i+1] − x[i]| and its lower index
// (−1 for fewer than two records).
func KZEqMaxStep(x []float64) (float64, int) { return KZMaxJumpAt(x) }

// ---- the raw file ---------------------------------------------------------------------------

// KZEqPoint is one (L, T) raw file as loaded.
type KZEqPoint struct {
	File      string
	Mode      string // "eq" | "smoke-eq"
	PreregSHA string
	L, LIndex int
	TIndex    int
	T, Eta    float64
	Seeds     []uint64
	BurnIn    int
	Sampling  int
	RecEvery  int
	Runs      []KZEqSeedResult
	Complete  bool // meta + eq + end with exit 0, and no abort record
	HaveEnd   bool
	AbortCode int
	ExitCode  int
}

type kzEqRecord struct {
	Kind      string           `json:"kind"`
	Mode      string           `json:"mode"`
	PreregSHA string           `json:"prereg_sha"`
	L         int              `json:"L"`
	LIndex    int              `json:"l_index"`
	TIndex    int              `json:"t_index"`
	T         float64          `json:"T"`
	Eta       float64          `json:"eta"`
	Seeds     []uint64         `json:"seeds"`
	BurnIn    int              `json:"burn_in"`
	Sampling  int              `json:"sampling"`
	RecEvery  int              `json:"rec_every"`
	Runs      []KZEqSeedResult `json:"runs"`
	Code      int              `json:"code"`
	ExitCode  *int             `json:"exit_code"`
	Tau       int              `json:"tau"`
}

// KZEqLoad reads one equilibrium raw JSONL. It needs a meta record of mode "eq" or
// "smoke-eq" (a 0b / grid / probe file is an error); the point is Complete only with an eq
// record, an end record with exit_code 0 (or none recorded) and no abort record.
func KZEqLoad(rd io.Reader) (KZEqPoint, error) {
	var p KZEqPoint
	sc := bufio.NewScanner(rd)
	sc.Buffer(make([]byte, 1<<20), 1<<27)
	haveMeta, haveEq := false, false
	for sc.Scan() {
		var m kzEqRecord
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			continue
		}
		switch m.Kind {
		case "meta":
			if haveMeta {
				return p, fmt.Errorf("two meta records")
			}
			if m.Mode != "eq" && m.Mode != "smoke-eq" {
				return p, fmt.Errorf("meta mode %q is not an equilibrium (eq) file (tau=%d)", m.Mode, m.Tau)
			}
			haveMeta = true
			p.Mode, p.PreregSHA, p.L, p.LIndex, p.TIndex, p.T, p.Eta = m.Mode, m.PreregSHA, m.L, m.LIndex, m.TIndex, m.T, m.Eta
			p.Seeds, p.BurnIn, p.Sampling, p.RecEvery = m.Seeds, m.BurnIn, m.Sampling, m.RecEvery
		case "eq":
			if haveEq {
				return p, fmt.Errorf("two eq records")
			}
			haveEq = true
			p.Runs = m.Runs
			if haveMeta && (m.L != p.L || m.TIndex != p.TIndex) {
				return p, fmt.Errorf("eq record (L=%d, t=%d) differs from meta (L=%d, t=%d)", m.L, m.TIndex, p.L, p.TIndex)
			}
		case "end":
			p.HaveEnd = true
			if m.ExitCode != nil {
				p.ExitCode = *m.ExitCode
			}
		case "abort":
			p.AbortCode = m.Code
			if p.AbortCode == 0 {
				p.AbortCode = 1
			}
		}
	}
	if err := sc.Err(); err != nil {
		return p, err
	}
	if !haveMeta {
		return p, fmt.Errorf("no meta record")
	}
	p.Complete = haveEq && p.HaveEnd && p.AbortCode == 0 && p.ExitCode == 0
	return p, nil
}

// ---- the decision -----------------------------------------------------------------------------

// KZEqRunStat is the per-run row of the analysis.
type KZEqRunStat struct {
	Seed              uint64
	Start             string
	TauIntRec         float64 // recomputed from the series
	SokalW            int
	NEff              float64
	Undersampled      bool
	TauRecorded       float64 // what the driver wrote
	MeanE, MeanS      float64
	SE10, SE50, SE250 float64 // blocked SE of ⟨e⟩ at block sizes 10, 50, 250 records
	MaxStepE          float64
	MaxStepEAt        int
	MaxStepS          float64
	MaxStepSAt        int
}

// KZEqStartStat is the start-dependence reading of one (L, T) point.
type KZEqStartStat struct {
	E, S KZEqStartDep
	D    float64 // max(D_e, D_S)
}

// KZEqPointResult is the analysis of one (L, T) point.
type KZEqPointResult struct {
	L, LIndex, TIndex int
	T                 float64
	File              string
	Hist, Smoothed    []float64
	Barrier           KZEqBarrier
	DeltaF            float64   // ΔF_L(T): Barrier.DeltaF (0 unless resolved)
	JackDeltaF        []float64 // leave-one-seed-out barriers on the fixed grid
	SigmaDeltaF       float64
	Binder            float64
	Start             KZEqStartStat
	StartDependent    bool
	Runs              []KZEqRunStat
	AnyUndersampled   bool
	SHist, SSmoothed  []float64
	SBarrier          KZEqBarrier
	WLow, WHigh       float64 // phase weights split at the valley (raw counts), if resolved
}

// KZEqLResult is one lattice size.
type KZEqLResult struct {
	L, LIndex  int
	Bins       KZEqBinning
	SBins      KZEqBinning
	Points     []KZEqPointResult // ascending t
	DeltaFStar float64
	TStar      float64
	TStarIdx   int // t index (into KZKzG2R3TTable); the lowest grid t by the tiebreak when all ΔF = 0
	TStarPos   int // position in Points
	SigmaStar  float64
	VMin       float64
	VMinT      float64
	DeltaEStar float64 // peak separation at T*(L) if resolved there, else 0
}

// KZEqDecision is the §3 reading of the 27 points.
type KZEqDecision struct {
	PreregSHA string
	Ls        [3]KZEqLResult // L = 8, 12, 16

	Eps1612, Eps128 float64
	A, B            bool   // the (a) and (b) ΔF clauses as such
	UnderA          bool   // an under-sampled run at (16, T*(16))
	UnderB          bool   // an under-sampled run at any (L ∈ {12, 16}, T)
	StartDep        bool   // a start-dependent point at any (L ∈ {12, 16}, T)
	MetaFlag        bool   // a start-dependent point (L ∈ {12, 16}) with no resolved two-peak structure
	Outcome         string // "H-FO" | "H-CONT" | "inconclusive"
	Rule            string // "(a)" | "(b)" | "(c)"
	SubLabel        string // "" | "sampling" | "metastability"

	// secondaries (non-gating)
	VStar               float64 // two-point 1/L³ extrapolation of V_min from L = 12, 16
	Secondary16Reported bool    // ΔF*(16) > 0
	CoexistN            int     // bimodal grid points at L = 16
	CoexistReadable     bool
	CoexistT            float64
	DeltaE16            float64 // peak separation at (16, T*(16))
	PhaseLockCount      int     // seeds at (16, T*(16)) whose e stays on one side of the valley
	Warnings            []string
}

// KZEqDecide applies §3 to exactly the 27 complete points (any file order). Fewer, more,
// duplicated, incomplete, smoke, sha-mismatched or off-table points are refused: the rule is
// never applied to a partial set (§3 harness-kill row, r6 "complete set").
func KZEqDecide(in []KZEqPoint) (KZEqDecision, error) {
	var d KZEqDecision
	const want = 27
	if len(in) != want {
		return d, fmt.Errorf("refusing: %d file(s) given; §3 is evaluated only on the complete set of %d (L, t) points — no partial-set reading (harness-kill row)", len(in), want)
	}
	prm := DefaultKZEqParams()
	nrec := KZEqRecords(prm)
	byPoint := map[[2]int]KZEqPoint{}
	for _, p := range in {
		if p.Mode == "smoke-eq" {
			return d, fmt.Errorf("refusing: %s is a smoke-eq file; smoke output is never analyzed", p.File)
		}
		if p.Mode != "eq" {
			return d, fmt.Errorf("refusing: %s has mode %q, not eq", p.File, p.Mode)
		}
		l, ok := KZKzG2R3LIndex(p.L)
		if !ok || l != p.LIndex || !KZKzG2R3Runs(l, p.TIndex) {
			return d, fmt.Errorf("refusing: %s is (L=%d, l=%d, t=%d), not a point of the frozen design", p.File, p.L, p.LIndex, p.TIndex)
		}
		key := [2]int{l, p.TIndex}
		if q, dup := byPoint[key]; dup {
			return d, fmt.Errorf("refusing: two files for (L=%d, t=%d): %s, %s", p.L, p.TIndex, q.File, p.File)
		}
		byPoint[key] = p
	}
	// (27 distinct valid points among 27 files ⇒ every point of the design is present)
	first := in[0]
	for _, p := range in {
		if p.PreregSHA != first.PreregSHA {
			return d, fmt.Errorf("refusing: prereg_sha differs between %s (%s) and %s (%s)", first.File, first.PreregSHA, p.File, p.PreregSHA)
		}
		if !p.Complete {
			return d, fmt.Errorf("refusing: %s (L=%d, t=%d) is incomplete (abort code %d, exit %d, end record %v): a harness kill voids the point; re-run it on its reserved sub-block under a dated amendment", p.File, p.L, p.TIndex, p.AbortCode, p.ExitCode, p.HaveEnd)
		}
		if p.BurnIn != prm.BurnIn || p.Sampling != prm.Sampling || p.RecEvery != prm.RecEvery {
			return d, fmt.Errorf("refusing: %s has burn-in/sampling/rec_every %d/%d/%d, not the frozen %d/%d/%d", p.File, p.BurnIn, p.Sampling, p.RecEvery, prm.BurnIn, prm.Sampling, prm.RecEvery)
		}
		if p.Eta != kzG2R3Eta {
			return d, fmt.Errorf("refusing: %s has eta %v, not the frozen %v", p.File, p.Eta, kzG2R3Eta)
		}
		if p.T != KZKzG2R3TTable[p.TIndex] {
			return d, fmt.Errorf("refusing: %s has T %v at t=%d; the frozen table says %v", p.File, p.T, p.TIndex, KZKzG2R3TTable[p.TIndex])
		}
		runSeeds, rerunSeeds := KZKzG2R3Seeds(p.LIndex, p.TIndex, false), KZKzG2R3Seeds(p.LIndex, p.TIndex, true)
		if !kzEqSameSeeds(p.Seeds, runSeeds) && !kzEqSameSeeds(p.Seeds, rerunSeeds) {
			return d, fmt.Errorf("refusing: %s has seeds %v; (L=%d, t=%d) runs the seed sub-block %v (re-run %v)", p.File, p.Seeds, p.L, p.TIndex, runSeeds, rerunSeeds)
		}
		if len(p.Runs) != len(p.Seeds) {
			return d, fmt.Errorf("refusing: %s has %d runs for %d seeds", p.File, len(p.Runs), len(p.Seeds))
		}
		starts := KZKzG2R3Starts()
		for j, r := range p.Runs {
			if r.Seed != p.Seeds[j] {
				return d, fmt.Errorf("refusing: %s run %d has seed %d, meta says %d", p.File, j, r.Seed, p.Seeds[j])
			}
			if r.Start != starts[j] {
				return d, fmt.Errorf("refusing: %s run %d has start %q, the frozen rule says %q", p.File, j, r.Start, starts[j])
			}
			if len(r.E) != nrec || len(r.S) != nrec {
				return d, fmt.Errorf("refusing: %s run %d has %d/%d records, want exactly %d", p.File, j, len(r.S), len(r.E), nrec)
			}
		}
	}
	d.PreregSHA = first.PreregSHA

	// ---- per L ------------------------------------------------------------------------------
	for l, L := range KZKzG2R3LSet {
		lr := KZEqLResult{L: L, LIndex: l}
		var pts []KZEqPoint
		for t := range KZKzG2R3TTable {
			if p, ok := byPoint[[2]int{l, t}]; ok {
				pts = append(pts, p)
			}
		}
		sort.Slice(pts, func(a, b int) bool { return pts[a].TIndex < pts[b].TIndex })
		var allE, allS [][]float64
		for _, p := range pts {
			for _, r := range p.Runs {
				allE, allS = append(allE, r.E), append(allS, r.S)
			}
		}
		lr.Bins, lr.SBins = KZEqBinningOf(allE...), KZEqBinningOf(allS...)
		for _, p := range pts {
			pr := KZEqPointResult{L: L, LIndex: l, TIndex: p.TIndex, T: p.T, File: p.File}
			es, ss := make([][]float64, len(p.Runs)), make([][]float64, len(p.Runs))
			for j, r := range p.Runs {
				es[j], ss[j] = r.E, r.S
			}
			pr.Hist = KZEqHistogram(lr.Bins, es...)
			pr.Smoothed = KZEqSmooth3(pr.Hist)
			pr.Barrier = KZEqBarrierOf(pr.Smoothed)
			pr.DeltaF = pr.Barrier.DeltaF
			for i := range es {
				var rest [][]float64
				for j := range es {
					if j != i {
						rest = append(rest, es[j])
					}
				}
				pr.JackDeltaF = append(pr.JackDeltaF, KZEqBarrierOf(KZEqSmooth3(KZEqHistogram(lr.Bins, rest...))).DeltaF)
			}
			pr.SigmaDeltaF = KZEqJackknifeSE(pr.JackDeltaF)
			pr.Binder = KZEqBinder(es...)
			pr.SHist = KZEqHistogram(lr.SBins, ss...)
			pr.SSmoothed = KZEqSmooth3(pr.SHist)
			pr.SBarrier = KZEqBarrierOf(pr.SSmoothed)
			if pr.Barrier.Resolved {
				for i, c := range pr.Hist {
					if i <= pr.Barrier.Valley {
						pr.WLow += c
					} else {
						pr.WHigh += c
					}
				}
			}
			var hotE, coldE, hotS, coldS [][]float64
			for j, r := range p.Runs {
				rs := KZEqRunStat{Seed: r.Seed, Start: r.Start, TauRecorded: r.TauIntRec}
				rs.TauIntRec, rs.SokalW = KZSokalTauInt(r.E, 6)
				rs.NEff, rs.Undersampled = KZEqNEff(len(r.E), rs.TauIntRec)
				if math.Abs(rs.TauIntRec-r.TauIntRec) > 1e-6*math.Max(1, math.Abs(r.TauIntRec)) || rs.Undersampled != r.Undersampled {
					d.Warnings = append(d.Warnings, fmt.Sprintf("(L=%d, t=%d) seed %d: recorded tau_int_rec %.4f (undersampled %v) differs from the recomputed %.4f (%v); the recomputed value is used", L, p.TIndex, r.Seed, r.TauIntRec, r.Undersampled, rs.TauIntRec, rs.Undersampled))
				}
				rs.MeanE, rs.SE10, _ = KZEqBlockedSE([][]float64{r.E}, 10)
				_, rs.SE50, _ = KZEqBlockedSE([][]float64{r.E}, 50)
				_, rs.SE250, _ = KZEqBlockedSE([][]float64{r.E}, KZEqBlock)
				rs.MeanS, _, _ = KZEqBlockedSE([][]float64{r.S}, len(r.S))
				rs.MaxStepE, rs.MaxStepEAt = KZEqMaxStep(r.E)
				rs.MaxStepS, rs.MaxStepSAt = KZEqMaxStep(r.S)
				pr.Runs = append(pr.Runs, rs)
				pr.AnyUndersampled = pr.AnyUndersampled || rs.Undersampled
				if r.Start == "aligned" {
					coldE, coldS = append(coldE, es[j]), append(coldS, ss[j])
				} else {
					hotE, hotS = append(hotE, es[j]), append(hotS, ss[j])
				}
			}
			pr.Start.E = KZEqStartDependence(hotE, coldE, KZEqBlock)
			pr.Start.S = KZEqStartDependence(hotS, coldS, KZEqBlock)
			pr.Start.D = math.Max(pr.Start.E.D, pr.Start.S.D)
			pr.StartDependent = pr.Start.D > KZEqDThreshold
			lr.Points = append(lr.Points, pr)
		}
		// primary: ΔF*(L), T*(L) first in ascending T on a tie (all zero → the lowest grid T)
		lr.DeltaFStar, lr.TStarPos = -1, 0
		lr.VMin = math.Inf(1)
		for i, pr := range lr.Points {
			if pr.DeltaF > lr.DeltaFStar {
				lr.DeltaFStar, lr.TStarPos = pr.DeltaF, i
			}
			if pr.Binder < lr.VMin {
				lr.VMin, lr.VMinT = pr.Binder, pr.T
			}
		}
		star := lr.Points[lr.TStarPos]
		lr.TStar, lr.TStarIdx, lr.SigmaStar = star.T, star.TIndex, star.SigmaDeltaF
		if star.Barrier.Resolved {
			lr.DeltaEStar = lr.Bins.Centre(star.Barrier.UpperPeak) - lr.Bins.Centre(star.Barrier.LowerPeak)
		}
		d.Ls[l] = lr
	}

	// ---- the rule ------------------------------------------------------------------------------
	l8, l12, l16 := d.Ls[0], d.Ls[1], d.Ls[2]
	d.Eps1612 = KZEqEpsilon(l16.SigmaStar, l12.SigmaStar)
	d.Eps128 = KZEqEpsilon(l12.SigmaStar, l8.SigmaStar)
	d.A = l16.DeltaFStar >= KZEqDeltaF0 && l16.DeltaFStar >= l12.DeltaFStar-d.Eps1612 && l12.DeltaFStar >= l8.DeltaFStar-d.Eps128
	d.B = true
	for _, lr := range []KZEqLResult{l12, l16} {
		for _, pr := range lr.Points {
			if pr.DeltaF != 0 {
				d.B = false
			}
			if pr.AnyUndersampled {
				d.UnderB = true
			}
			if pr.StartDependent {
				d.StartDep = true
				if !pr.Barrier.Resolved {
					d.MetaFlag = true
				}
			}
		}
	}
	d.UnderA = l16.Points[l16.TStarPos].AnyUndersampled
	aFires := d.A && !d.UnderA
	switch {
	case aFires:
		d.Outcome, d.Rule = "H-FO", "(a)"
	case d.A && d.UnderA:
		d.Outcome, d.Rule, d.SubLabel = "inconclusive", "(c)", "sampling"
	case d.B && d.UnderB:
		d.Outcome, d.Rule, d.SubLabel = "inconclusive", "(c)", "sampling"
	case d.B && d.StartDep:
		d.Outcome, d.Rule, d.SubLabel = "inconclusive", "(c)", "metastability"
	case d.B:
		d.Outcome, d.Rule = "H-CONT", "(b)"
	default:
		d.Outcome, d.Rule = "inconclusive", "(c)"
		if d.MetaFlag {
			d.SubLabel = "metastability"
		}
	}

	// ---- secondaries ------------------------------------------------------------------------------
	x12, x16 := 1/math.Pow(12, 3), 1/math.Pow(16, 3)
	slope := (l12.VMin - l16.VMin) / (x12 - x16)
	d.VStar = l16.VMin - slope*x16
	d.Secondary16Reported = l16.DeltaFStar > 0
	star16 := l16.Points[l16.TStarPos]
	if d.Secondary16Reported && star16.Barrier.Resolved {
		d.DeltaE16 = l16.DeltaEStar
		valley := l16.Bins.Centre(star16.Barrier.Valley)
		for _, r := range byPoint[[2]int{2, star16.TIndex}].Runs {
			below, above := 0, 0
			for _, e := range r.E {
				if e < valley {
					below++
				} else {
					above++
				}
			}
			if below == 0 || above == 0 {
				d.PhaseLockCount++
			}
		}
	}
	// equal-weight coexistence T at L = 16: the sign change of (w_low − w_high) between
	// consecutive bimodal grid points, linearly interpolated; needs ≥ 2 bimodal points
	type wt struct{ T, diff float64 }
	var ws []wt
	for _, pr := range l16.Points {
		if pr.Barrier.Resolved {
			ws = append(ws, wt{pr.T, (pr.WLow - pr.WHigh) / (pr.WLow + pr.WHigh)})
		}
	}
	d.CoexistN = len(ws)
	if len(ws) >= 2 {
		for i := 0; i+1 < len(ws); i++ {
			a, b := ws[i], ws[i+1]
			if a.diff == 0 {
				d.CoexistReadable, d.CoexistT = true, a.T
				break
			}
			if (a.diff > 0) != (b.diff > 0) {
				d.CoexistReadable = true
				d.CoexistT = a.T + (b.T-a.T)*a.diff/(a.diff-b.diff)
				break
			}
		}
		if !d.CoexistReadable && ws[len(ws)-1].diff == 0 {
			d.CoexistReadable, d.CoexistT = true, ws[len(ws)-1].T
		}
	}
	return d, nil
}

func kzEqSameSeeds(a, b []uint64) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ---- report and plot data --------------------------------------------------------------------

// KZEqFormatReport renders the decision, tables first: the per-(L, T) barrier table, the
// per-run τ table, the primary ΔF*(L) table, the rule with its gates, the outcome, the
// secondaries and the warnings.
func KZEqFormatReport(d KZEqDecision) string {
	var b strings.Builder
	fmt.Fprintf(&b, "qbp-kz-g2 round 3 — equilibrium analysis (FROZEN-v1 §3)  prereg_sha %s\n", d.PreregSHA)
	fmt.Fprintf(&b, "bins: 60 per L over the pooled e; smoothing 3-bin (truncated edges); floor %g sigma; dF0 = ln 2 = %.4f; eps floor %.2f; block %d; D > %g\n\n",
		KZEqSigmaFloor, KZEqDeltaF0, KZEqEpsFloor, KZEqBlock, KZEqDThreshold)
	fmt.Fprintf(&b, "| L | t | T | dF_L(T) | sigma_dF | resolved | peaks (bins) | h_lower | h_valley | depth/sigma | V_L(T) | D_e | D_S | start-dep | undersampled | S bimodal | file |\n")
	fmt.Fprintf(&b, "|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, lr := range d.Ls {
		for _, p := range lr.Points {
			peaks := "-"
			if p.Barrier.Candidate {
				peaks = fmt.Sprintf("%d,%d (valley %d)", p.Barrier.LowerPeak, p.Barrier.UpperPeak, p.Barrier.Valley)
			}
			fmt.Fprintf(&b, "| %d | %d | %.2f | %.4f | %.4f | %v | %s | %.1f | %.1f | %.2f | %.6f | %.2f | %.2f | %v | %v | %v | %s |\n",
				p.L, p.TIndex, p.T, p.DeltaF, p.SigmaDeltaF, p.Barrier.Resolved, peaks, p.Barrier.HLower, p.Barrier.HValley, p.Barrier.Depth,
				p.Binder, p.Start.E.D, p.Start.S.D, p.StartDependent, p.AnyUndersampled, p.SBarrier.Resolved, p.File)
		}
	}
	fmt.Fprintf(&b, "\nper run (tau_int in RECORD units; n_eff = 5000 / (2 tau); undersampled iff n_eff < 20; SE of <e> at blocks 10/50/250):\n")
	fmt.Fprintf(&b, "| L | t | seed | start | tau_int_rec | W | n_eff | undersampled | tau recorded | <e> | SE10 | SE50 | SE250 | <S> | max step e (rec) | max step S (rec) |\n")
	fmt.Fprintf(&b, "|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, lr := range d.Ls {
		for _, p := range lr.Points {
			for _, r := range p.Runs {
				fmt.Fprintf(&b, "| %d | %d | %d | %s | %.3f | %d | %.1f | %v | %.3f | %.6f | %.2e | %.2e | %.2e | %.6f | %.4f (%d) | %.4f (%d) |\n",
					p.L, p.TIndex, r.Seed, r.Start, r.TauIntRec, r.SokalW, r.NEff, r.Undersampled, r.TauRecorded, r.MeanE, r.SE10, r.SE50, r.SE250, r.MeanS,
					r.MaxStepE, r.MaxStepEAt, r.MaxStepS, r.MaxStepSAt)
			}
		}
	}
	fmt.Fprintf(&b, "\nprimary:\n| L | dF*(L) | T*(L) (t) | sigma*(L) | V_min(L) at T | bins [lo, hi] | de at T* |\n|---|---|---|---|---|---|---|\n")
	for _, lr := range d.Ls {
		note := ""
		if lr.DeltaFStar == 0 {
			note = " (tiebreak: lowest grid T, no physical meaning)"
		}
		fmt.Fprintf(&b, "| %d | %.4f | %.2f (%d)%s | %.4f | %.6f at %.2f | [%.5f, %.5f] | %.4f |\n", lr.L, lr.DeltaFStar, lr.TStar, lr.TStarIdx, note, lr.SigmaStar, lr.VMin, lr.VMinT, lr.Bins.Lo, lr.Bins.Hi, lr.DeltaEStar)
	}
	l8, l12, l16 := d.Ls[0], d.Ls[1], d.Ls[2]
	fmt.Fprintf(&b, "\nrule (precedence (a) -> (b) -> (c)):\n")
	fmt.Fprintf(&b, "  eps(16,12) = %.4f  eps(12,8) = %.4f\n", d.Eps1612, d.Eps128)
	fmt.Fprintf(&b, "  (a) dF*(16) = %.4f >= ln 2 = %.4f ? %v;  dF*(16) >= dF*(12) - eps = %.4f ? %v;  dF*(12) = %.4f >= dF*(8) - eps = %.4f ? %v  => (a) clauses %v\n",
		l16.DeltaFStar, KZEqDeltaF0, l16.DeltaFStar >= KZEqDeltaF0, l12.DeltaFStar-d.Eps1612, l16.DeltaFStar >= l12.DeltaFStar-d.Eps1612,
		l12.DeltaFStar, l8.DeltaFStar-d.Eps128, l12.DeltaFStar >= l8.DeltaFStar-d.Eps128, d.A)
	fmt.Fprintf(&b, "  (a)-side gate: undersampled run at (16, T*(16) = %.2f)? %v\n", l16.TStar, d.UnderA)
	fmt.Fprintf(&b, "  (b) dF_16(T) = 0 and dF_12(T) = 0 at every T ? %v\n", d.B)
	fmt.Fprintf(&b, "  (b)-side gate: undersampled run at any (L in {12,16}, T)? %v;  start-dependent point at any (L in {12,16}, T)? %v (without a resolved pair: %v)\n", d.UnderB, d.StartDep, d.MetaFlag)
	out := d.Outcome
	if d.SubLabel != "" {
		out += " (" + d.SubLabel + ")"
	}
	fmt.Fprintf(&b, "\nOUTCOME: %s by rule %s\n", out, d.Rule)
	fmt.Fprintf(&b, "\nsecondaries (non-gating):\n")
	fmt.Fprintf(&b, "  V_min: L=8 %.6f, L=12 %.6f, L=16 %.6f; two-point 1/L^3 extrapolation V* = %.6f (2/3 = %.6f)\n", l8.VMin, l12.VMin, l16.VMin, d.VStar, 2.0/3)
	if d.Secondary16Reported {
		fmt.Fprintf(&b, "  at (16, T*(16) = %.2f): de (peak separation) = %.4f; phase-locking count = %d of %d seeds\n", l16.TStar, d.DeltaE16, d.PhaseLockCount, len(l16.Points[l16.TStarPos].Runs))
	} else {
		fmt.Fprintf(&b, "  at (16, T*(16)): not reported (dF*(16) = 0; T*(16) is the tiebreak value)\n")
	}
	fmt.Fprintf(&b, "  de trend at each L's own T*: L=8 %.4f, L=12 %.4f, L=16 %.4f (0 = no resolved pair there)\n", l8.DeltaEStar, l12.DeltaEStar, l16.DeltaEStar)
	if d.CoexistReadable {
		fmt.Fprintf(&b, "  equal-weight coexistence T at L = 16: %.4f (from %d bimodal grid points)\n", d.CoexistT, d.CoexistN)
	} else {
		fmt.Fprintf(&b, "  no coexistence T readable at L = 16 (%d bimodal grid point(s); needs >= 2 with an equal-weight crossing)\n", d.CoexistN)
	}
	nS := 0
	for _, lr := range d.Ls {
		for _, p := range lr.Points {
			if p.SBarrier.Resolved {
				nS++
			}
		}
	}
	fmt.Fprintf(&b, "  S histograms: %d of 27 points resolved-bimodal (descriptive only)\n", nS)
	for _, w := range d.Warnings {
		fmt.Fprintf(&b, "  WARNING: %s\n", w)
	}
	return b.String()
}

// kzEqPlot is the plot-data JSON shape.
type kzEqPlot struct {
	PreregSHA string         `json:"prereg_sha"`
	Outcome   string         `json:"outcome"`
	Rule      string         `json:"rule"`
	SubLabel  string         `json:"sub_label"`
	Ls        []kzEqPlotL    `json:"L"`
	Eps1612   float64        `json:"eps_16_12"`
	Eps128    float64        `json:"eps_12_8"`
	VStar     float64        `json:"v_star"`
	Coexist   map[string]any `json:"coexistence"`
	Warnings  []string       `json:"warnings"`
}

type kzEqPlotL struct {
	L          int             `json:"L"`
	BinLo      float64         `json:"bin_lo"`
	BinHi      float64         `json:"bin_hi"`
	SBinLo     float64         `json:"s_bin_lo"`
	SBinHi     float64         `json:"s_bin_hi"`
	DeltaFStar float64         `json:"delta_f_star"`
	TStar      float64         `json:"t_star"`
	SigmaStar  float64         `json:"sigma_star"`
	VMin       float64         `json:"v_min"`
	Points     []kzEqPlotPoint `json:"points"`
}

type kzEqPlotPoint struct {
	T              float64       `json:"T"`
	TIndex         int           `json:"t_index"`
	Hist           []float64     `json:"hist"`
	Smoothed       []float64     `json:"smoothed"`
	SHist          []float64     `json:"s_hist"`
	DeltaF         float64       `json:"delta_f"`
	SigmaDeltaF    float64       `json:"sigma_delta_f"`
	Resolved       bool          `json:"resolved"`
	Peaks          []int         `json:"peaks"`
	Valley         int           `json:"valley"`
	Binder         float64       `json:"binder"`
	DE             float64       `json:"d_e"`
	DS             float64       `json:"d_s"`
	StartDependent bool          `json:"start_dependent"`
	Runs           []kzEqPlotRun `json:"runs"`
}

type kzEqPlotRun struct {
	Seed         uint64  `json:"seed"`
	Start        string  `json:"start"`
	TauIntRec    float64 `json:"tau_int_rec"`
	NEff         float64 `json:"n_eff"`
	Undersampled bool    `json:"undersampled"`
	MeanE        float64 `json:"mean_e"`
	MaxStepE     float64 `json:"max_step_e"`
	MaxStepS     float64 `json:"max_step_s"`
}

// KZEqPlotJSON is the plot data of a decision (histograms, smoothed curves, S histograms,
// V_L(T), per-run τ) as JSON. The per-seed traces for Human Visual Review are read from the
// raw files directly (they are the e_series / s_series themselves).
func KZEqPlotJSON(d KZEqDecision) ([]byte, error) {
	p := kzEqPlot{PreregSHA: d.PreregSHA, Outcome: d.Outcome, Rule: d.Rule, SubLabel: d.SubLabel, Eps1612: d.Eps1612, Eps128: d.Eps128, VStar: d.VStar,
		Coexist:  map[string]any{"readable": d.CoexistReadable, "T": d.CoexistT, "n_bimodal": d.CoexistN, "delta_e_16": d.DeltaE16, "phase_lock_count": d.PhaseLockCount, "reported_16": d.Secondary16Reported},
		Warnings: d.Warnings}
	for _, lr := range d.Ls {
		pl := kzEqPlotL{L: lr.L, BinLo: lr.Bins.Lo, BinHi: lr.Bins.Hi, SBinLo: lr.SBins.Lo, SBinHi: lr.SBins.Hi, DeltaFStar: lr.DeltaFStar, TStar: lr.TStar, SigmaStar: lr.SigmaStar, VMin: lr.VMin}
		for _, pt := range lr.Points {
			pp := kzEqPlotPoint{T: pt.T, TIndex: pt.TIndex, Hist: pt.Hist, Smoothed: pt.Smoothed, SHist: pt.SHist, DeltaF: pt.DeltaF, SigmaDeltaF: pt.SigmaDeltaF,
				Resolved: pt.Barrier.Resolved, Peaks: pt.Barrier.Peaks, Valley: pt.Barrier.Valley, Binder: pt.Binder, DE: pt.Start.E.D, DS: pt.Start.S.D, StartDependent: pt.StartDependent}
			for _, r := range pt.Runs {
				pp.Runs = append(pp.Runs, kzEqPlotRun{Seed: r.Seed, Start: r.Start, TauIntRec: r.TauIntRec, NEff: r.NEff, Undersampled: r.Undersampled, MeanE: r.MeanE, MaxStepE: r.MaxStepE, MaxStepS: r.MaxStepS})
			}
			pl.Points = append(pl.Points, pp)
		}
		p.Ls = append(p.Ls, pl)
	}
	return json.MarshalIndent(p, "", " ")
}
