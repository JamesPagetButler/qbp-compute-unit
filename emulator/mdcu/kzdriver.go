package mdcu

// Driver-side pieces of the mdcu-kz run that are worth testing without a binary: the
// frozen seed lists, the outcome-blind probe and its validation, and the loader that
// turns a run's JSONL back into the analyzer's input (refusing partial data).

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"time"
)

// KZSeeds is every frozen seed list of the round. The probe sub-ranges are a HARNESS
// split of 9930-9949 (the frozen text pins only the range).
type KZSeeds struct {
	Main, Null               []uint64
	MainStage0a, MainStage0b []uint64
	NullStage0a, NullStage0b []uint64
	Saturation               []uint64
	Eta, Step16, Step24, Det []uint64
}

func kzSeq(lo uint64, n int) []uint64 {
	s := make([]uint64, n)
	for i := range s {
		s[i] = lo + uint64(i)
	}
	return s
}

// KZFrozenSeeds returns the frozen lists. Stage 0a uses the first three seeds of each
// five-seed range (5600-5604 main, 5605-5609 null; HARNESS), Stage 0b the ten seeds of
// 5610-5619 (main) and 5620-5629 (null).
func KZFrozenSeeds() KZSeeds {
	return KZSeeds{
		Main: kzSeq(5000, 200), Null: kzSeq(5200, 200),
		MainStage0a: kzSeq(5600, 3), NullStage0a: kzSeq(5605, 3),
		MainStage0b: kzSeq(5610, 10), NullStage0b: kzSeq(5620, 10),
		Saturation: kzSeq(5630, 20),
		Eta:        kzSeq(9930, 5), Step16: kzSeq(9935, 5), Step24: kzSeq(9940, 5), Det: kzSeq(9945, 5),
	}
}

// KZFrozenRunConfig is the run configuration on the frozen seeds; the caller adds the
// field factories, Check and Emit. R seeds are the first R of each range.
func KZFrozenRunConfig(L, R int, eta float64) KZRunConfig {
	s := KZFrozenSeeds()
	return KZRunConfig{
		L: L, R: R, Eta: eta, Ladder: KZLadder, Stage0: DefaultKZStage0Params(),
		MainSeed0: s.Main[0], NullSeed0: s.Null[0],
		MainStage0a: s.MainStage0a, MainStage0b: s.MainStage0b,
		NullStage0a: s.NullStage0a, NullStage0b: s.NullStage0b,
		SatSeed0: s.Saturation[0], SatN: len(s.Saturation),
		MkMain: func(seed uint64) KZArm { return &KZMainArm{F: NewKZPlaneFieldHaar(L, seed)} },
		MkNull: func(seed uint64) KZArm { return &KZNullArm{F: NewKZS3FieldHaar(L, seed)} },
	}
}

// ---- probe ------------------------------------------------------------------------------

// KZProbeSpec says what the probe runs on; KZFrozenProbeSpec is the real one.
type KZProbeSpec struct {
	L1, L2                   int // 16 and 24
	Steps                    int // timed steps per seed per arm
	Eta, Step16, Step24, Det []uint64
}

func KZFrozenProbeSpec() KZProbeSpec {
	s := KZFrozenSeeds()
	return KZProbeSpec{L1: 16, L2: 24, Steps: 50, Eta: s.Eta, Step16: s.Step16, Step24: s.Step24, Det: s.Det}
}

// KZProbeReport is the outcome-blind probe: it reads timings and the descent test only,
// never a defect count.
type KZProbeReport struct {
	Kind        string          `json:"kind"`
	Eta         float64         `json:"eta"`
	EtaOK       bool            `json:"eta_ok"`
	EtaSeeds    []uint64        `json:"eta_seeds"`
	Step16Seeds []uint64        `json:"step16_seeds"`
	Step24Seeds []uint64        `json:"step24_seeds"`
	DetSeeds    []uint64        `json:"det_seeds"`
	TS16        []float64       `json:"ts16"`
	TS24        []float64       `json:"ts24"`
	TD16        []float64       `json:"td16"`
	TD24        []float64       `json:"td24"`
	Timing      KZProbeTiming   `json:"timing"`
	Decision    KZProbeDecision `json:"decision"`
}

// kzTimeSteps times n Langevin steps of one arm pair at size L, seconds per step with
// both arms averaged (HARNESS), at T = 1 from the Haar start.
func kzTimeSteps(L int, seed uint64, eta float64, n int, check func() error) (float64, error) {
	if check != nil {
		if err := check(); err != nil {
			return 0, err
		}
	}
	rng := KZNoiseRNG(seed, 7)
	m := NewKZPlaneFieldHaar(L, seed)
	t0 := time.Now()
	for i := 0; i < n; i++ {
		m.Step(eta, 1, rng)
	}
	dm := time.Since(t0).Seconds()
	s := NewKZS3FieldHaar(L, seed)
	t0 = time.Now()
	for i := 0; i < n; i++ {
		s.Step(eta, 1, rng)
	}
	ds := time.Since(t0).Seconds()
	return (dm + ds) / (2 * float64(n)), nil
}

// kzTimeDetect times one detector pass per arm on a Haar field, seconds per pass, both
// arms averaged (HARNESS).
func kzTimeDetect(L int, seed uint64, check func() error) (float64, error) {
	if check != nil {
		if err := check(); err != nil {
			return 0, err
		}
	}
	p := Amend1Z2Params()
	m := &KZMainArm{F: NewKZPlaneFieldHaar(L, seed)}
	s := &KZNullArm{F: NewKZS3FieldHaar(L, seed)}
	t0 := time.Now()
	m.Detect(p)
	dm := time.Since(t0).Seconds()
	t0 = time.Now()
	s.Detect(p)
	ds := time.Since(t0).Seconds()
	return (dm + ds) / 2, nil
}

// KZRunProbe runs the descent test for eta and the timings, and applies the frozen
// probe rule. check, if non-nil, is polled between seeds; its error aborts the probe.
func KZRunProbe(spec KZProbeSpec, check func() error) (KZProbeReport, error) {
	rep := KZProbeReport{Kind: "probe", EtaSeeds: spec.Eta, Step16Seeds: spec.Step16, Step24Seeds: spec.Step24, DetSeeds: spec.Det}
	if check != nil {
		if err := check(); err != nil {
			return rep, err
		}
	}
	rep.Eta, rep.EtaOK = KZChooseEta(spec.L1, spec.Eta)
	var err error
	for _, sd := range spec.Step16 {
		var v float64
		if v, err = kzTimeSteps(spec.L1, sd, rep.Eta, spec.Steps, check); err != nil {
			return rep, err
		}
		rep.TS16 = append(rep.TS16, v)
	}
	for _, sd := range spec.Step24 {
		var v float64
		if v, err = kzTimeSteps(spec.L2, sd, rep.Eta, spec.Steps, check); err != nil {
			return rep, err
		}
		rep.TS24 = append(rep.TS24, v)
	}
	for _, sd := range spec.Det {
		var a, b float64
		if a, err = kzTimeDetect(spec.L1, sd, check); err != nil {
			return rep, err
		}
		if b, err = kzTimeDetect(spec.L2, sd, check); err != nil {
			return rep, err
		}
		rep.TD16 = append(rep.TD16, a)
		rep.TD24 = append(rep.TD24, b)
	}
	rep.Timing = KZProbeTiming{TS16: KZMedian(rep.TS16), TD16: KZMedian(rep.TD16), TS24: KZMedian(rep.TS24), TD24: KZMedian(rep.TD24)}
	rep.Decision = KZProbeDecide(rep.Timing)
	return rep, nil
}

func kzSameSeeds(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func kzClose(a, b float64) bool { return math.Abs(a-b) <= 1e-12*math.Max(1, math.Abs(b)) }

// KZValidateProbe re-derives everything a probe file claims: the seed lists are the
// frozen ones, eta is on the ladder and passed the descent test, the timings are the
// medians of the per-seed arrays, and the decision follows from the timings. It returns
// the decision and refuses a probe whose rule says the round does not run.
func KZValidateProbe(r KZProbeReport) (KZProbeDecision, error) {
	s := KZFrozenSeeds()
	switch {
	case r.Kind != "probe":
		return KZProbeDecision{}, fmt.Errorf("record kind %q is not a probe", r.Kind)
	case !kzSameSeeds(r.EtaSeeds, s.Eta), !kzSameSeeds(r.Step16Seeds, s.Step16),
		!kzSameSeeds(r.Step24Seeds, s.Step24), !kzSameSeeds(r.DetSeeds, s.Det):
		return KZProbeDecision{}, fmt.Errorf("probe was not run on the frozen probe seeds 9930-9949")
	case len(r.TS16) != len(s.Step16), len(r.TS24) != len(s.Step24), len(r.TD16) != len(s.Det), len(r.TD24) != len(s.Det):
		return KZProbeDecision{}, fmt.Errorf("probe timing arrays do not match the seed lists")
	case !r.EtaOK:
		return KZProbeDecision{}, fmt.Errorf("no eta passed the descent test")
	}
	onLadder := false
	for _, e := range KZEtaLadder {
		if e == r.Eta {
			onLadder = true
		}
	}
	if !onLadder {
		return KZProbeDecision{}, fmt.Errorf("eta %g is not on the frozen ladder", r.Eta)
	}
	want := KZProbeTiming{TS16: KZMedian(r.TS16), TD16: KZMedian(r.TD16), TS24: KZMedian(r.TS24), TD24: KZMedian(r.TD24)}
	if !kzClose(r.Timing.TS16, want.TS16) || !kzClose(r.Timing.TD16, want.TD16) ||
		!kzClose(r.Timing.TS24, want.TS24) || !kzClose(r.Timing.TD24, want.TD24) {
		return KZProbeDecision{}, fmt.Errorf("probe timings are not the medians of its per-seed arrays")
	}
	d := KZProbeDecide(want)
	if d.Run != r.Decision.Run || d.L != r.Decision.L || d.R != r.Decision.R || d.LowPower != r.Decision.LowPower {
		return KZProbeDecision{}, fmt.Errorf("probe decision (L=%d R=%d) does not follow from its timings (rule gives L=%d R=%d)",
			r.Decision.L, r.Decision.R, d.L, d.R)
	}
	if !d.Run {
		return d, fmt.Errorf("the probe rule says the round does not run (even R=25 at L=16 exceeds the budget)")
	}
	return d, nil
}

// ---- reading a run back -------------------------------------------------------------------

// KZLoadRunRecords reads a run's JSONL and returns the per-rung data, main and null, in
// ladder order. It refuses anything but a complete run: an "end" record with verdict
// "complete", exactly R replicates per arm and rung, no duplicates, no unknown rungs.
func KZLoadRunRecords(rd io.Reader, ladder []int, R int) (main, null []KZRungData, err error) {
	type rec struct {
		Kind      string `json:"kind"`
		Arm       string `json:"arm"`
		Tau       int    `json:"tau"`
		Rep       int    `json:"rep"`
		Cells     int    `json:"cells"`
		Defects   int    `json:"defects"`
		Ambiguous int    `json:"ambiguous"`
		Neg       int    `json:"neg"`
		Verdict   string `json:"verdict"`
		Reason    string `json:"reason"`
	}
	idx := map[int]int{}
	for i, t := range ladder {
		idx[t] = i
	}
	mk := func() []KZRungData {
		d := make([]KZRungData, len(ladder))
		for i, t := range ladder {
			d[i] = KZRungData{Tau: t, Defects: make([]int, R), Ambiguous: make([]int, R), Neg: make([]int, R)}
		}
		return d
	}
	main, null = mk(), mk()
	seen := map[[3]int]bool{}
	var ended bool
	var verdict string
	sc := bufio.NewScanner(rd)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	line := 0
	for sc.Scan() {
		line++
		var r rec
		if e := json.Unmarshal(sc.Bytes(), &r); e != nil {
			return nil, nil, fmt.Errorf("line %d: %v", line, e)
		}
		switch r.Kind {
		case "abort":
			return nil, nil, fmt.Errorf("the run aborted: %s", r.Reason)
		case "end":
			ended, verdict = true, r.Verdict
		case "run":
			ti, ok := idx[r.Tau]
			arm := 0
			switch r.Arm {
			case "main":
			case "null":
				arm = 1
			default:
				ok = false
			}
			if !ok || r.Rep < 0 || r.Rep >= R {
				return nil, nil, fmt.Errorf("line %d: run record outside the plan (arm %q tau %d rep %d)", line, r.Arm, r.Tau, r.Rep)
			}
			key := [3]int{arm, ti, r.Rep}
			if seen[key] {
				return nil, nil, fmt.Errorf("line %d: duplicate run record (arm %q tau %d rep %d)", line, r.Arm, r.Tau, r.Rep)
			}
			seen[key] = true
			d := main
			if arm == 1 {
				d = null
			}
			d[ti].Cells = r.Cells
			d[ti].Defects[r.Rep], d[ti].Ambiguous[r.Rep], d[ti].Neg[r.Rep] = r.Defects, r.Ambiguous, r.Neg
		}
	}
	if e := sc.Err(); e != nil {
		return nil, nil, e
	}
	if !ended {
		return nil, nil, fmt.Errorf("no end record: the run is partial")
	}
	if verdict != "complete" {
		return nil, nil, fmt.Errorf("the run ended with verdict %q: there is no grid to analyze", verdict)
	}
	if want := 2 * R * len(ladder); len(seen) != want {
		return nil, nil, fmt.Errorf("%d run records, want %d", len(seen), want)
	}
	return main, null, nil
}
