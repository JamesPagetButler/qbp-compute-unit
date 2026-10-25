package mdcu

// R3-Q2 host-ns benchmarks (frozen §3.3). One benchmark per (workload, path),
// each over a batch of 64 fixed-seed inputs, nil tape (no recording overhead).
//
// DO NOT RUN until deming has scheduled the idle-box window. Run as:
//   go test -c -o mdcu.test ./mdcu/
//   GOMAXPROCS=1 nice -n 19 taskset -c <k> ./mdcu.test -test.run '^$' \
//     -test.bench 'BenchmarkR3Q2' -test.count 10 -test.benchtime 200ms

import "testing"

const r3q2BenchSeed = 9820 // out-of-grid, fixed

var r3q2Sink float64
var r3q2Sink8 [8]float64

func benchW3(b *testing.B, f func(*V3In, *Tape) [8]float64) {
	ins := NewV3Ins(64, r3q2BenchSeed)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for k := range ins {
			r3q2Sink8 = f(&ins[k], nil)
		}
	}
}

func benchW2(b *testing.B, f func(*V3In, *Tape) float64) {
	ins := NewV3Ins(64, r3q2BenchSeed+1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for k := range ins {
			r3q2Sink = f(&ins[k], nil)
		}
	}
}

func benchW1(b *testing.B, f func(*Q2Inst, *Tape) float64) {
	ins := NewQ2Insts(64, r3q2BenchSeed+2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for k := range ins {
			r3q2Sink = f(&ins[k], nil)
		}
	}
}

func BenchmarkR3Q2W1CD_B1lean(b *testing.B)     { benchW1(b, DetB1Lean) }
func BenchmarkR3Q2W1NonCD_B0b(b *testing.B)     { benchW1(b, DetB0b) }
func BenchmarkR3Q2W1NonCD_B0cross(b *testing.B) { benchW1(b, DetB0Cross) }
func BenchmarkR3Q2W2CD(b *testing.B)            { benchW2(b, W2CD) }
func BenchmarkR3Q2W2NonCD(b *testing.B)         { benchW2(b, W2NonCD) }
func BenchmarkR3Q2W3CD(b *testing.B)            { benchW3(b, W3CD) }
func BenchmarkR3Q2W3NonCD(b *testing.B)         { benchW3(b, W3NonCD) }
