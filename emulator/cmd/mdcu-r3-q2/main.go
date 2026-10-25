// Command mdcu-r3-q2 reports R3-Q2 (frozen §3): the deterministic multiply-count
// closure (tape counts, no host timing) and, given `go test -bench` output, the
// benchstat-equivalent host-ns verdict and the combined R3-Q2 verdict.
//
//	go run ./cmd/mdcu-r3-q2                       closure only (safe any time)
//	go run ./cmd/mdcu-r3-q2 -bench bench.txt -load-start 0.12 -load-end 0.31
//
// The closure is pure arithmetic on fixed out-of-grid inputs and is safe to run
// on a busy box. The bench file must be produced in deming's idle window as
// specified in §3.3; this command never runs a benchmark itself.
//
// Interpretation (not spelled out in the frozen text; documented here):
//   - closure overall = "yes" iff ANY workload shows a ≥ 2× advantage (mirrors
//     the benchstat rule);
//   - benchstat "inconclusive" with a decided closure: the verdict follows the
//     closure and is labelled as such.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/JamesPagetButler/qbp-compute-unit/emulator/mdcu"
)

const disclosure = "Host ns measured on an FX-8350 without AVX2. The CD-MAC's wide-lane parallelism is serialised, so host ns is biased against CD."

func main() {
	benchPath := flag.String("bench", "", "go test -bench output (10 counts) from the idle-box window")
	ls := flag.Float64("load-start", -1, "1-min loadavg at the start of the bench window")
	le := flag.Float64("load-end", -1, "1-min loadavg at the end of the bench window")
	flag.Parse()

	fmt.Println("# R3-Q2 (frozen §3)")
	fmt.Println("\n## Multiply-count closure (tape-counted; data×data multiplies; ±1 and ×2 not counted)")
	fmt.Println("| workload | CD path | CD mults | best non-CD path | non-CD mults | advantage (non-CD/CD) | ≥2× for CD? | adds CD / non-CD (analytic, not tape) |")
	fmt.Println("|---|---|---|---|---|---|---|---|")
	closureYes := false
	for _, r := range mdcu.R3Q2Closure() {
		fmt.Printf("| %s | %s | %d | %s | %d | %.3f | %v | %d / %d |\n", r.Workload, r.CDPath, r.CDMults, r.NonCDPath, r.NonCDMults, r.CDAdvantage, r.CDYes, r.CDAdds, r.NonCDAdds)
		closureYes = closureYes || r.CDYes
	}
	closure := map[bool]string{true: "yes", false: "no"}[closureYes]
	fmt.Printf("\nClosure verdict: **%s** (yes iff any workload has a ≥ 2× CD multiply-count advantage)\n", closure)

	if *benchPath == "" {
		fmt.Println("\nNo bench output supplied: host-ns benchstat not evaluated. Combined verdict pending the idle-box window.")
		fmt.Printf("\n> %s\n", disclosure)
		return
	}

	b, err := os.ReadFile(*benchPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rows, err := mdcu.BenchRows(mdcu.ParseBenchOutput(string(b)))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("\n## Host-ns (Mann–Whitney U, α = 0.05; benchstat-equivalent — benchstat itself not installed)")
	fmt.Println("| workload | n(CD) | CD median ns | n(non-CD) | non-CD median ns | CD/non-CD | p |")
	fmt.Println("|---|---|---|---|---|---|---|")
	for _, r := range rows {
		mc, mn := mdcu.Median(r.CD), mdcu.Median(r.NonCD)
		fmt.Printf("| %s | %d | %.0f | %d | %.0f | %.3f | %.2g |\n", r.Workload, len(r.CD), mc, len(r.NonCD), mn, mc/mn, mdcu.MannWhitneyP(r.CD, r.NonCD))
	}

	bench := mdcu.BenchVerdict(rows)
	switch {
	case *ls < 0 || *le < 0:
		fmt.Println("\nLoad averages not supplied: benchstat verdict withheld (-load-start / -load-end required).")
		bench = "inconclusive (load not recorded)"
	case !mdcu.LoadOK(*ls, *le):
		fmt.Printf("\nLoad ceiling violated (start %.2f > 1.0 or end %.2f > 1.5): rerun once per §3.3; if it persists the benchstat is \"inconclusive (load)\".\n", *ls, *le)
		bench = "inconclusive (load)"
	}
	fmt.Printf("\nBenchstat verdict: **%s**\n", bench)

	final, why := closure, "closure and benchstat agree"
	switch {
	case bench == "yes" || bench == "no":
		if bench != closure {
			final, why = bench, "(host-ns tiebreak)"
		}
	default:
		why = "benchstat " + bench + "; verdict follows the closure"
	}
	fmt.Printf("\n## R3-Q2 verdict: **%s** — %s\n\n> %s\n", final, why, disclosure)
}
