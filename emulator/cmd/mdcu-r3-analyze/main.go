// Command mdcu-r3-analyze evaluates the FROZEN round-3 R3-Q1 rules
// (doc/spikes/prereg/mdcu-round3-prereg-FROZEN-v1.md) on the JSONL written by
// mdcu-r3. The decision logic lives in mdcu/r3rules.go and is unit-tested on
// synthetic data; this command only reads, groups and prints.
//
//	go run ./cmd/mdcu-r3-analyze -in ../doc/spikes/raw/mdcu-r3-<date>.jsonl
//
// Interpretation choices NOT spelled out in the frozen text (documented here,
// fixed before any grid output exists):
//   - Q1b rates/Δ use a stratum's paired seeds (both K-phi and K-psi present);
//     the φ-only precondition (rate ≥ 50%) uses K-phi on those same seeds.
//   - Spontaneity KS is computed per arm with ≥ 20 three-locking runs in the
//     main cell, on those runs' principal planes (the round-2 definition
//     assumed every seed locked).
//   - R3-Q2 is analysed by a separate command (cmd/mdcu-r3-q2).
//
// A "smoke" input is analysed for pipeline testing only and is labelled so.
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

type psiRec struct {
	Seed         uint64   `json:"seed"`
	Stratum      string   `json:"stratum"`
	Sampler      string   `json:"sampler"`
	S1Draws      int      `json:"s1_draws"`
	S1Degenerate int      `json:"s1_degenerate"`
	S1Miss       bool     `json:"s1_miss"`
	S2S          *float64 `json:"s2_s"`
	S2Dist       *float64 `json:"s2_dist"`
	S2Cond       *float64 `json:"s2_cond"`
}

type line struct {
	Type string `json:"type"`
}
type runLine struct {
	Type string `json:"type"`
	mdcu.R3Run
}

type data struct {
	meta  map[string]any
	psi   map[uint64]psiRec
	runs  map[string]map[mdcu.R3Arm][]mdcu.R3Run // cell → arm → runs sorted by seed
	ended string
}

func load(path string) (*data, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	d := &data{psi: map[uint64]psiRec{}, runs: map[string]map[mdcu.R3Arm][]mdcu.R3Run{}}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<22), 1<<22)
	for sc.Scan() {
		var l line
		if json.Unmarshal(sc.Bytes(), &l) != nil {
			continue
		}
		switch l.Type {
		case "meta":
			_ = json.Unmarshal(sc.Bytes(), &d.meta)
		case "psi":
			var p psiRec
			if json.Unmarshal(sc.Bytes(), &p) == nil {
				d.psi[p.Seed] = p
			}
		case "run":
			var r runLine
			if json.Unmarshal(sc.Bytes(), &r) == nil {
				if d.runs[r.Cell] == nil {
					d.runs[r.Cell] = map[mdcu.R3Arm][]mdcu.R3Run{}
				}
				d.runs[r.Cell][r.Arm] = append(d.runs[r.Cell][r.Arm], r.R3Run)
			}
		case "end", "abort":
			var m map[string]any
			_ = json.Unmarshal(sc.Bytes(), &m)
			d.ended = fmt.Sprintf("%s: %v", l.Type, m["reason"])
		}
	}
	for _, am := range d.runs {
		for _, rs := range am {
			sort.Slice(rs, func(i, j int) bool { return rs[i].Seed < rs[j].Seed })
		}
	}
	return d, sc.Err()
}

func (d *data) get(cell string, arm mdcu.R3Arm, stratum string) []mdcu.R3Run {
	var out []mdcu.R3Run
	for _, r := range d.runs[cell][arm] {
		if stratum == "" || r.Stratum == stratum {
			out = append(out, r)
		}
	}
	return out
}

func classRate(rs []mdcu.R3Run, class string) float64 {
	if len(rs) == 0 {
		return math.NaN()
	}
	n := 0
	for _, r := range rs {
		if r.Stats.Class == class {
			n++
		}
	}
	return float64(n) / float64(len(rs))
}

func col(rs []mdcu.R3Run, f func(mdcu.R3Stats) float64) []float64 {
	o := make([]float64, len(rs))
	for i, r := range rs {
		o[i] = f(r.Stats)
	}
	return o
}

func mean(s []float64) float64 {
	if len(s) == 0 {
		return math.NaN()
	}
	t := 0.0
	for _, v := range s {
		t += v
	}
	return t / float64(len(s))
}

func pct(x float64) string {
	if math.IsNaN(x) {
		return "  n/a"
	}
	return fmt.Sprintf("%5.1f%%", 100*x)
}

func armTable(d *data, cell, stratum string) {
	fmt.Println("| arm | n | 3-lock | 2-collapse | diffuse | median S3 | median λ₃ share |")
	fmt.Println("|---|---|---|---|---|---|---|")
	for _, a := range mdcu.R3AllArms {
		rs := d.get(cell, a, stratum)
		if len(rs) == 0 {
			continue
		}
		fmt.Printf("| %s | %d | %s | %s | %s | %.3f | %.3f |\n", a, len(rs),
			pct(classRate(rs, "3-lock")), pct(classRate(rs, "2-collapse")), pct(classRate(rs, "diffuse")),
			mdcu.Median(col(rs, func(s mdcu.R3Stats) float64 { return s.S3 })),
			mdcu.Median(col(rs, func(s mdcu.R3Stats) float64 { return s.L3Share })))
	}
}

func contrastLine(c mdcu.ContrastStat) string {
	return fmt.Sprintf("%s: n=%d rates %s vs %s, Δ median %+.3f, 95%% CI [%+.3f, %+.3f]",
		c.Label, c.N, pct(c.RateA), pct(c.RateB), c.Median, c.Lo, c.Hi)
}

// ---- round-2 spontaneity machinery (copied verbatim from cmd/mdcu-q1-analyze) ----

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

func spontaneity(planes [][3][7]float64, reps int) (dObs, p float64) {
	obs := pairwise(planes)
	rr := rand.New(rand.NewPCG(1000, 0xdeadbeef))
	dObs = ksD(obs, pairwise(refSet(rr, len(planes))))
	ge := 0
	for m := 0; m < reps; m++ {
		if ksD(pairwise(refSet(rr, len(planes))), pairwise(refSet(rr, len(planes)))) >= dObs {
			ge++
		}
	}
	return dObs, float64(ge) / float64(reps)
}

func main() {
	in := flag.String("in", "mdcu-r3-smoke.jsonl", "input JSONL from mdcu-r3")
	reps := flag.Int("mc", 1000, "Monte-Carlo replicates for the spontaneity p-value (round-2 definition: 1000)")
	skipKS := flag.Bool("skip-ks", false, "skip the spontaneity secondary")
	flag.Parse()
	d, err := load(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	mode := fmt.Sprint(d.meta["mode"])
	fmt.Println("# R3-Q1 analysis (frozen rules, prereg sha256 070aeab8…c1848)")
	fmt.Printf("input %s · mode=%s · first_seed=%v seeds=%v · run end: %s\n", *in, mode, d.meta["first_seed"], d.meta["seeds"], d.ended)
	if mode != "grid" {
		fmt.Println("\n> **SMOKE / NON-GRID DATA — pipeline test only. Nothing below is an R3 result; no verdict may be quoted.**")
	}
	if d.ended == "" || d.ended[:3] != "end" {
		fmt.Println("\n> **INCOMPLETE RUN (no clean end record): verdicts below are over the cells that finished and must be reported as such.**")
	}

	// ---- sampler record ----------------------------------------------------
	fmt.Println("\n## Sampler record (reported alongside every verdict, arch seq 42)")
	bins := []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384}
	for _, st := range []string{"definite", "split"} {
		var draws []float64
		hist := make([]int, len(bins)+1)
		miss, n := 0, 0
		for _, p := range d.psi {
			if p.Stratum != st {
				continue
			}
			n++
			if p.S1Miss {
				miss++
			}
			draws = append(draws, float64(p.S1Draws))
			b := 0
			for b < len(bins) && p.S1Draws >= bins[b] {
				b++
			}
			if b > 0 {
				hist[b-1]++
			}
		}
		if n == 0 {
			continue
		}
		sort.Float64s(draws)
		role := "S1 (sensitivity K-psi-S1; primary ψ is S2)"
		if st == "split" {
			role = "S1 (primary)"
		}
		fmt.Printf("- **%s** (%d seeds, %s): S1 draws min %.0f · median %.0f · max %.0f · mean %.0f · misses %d\n  histogram (draws ≥ bin): ", st, n, role, draws[0], mdcu.Median(draws), draws[len(draws)-1], mean(draws), miss)
		for i, c := range hist[:len(bins)] {
			if c > 0 {
				fmt.Printf("[%d+]=%d ", bins[i], c)
			}
		}
		fmt.Println()
	}
	var sS, sD, sC []float64
	for _, p := range d.psi {
		if p.S2S != nil {
			sS, sD, sC = append(sS, *p.S2S), append(sD, *p.S2Dist), append(sC, *p.S2Cond)
		}
	}
	if len(sS) > 0 {
		fmt.Printf("- **S2** (definite seeds, n=%d): s median %.3f · distance d median %.3f [%.3f, %.3f] · metric condition median %.1f\n",
			len(sS), mdcu.Median(sS), mdcu.Median(sD), quant(sD, 0), quant(sD, 1), mdcu.Median(sC))
	}

	// ---- main cell ----------------------------------------------------------
	if len(d.runs["main"]) == 0 {
		fmt.Println("\nmain cell not present; no verdict")
		return
	}
	fmt.Println("\n## Main cell (N=24, σ=0.1, 250 steps) — all seeds")
	armTable(d, "main", "")
	for _, st := range []string{"definite", "split"} {
		fmt.Printf("\n### stratum: %s\n", st)
		armTable(d, "main", st)
	}

	full, gram := d.get("main", mdcu.ArmFull, ""), d.get("main", mdcu.ArmGram, "")
	gateRate := mdcu.LockRate(full)
	gate := gateRate >= 0.90-1e-9
	fmt.Printf("\n## Decision rules\n\n**Replication gate**: K-full 3-lock rate %s (needs ≥ 90%%) → **%v**\n", pct(gateRate), map[bool]string{true: "pass", false: "FAIL (all R3-Q1 verdicts: inconclusive, non-replication)"}[gate])

	cA := mdcu.R3Contrast("Δ(K-full, K-gram)", full, gram)
	q1a := mdcu.VerdictQ1a(cA, cA.RateB)
	fmt.Printf("\n**Q1a** — %s\n→ **%s**\n", contrastLine(cA), q1a)

	q1b := map[string]string{}
	for _, st := range []string{"definite", "split"} {
		phi, psi := d.get("main", mdcu.ArmPhi, st), d.get("main", mdcu.ArmPsi, st)
		pphi, ppsi := mdcu.PairBySeed(phi, psi)
		c := mdcu.R3Contrast("Δ(K-phi, K-psi) "+st, phi, psi)
		q1b[st] = mdcu.VerdictQ1b(c, pphi, ppsi)
		fmt.Printf("\n**Q1b (%s)** — %s\n→ **%s**\n", st, contrastLine(c), q1b[st])
	}
	if !gate {
		q1a, q1b["definite"], q1b["split"] = "inconclusive (non-replication)", "inconclusive (non-replication)", "inconclusive (non-replication)"
	}
	fmt.Printf("\n**Reading** (Q1a=%s, Q1b definite=%s, Q1b split=%s):\n> %s\n", q1a, q1b["definite"], q1b["split"], mdcu.ReadingRow(gate, q1a, q1b["definite"], q1b["split"]))

	// ---- secondaries ----------------------------------------------------------
	fmt.Println("\n## Secondaries (diagnostic only; never a decision input)")

	// Dose-response first (oppenheimer's request).
	phiD, psiD := mdcu.PairBySeed(d.get("main", mdcu.ArmPhi, "definite"), d.get("main", mdcu.ArmPsi, "definite"))
	var gap, dist []float64
	for i := range phiD {
		if p, ok := d.psi[phiD[i].Seed]; ok && p.S2Dist != nil {
			gap = append(gap, phiD[i].Stats.S3-psiD[i].Stats.S3)
			dist = append(dist, *p.S2Dist)
		}
	}
	fmt.Printf("\n**Dose-response** (definite seeds; Spearman of paired S3(K-phi)−S3(K-psi) vs S2 distance d): n=%d, ρ=%.3f\n", len(gap), mdcu.Spearman(gap, dist))
	fmt.Println("(Euclidean-vs-g_ψ mismatch is the Q1b question: ψ is measured in the Euclidean ring while its own metric g_ψ differs, per qbp-oppenheimer.)")

	sec := func(a, b mdcu.R3Arm, st string) {
		fmt.Printf("- %s\n", contrastLine(mdcu.R3Contrast(fmt.Sprintf("Δ(%s, %s) %s", a, b, orAll(st)), d.get("main", a, st), d.get("main", b, st))))
	}
	fmt.Println("\n**K-phi vs K-full / K-psigram vs K-full**")
	sec(mdcu.ArmPhi, mdcu.ArmFull, "")
	for _, st := range []string{"definite", "split"} {
		sec(mdcu.ArmPsiGram, mdcu.ArmFull, st)
	}

	// S1 sensitivity: Q1b rule with K-psi-S1 on definite seeds.
	phiS, s1 := d.get("main", mdcu.ArmPhi, "definite"), d.get("main", mdcu.ArmPsiS1, "definite")
	if len(s1) > 0 {
		pa, pb := mdcu.PairBySeed(phiS, s1)
		c := mdcu.R3Contrast("Δ(K-phi, K-psi-S1) definite", phiS, s1)
		fmt.Printf("\n**Sampler sensitivity** (Q1b rule with K-psi-S1; not a decision input) — %s\n→ %s\n", contrastLine(c), mdcu.VerdictQ1b(c, pa, pb))
	}

	fmt.Println("\n**Cross-calibration** (main cell; own-form calibration is tautological) — mean |φ₀| plane / |φ₀|≥0.95 triple fraction / |ψ₀| plane / |ψ₀|≥0.95 fraction")
	fmt.Println("| arm | stratum | φ₀ plane | φ₀ frac | ψ₀ plane | ψ₀ frac |")
	fmt.Println("|---|---|---|---|---|---|")
	for _, a := range mdcu.R3AllArms {
		for _, st := range []string{"definite", "split"} {
			rs := d.get("main", a, st)
			if len(rs) == 0 {
				continue
			}
			fmt.Printf("| %s | %s | %.3f | %.3f | %.3f | %.3f |\n", a, st,
				mean(col(rs, func(s mdcu.R3Stats) float64 { return s.PlanePhi0 })),
				mean(col(rs, func(s mdcu.R3Stats) float64 { return s.Phi0Frac })),
				mean(col(rs, func(s mdcu.R3Stats) float64 { return s.PlanePsi0 })),
				mean(col(rs, func(s mdcu.R3Stats) float64 { return s.Psi0Frac })))
		}
	}
	fmt.Println("(|ψ₀| is not bounded by 1: ψ is Frobenius-matched to φ's norm but is not Euclidean-compatible.)")

	fmt.Println("\n**Persistence** (secondary; same rules applied for information only)")
	for _, cell := range []string{"N12", "N48"} {
		if len(d.runs[cell]) == 0 {
			continue
		}
		fmt.Printf("\n### %s (250 steps)\n", cell)
		armTable(d, cell, "")
		f, g := d.get(cell, mdcu.ArmFull, ""), d.get(cell, mdcu.ArmGram, "")
		ca := mdcu.R3Contrast("Δ(K-full, K-gram) "+cell, f, g)
		fmt.Printf("\n- Q1a-rule: %s → %s\n", contrastLine(ca), mdcu.VerdictQ1a(ca, ca.RateB))
		for _, st := range []string{"definite", "split"} {
			phi, psi := d.get(cell, mdcu.ArmPhi, st), d.get(cell, mdcu.ArmPsi, st)
			pa, pb := mdcu.PairBySeed(phi, psi)
			c := mdcu.R3Contrast("Δ(K-phi, K-psi) "+cell+" "+st, phi, psi)
			fmt.Printf("- Q1b-rule %s: %s → %s\n", st, contrastLine(c), mdcu.VerdictQ1b(c, pa, pb))
		}
	}

	if len(d.runs["N48s1000"]) > 0 {
		fmt.Println("\n**N=48 convergence (K-full, K-gram at 250 vs 1000 steps)**")
		fmt.Println("| arm | steps | n | 3-lock | median S3 |")
		fmt.Println("|---|---|---|---|---|")
		for _, a := range []mdcu.R3Arm{mdcu.ArmFull, mdcu.ArmGram} {
			for _, p := range []struct {
				cell  string
				steps int
			}{{"N48", 250}, {"N48s1000", 1000}} {
				rs := d.get(p.cell, a, "")
				if len(rs) > 0 {
					fmt.Printf("| %s | %d | %d | %s | %.3f |\n", a, p.steps, len(rs), pct(mdcu.LockRate(rs)), mdcu.Median(col(rs, func(s mdcu.R3Stats) float64 { return s.S3 })))
				}
			}
		}
	}

	if !*skipKS {
		fmt.Printf("\n**Spontaneity** (round-2 definition: KS D of pairwise plane distances vs Haar-G₂ reference, MC p over %d reps; arms with ≥ 20 three-locking runs; spontaneous iff p > 0.05)\n", *reps)
		for _, a := range mdcu.R3AllArms {
			var planes [][3][7]float64
			for _, r := range d.get("main", a, "") {
				if r.Stats.Class == "3-lock" {
					planes = append(planes, r.Stats.Plane)
				}
			}
			if len(planes) < 20 {
				continue
			}
			dObs, p := spontaneity(planes, *reps)
			fmt.Printf("- %s: n=%d locked planes, D=%.4f, p=%.3f → %s\n", a, len(planes), dObs, p, map[bool]string{true: "consistent with Haar-G₂ (spontaneous)", false: "not Haar-G₂ (non-spontaneous)"}[p > 0.05])
		}
	}

	fmt.Println("\n**On-file prediction (r1 + addendum, unedited in the frozen file):** Q1a = no; Q1b = no in both strata; reading row 2. Score it against the verdicts above in the results document.")
}

func orAll(s string) string {
	if s == "" {
		return "all"
	}
	return s
}

func quant(s []float64, q float64) float64 {
	c := append([]float64(nil), s...)
	sort.Float64s(c)
	if len(c) == 0 {
		return math.NaN()
	}
	return c[int(q*float64(len(c)-1))]
}
