// Command mdcu-q1-analyze evaluates the FROZEN round-2 Q1 rules
// (doc/spikes/prereg/mdcu-round2-prereg-FROZEN-v3.md) on the JSONL written by
// mdcu-q1. It was written from the frozen text, before reading any grid output.
//
//	go run ./cmd/mdcu-q1-analyze -in ../doc/spikes/raw/mdcu-q1-2026-10-23.jsonl
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"sort"

	"github.com/JamesPagetButler/qbp-compute-unit/emulator/mdcu"
)

type key struct {
	N     int
	Sigma float64
	Kind  string
}

func quantile(s []float64, q float64) float64 {
	c := append([]float64(nil), s...)
	sort.Float64s(c)
	if len(c) == 0 {
		return math.NaN()
	}
	pos := q * float64(len(c)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	return c[lo] + (c[hi]-c[lo])*(pos-float64(lo))
}

func fracGE(s []float64, t float64) float64 {
	n := 0
	for _, v := range s {
		if v >= t {
			n++
		}
	}
	return float64(n) / float64(len(s))
}

func pickQ(runs []mdcu.Q1Run, f func(mdcu.Q1Stats) float64) []float64 {
	out := make([]float64, len(runs))
	for i, r := range runs {
		out[i] = f(r.Final)
	}
	return out
}

func qline(name string, s []float64) string {
	return fmt.Sprintf("%-10s q5=%.3f q25=%.3f q50=%.3f q75=%.3f q95=%.3f", name,
		quantile(s, .05), quantile(s, .25), quantile(s, .5), quantile(s, .75), quantile(s, .95))
}

// planeDist2 = 3 - ||U^T V||_F^2  (sum of sin^2 of the principal angles).
func planeDist(a, b [3][7]float64) float64 {
	s := 0.0
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			d := 0.0
			for k := 0; k < 7; k++ {
				d += a[i][k] * b[j][k]
			}
			s += d * d
		}
	}
	return math.Sqrt(math.Max(0, 3-s))
}

func pairwise(p [][3][7]float64) []float64 {
	out := make([]float64, 0, len(p)*(len(p)-1)/2)
	for i := range p {
		for j := i + 1; j < len(p); j++ {
			out = append(out, planeDist(p[i], p[j]))
		}
	}
	return out
}

func ksD(a, b []float64) float64 {
	x := append([]float64(nil), a...)
	y := append([]float64(nil), b...)
	sort.Float64s(x)
	sort.Float64s(y)
	i, j, d := 0, 0, 0.0
	for i < len(x) && j < len(y) {
		if x[i] <= y[j] {
			i++
		} else {
			j++
		}
		d = math.Max(d, math.Abs(float64(i)/float64(len(x))-float64(j)/float64(len(y))))
	}
	return d
}

// refPlane: span(x, y, xy), x uniform on S^6, y uniform on S^5 orthogonal to x
// (the G2-invariant measure on associative 3-planes).
func refPlane(r *rand.Rand) (p [3][7]float64) {
	unit := func() []float64 {
		v := make([]float64, 7)
		n := 0.0
		for i := range v {
			v[i] = r.NormFloat64()
			n += v[i] * v[i]
		}
		n = math.Sqrt(n)
		for i := range v {
			v[i] /= n
		}
		return v
	}
	x := unit()
	y := unit()
	d := 0.0
	for i := range x {
		d += x[i] * y[i]
	}
	n := 0.0
	for i := range y {
		y[i] -= d * x[i]
		n += y[i] * y[i]
	}
	n = math.Sqrt(n)
	for i := range y {
		y[i] /= n
	}
	x8, y8 := make([]float64, 8), make([]float64, 8)
	copy(x8[1:], x)
	copy(y8[1:], y)
	xy := mdcu.MulRef(x8, y8)
	for k := 0; k < 7; k++ {
		p[0][k], p[1][k], p[2][k] = x[k], y[k], xy[k+1]
	}
	return
}

func refSet(r *rand.Rand, n int) [][3][7]float64 {
	out := make([][3][7]float64, n)
	for i := range out {
		out[i] = refPlane(r)
	}
	return out
}

func bootCI(diff []float64, r *rand.Rand, B int) (lo, hi float64) {
	meds := make([]float64, B)
	buf := make([]float64, len(diff))
	for b := 0; b < B; b++ {
		for i := range buf {
			buf[i] = diff[r.IntN(len(diff))]
		}
		meds[b] = quantile(buf, .5)
	}
	return quantile(meds, .025), quantile(meds, .975)
}

func main() {
	in := flag.String("in", "q1.jsonl", "input JSONL from mdcu-q1")
	reps := flag.Int("mc", 1000, "Monte-Carlo replicates for the spontaneity p-value (pre-registered: 1000)")
	flag.Parse()

	f, err := os.Open(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	by := map[key][]mdcu.Q1Run{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var r mdcu.Q1Run
		if json.Unmarshal(sc.Bytes(), &r) != nil {
			continue
		}
		k := key{r.N, r.Sigma, r.Kind}
		by[k] = append(by[k], r)
	}
	for _, v := range by {
		sort.Slice(v, func(i, j int) bool { return v[i].Seed < v[j].Seed })
	}

	fmt.Println("# Q1 analysis (frozen rules v3)")
	keys := make([]key, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.N != b.N {
			return a.N < b.N
		}
		if a.Sigma != b.Sigma {
			return a.Sigma < b.Sigma
		}
		return a.Kind < b.Kind
	})
	fmt.Println("\n## Distributions (final SVD-3 primary; Local, phi0 diagnostic)")
	for _, k := range keys {
		rs := by[k]
		fmt.Printf("N=%d sigma=%.1f %-8s n=%d\n  %s | >=0.95: %.2f >=0.90: %.2f\n  %s\n  %s\n  %s\n", k.N, k.Sigma, k.Kind, len(rs),
			qline("SVD3", pickQ(rs, func(s mdcu.Q1Stats) float64 { return s.SVD3 })),
			fracGE(pickQ(rs, func(s mdcu.Q1Stats) float64 { return s.SVD3 }), .95),
			fracGE(pickQ(rs, func(s mdcu.Q1Stats) float64 { return s.SVD3 }), .90),
			qline("Local", pickQ(rs, func(s mdcu.Q1Stats) float64 { return s.Local })),
			qline("Phi0Frac", pickQ(rs, func(s mdcu.Q1Stats) float64 { return s.Phi0Frac })),
			qline("PlanePhi0", pickQ(rs, func(s mdcu.Q1Stats) float64 { return s.PlanePhi0 })))
	}

	cellMain := func(kind string) []mdcu.Q1Run { return by[key{24, 0.1, kind}] }
	ov, rand3, none, dis := cellMain("overlap"), cellMain("rand3"), cellMain("none"), cellMain("disjoint")
	if len(ov) == 0 {
		fmt.Println("\nmain cell (N=24, sigma=0.1) not present; no verdict")
		return
	}
	svd := func(rs []mdcu.Q1Run) []float64 { return pickQ(rs, func(s mdcu.Q1Stats) float64 { return s.SVD3 }) }

	fmt.Println("\n## Frozen decision rule, main cell N=24 sigma=0.1 (n =", len(ov), "seeds)")
	c1 := fracGE(svd(ov), .95) >= .90
	fmt.Printf("C1 overlap SVD3>=0.95 in >=90%% of seeds: %.3f -> %v\n", fracGE(svd(ov), .95), c1)
	c2 := true
	for _, p := range []struct {
		n  string
		rs []mdcu.Q1Run
	}{{"none (null 1)", none}, {"disjoint (null 2)", dis}, {"rand3 (null 3)", rand3}} {
		q := quantile(svd(p.rs), .95)
		fmt.Printf("C2 %s 95th percentile SVD3 = %.3f (< 0.80 required)\n", p.n, q)
		if !(q < .80) {
			c2 = false
		}
	}
	fmt.Printf("C2 -> %v\n", c2)
	c3 := true
	for _, N := range []int{12, 48} {
		rs := by[key{N, 0.1, "overlap"}]
		fr := fracGE(svd(rs), .90)
		fmt.Printf("C3 N=%d overlap SVD3>=0.90 in %.3f of seeds (>=0.80 required, n=%d)\n", N, fr, len(rs))
		if len(rs) == 0 || fr < .80 {
			c3 = false
		}
	}
	fmt.Printf("C3 -> %v\n", c3)
	r := rand.New(rand.NewPCG(2026, 1023))
	diff := make([]float64, 0, len(ov))
	for i := range ov {
		if i < len(rand3) && ov[i].Seed == rand3[i].Seed {
			diff = append(diff, ov[i].Final.SVD3-rand3[i].Final.SVD3)
		}
	}
	lo, hi := bootCI(diff, r, 2000)
	med := quantile(diff, .5)
	c4 := med >= .15 && lo > 0
	fmt.Printf("C4 median paired SVD3 gap overlap-rand3 = %.3f, bootstrap 95%% CI [%.3f, %.3f] (needs >=0.15 and CI excluding 0) -> %v\n", med, lo, hi, c4)

	planes := make([][3][7]float64, len(ov))
	for i, rn := range ov {
		planes[i] = rn.Final.Plane
	}
	obs := pairwise(planes)
	rr := rand.New(rand.NewPCG(1000, 0xdeadbeef))
	ref := pairwise(refSet(rr, len(ov)))
	dObs := ksD(obs, ref)
	ge := 0
	for m := 0; m < *reps; m++ {
		a := pairwise(refSet(rr, len(ov)))
		b := pairwise(refSet(rr, len(ov)))
		if ksD(a, b) >= dObs {
			ge++
		}
	}
	pv := float64(ge) / float64(*reps)
	c5 := pv > .05
	fmt.Printf("C5 spontaneity: KS D(observed, Haar-G2 reference)=%.4f, Monte-Carlo p=%.3f (spontaneous iff p>0.05) -> %v\n", dObs, pv, c5)

	noRule := fracGE(svd(dis), .95) >= .5 || fracGE(svd(rand3), .95) >= .5
	fmt.Printf("\nNO-rule: nulls 2 or 3 reach SVD3>=0.95 in >=50%% of seeds: disjoint %.3f, rand3 %.3f -> %v\n", fracGE(svd(dis), .95), fracGE(svd(rand3), .95), noRule)
	verdict := "inconclusive"
	if noRule {
		verdict = "NO"
	} else if c1 && c2 && c3 && c4 && c5 {
		verdict = "YES"
	}
	fmt.Println("\n## Q1 verdict (frozen rules):", verdict)

	fmt.Println("\n## Secondary: Fano-line histogram, overlap main cell (flag if any line > 30%)")
	h := make([]int, 8)
	for _, rn := range ov {
		h[rn.Final.Frame]++
	}
	for l := 1; l <= 7; l++ {
		fl := ""
		if float64(h[l])/float64(len(ov)) > .30 {
			fl = "  <-- >30%"
		}
		fmt.Printf("  line %d: %d (%.1f%%)%s\n", l, h[l], 100*float64(h[l])/float64(len(ov)), fl)
	}
}
