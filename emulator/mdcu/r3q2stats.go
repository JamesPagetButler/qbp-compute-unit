package mdcu

// R3-Q2 host-ns statistics (frozen §3.3).
//
// The frozen text compares benchmarks "with benchstat (default test, α = 0.05)".
// benchstat's default test is the two-sided Mann–Whitney U test. benchstat is
// not installed on the spike box (no network), so MannWhitneyP reimplements the
// same test: exact null distribution when there are no ties, normal
// approximation with tie and continuity correction otherwise. This is disclosed
// in the results; a real benchstat cross-check can be run on the same raw
// `go test -bench` output if the tool becomes available.

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
)

// MannWhitneyP is the two-sided Mann–Whitney U p-value for samples a and b.
func MannWhitneyP(a, b []float64) float64 {
	n1, n2 := len(a), len(b)
	if n1 == 0 || n2 == 0 {
		return math.NaN()
	}
	n := n1 + n2
	all := make([]float64, 0, n)
	all = append(all, a...)
	all = append(all, b...)
	ranks := avgRanks(all)
	r1 := 0.0
	for i := 0; i < n1; i++ {
		r1 += ranks[i]
	}
	u1 := r1 - float64(n1*(n1+1))/2
	u := math.Min(u1, float64(n1*n2)-u1)

	// tie groups
	sorted := append([]float64(nil), all...)
	sort.Float64s(sorted)
	ties := 0.0
	hasTies := false
	for i := 0; i < n; {
		j := i
		for j+1 < n && sorted[j+1] == sorted[i] {
			j++
		}
		if t := float64(j - i + 1); t > 1 {
			hasTies = true
			ties += t*t*t - t
		}
		i = j + 1
	}

	if !hasTies && n1 <= 30 && n2 <= 30 {
		// exact: f[k][s] = number of ways to pick k of the ranks 1..n with sum s
		maxS := n * (n + 1) / 2
		f := make([][]float64, n1+1)
		for k := range f {
			f[k] = make([]float64, maxS+1)
		}
		f[0][0] = 1
		for r := 1; r <= n; r++ {
			for k := min(n1, r); k >= 1; k-- {
				for s := maxS; s >= r; s-- {
					f[k][s] += f[k-1][s-r]
				}
			}
		}
		total, cum := 0.0, 0.0
		umax := int(u + 0.5)
		for s := 0; s <= maxS; s++ {
			total += f[n1][s]
			if s-n1*(n1+1)/2 <= umax {
				cum += f[n1][s]
			}
		}
		return math.Min(1, 2*cum/total)
	}

	mu := float64(n1*n2) / 2
	v := float64(n1*n2) / 12 * (float64(n+1) - ties/float64(n*(n-1)))
	if v <= 0 {
		return 1
	}
	z := (math.Abs(u-mu) - 0.5) / math.Sqrt(v)
	if z < 0 {
		z = 0
	}
	return math.Min(1, math.Erfc(z/math.Sqrt2))
}

// BenchRow: per workload, ns/op samples of the best CD path and best non-CD path.
type BenchRow struct {
	Workload string
	CD       []float64
	NonCD    []float64
}

// BenchVerdict is the frozen benchstat verdict (§3.3):
//
//	yes  iff on ≥ 1 workload the best CD median ≤ 0.5 × best non-CD median, p < 0.05
//	no   iff on every workload the best CD path is not significantly faster
//	else inconclusive
func BenchVerdict(rows []BenchRow) string {
	allNot := true
	for _, r := range rows {
		mc, mn := Median(r.CD), Median(r.NonCD)
		p := MannWhitneyP(r.CD, r.NonCD)
		if mc <= 0.5*mn && p < 0.05 {
			return "yes"
		}
		if mc < mn && p < 0.05 {
			allNot = false
		}
	}
	if allNot {
		return "no"
	}
	return "inconclusive"
}

// LoadOK applies the frozen load ceiling to one run: 1-min loadavg ≤ 1.0 at
// the start and ≤ 1.5 at the end.
func LoadOK(start, end float64) bool { return start <= 1.0 && end <= 1.5 }

var benchLine = regexp.MustCompile(`^(Benchmark\S+?)(?:-\d+)?\s+\d+\s+([0-9.]+)\s+ns/op`)

// ParseBenchOutput collects ns/op samples by benchmark name (cpu suffix stripped).
func ParseBenchOutput(txt string) map[string][]float64 {
	out := map[string][]float64{}
	start := 0
	for i := 0; i <= len(txt); i++ {
		if i == len(txt) || txt[i] == '\n' {
			if m := benchLine.FindStringSubmatch(txt[start:i]); m != nil {
				if v, err := strconv.ParseFloat(m[2], 64); err == nil {
					out[m[1]] = append(out[m[1]], v)
				}
			}
			start = i + 1
		}
	}
	return out
}

var benchName = regexp.MustCompile(`^BenchmarkR3Q2W(\d)(CD|NonCD)(?:_\w+)?$`)

// BenchRows picks, per workload, the best (lowest-median) CD path and the best
// non-CD path from parsed benchmark samples.
func BenchRows(m map[string][]float64) ([]BenchRow, error) {
	type best struct {
		s   []float64
		med float64
	}
	cd, nc := map[string]best{}, map[string]best{}
	for name, s := range m {
		g := benchName.FindStringSubmatch(name)
		if g == nil {
			continue
		}
		w := "W" + g[1]
		tgt := cd
		if g[2] == "NonCD" {
			tgt = nc
		}
		if cur, ok := tgt[w]; !ok || Median(s) < cur.med {
			tgt[w] = best{s, Median(s)}
		}
	}
	var rows []BenchRow
	for _, w := range []string{"W1", "W2", "W3"} {
		c, okc := cd[w]
		n, okn := nc[w]
		if !okc || !okn {
			return nil, fmt.Errorf("workload %s: missing CD or non-CD benchmark samples", w)
		}
		rows = append(rows, BenchRow{w, c.s, n.s})
	}
	return rows, nil
}
