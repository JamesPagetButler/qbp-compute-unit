// mdcu-q2 runs the frozen round-2 Q2 measurement (plan v3 section 4):
// Albert determinant, B1 (CD-MAC) vs B0 (non-CD), as three separate columns
// (MACs, model cycles under two bounds, host ns/op). It decides nothing; the
// decision rule is applied by reading the printed table against the frozen text.
//
// Run only under the gates (plan frozen + addendum, deming box-safety, beekeeper go):
//   cd ~/Documents/QBP-CU-mdcu/emulator && go build -o bin/mdcu-q2 ./cmd/mdcu-q2
//   nice -n 19 ./bin/mdcu-q2 -out ../doc/spikes/raw/mdcu-q2-YYYY-MM-DD.json
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

type cycles struct {
	Ideal int `json:"ideal"`
	Dep   int `json:"dep"`
}

type row struct {
	Variant string `json:"variant"`
	Side    string `json:"side"`
	// MACs (multiplies) per instance, measured from the tape.
	MACs int `json:"macs_per_instance"`
	// Model cycles at each P for a batch of Batch independent instances.
	BatchCycles map[int]cycles `json:"batch_cycles"`
	// Model cycles at each P for a single instance (latency).
	SingleCycles map[int]cycles `json:"single_cycles"`
	CritPath     int            `json:"critical_path_batches"`
	// Host ns per instance: 6 medians-of-benchmark samples.
	HostNs       []float64 `json:"host_ns_samples,omitempty"`
	HostNsMedian float64   `json:"host_ns_median,omitempty"`
}

type report struct {
	Date       string   `json:"date"`
	Batch      int      `json:"batch"`
	Agreement  float64  `json:"agreement_worst_normwise"`
	LoadAvgAt  string   `json:"loadavg_at_start"`
	GoMaxProcs int      `json:"gomaxprocs"`
	Rows       []row    `json:"rows"`
	Notes      []string `json:"notes"`
}

func median(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[len(s)/2]
}

func main() {
	dry := flag.Bool("dry", false, "skip host-ns benchmarks")
	out := flag.String("out", "", "write JSON here (<= 50 MB cap is trivially met)")
	batch := flag.Int("batch", 64, "independent instances in the throughput workload")
	capSec := flag.Int("cap", 600, "wall-clock cap in seconds (plan section 10: Q2 10 min)")
	flag.Parse()

	runtime.GOMAXPROCS(1)
	start := time.Now()
	la, _ := os.ReadFile("/proc/loadavg")
	rep := report{Date: time.Now().Format("2006-01-02"), Batch: *batch,
		LoadAvgAt: strings.TrimSpace(string(la)), GoMaxProcs: runtime.GOMAXPROCS(0)}

	// Agreement gate first: if it fails the comparison does not count.
	rep.Agreement = mdcu.Q2Agree(mdcu.NewQ2Insts(20000, 77))
	if rep.Agreement > 1e-12 {
		fmt.Fprintf(os.Stderr, "AGREEMENT GATE FAILED: %.3g > 1e-12; comparison does not count\n", rep.Agreement)
		os.Exit(2)
	}

	ins := mdcu.NewQ2Insts(*batch, 83)
	for _, v := range mdcu.Q2Variants {
		var one, many mdcu.Tape
		v.Fn(&ins[0], &one)
		for i := range ins {
			v.Fn(&ins[i], &many)
		}
		r := row{Variant: v.Name, Side: v.Side, MACs: one.TotalMults(),
			BatchCycles: map[int]cycles{}, SingleCycles: map[int]cycles{}, CritPath: one.CritPath()}
		for _, P := range mdcu.Tiles {
			r.BatchCycles[P] = cycles{many.IdealCycles(P), many.DepCycles(P)}
			r.SingleCycles[P] = cycles{one.IdealCycles(P), one.DepCycles(P)}
		}
		rep.Rows = append(rep.Rows, r)
	}

	if !*dry {
		pool := mdcu.NewQ2Insts(4096, 82)
		for ri := range rep.Rows {
			v := mdcu.Q2Variants[ri]
			for s := 0; s < 6; s++ {
				if time.Since(start) > time.Duration(*capSec)*time.Second {
					fmt.Fprintln(os.Stderr, "wall-clock cap hit; aborting with partial host-ns")
					goto done
				}
				res := testing.Benchmark(func(b *testing.B) {
					var sink float64
					for i := 0; i < b.N; i++ {
						sink += v.Fn(&pool[i&4095], nil)
					}
					_ = sink
				})
				rep.Rows[ri].HostNs = append(rep.Rows[ri].HostNs, float64(res.T.Nanoseconds())/float64(res.N))
			}
			rep.Rows[ri].HostNsMedian = median(rep.Rows[ri].HostNs)
		}
	}
done:
	rep.Notes = []string{
		"MACs = multiplies counted on the tape; model cycles = symmetric tile model (64 lanes/tile), assumed, not silicon;",
		"host ns = measured on this box; the three columns are never merged.",
		"Bound (i) ideal = ceil(mults/(64P)); bound (ii) dep = greedy list schedule on P tiles.",
		"Admission rule is asserted by `go test ./mdcu/ -run 'TestRT|TestQ2_Admission'`.",
	}

	fmt.Printf("agreement (worst normwise) %.3g   load at start: %s\n\n", rep.Agreement, rep.LoadAvgAt)
	fmt.Printf("%-7s %-3s %5s | batch=%d P=8: ideal dep | single P=8: ideal dep | crit | host ns/op (median of 6)\n",
		"variant", "", "MACs", rep.Batch)
	for _, r := range rep.Rows {
		fmt.Printf("%-7s %-3s %5d |        %6d %4d |         %6d %4d | %4d | %.1f\n",
			r.Variant, r.Side, r.MACs, r.BatchCycles[8].Ideal, r.BatchCycles[8].Dep,
			r.SingleCycles[8].Ideal, r.SingleCycles[8].Dep, r.CritPath, r.HostNsMedian)
	}
	if *out != "" {
		b, _ := json.MarshalIndent(rep, "", " ")
		if err := os.WriteFile(*out, b, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("wrote", *out)
	}
}
