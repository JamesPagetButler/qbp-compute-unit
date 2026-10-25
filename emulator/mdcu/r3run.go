package mdcu

// Round-3 run layer (frozen §1.2 samplers and miss rules, §1.4 grid).
//
// BuildR3Psi draws everything stochastic-but-dynamics-free for one seed (ψ for
// each role, draw counts, S2 deformation record). RunR3Seed then runs every
// applicable arm of one cell on that seed with a shared initial condition.
//
// Process guard: R3InGrid reports whether a seed is in the pre-registered R3
// grid (3000–3199). The driver refuses those seeds unless run with an explicit
// -run-grid flag; every test and smoke run uses out-of-grid seeds only.

const (
	R3GridLo = 3000
	R3GridHi = 3199
)

// R3InGrid reports whether seed lies in the frozen R3 grid.
func R3InGrid(seed uint64) bool { return seed >= R3GridLo && seed <= R3GridHi }

// R3Psi is everything the ψ arms need for one seed.
type R3Psi struct {
	Seed    uint64
	Stratum string // "definite" | "split" (by seed parity)
	Sampler string // sampler of Primary: "S2" (definite) | "S1" (split)

	Primary *Form3 // ψ for K-psi and K-psigram; nil if the primary sampler missed

	// S1 record. On odd seeds S1 produces Primary; on even seeds S1 produces the
	// sensitivity ψ (S1Def) only. S1Miss refers to whichever S1 call was made.
	S1Draws      int
	S1Degenerate int
	S1Miss       bool
	S1Def        *Form3 // even seeds only; nil on a miss

	S2 *S2Draw // even seeds only
}

// buildR3Psi is BuildR3Psi with an injectable S1 sampler (for miss-rule tests).
func buildR3Psi(seed uint64, s1 func(uint64, string) (S1Draw, bool)) *R3Psi {
	p := &R3Psi{Seed: seed, Stratum: R3Stratum(seed)}
	if p.Stratum == "definite" {
		s2 := SampleS2(seed)
		p.Sampler, p.S2, p.Primary = "S2", &s2, s2.Form
		d, ok := s1(seed, "definite")
		p.S1Draws, p.S1Degenerate, p.S1Miss = d.Draws, d.Degenerate, !ok
		if ok {
			p.S1Def = d.Form
		}
		return p
	}
	p.Sampler = "S1"
	d, ok := s1(seed, "split")
	p.S1Draws, p.S1Degenerate, p.S1Miss = d.Draws, d.Degenerate, !ok
	if ok {
		p.Primary = d.Form
	}
	return p
}

// BuildR3Psi draws the seed's ψ forms with the frozen samplers.
func BuildR3Psi(seed uint64) *R3Psi { return buildR3Psi(seed, SampleS1) }

// R3ArmsFor lists the arms that run on this seed, in reporting order.
//
//	always:        K-full, K-gram, K-phi, K-rand3, K-none
//	K-psi, K-psigram: need Primary (S2 on even seeds — always; S1 split on odd — unless miss)
//	K-psi-S1:      even seeds with an S1-definite form only
func R3ArmsFor(p *R3Psi) []R3Arm {
	var out []R3Arm
	for _, a := range R3AllArms {
		switch a {
		case ArmPsi, ArmPsiGram:
			if p.Primary == nil {
				continue
			}
		case ArmPsiS1:
			if p.S1Def == nil {
				continue
			}
		}
		out = append(out, a)
	}
	return out
}

// R3Cell is one (N, steps, σ) cell; Arms restricts the arms (nil = all that apply).
type R3Cell struct {
	Name  string
	N     int
	Steps int
	Sigma float64
	Arms  []R3Arm
}

// R3Run is one (cell, seed, arm) record.
type R3Run struct {
	Cell    string
	N       int
	Steps   int
	Sigma   float64
	Seed    uint64
	Stratum string
	Arm     R3Arm
	Stats   R3Stats
}

// armForm returns the ψ an arm is driven by (nil for non-ψ arms).
func armForm(arm R3Arm, p *R3Psi) *Form3 {
	switch arm {
	case ArmPsi, ArmPsiGram:
		return p.Primary
	case ArmPsiS1:
		return p.S1Def
	}
	return nil
}

// RunR3Seed runs all applicable (and requested) arms of cell c on seed p.Seed.
// Every arm starts from the same Q1Init draw. Calibration ψ₀ is measured under
// the arm's own ψ where it has one, else under the seed's Primary ψ (the
// cross-calibration diagnostic), else under φ.
func RunR3Seed(c R3Cell, p *R3Psi) []R3Run {
	z0 := Q1Init(c.N, c.Sigma, newRNG(p.Seed))
	allowed := map[R3Arm]bool{}
	for _, a := range c.Arms {
		allowed[a] = true
	}
	var out []R3Run
	for _, arm := range R3ArmsFor(p) {
		if len(c.Arms) > 0 && !allowed[arm] {
			continue
		}
		var rand3 *AltTensor
		if arm == ArmRand3 {
			rand3 = NewRandomAlt(newRNG(p.Seed ^ r3SeedRand3))
		}
		psi := armForm(arm, p)
		en := R3EnergyFor(arm, psi, rand3)
		Z := RunRingEnergy(z0, en, c.Steps)
		calib := psi
		if calib == nil {
			calib = p.Primary
		}
		if calib == nil {
			calib = Phi()
		}
		out = append(out, R3Run{
			Cell: c.Name, N: c.N, Steps: c.Steps, Sigma: c.Sigma,
			Seed: p.Seed, Stratum: p.Stratum, Arm: arm,
			Stats: R3Measure(Z, calib),
		})
	}
	return out
}
