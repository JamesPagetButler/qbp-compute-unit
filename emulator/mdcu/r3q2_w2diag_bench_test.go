package mdcu

// POST-HOC DIAGNOSTIC benchmark (arch seq 89). Same 64 inputs and sink as the
// frozen BenchmarkR3Q2W2* (seed r3q2BenchSeed+1). Named BenchmarkPostHoc* so the
// frozen 'BenchmarkR3Q2' selector never picks it up. Run, with the frozen W2
// pair in the same process for a same-load comparison, as:
//   go test -c -o mdcu.test ./mdcu/
//   GOMAXPROCS=1 nice -n 19 taskset -c <k> ./mdcu.test -test.run '^$' \
//     -test.bench 'BenchmarkR3Q2W2|BenchmarkPostHocW2' -test.count 10 -test.benchtime 200ms

import "testing"

func BenchmarkPostHocW2NonCDUnrolled(b *testing.B) { benchW2(b, W2NonCDUnrolled) }
