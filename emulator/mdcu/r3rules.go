package mdcu

// Round-3 decision rules (frozen §1.5), kept in the library so they are
// unit-tested on synthetic data with known answers. The analyzer command is a
// thin reader/printer around these functions.
//
// Rules transcribed from doc/spikes/prereg/mdcu-round3-prereg-FROZEN-v1.md.
// "Rate" = 3-lock rate (Class == "3-lock", frozen §1.3). Δ(X,Y) = median paired
// S3(X) − S3(Y) with a 95% percentile bootstrap CI over 10,000 resamples.
// Boundary comparisons carry a 1e-9 tolerance so that k/n rates sitting exactly
// on a threshold (e.g. a 10-point gap, a 90% rate) are decided as written.

import (
	"hash/fnv"
	"math"
	"math/rand/v2"
	"sort"
)

const (
	r3Boot     = 10000
	r3Eps      = 1e-9
	r3RateTol  = 0.10 // ±10 points
	r3DeltaEq  = 0.05 // equivalence margin on Δ
	r3DeltaBig = 0.15 // "clearly different" on Δ
	r3PhiPre   = 0.50 // Q1b precondition: K-phi rate
	r3Rate40   = 0.40 // Q1b yes via rate gap
	r3MinPairs = 50   // stratum power floor
)

// LockRate is the fraction of runs classified "3-lock".
func LockRate(rs []R3Run) float64 {
	if len(rs) == 0 {
		return math.NaN()
	}
	n := 0
	for _, r := range rs {
		if r.Stats.Class == "3-lock" {
			n++
		}
	}
	return float64(n) / float64(len(rs))
}

// PairBySeed returns the runs of a and b restricted to seeds present in both,
// ordered by seed (position i of each result refers to the same seed).
func PairBySeed(a, b []R3Run) (pa, pb []R3Run) {
	idx := map[uint64]R3Run{}
	for _, r := range b {
		idx[r.Seed] = r
	}
	sorted := append([]R3Run(nil), a...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Seed < sorted[j].Seed })
	for _, r := range sorted {
		if rb, ok := idx[r.Seed]; ok {
			pa, pb = append(pa, r), append(pb, rb)
		}
	}
	return
}

// Median of s (NaN for empty).
func Median(s []float64) float64 { return quantileF(s, 0.5) }

func quantileF(s []float64, q float64) float64 {
	if len(s) == 0 {
		return math.NaN()
	}
	c := append([]float64(nil), s...)
	sort.Float64s(c)
	pos := q * float64(len(c)-1)
	lo, hi := int(math.Floor(pos)), int(math.Ceil(pos))
	return c[lo] + (c[hi]-c[lo])*(pos-float64(lo))
}

// ContrastStat is Δ(A,B) on the paired seeds, plus both rates on those seeds.
type ContrastStat struct {
	Label        string
	N            int
	RateA, RateB float64
	Median       float64 // median paired S3(A) − S3(B)
	Lo, Hi       float64 // bootstrap 95% CI of that median
}

func bootSeed(label string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(label))
	return h.Sum64()
}

// R3Contrast pairs a and b by seed and computes Δ(a,b). The bootstrap RNG is
// seeded from the label, so results do not depend on evaluation order.
func R3Contrast(label string, a, b []R3Run) ContrastStat {
	pa, pb := PairBySeed(a, b)
	c := ContrastStat{Label: label, N: len(pa), RateA: LockRate(pa), RateB: LockRate(pb)}
	if len(pa) == 0 {
		c.Median, c.Lo, c.Hi = math.NaN(), math.NaN(), math.NaN()
		return c
	}
	d := make([]float64, len(pa))
	for i := range pa {
		d[i] = pa[i].Stats.S3 - pb[i].Stats.S3
	}
	c.Median = Median(d)
	rng := rand.New(rand.NewPCG(bootSeed(label), 0x3243f6a8885a308d))
	meds := make([]float64, r3Boot)
	buf := make([]float64, len(d))
	for b := range meds {
		for i := range buf {
			buf[i] = d[rng.IntN(len(d))]
		}
		meds[b] = Median(buf)
	}
	c.Lo, c.Hi = quantileF(meds, .025), quantileF(meds, .975)
	return c
}

// ciInsideEq: CI lies inside ±0.05.
func (c ContrastStat) ciInsideEq() bool { return c.Lo >= -r3DeltaEq-r3Eps && c.Hi <= r3DeltaEq+r3Eps }

// bigPositive: Δ ≥ 0.15 with the CI excluding 0.
func (c ContrastStat) bigPositive() bool { return c.Median >= r3DeltaBig-r3Eps && c.Lo > 0 }

// bigNegative: Δ ≤ −0.15 with the CI excluding 0.
func (c ContrastStat) bigNegative() bool { return c.Median <= -r3DeltaBig+r3Eps && c.Hi < 0 }

func within10(a, b float64) bool { return math.Abs(a-b) <= r3RateTol+r3Eps }

// VerdictQ1a is the frozen Q1a rule for c = Δ(K-full, K-gram) over all seeds.
// Returns "yes", "no" or "inconclusive".
func VerdictQ1a(c ContrastStat, gramRate float64) string {
	if gramRate >= 0.90-r3Eps && within10(c.RateA, c.RateB) && c.ciInsideEq() {
		return "yes"
	}
	if gramRate < 0.50-r3Eps || c.bigPositive() {
		return "no"
	}
	return "inconclusive"
}

// VerdictQ1b is the frozen Q1b rule for c = Δ(K-phi, K-psi) on one stratum's
// paired seeds (phi, psi are that stratum's runs; c.N is the paired count).
func VerdictQ1b(c ContrastStat, phi, psi []R3Run) string {
	if c.N < r3MinPairs {
		return "inconclusive (underpowered)"
	}
	if c.RateA < r3PhiPre-r3Eps {
		return "inconclusive (φ-only does not lock)"
	}
	if within10(c.RateA, c.RateB) && c.ciInsideEq() {
		return "no"
	}
	if c.bigPositive() || c.RateA-c.RateB >= r3Rate40-r3Eps {
		return "yes"
	}
	if c.bigNegative() {
		return "yes-reversed"
	}
	return "inconclusive"
}

// ReadingRow is the pre-registered reading table (frozen §1.5). gate is the
// replication gate (K-full 3-lock ≥ 90%); a failed gate overrides everything.
func ReadingRow(gate bool, q1a, bdef, bsplit string) string {
	if !gate {
		return "Inconclusive (non-replication): K-full did not 3-lock in ≥ 90% of seeds, so no R3-Q1 reading is made."
	}
	switch {
	case q1a == "yes":
		return "Locking is reproduced by the O(7) Gram term alone. Round-2 locking is not G₂ evidence, consistent with the null-3 = Gram + noise hypothesis. (Row 1)"
	case q1a == "no" && bdef == "no" && bsplit == "no":
		return "The 3-form term contributes, but any 3-form does. Not G₂-specific. (Row 2)"
	case q1a == "no" && bdef == "no" && bsplit == "yes":
		return "The GL(7) type matters (compact-type ψ behaves like φ; split does not). G₂-type matters, but not Euclidean-compatible φ specifically. (Row 3)"
	case q1a == "no" && bdef == "yes" && bsplit == "yes":
		return "φ specifically matters. This is the only cell that supports a G₂-specific reading. (Row 4)"
	}
	return "Inconclusive, or a mixed result reported as-is. (Row 5)"
}

// Spearman is the rank correlation (average ranks for ties); NaN if either
// input is constant or the lengths differ / are < 3.
func Spearman(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 3 {
		return math.NaN()
	}
	rx, ry := avgRanks(x), avgRanks(y)
	var mx, my float64
	for i := range rx {
		mx += rx[i]
		my += ry[i]
	}
	mx /= float64(len(rx))
	my /= float64(len(ry))
	var sxy, sxx, syy float64
	for i := range rx {
		sxy += (rx[i] - mx) * (ry[i] - my)
		sxx += (rx[i] - mx) * (rx[i] - mx)
		syy += (ry[i] - my) * (ry[i] - my)
	}
	if sxx == 0 || syy == 0 {
		return math.NaN()
	}
	return sxy / math.Sqrt(sxx*syy)
}

func avgRanks(v []float64) []float64 {
	idx := make([]int, len(v))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return v[idx[a]] < v[idx[b]] })
	r := make([]float64, len(v))
	for i := 0; i < len(idx); {
		j := i
		for j+1 < len(idx) && v[idx[j+1]] == v[idx[i]] {
			j++
		}
		avg := float64(i+j)/2 + 1
		for k := i; k <= j; k++ {
			r[idx[k]] = avg
		}
		i = j + 1
	}
	return r
}
