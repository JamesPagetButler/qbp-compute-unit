// Command mdcu-q1 runs the pre-registered round-2 Q1 grid (Test B with nulls).
//
// It must NOT be run for real until the plan is frozen and @deming has given the
// box-safety go. -dry-run prints the pre-registered run order and cell counts and
// runs nothing.
//
//	go run ./cmd/mdcu-q1 -dry-run
//	nice -n 19 go run ./cmd/mdcu-q1 -out q1.jsonl -max-wall 2h   (only after the go)
//
// Kill conditions (draft v2 / box-safety request): wall-clock cap, RSS cap, free
// disk floor, output-size cap, NaN. On any of them it stops and reports the cells
// completed and skipped (the pre-registered order makes the cap truncate the least
// important cells).
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/emulator/mdcu"
)

type cell struct {
	N     int
	Sigma float64
}

// Pre-registered run order: (1) main cell; (2) N=12 and N=48 at σ=0.1; (3) the rest.
var order = []cell{
	{24, 0.1},
	{12, 0.1}, {48, 0.1},
	{24, 0.0}, {24, 0.3},
	{12, 0.0}, {12, 0.3}, {48, 0.0}, {48, 0.3},
}

var kinds = []string{"overlap", "local", "disjoint", "none", "rand3", "shuffle"}

func main() {
	dry := flag.Bool("dry-run", false, "print the plan and exit; run nothing")
	out := flag.String("out", "q1.jsonl", "output JSONL (one line per run)")
	seeds := flag.Int("seeds", 200, "seeds per cell (pre-registered: 200, seeds 1000..)")
	first := flag.Uint64("first-seed", 1000, "first seed (pre-registered: 1000)")
	maxWall := flag.Duration("max-wall", 2*time.Hour, "wall-clock cap")
	maxRSSMB := flag.Uint64("max-rss-mb", 500, "abort if Go heap sys exceeds this")
	minFreeGB := flag.Uint64("min-free-gb", 15, "abort if free disk on the output dir drops below this")
	maxOutMB := flag.Int64("max-out-mb", 200, "abort if output exceeds this")
	flag.Parse()

	fmt.Printf("plan: %d cells x %d seeds x %d kinds = %d runs; order:\n", len(order), *seeds, len(kinds), len(order)**seeds*len(kinds))
	for i, c := range order {
		fmt.Printf("  %d. N=%d sigma=%.1f\n", i+1, c.N, c.Sigma)
	}
	if *dry {
		fmt.Println("dry-run: nothing executed")
		return
	}

	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	enc := json.NewEncoder(w)

	start := time.Now()
	var done, skipped []string
	abort := func(why string) bool {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		if time.Since(start) > *maxWall {
			fmt.Println("ABORT: wall-clock cap")
			return true
		}
		if ms.Sys>>20 > *maxRSSMB {
			fmt.Println("ABORT: memory cap")
			return true
		}
		var st syscall.Statfs_t
		if syscall.Statfs(".", &st) == nil && (st.Bavail*uint64(st.Bsize))>>30 < *minFreeGB {
			fmt.Println("ABORT: free disk below floor")
			return true
		}
		if fi, err := f.Stat(); err == nil && fi.Size()>>20 > *maxOutMB {
			fmt.Println("ABORT: output size cap")
			return true
		}
		_ = why
		return false
	}

	for ci, c := range order {
		name := fmt.Sprintf("N=%d sigma=%.1f", c.N, c.Sigma)
		stopped := false
		for s := 0; s < *seeds && !stopped; s++ {
			if abort(name) {
				stopped = true
				break
			}
			for _, k := range kinds {
				r := mdcu.RunQ1Kind(c.N, c.Sigma, *first+uint64(s), k)
				if math.IsNaN(r.Final.SVD3) {
					fmt.Println("ABORT: NaN in", name, k, r.Seed)
					stopped = true
					break
				}
				_ = enc.Encode(r)
			}
			if s%20 == 0 {
				w.Flush()
				fmt.Printf("[%s] %s seed %d/%d elapsed %.0fs\n", time.Now().Format("15:04:05"), name, s, *seeds, time.Since(start).Seconds())
			}
		}
		if stopped {
			skipped = append(skipped, name+" (partial or not run)")
			for _, c2 := range order[ci+1:] {
				skipped = append(skipped, fmt.Sprintf("N=%d sigma=%.1f", c2.N, c2.Sigma))
			}
			break
		}
		done = append(done, name)
	}
	w.Flush()
	fmt.Printf("completed cells: %v\nskipped: %v\nwall: %.1fs\n", done, skipped, time.Since(start).Seconds())
}
