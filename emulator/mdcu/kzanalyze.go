package mdcu

// mdcu-kz analyzer (frozen mdcu-kz prereg §3 observable, §5 Q5a/Q5b/Q5c and kills).
// Pure functions of per-replicate counts; no dynamics. HARNESS marks choices the frozen
// text leaves open; they are listed in the check-in.

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
)

const (
	KZBootstrapResamples = 10000
	kzBootSeed           = 0xB007
)

// KZRungData is one rung and arm: per-replicate counts of one detector read each.
type KZRungData struct {
	Tau       int
	Cells     int
	Defects   []int // -1 cells with no ambiguous triangle
	Ambiguous []int // ambiguous cells
	Neg       []int // raw -1 cells (parity / saturation only)
}

// KZRungStat is n(τ_Q) and the quantities the rules read at one rung.
type KZRungStat struct {
	Tau           int
	N             float64 // mean over replicates of Defects/Cells
	CILo, CIHi    float64 // 95% percentile bootstrap CI, 10,000 resamples of the replicates
	PooledDefects int
	AmbigFrac     float64 // pooled ambiguous cells / pooled cells
}

func kzBootRNG(stream uint64) *rand.Rand { return rand.New(rand.NewPCG(kzBootSeed, stream)) }

// KZRungStats computes n and its bootstrap CI; stream selects the bootstrap generator
// (HARNESS: the analyzer passes 2*τ_Q for the main arm and 2*τ_Q+1 for the null).
func KZRungStats(d KZRungData, stream uint64) KZRungStat {
	R := len(d.Defects)
	per := make([]float64, R)
	st := KZRungStat{Tau: d.Tau}
	amb := 0
	for j := 0; j < R; j++ {
		per[j] = float64(d.Defects[j]) / float64(d.Cells)
		st.N += per[j]
		st.PooledDefects += d.Defects[j]
		amb += d.Ambiguous[j]
	}
	st.N /= float64(R)
	st.AmbigFrac = float64(amb) / float64(R*d.Cells)
	r := kzBootRNG(stream)
	means := make([]float64, KZBootstrapResamples)
	for b := range means {
		s := 0.0
		for j := 0; j < R; j++ {
			s += per[r.IntN(R)]
		}
		means[b] = s / float64(R)
	}
	sort.Float64s(means)
	st.CILo = means[int(0.025*float64(KZBootstrapResamples))]
	st.CIHi = means[int(0.975*float64(KZBootstrapResamples))-1]
	if st.CILo > st.N {
		st.CILo = math.Min(st.CILo, st.N)
	}
	if st.CIHi < st.N {
		st.CIHi = math.Max(st.CIHi, st.N)
	}
	return st
}

// ---- log-log fits ----------------------------------------------------------------------

// KZFit holds the linear and quadratic fits of y on x (both log10).
type KZFit struct {
	Slope, Intercept float64
	QuadC            float64 // coefficient of x² in the quadratic fit
	RSSLin, RSSQuad  float64
	DeltaAIC         float64 // AIC(quadratic) - AIC(linear)
}

// kzRSSFloor keeps the AIC finite for noise-free data (HARNESS): on the log10 scale it
// is far below any real scatter.
const kzRSSFloor = 1e-12

// KZFitLogLog fits y = a + b x and y = a + b x + c x² by least squares; AIC = N ln(RSS/N)
// + 2k with k = 2 and 3, RSS floored at kzRSSFloor.
func KZFitLogLog(x, y []float64) KZFit {
	n := float64(len(x))
	mx := 0.0
	for _, v := range x {
		mx += v
	}
	mx /= n
	var sxx, sxy, sy float64
	for i := range x {
		dx := x[i] - mx
		sxx += dx * dx
		sxy += dx * y[i]
		sy += y[i]
	}
	f := KZFit{}
	f.Slope = sxy / sxx
	f.Intercept = sy/n - f.Slope*mx
	for i := range x {
		r := y[i] - f.Intercept - f.Slope*x[i]
		f.RSSLin += r * r
	}
	// quadratic on centred x: columns 1, d, d²
	var A [3][4]float64
	for i := range x {
		d := x[i] - mx
		row := [3]float64{1, d, d * d}
		for a := 0; a < 3; a++ {
			for b := 0; b < 3; b++ {
				A[a][b] += row[a] * row[b]
			}
			A[a][3] += row[a] * y[i]
		}
	}
	for c := 0; c < 3; c++ { // Gauss-Jordan with partial pivoting
		p := c
		for r := c + 1; r < 3; r++ {
			if math.Abs(A[r][c]) > math.Abs(A[p][c]) {
				p = r
			}
		}
		A[c], A[p] = A[p], A[c]
		for r := 0; r < 3; r++ {
			if r == c {
				continue
			}
			m := A[r][c] / A[c][c]
			for k := c; k < 4; k++ {
				A[r][k] -= m * A[c][k]
			}
		}
	}
	q0, q1, q2 := A[0][3]/A[0][0], A[1][3]/A[1][1], A[2][3]/A[2][2]
	f.QuadC = q2
	for i := range x {
		d := x[i] - mx
		r := y[i] - q0 - q1*d - q2*d*d
		f.RSSQuad += r * r
	}
	aic := func(rss float64, k float64) float64 { return n*math.Log(math.Max(rss, kzRSSFloor)/n) + 2*k }
	f.DeltaAIC = aic(f.RSSQuad, 3) - aic(f.RSSLin, 2)
	return f
}

// ---- decision ---------------------------------------------------------------------------

// KZDecision is the mechanical reading of one grid.
type KZDecision struct {
	Q5a, Q5b, Q5c           string
	Q5cLiteral              bool // the literal "main CI above null CI at every used rung", whatever the kills say
	NullFires               bool
	NullReason              string
	Alpha, AlphaLo, AlphaHi float64
	QuadC, QuadCLo, QuadCHi float64
	DeltaAIC                float64
	SpanDecades             float64
	Used                    []int // indices into the input rungs
	RungStatus              []string
	MainStats, NullStats    []KZRungStat
	Resamples               int
}

const (
	kzAmbigMax    = 0.05
	kzCountFloor  = 20
	kzSpanMin     = 1.5
	kzMinFitRungs = 4 // HARNESS guard: with 3 points the quadratic is exact
	kzBandLo      = 0.6
	kzBandHi      = 1.0
)

// KZAnalyze applies §5 to per-replicate counts of the main arm and the S³ null (same
// ladder order). "n" everywhere is the defect count, never the raw -1 count.
func KZAnalyze(main, null []KZRungData) KZDecision {
	d := KZDecision{RungStatus: make([]string, len(main))}
	for i := range main {
		d.MainStats = append(d.MainStats, KZRungStats(main[i], 2*uint64(main[i].Tau)))
		d.NullStats = append(d.NullStats, KZRungStats(null[i], 2*uint64(null[i].Tau)+1))
	}
	// the null kills
	for i := range null {
		for j, df := range null[i].Defects {
			if df > 0 {
				d.NullFires = true
				d.NullReason = fmt.Sprintf("null replicate %d at tau %d has %d non-ambiguous -1 cell(s)", j, null[i].Tau, df)
				break
			}
		}
	}
	// rung status
	anyResolution := false
	for i, s := range d.MainStats {
		switch {
		case s.AmbigFrac >= kzAmbigMax:
			d.RungStatus[i] = "inconclusive (resolution)"
			anyResolution = true
		case s.PooledDefects < kzCountFloor:
			d.RungStatus[i] = "inconclusive (finite size)"
		default:
			d.RungStatus[i] = "used"
			d.Used = append(d.Used, i)
		}
	}
	if !d.NullFires {
		for _, i := range d.Used {
			if d.NullStats[i].CILo > 0.10*d.MainStats[i].CIHi {
				d.NullFires = true
				d.NullReason = fmt.Sprintf("null n above 10%% of main n with non-overlapping CIs at tau %d", main[i].Tau)
				break
			}
		}
	}
	// literal Q5c
	d.Q5cLiteral = len(d.Used) > 0
	for _, i := range d.Used {
		if !(d.MainStats[i].CILo > d.NullStats[i].CIHi) {
			d.Q5cLiteral = false
		}
	}
	if d.NullFires {
		d.Q5a, d.Q5b, d.Q5c = "inconclusive (detector)", "inconclusive (detector)", "inconclusive (detector)"
		return d
	}
	// span
	if len(d.Used) > 0 {
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, i := range d.Used {
			x := math.Log10(float64(main[i].Tau))
			lo, hi = math.Min(lo, x), math.Max(hi, x)
		}
		d.SpanDecades = hi - lo
	}
	switch {
	case len(d.Used) == 0 && anyResolution:
		d.Q5a, d.Q5b = "inconclusive (resolution)", "inconclusive (resolution)"
		d.Q5c = "inconclusive (resolution)"
		return d
	case len(d.Used) < kzMinFitRungs || d.SpanDecades < kzSpanMin-1e-9:
		d.Q5a, d.Q5b = "inconclusive (span)", "inconclusive (span)"
		d.Q5c = "inconclusive (span)"
		if len(d.Used) > 0 {
			d.Q5c = kzYesNo(d.Q5cLiteral)
		}
		return d
	}
	// fits: point estimates on the means, bootstrap CIs on the replicates resampled
	// jointly across rungs (the design is paired: replicate j has the same start state at
	// every rung). HARNESS.
	x := make([]float64, len(d.Used))
	y := make([]float64, len(d.Used))
	for k, i := range d.Used {
		x[k] = math.Log10(float64(main[i].Tau))
		y[k] = math.Log10(d.MainStats[i].N)
	}
	fit := KZFitLogLog(x, y)
	d.Alpha, d.QuadC, d.DeltaAIC = -fit.Slope, fit.QuadC, fit.DeltaAIC
	R := len(main[d.Used[0]].Defects)
	per := make([][]float64, len(d.Used))
	for k, i := range d.Used {
		per[k] = make([]float64, R)
		for j := 0; j < R; j++ {
			per[k][j] = float64(main[i].Defects[j]) / float64(main[i].Cells)
		}
	}
	r := kzBootRNG(0xF17)
	alphas := make([]float64, 0, KZBootstrapResamples)
	quads := make([]float64, 0, KZBootstrapResamples)
	yb := make([]float64, len(d.Used))
	idx := make([]int, R)
	for b := 0; b < KZBootstrapResamples; b++ {
		for j := range idx {
			idx[j] = r.IntN(R)
		}
		ok := true
		for k := range d.Used {
			s := 0.0
			for _, j := range idx {
				s += per[k][j]
			}
			s /= float64(R)
			if s <= 0 {
				ok = false
				break
			}
			yb[k] = math.Log10(s)
		}
		if !ok {
			continue
		}
		fb := KZFitLogLog(x, yb)
		alphas = append(alphas, -fb.Slope)
		quads = append(quads, fb.QuadC)
	}
	d.Resamples = len(alphas)
	sort.Float64s(alphas)
	sort.Float64s(quads)
	q := func(v []float64, p float64) float64 { return v[int(p*float64(len(v)))] }
	d.AlphaLo, d.AlphaHi = q(alphas, 0.025), q(alphas, 0.975)
	d.QuadCLo, d.QuadCHi = q(quads, 0.025), q(quads, 0.975)
	noCurv := d.QuadCLo <= 0 && d.QuadCHi >= 0 && d.DeltaAIC > -2
	alphaNonzero := d.AlphaLo > 0 || d.AlphaHi < 0
	d.Q5a = kzYesNo(noCurv && alphaNonzero)
	d.Q5b = kzYesNo(d.Alpha >= kzBandLo && d.Alpha <= kzBandHi)
	d.Q5c = kzYesNo(d.Q5cLiteral)
	return d
}

func kzYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
