// Command mdcu-r3-comass prints the POST-HOC comass diagnostic for the R3-Q1 grid.
//
// POST-HOC. It is not part of the frozen R3-Q1 analysis and changes no verdict
// (the frozen analyzer's output is doc/spikes/raw/mdcu-r3-analysis-2026-10-26.md).
// It joins per-seed comass quantities, recomputed from each seed's own sampler
// streams (mdcu/r3comass.go), with the raw main-cell outcomes, and reports
// descriptive associations only. No ring runs are performed.
//
//	go run ./cmd/mdcu-r3-comass -in ../doc/spikes/raw/mdcu-r3-2026-10-26.jsonl
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
	Seed   uint64   `json:"seed"`
	S2Dist *float64 `json:"s2_dist"`
}

type runLine struct {
	Type string `json:"type"`
	mdcu.R3Run
}

func load(path string) (seeds []uint64, dist map[uint64]float64, runs map[mdcu.R3Arm]map[uint64]mdcu.R3Run, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}
	defer f.Close()
	dist = map[uint64]float64{}
	runs = map[mdcu.R3Arm]map[uint64]mdcu.R3Run{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<22), 1<<22)
	for sc.Scan() {
		var h struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(sc.Bytes(), &h) != nil {
			continue
		}
		switch h.Type {
		case "psi":
			var p psiRec
			if json.Unmarshal(sc.Bytes(), &p) == nil {
				seeds = append(seeds, p.Seed)
				if p.S2Dist != nil {
					dist[p.Seed] = *p.S2Dist
				}
			}
		case "run":
			var r runLine
			if json.Unmarshal(sc.Bytes(), &r) == nil && r.Cell == "main" {
				if runs[r.Arm] == nil {
					runs[r.Arm] = map[uint64]mdcu.R3Run{}
				}
				runs[r.Arm][r.Seed] = r.R3Run
			}
		}
	}
	sort.Slice(seeds, func(i, j int) bool { return seeds[i] < seeds[j] })
	return seeds, dist, runs, sc.Err()
}

func median(s []float64) float64 { return mdcu.Median(s) }

func quant(s []float64, q float64) float64 {
	c := append([]float64(nil), s...)
	sort.Float64s(c)
	if len(c) == 0 {
		return math.NaN()
	}
	return c[int(math.Round(q*float64(len(c)-1)))]
}

// partial is the first-order partial Spearman rho(x,y | z).
func partial(x, y, z []float64) float64 {
	rxy, rxz, ryz := mdcu.Spearman(x, y), mdcu.Spearman(x, z), mdcu.Spearman(y, z)
	return (rxy - rxz*ryz) / math.Sqrt((1-rxz*rxz)*(1-ryz*ryz))
}

type row struct {
	ph  mdcu.PostHoc
	run map[mdcu.R3Arm]mdcu.R3Run
}

func lockRate(rs []mdcu.R3Run) float64 { return mdcu.LockRate(rs) }

func tercileTable(title string, rows []row, comass func(row) float64, arms []mdcu.R3Arm) {
	if len(rows) < 30 {
		return
	}
	idx := make([]int, len(rows))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return comass(rows[idx[a]]) < comass(rows[idx[b]]) })
	fmt.Printf("\n%s (n=%d)\n\n| comass tercile | comass range |", title, len(rows))
	for _, a := range arms {
		fmt.Printf(" %s 3-lock | %s median S3 |", a, a)
	}
	fmt.Println("\n|---|---|" + func() (s string) {
		for range arms {
			s += "---|---|"
		}
		return
	}())
	for t := 0; t < 3; t++ {
		lo, hi := t*len(idx)/3, (t+1)*len(idx)/3
		var cs []float64
		sel := map[mdcu.R3Arm][]mdcu.R3Run{}
		for _, i := range idx[lo:hi] {
			cs = append(cs, comass(rows[i]))
			for _, a := range arms {
				if r, ok := rows[i].run[a]; ok {
					sel[a] = append(sel[a], r)
				}
			}
		}
		fmt.Printf("| %d | %.3f – %.3f |", t+1, quant(cs, 0), quant(cs, 1))
		for _, a := range arms {
			var s3 []float64
			for _, r := range sel[a] {
				s3 = append(s3, r.Stats.S3)
			}
			fmt.Printf(" %.1f%% | %.3f |", 100*lockRate(sel[a]), median(s3))
		}
		fmt.Println()
	}
}

func main() {
	in := flag.String("in", "", "R3 raw JSONL (grid output)")
	flag.Parse()
	seeds, dist, runs, err := load(*in)
	if err != nil || len(seeds) == 0 {
		fmt.Fprintln(os.Stderr, "mdcu-r3-comass: cannot read grid input:", err)
		os.Exit(1)
	}

	fmt.Println("# R3-Q1 comass diagnostic — POST-HOC")
	fmt.Println()
	fmt.Println("**POST-HOC.** Not part of the frozen R3-Q1 analysis; no verdict, reading row or rule changes.")
	fmt.Println("Per-seed comass quantities are recomputed from each seed's own sampler streams (the frozen samplers are")
	fmt.Println("called unchanged); no ring runs. Associations are descriptive. `comass` here = max |f(u,v,w)| over orthonormal")
	fmt.Println("triples (multistart ascent, so a lower bound); comass(φ) = 1, so the number is also the ratio to φ.")
	fmt.Printf("\nInput: `%s` (%d seeds).\n", *in, len(seeds))

	phiC := mdcu.ComassEstimate(mdcu.Phi(), rand.New(rand.NewPCG(1, 2)), 40, 300)
	fmt.Printf("\n**Calibration**: estimator on φ = %.9f (exact: 1).\n", phiC)

	var rows []row
	for _, s := range seeds {
		r := row{ph: mdcu.PostHocComass(s), run: map[mdcu.R3Arm]mdcu.R3Run{}}
		for a, m := range runs {
			if x, ok := m[s]; ok {
				r.run[a] = x
			}
		}
		rows = append(rows, r)
	}

	var rel []float64
	for _, r := range rows {
		if r.ph.PsiComass > 0 {
			rel = append(rel, math.Abs(r.ph.PsiComass-r.ph.PsiComassChk)/r.ph.PsiComass)
		}
	}
	fmt.Printf("**Convergence check** (two independent restart sets per form): max relative difference %.2e, median %.2e.\n", quant(rel, 1), median(rel))

	var def, spl []row
	for _, r := range rows {
		if r.ph.Stratum == "definite" {
			def = append(def, r)
		} else {
			spl = append(spl, r)
		}
	}
	col := func(rs []row, f func(row) float64) (o []float64) {
		for _, r := range rs {
			o = append(o, f(r))
		}
		return
	}

	fmt.Println("\n## Comass of the ψ forms actually used (Frobenius-matched to φ)")
	fmt.Println("\n| form | n | min | median | max |")
	fmt.Println("|---|---|---|---|---|")
	line := func(name string, v []float64) {
		fmt.Printf("| %s | %d | %.3f | %.3f | %.3f |\n", name, len(v), quant(v, 0), median(v), quant(v, 1))
	}
	line("K-psi, definite seeds (S2)", col(def, func(r row) float64 { return r.ph.PsiComass }))
	line("K-psi-S1, definite seeds (S1-definite)", col(def, func(r row) float64 { return r.ph.PsiS1Comass }))
	line("K-psi, split seeds (S1-split)", col(spl, func(r row) float64 { return r.ph.PsiComass }))
	line("‖g‖_op³ (S2 bound, pre-match)", col(def, func(r row) float64 { return r.ph.GOp3 }))
	cc := col(def, func(r row) float64 { return r.ph.PsiComass })
	var sq []float64
	for _, c := range cc {
		sq = append(sq, c*c)
	}
	fmt.Printf("\nDescriptive: with −4ψ², the peak coupling is 4·comass², i.e. comass² times φ's. Median comass² on S2 = %.2f, on S1-definite = %.2f, on S1-split = %.2f.\n",
		median(sq), median(sqs(col(def, func(r row) float64 { return r.ph.PsiS1Comass }))), median(sqs(col(spl, func(r row) float64 { return r.ph.PsiComass }))))
	pooled := append(col(def, func(r row) float64 { return r.ph.PsiComass }), col(spl, func(r row) float64 { return r.ph.PsiComass })...)
	fmt.Printf("\nPooled K-psi forms (S2 on definite seeds + S1-split on split seeds, n=%d): comass median %.3f [%.3f, %.3f], comass² median %.3f.\n",
		len(pooled), median(pooled), quant(pooled, 0), quant(pooled, 1), median(sqs(pooled)))

	d := col(def, func(r row) float64 { return dist[r.ph.Seed] })
	sv := col(def, func(r row) float64 { return r.ph.S2S })
	g3 := col(def, func(r row) float64 { return r.ph.GOp3 })
	fmt.Println("\n## Does comass grow with the deformation? (definite seeds, S2)")
	fmt.Printf("\n- Spearman(comass, d) = %.3f; Spearman(comass, s) = %.3f; Spearman(‖g‖_op³, d) = %.3f; Spearman(‖g‖_op³, comass) = %.3f (n=%d)\n",
		mdcu.Spearman(cc, d), mdcu.Spearman(cc, sv), mdcu.Spearman(g3, d), mdcu.Spearman(g3, cc), len(cc))

	fmt.Println("\n## Does comass track locking? (main cell, per stratum)")
	sat, tot := 0, 0
	for _, r := range rows {
		if x, ok := r.run[mdcu.ArmPsi]; ok {
			tot++
			if x.Stats.S3 >= 0.999 {
				sat++
			}
		}
	}
	fmt.Printf("\nCaveat: S3 is saturated near 1 (%d of %d K-psi runs have S3 ≥ 0.999), so rank correlations on S3 are driven by tiny differences;\nthe 3-lock rate by tercile below is the cleaner view.\n", sat, tot)
	for _, st := range []struct {
		name string
		rs   []row
	}{{"definite", def}, {"split", spl}} {
		var c, s3, s3g []float64
		for _, r := range st.rs {
			p, ok1 := r.run[mdcu.ArmPsi]
			g, ok2 := r.run[mdcu.ArmPsiGram]
			if ok1 && ok2 {
				c, s3, s3g = append(c, r.ph.PsiComass), append(s3, p.Stats.S3), append(s3g, g.Stats.S3)
			}
		}
		fmt.Printf("\n- **%s** (n=%d): Spearman(comass, S3 K-psi) = %.3f; Spearman(comass, S3 K-psigram) = %.3f\n", st.name, len(c), mdcu.Spearman(c, s3), mdcu.Spearman(c, s3g))
	}
	tercileTable("### K-psi / K-psigram by comass tercile — definite (S2)", def, func(r row) float64 { return r.ph.PsiComass }, []mdcu.R3Arm{mdcu.ArmPsi, mdcu.ArmPsiGram})
	tercileTable("### K-psi-S1 by comass tercile — definite (S1-definite form)", def, func(r row) float64 { return r.ph.PsiS1Comass }, []mdcu.R3Arm{mdcu.ArmPsiS1})
	tercileTable("### K-psi / K-psigram by comass tercile — split (S1-split)", spl, func(r row) float64 { return r.ph.PsiComass }, []mdcu.R3Arm{mdcu.ArmPsi, mdcu.ArmPsiGram})

	fmt.Println("\n## Dose-response, with and without comass (definite seeds)")
	var gap, dd, cm []float64
	for _, r := range def {
		ph, ok1 := r.run[mdcu.ArmPhi]
		ps, ok2 := r.run[mdcu.ArmPsi]
		if ok1 && ok2 {
			gap = append(gap, ph.Stats.S3-ps.Stats.S3)
			dd = append(dd, dist[r.ph.Seed])
			cm = append(cm, r.ph.PsiComass)
		}
	}
	fmt.Printf("\nSame quantity as the frozen analyzer, S3(K-phi) − S3(K-psi) vs d: n=%d, ρ = %.3f (the frozen analyzer reports −0.258).\n", len(gap), mdcu.Spearman(gap, dd))
	fmt.Printf("- Spearman(gap, comass) = %.3f\n- Partial Spearman(gap, d | comass) = %.3f\n", mdcu.Spearman(gap, cm), partial(gap, dd, cm))
	fmt.Printf("\nCaveats: (1) comass and d are rank-collinear on S2 (Spearman %.3f), so the partial is poorly determined and cannot separate them.\n", mdcu.Spearman(cc, d))
	fmt.Printf("(2) Every ψ form has comass above φ's (φ = 1.000; smallest ψ = %.3f), so R3 has no ψ at φ's coupling scale: the comass\n", math.Min(quant(cc, 0), math.Min(quant(col(def, func(r row) float64 { return r.ph.PsiS1Comass }), 0), quant(col(spl, func(r row) float64 { return r.ph.PsiComass }), 0))))
	fmt.Println("confound between arms cannot be tested within R3 data. Within-form comass variation is the only handle, and it is weak and non-monotone.")
	fmt.Println("R4's rescaled-ψ arm (max|ψ| = 1) is the pre-registered test; this report only motivates it. No verdict is changed.")
}

func sqs(v []float64) (o []float64) {
	for _, x := range v {
		o = append(o, x*x)
	}
	return
}
