// Command mdcu-r4-analyze evaluates the FROZEN round-4 rules
// (doc/spikes/prereg/mdcu-round4-prereg-FROZEN-v1.md, sha256 699a8a21…587a5d) on the
// JSONL written by mdcu-r4. The decision logic lives in mdcu/r4rules.go and
// mdcu/r4spont.go and is unit-tested on synthetic data; this command only reads,
// groups and prints.
//
//	go run ./cmd/mdcu-r4-analyze -in ../doc/spikes/raw/mdcu-r4-<date>.jsonl
//
// The interpretation choices the frozen text does not spell out are listed at the top
// of mdcu/r4rules.go and printed with every report. A "smoke" input is analysed for
// pipeline testing only and is labelled so: no verdict may be quoted from it.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/JamesPagetButler/qbp-compute-unit/emulator/mdcu"
)

func pct(x float64) string {
	if math.IsNaN(x) {
		return "  n/a"
	}
	return fmt.Sprintf("%5.1f%%", 100*x)
}

func col(rs []mdcu.R3Run, f func(mdcu.R3Stats) float64) []float64 {
	o := make([]float64, len(rs))
	for i, r := range rs {
		o[i] = f(r.Stats)
	}
	return o
}

func contrastLine(c mdcu.ContrastStat) string {
	return fmt.Sprintf("%s: n=%d rates %s vs %s, Δ median %+.3f, 95%% CI [%+.3f, %+.3f]", c.Label, c.N, pct(c.RateA), pct(c.RateB), c.Median, c.Lo, c.Hi)
}

func quant(s []float64, q float64) float64 {
	c := append([]float64(nil), s...)
	sort.Float64s(c)
	if len(c) == 0 {
		return math.NaN()
	}
	return c[int(q*float64(len(c)-1))]
}

var cellOrder = []string{"A250", "B1000", "C4000", "SD250", "SD1000"}

func armTable(d *mdcu.R4Data, cell, stratum string) {
	fmt.Println("| arm | n | 3-lock | 2-collapse | diffuse | median S3 | median λ₃ share |")
	fmt.Println("|---|---|---|---|---|---|---|")
	var arms []mdcu.R3Arm
	for a := range d.Runs[cell] {
		arms = append(arms, a)
	}
	order := map[mdcu.R3Arm]int{}
	for i, a := range append(append(append([]mdcu.R3Arm{}, mdcu.R4TierAArms...), mdcu.ArmPsi1), mdcu.ArmPhi) {
		if _, ok := order[a]; !ok {
			order[a] = i
		}
	}
	sort.Slice(arms, func(i, j int) bool { return order[arms[i]] < order[arms[j]] })
	for _, a := range arms {
		rs := d.Get(cell, a, stratum)
		if len(rs) == 0 {
			continue
		}
		cls := func(c string) float64 {
			n := 0
			for _, r := range rs {
				if r.Stats.Class == c {
					n++
				}
			}
			return float64(n) / float64(len(rs))
		}
		fmt.Printf("| %s | %d | %s | %s | %s | %.3f | %.3f |\n", a, len(rs), pct(cls("3-lock")), pct(cls("2-collapse")), pct(cls("diffuse")),
			mdcu.Median(col(rs, func(s mdcu.R3Stats) float64 { return s.S3 })), mdcu.Median(col(rs, func(s mdcu.R3Stats) float64 { return s.L3Share })))
	}
}

func main() {
	in := flag.String("in", "mdcu-r4-smoke.jsonl", "input JSONL from mdcu-r4")
	reps := flag.Int("mc", 1000, "Monte-Carlo replicates for the spontaneity p-values (round-2/3 definition: 1000)")
	skipKS := flag.Bool("skip-ks", false, "skip the spontaneity secondaries (and the mechanism kills that need them)")
	flag.Parse()
	d, m, err := mdcu.LoadR4(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	mode := fmt.Sprint(m.Meta["mode"])
	fmt.Println("# R4 analysis (frozen rules, prereg sha256 699a8a21…587a5d)")
	fmt.Printf("input %s · mode=%s · grid seeds=%v · S-D seeds=%v · Tier C seeds=%v · run end: %s\n", *in, mode, m.Meta["grid_seeds"], m.Meta["sd_seeds"], m.Meta["tierc_seeds"], m.Ended)
	if mode != "grid" {
		fmt.Println("\n> **SMOKE / NON-GRID DATA — pipeline test only. Nothing below is an R4 result; no verdict may be quoted.**")
	}
	if !strings.HasPrefix(m.Ended, "end") {
		fmt.Println("\n> **INCOMPLETE RUN (no clean end record): verdicts below are over what finished and must be reported as such.**")
	}
	fmt.Println("\n> No CD speed claim is permitted at any reading. Branch B, spike/napkin grade. Interpretation choices not spelled out in the frozen text are fixed in mdcu/r4rules.go (header comment) and apply here.")

	// ---- seed record, kill conditions ---------------------------------------------
	k := mdcu.R4Kills(d.Seeds)
	fmt.Println("\n## Sampler and comass record")
	fmt.Printf("- S1-split sampler: %d misses in %d calls (%.1f%%) → %s\n", k.Misses, k.S1Calls, 100*k.MissRate, map[bool]string{true: "**KILL: inconclusive (sampler)** (> 5%)", false: "ok"}[k.Sampler])
	fmt.Printf("- Comass two-set check: %d unconverged of %d grid seeds with a ψ (%.1f%%) → %s\n", k.Unconverged, k.WithPsi, 100*k.UnconvRate, map[bool]string{true: "**KILL: rescaled arms inconclusive (comass)** (> 5%)", false: "ok"}[k.Comass])
	var comass, rel []float64
	sdUnconv, sdN := 0, 0
	var sdSeeds []uint64
	for s, sd := range d.Seeds {
		switch sd.Kind {
		case "grid":
			if !sd.Miss {
				comass, rel = append(comass, sd.Comass), append(rel, sd.RelDiff)
			}
		case "sd":
			sdN++
			sdSeeds = append(sdSeeds, s)
			if !sd.Converged {
				sdUnconv++
			}
		}
	}
	if len(comass) > 0 {
		fmt.Printf("- Native ψ comass (grid): median %.3f [min %.3f, max %.3f]; two-set relative difference max %.2e (tolerance %.0e)\n", mdcu.Median(comass), quant(comass, 0), quant(comass, 1), quant(rel, 1), mdcu.R4ComassTol)
	}
	if sdN > 0 {
		fmt.Printf("- S-D: %d seeds, %d unconverged (their K-psi@1 arm is dropped)\n", sdN, sdUnconv)
	}

	// ---- cells -----------------------------------------------------------------------
	for _, cell := range cellOrder {
		if len(d.Runs[cell]) == 0 {
			continue
		}
		fmt.Printf("\n## Cell %s — all seeds\n", cell)
		armTable(d, cell, "")
		if strings.HasPrefix(cell, "SD") {
			continue
		}
		for _, st := range []string{"definite", "split"} {
			fmt.Printf("\n### %s · stratum %s\n", cell, st)
			armTable(d, cell, st)
		}
	}
	if len(d.Runs["A250"]) == 0 {
		fmt.Println("\nTier A not present; no verdict")
		return
	}

	// ---- gate and decisions ------------------------------------------------------
	gate, gateRate := mdcu.R4Gate(d)
	fmt.Printf("\n## Decision rules\n\n**G4 (replication gate)**: K-full 3-lock rate at 250 steps %s (needs ≥ 90%%) → **%s**\n", pct(gateRate),
		map[bool]string{true: "pass", false: "FAIL (every R4 verdict: inconclusive, non-replication)"}[gate])

	q4a := mdcu.R4Q4a(d)
	fmt.Printf("\n**Q4a — does φ lock if given more steps?** K-phi:")
	for _, r := range q4a.Rungs {
		fmt.Printf(" %d steps: n=%d %s;", r.Steps, r.N, pct(r.Rate))
	}
	fmt.Printf("\n→ **%s**", q4a.Verdict)
	if q4a.RStar > 0 {
		fmt.Printf(" (R* = %d)", q4a.RStar)
	}
	fmt.Println()

	q4b := mdcu.R4Q4b(d)
	q4bV := q4b.Verdict
	switch {
	case k.Comass:
		q4bV = "inconclusive (comass)"
	case k.Sampler:
		q4bV = "inconclusive (sampler)"
	}
	fmt.Printf("\n**Q4b — coupling scale or form?** (Tier A, %d paired seeds at least)\n", q4b.N)
	fmt.Println("| arm | all | definite | split |")
	fmt.Println("|---|---|---|---|")
	for _, a := range []mdcu.R3Arm{mdcu.ArmPhi, mdcu.ArmPhiK, mdcu.ArmPsi1, mdcu.ArmPsiK} {
		fmt.Printf("| %s | %s | %s | %s |\n", a, pct(q4b.Rates[a]), pct(q4b.RatesByStratum["definite"][a]), pct(q4b.RatesByStratum["split"][a]))
	}
	for _, key := range []string{"Δ(K-psi@1, K-phi)", "Δ(K-psi@κ, K-phi@κ)", "Δ(K-phi@κ, K-phi)"} {
		fmt.Println("- " + contrastLine(q4b.Contrasts[key]))
	}
	f := q4b.Flags
	fmt.Printf("- equivalent(K-psi@1, K-phi)=%v · equivalent(K-psi@κ, K-phi@κ)=%v · clearly-higher(K-phi@κ, K-phi)=%v · clearly-higher(K-psi@1, K-phi)=%v · clearly-higher(K-psi@κ, K-phi@κ)=%v\n",
		f.EqPsi1Phi, f.EqPsiKPhiK, f.PhiKHigher, f.FormAt1, f.FormAtK)
	if len(q4b.FloorPairs) > 0 {
		fmt.Printf("- **floor-equivalent (both rates < 10%%; a floor, not a finding; disclosure only):** %s\n", strings.Join(q4b.FloorPairs, "; "))
	}
	fmt.Printf("→ **%s**", q4bV)
	if q4bV != q4b.Verdict {
		fmt.Printf("  (rule value before the kill override: %s)", q4b.Verdict)
	}
	fmt.Println()

	q4c := map[string]mdcu.Q4cResult{}
	q4cV := map[string]string{}
	for _, st := range []string{"definite", "split"} {
		r := mdcu.R4Q4c(d, st)
		q4c[st], q4cV[st] = r, r.Verdict
		switch {
		case k.Comass:
			q4cV[st] = "inconclusive (comass)"
		case k.Sampler:
			q4cV[st] = "inconclusive (sampler)"
		}
		cell := r.Cell
		if cell == "" {
			cell = "none"
		}
		fmt.Printf("\n**Q4c (%s) — does φ specifically matter at matched coupling?** first qualifying cell: %s\n", st, cell)
		if r.Cell != "" {
			fmt.Println("- " + contrastLine(r.Stat))
		}
		fmt.Printf("→ **%s**", q4cV[st])
		if q4cV[st] != r.Verdict {
			fmt.Printf("  (rule value before the kill override: %s)", r.Verdict)
		}
		fmt.Println()
	}
	fmt.Printf("\n**Reading** (G4=%v, Q4b=%s, Q4c definite=%s, split=%s):\n> %s\n", gate, q4bV, q4cV["definite"], q4cV["split"], mdcu.R4Reading(gate, q4bV, q4cV["definite"], q4cV["split"]))

	// ---- spontaneity, mechanism, S-A ---------------------------------------------
	fmt.Println("\n## Secondaries (each carries its own hypothesis and kill; never a decision input)")
	spon := func(cell string, arm mdcu.R3Arm) mdcu.SponResult {
		return mdcu.R4Spontaneity(d.Get(cell, arm, ""), arm, cell, *reps)
	}
	if !*skipKS {
		full := spon("A250", mdcu.ArmFull)
		var psiArms, phiArms []mdcu.SponResult
		fmt.Printf("\n**S-A Spontaneity** (Haar-G₂ KS, MC p over %d reps; arms with ≥ %d three-locked runs)\n", *reps, mdcu.R4LockedMin)
		show := func(s mdcu.SponResult) {
			if s.N < mdcu.R4LockedMin {
				fmt.Printf("- %s @ %s: %d locked runs → untestable\n", s.Arm, s.Cell, s.N)
				return
			}
			fmt.Printf("- %s @ %s: n=%d locked planes, D=%.4f, p=%.3f → %s\n", s.Arm, s.Cell, s.N, s.D, s.P,
				map[bool]string{true: "consistent with Haar-G₂", false: "not Haar-G₂"}[s.P > 0.05])
		}
		show(full)
		var phiKres, psi1res mdcu.SponResult
		for _, a := range []mdcu.R3Arm{mdcu.ArmPsi, mdcu.ArmPsi1, mdcu.ArmPsiK} {
			r := spon("A250", a)
			show(r)
			psiArms = append(psiArms, r)
			if a == mdcu.ArmPsi1 {
				psi1res = r
			}
		}
		for _, p := range []struct {
			cell string
			arm  mdcu.R3Arm
		}{{"A250", mdcu.ArmPhiK}, {"B1000", mdcu.ArmPhiK}, {"A250", mdcu.ArmPhi}, {"B1000", mdcu.ArmPhi}, {"C4000", mdcu.ArmPhi}} {
			if len(d.Runs[p.cell][p.arm]) == 0 {
				continue
			}
			r := spon(p.cell, p.arm)
			show(r)
			phiArms = append(phiArms, r)
			if p.cell == "A250" && p.arm == mdcu.ArmPhiK {
				phiKres = r
			}
		}
		sa := mdcu.R4SA(full, psiArms, phiArms)
		fmt.Printf("- H-A1 (K-full Haar-consistent; R3 p=0.545): **%s** · H-A2 (every ψ arm non-Haar): **%s** · H-A3 (every locking φ arm Haar-consistent): **%s**\n", sa.A1, sa.A2, sa.A3)

		phiK, psi1 := d.Get("A250", mdcu.ArmPhiK, ""), d.Get("A250", mdcu.ArmPsi1, "")
		cLag := mdcu.R3Contrast("Δ(K-phi@κ, K-psi@1)", phiK, psi1)
		mech := mdcu.R4Mechanism(q4b.Contrasts["Δ(K-psi@κ, K-phi@κ)"], cLag, phiKres, psi1res)
		fmt.Printf("\n**Mechanism reading** (oppenheimer seq 77; no claim): %s\n- %s\n- Kill FORM=%v · Kill COUPLING-ONLY=%v\n", mech.Reading, contrastLine(cLag), mech.KillForm, mech.KillCouplingOnly)
	} else {
		fmt.Println("\nS-A and the mechanism kills skipped (-skip-ks).")
	}

	fmt.Println("\n**S-B Dose-response on the rescaled arm** (definite seeds, 250 steps; Spearman of paired s₃(K-phi@L) − s₃(K-psi@L) vs d; R3: ρ = −0.258 raw)")
	sb := map[string][3]float64{}
	for _, lv := range []struct {
		name     string
		phi, psi mdcu.R3Arm
	}{{"L=κ", mdcu.ArmPhiK, mdcu.ArmPsiK}, {"L=1", mdcu.ArmPhi, mdcu.ArmPsi1}} {
		pa, pb := mdcu.PairBySeed(d.Get("A250", lv.phi, "definite"), d.Get("A250", lv.psi, "definite"))
		var gap, dist []float64
		for i := range pa {
			if sd, ok := d.Seeds[pa[i].Seed]; ok && sd.S2Dist != nil {
				gap, dist = append(gap, pa[i].Stats.S3-pb[i].Stats.S3), append(dist, *sd.S2Dist)
			}
		}
		rho, lo, hi, n := mdcu.R4SBStat("S-B "+lv.name, gap, dist)
		sb[lv.name] = [3]float64{rho, lo, hi}
		fmt.Printf("- %s: n=%d, ρ=%.3f, 95%% CI [%.3f, %.3f] → H-B **%s**\n", lv.name, n, rho, lo, hi, mdcu.R4SBVerdict(rho, lo, hi))
	}

	fmt.Printf("\n**S-C Finding (a) at matched coupling:** %s\n", mdcu.R4SC(d))

	if len(d.Runs["SD250"]) > 0 {
		sdr := mdcu.R4SD(d)
		fmt.Println("\n**S-D Dose ladder** (no decision weight; K-psi@1 vs K-phi; 'falls continuously' has no numeric rule, so only the kill is evaluated)")
		fmt.Println("| s | cell | n | K-psi@1 | K-phi |")
		fmt.Println("|---|---|---|---|---|")
		for _, r := range sdr.Rungs {
			fmt.Printf("| %.3f | %s | %d | %s | %s |\n", r.S, r.Cell, r.N, pct(r.RatePsi1), pct(r.RatePhi))
		}
		fmt.Printf("→ H-D kill (at s=0.005, rate(K-psi@1) − rate(K-phi) ≥ 50 points, at either cell): **%v**\n", sdr.Killed)
	}

	// ---- diagnostics -------------------------------------------------------------
	psiN := d.Get("A250", mdcu.ArmPsi, "")
	fmt.Printf("\n**R3 K-psi replication (diagnostic, not a gate):** native K-psi 3-lock rate %s vs R3's 79%% → %s\n", pct(mdcu.LockRate(psiN)),
		map[bool]string{true: "within ±15 points", false: "outside ±15 points"}[math.Abs(mdcu.LockRate(psiN)-0.79) <= 0.15+1e-9])

	// ---- prediction on file --------------------------------------------------------
	fmt.Println("\n## On-file prediction (§1.6, unedited in the frozen file) — mechanical scoring, record only")
	r := q4b.Rates
	p1 := q4bV == "MIXED" && r[mdcu.ArmPhiK] >= 0.20-1e-9 && r[mdcu.ArmPhiK] <= 0.80+1e-9 && r[mdcu.ArmPsi1] >= 0.30-1e-9 && r[mdcu.ArmPsi1] <= 0.80+1e-9
	fmt.Printf("- P1 (Q4b = MIXED; K-phi@κ 20–80%%; K-psi@1 30–80%%): Q4b=%s, K-phi@κ %s, K-psi@1 %s → **%v**\n", q4bV, pct(r[mdcu.ArmPhiK]), pct(r[mdcu.ArmPsi1]), p1)
	fmt.Printf("- P2 (Q4a = locks slowly): Q4a=%s → **%v**\n", q4a.Verdict, q4a.Verdict == "locks slowly")
	fmt.Printf("- P3 (S-C holds): **%v**\n", mdcu.R4SC(d) == "holds")
	if kv, ok := sb["L=κ"]; ok {
		k1 := sb["L=1"]
		res := "n/a (ρ undefined)"
		if !math.IsNaN(kv[0]) && !math.IsNaN(k1[0]) {
			res = fmt.Sprint(math.Abs(kv[0]) < 0.15 && math.Abs(k1[0]) < 0.15)
		}
		fmt.Printf("- P4 (S-B |ρ| < 0.15 on the rescaled arm; scored as true iff both L=κ and L=1 are below 0.15): ρ(κ)=%.3f, ρ(1)=%.3f → **%s**\n", kv[0], k1[0], res)
	}
	fmt.Println("- P5 (K-psi@1 stays non-Haar, p < 0.01): see the S-A line for K-psi@1 above.")
}
