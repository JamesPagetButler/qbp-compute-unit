package mdcu

// Calibration of the frozen Q5a rule on synthetic data with a TRUE power law and zero
// null (no dynamics, no frozen seed). It logs how often the frozen no-curvature test
// (quadratic CI includes 0 and ΔAIC > -2) rejects a clean power law, so the signers can
// see the rule's false-negative rate. The only assertion is a loose sanity bound.

import (
	"os"
	"testing"
)

func TestKZQ5aCalibrationOnTruePowerLaw(t *testing.T) {
	if os.Getenv("KZ_CALIB") != "1" {
		t.Skip("set KZ_CALIB=1 to run (about 40 s)")
	}
	for _, R := range []int{25, 50, 100} {
		const cells = 4096
		sims, yes, q5b := 100, 0, 0
		for i := 0; i < sims; i++ {
			main := kzSynthArm(uint64(12500+i), R, cells, kzPowerLaw(0.2, 0.75), nil)
			d := KZAnalyze(main, kzZeroArm(R, cells))
			if d.Q5a == "yes" {
				yes++
			}
			if d.Q5b == "yes" {
				q5b++
			}
		}
		t.Logf("true alpha = 0.75, L^3=%d cells, R=%d: Q5a yes %d/%d, Q5b yes %d/%d", cells, R, yes, sims, q5b, sims)
		if float64(yes)/float64(sims) < 0.6 {
			t.Fatalf("R=%d: the frozen Q5a rule rejects a clean power law too often (%d/%d)", R, sims-yes, sims)
		}
	}
}
