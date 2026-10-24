// mdcu-q2 runs the frozen round-2 Q2 measurement (plan v3 section 4 plus the
// Q2 amendment prereg/mdcu-round2-q2-amendment-FROZEN-v3a.md): Albert
// determinant, B1 (CD-MAC) vs B0 (non-CD), as three separate columns (MACs,
// model cycles under two bounds, host ns/op). The verdict is applied
// mechanically from the frozen amendment section 3; nothing is judged by hand.
//
// Run only under the gates (amendment frozen, deming box-safety, beekeeper go):
//
//	cd ~/Documents/QBP-CU-mdcu/emulator && go build -o bin/mdcu-q2 ./cmd/mdcu-q2
//	nice -n 19 ./bin/mdcu-q2 -out ../doc/spikes/raw/mdcu-q2-YYYY-MM-DD.json
//
// -dry skips the host-ns benchmarks (counts and cycle bounds only).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/emulator/mdcu"
)

// Decisive cell and secondary sweep (amendment section 2).
const (
	decBatch = 64
	decP     = 8
)

var (
	sweepBatches = []int{1, 64, 512}
	sweepP       = []int{1, 8}
)

type cell struct {
	Batch int `json:"batch"`
	P     int `json:"p"`
	Ideal int `json:"ideal_cycles"`
	Dep   int `json:"dep_cycles"`
}

type row struct {
	Variant      string    `json:"variant"`
	Side         string    `json:"side"`
	MACs         int       `json:"macs_per_det"` // fixed by construction, not evidence
	CritPath     int       `json:"critical_path_batches"`
	Cells        []cell    `json:"cells"`
	HostNs       []float64 `json:"host_ns_samples,omitempty"`
	HostNsMedian float64   `json:"host_ns_median,omitempty"`
}

func (r *row) at(batch, P int) cell {
	for _, c := range r.Cells {
		if c.Batch == batch && c.P == P {
			return c
		}
	}
	panic("missing cell")
}

type verdict struct {
	B1       string `json:"b1"`
	B0       string `json:"b0"`
	Yes      bool   `json:"yes_rule"`
	No       bool   `json:"no_rule"`
	Result   string `json:"result"`
	Ordering string `json:"ordering_measured_columns"` // per column: "B0" if B0<=B1 else "B1"
}

type report struct {
	Date        string   `json:"date"`
	Agreement   float64  `json:"agreement_worst_normwise"`
	LoadAtStart string   `json:"loadavg_at_start"`
	LoadAtEnd   string   `json:"loadavg_at_end"`
	WallSec     float64  `json:"wall_seconds"`
	HostNsDone  bool     `json:"host_ns_complete"`
	Rows        []row    `json:"rows"`
	Primary     verdict  `json:"verdict_primary"`
	Sensitivity verdict  `json:"verdict_b1lean_sensitivity"`
	Final       string   `json:"verdict_of_record"`
	Notes       []string `json:"notes"`
}

func median(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[len(s)/2]
}

func loadavg() string {
	la, _ := os.ReadFile("/proc/loadavg")
	return strings.TrimSpace(string(la))
}

// decide applies amendment section 3 to one (B1, B0) pair.
func decide(b1, b0 *row, hostOK bool) verdict {
	c1, c0 := b1.at(decBatch, decP), b0.at(decBatch, decP)
	v := verdict{B1: b1.Variant, B0: b0.Variant}
	v.Yes = 2*c1.Ideal <= c0.Ideal && 2*c1.Dep <= c0.Dep
	ord := func(b0v, b1v float64) string {
		if b0v <= b1v {
			return "B0"
		}
		return "B1"
	}
	cols := []string{ord(float64(c0.Ideal), float64(c1.Ideal)), ord(float64(c0.Dep), float64(c1.Dep))}
	if hostOK {
		cols = append(cols, ord(b0.HostNsMedian, b1.HostNsMedian))
	}
	v.Ordering = strings.Join(cols, ",")
	v.No = hostOK && b0.MACs <= b1.MACs && !strings.Contains(v.Ordering, "B1")
	switch {
	case v.Yes:
		v.Result = "yes"
	case v.No:
		v.Result = "no"
	default:
		v.Result = "inconclusive"
	}
	return v
}

func main() {
	dry := flag.Bool("dry", false, "skip host-ns benchmarks")
	out := flag.String("out", "", "write JSON here")
	capSec := flag.Int("cap", 600, "wall-clock cap in seconds (plan section 10: Q2 10 min)")
	flag.Parse()

	runtime.GOMAXPROCS(1)
	start := time.Now()
	rep := report{Date: start.Format("2006-01-02"), LoadAtStart: loadavg()}

	// Agreement gate first: if it fails the comparison does not count.
	rep.Agreement = mdcu.Q2Agree(mdcu.NewQ2Insts(20000, 77))
	if rep.Agreement > 1e-12 {
		fmt.Fprintf(os.Stderr, "AGREEMENT GATE FAILED: %.3g > 1e-12; comparison does not count\n", rep.Agreement)
		os.Exit(2)
	}

	pool := mdcu.NewQ2Insts(512, 83)
	for _, v := range mdcu.Q2Variants {
		var one mdcu.Tape
		v.Fn(&pool[0], &one)
		r := row{Variant: v.Name, Side: v.Side, MACs: one.TotalMults(), CritPath: one.CritPath()}
		for _, b := range sweepBatches {
			var tp mdcu.Tape
			for i := 0; i < b; i++ {
				v.Fn(&pool[i], &tp)
			}
			for _, P := range mdcu.Tiles {
				r.Cells = append(r.Cells, cell{b, P, tp.IdealCycles(P), tp.DepCycles(P)})
			}
		}
		rep.Rows = append(rep.Rows, r)
	}

	rep.HostNsDone = !*dry
	if !*dry {
		bench := mdcu.NewQ2Insts(4096, 82)
	outer:
		for ri := range rep.Rows {
			v := mdcu.Q2Variants[ri]
			for s := 0; s < 6; s++ {
				if time.Since(start) > time.Duration(*capSec)*time.Second {
					fmt.Fprintln(os.Stderr, "wall-clock cap hit; aborting with partial host-ns")
					rep.HostNsDone = false
					break outer
				}
				res := testing.Benchmark(func(b *testing.B) {
					var sink float64
					for i := 0; i < b.N; i++ {
						sink += v.Fn(&bench[i&4095], nil)
					}
					_ = sink
				})
				rep.Rows[ri].HostNs = append(rep.Rows[ri].HostNs, float64(res.T.Nanoseconds())/float64(res.N))
			}
			rep.Rows[ri].HostNsMedian = median(rep.Rows[ri].HostNs)
		}
	}

	// Verdict (amendment section 3).
	byName := map[string]*row{}
	for i := range rep.Rows {
		byName[rep.Rows[i].Variant] = &rep.Rows[i]
	}
	b0 := byName["B0a"]
	if byName["B0b"].MACs < b0.MACs {
		b0 = byName["B0b"]
	}
	rep.Primary = decide(byName["B1lit"], b0, rep.HostNsDone)
	rep.Sensitivity = decide(byName["B1lean"], b0, rep.HostNsDone)
	rep.Final = rep.Primary.Result
	if rep.Sensitivity.Ordering != rep.Primary.Ordering {
		rep.Final = "inconclusive (B1-variant-sensitive)"
	}
	if !rep.HostNsDone && rep.Final == "no" {
		rep.Final = "inconclusive (host ns incomplete)"
	}
	rep.LoadAtEnd = loadavg()
	rep.WallSec = time.Since(start).Seconds()
	rep.Notes = []string{
		"MACs are fixed by construction (arithmetic, not evidence).",
		"Model cycles: symmetric tile model, 64 lanes/tile, one batch per tile-cycle, identical parameters both sides; assumed, not silicon.",
		"Bound (i) ideal = ceil(mults/(64P)); bound (ii) dep = greedy list schedule on P tiles.",
		"Host ns measured on this box, GOMAXPROCS=1, median of 6. Columns never merged.",
		"Decisive cell batch 64, P=8. Batch 1/512 at P=1/8 is a secondary, non-deciding sweep.",
	}

	fmt.Printf("agreement %.3g | load start %s | load end %s | wall %.1fs\n\n",
		rep.Agreement, rep.LoadAtStart, rep.LoadAtEnd, rep.WallSec)
	fmt.Printf("%-7s %-3s %5s %5s %12s\n", "variant", "", "MACs", "crit", "host ns/det")
	for _, r := range rep.Rows {
		fmt.Printf("%-7s %-3s %5d %5d %12.1f\n", r.Variant, r.Side, r.MACs, r.CritPath, r.HostNsMedian)
	}
	fmt.Println("\nmodel cycles  ideal/dep")
	fmt.Printf("%-7s", "")
	for _, b := range sweepBatches {
		for _, P := range sweepP {
			mark := ""
			if b == decBatch && P == decP {
				mark = "*"
			}
			fmt.Printf(" %12s", fmt.Sprintf("b%d P%d%s", b, P, mark))
		}
	}
	fmt.Println()
	for _, r := range rep.Rows {
		fmt.Printf("%-7s", r.Variant)
		for _, b := range sweepBatches {
			for _, P := range sweepP {
				c := r.at(b, P)
				fmt.Printf(" %12s", fmt.Sprintf("%d/%d", c.Ideal, c.Dep))
			}
		}
		fmt.Println()
	}
	fmt.Printf("\nprimary  %s vs %s: yes=%v no=%v ordering[ideal,dep,host]=%s -> %s\n",
		rep.Primary.B1, rep.Primary.B0, rep.Primary.Yes, rep.Primary.No, rep.Primary.Ordering, rep.Primary.Result)
	fmt.Printf("sens.    %s vs %s: yes=%v no=%v ordering[ideal,dep,host]=%s -> %s\n",
		rep.Sensitivity.B1, rep.Sensitivity.B0, rep.Sensitivity.Yes, rep.Sensitivity.No, rep.Sensitivity.Ordering, rep.Sensitivity.Result)
	fmt.Printf("VERDICT OF RECORD: %s\n", rep.Final)

	if *out != "" {
		b, _ := json.MarshalIndent(rep, "", " ")
		if err := os.WriteFile(*out, b, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("wrote", *out)
	}
}
