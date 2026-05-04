// Package spinchain implements the 1D Heisenberg spin-chain benchmark.
//
// This is Action 3 of the Crawl Phase: the first proof-of-concept for
// QBP-algebraic computation. We simulate the same physical system two ways:
//
//  1. QBP-Algebraic: Spin states as unit quaternions. Time evolution as
//     quaternion rotation (exp of pure quaternion). Norm preservation is
//     an algebraic property of the multiplication.
//
//  2. Float64-Scalar: Spin states as 2x2 complex matrices. Time evolution
//     via matrix exponentiation. Unitarity must be maintained numerically.
//
// The hypothesis: QBP-algebraic simulation maintains unitarity (norm
// preservation) for longer durations with fewer renormalisations because
// quaternion multiplication is norm-preserving by algebraic structure,
// not by numerical accident.
//
// The benchmark measures:
//   - Operation count for equivalent simulation time
//   - Cumulative norm drift (algebraic curvature)
//   - Maximum unitarity defect
//   - Wall-clock time
//
// Physical system: 1D Heisenberg model
//
//	H = J Σᵢ Sᵢ · Sᵢ₊₁
//
// For spin-½: Sᵢ = σᵢ/2 (Pauli matrices / 2)
// Time evolution: U(dt) = exp(-iHdt)
// For nearest-neighbour pair: U_pair(dt) = exp(-iJ(S·S)dt)
//
// The spin-½ exchange operator decomposes into a quaternion rotation:
// exp(-iJ(σ·σ/4)dt) = cos(Jdt/4)I - i·sin(Jdt/4)(σ·σ)
// which IS a unit quaternion.
package spinchain

import (
	"math"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/watchdog"
)

// ─── QBP-Algebraic simulation ──────────────────────────────────────────────

// QBPState represents the spin chain as an array of unit quaternions.
// Each quaternion encodes the spin-½ state at one site.
//
// Convention: |↑⟩ = (1,0,0,0), |↓⟩ = (0,1,0,0)
// General state: q = cos(θ/2)|↑⟩ + e^{iφ}sin(θ/2)|↓⟩
// maps to quaternion q = (cos(θ/2), sin(θ/2)cos(φ), sin(θ/2)sin(φ), 0)
type QBPState struct {
	Spins []quat.Quat
	N     int // number of sites
}

// NewQBPState creates a spin chain with N sites, all initialised to |↑⟩.
func NewQBPState(n int) *QBPState {
	s := &QBPState{
		Spins: make([]quat.Quat, n),
		N:     n,
	}
	for i := range s.Spins {
		s.Spins[i] = quat.Identity() // |↑⟩
	}
	return s
}

// NewNeelState creates an antiferromagnetic Néel state: |↑↓↑↓...⟩
// This is a more interesting initial condition for dynamics.
func NewNeelState(n int) *QBPState {
	s := &QBPState{
		Spins: make([]quat.Quat, n),
		N:     n,
	}
	for i := range s.Spins {
		if i%2 == 0 {
			s.Spins[i] = quat.Identity() // |↑⟩
		} else {
			// |↓⟩ = rotation by π around x-axis: (0,1,0,0)
			s.Spins[i] = quat.New(0, 1, 0, 0)
		}
	}
	return s
}

// PairEvolution computes the time-evolution quaternion for a nearest-neighbour
// Heisenberg exchange interaction with coupling J and timestep dt.
//
// For the isotropic Heisenberg model, the pair evolution operator is:
//
//	U_pair = exp(-iJ·(S_i · S_{i+1})·dt)
//
// For spin-½, S·S has eigenvalues 3/4 (triplet) and -1/4 (singlet).
// The evolution operator decomposes as a rotation in the product space.
//
// In the Trotter decomposition (first order), we apply pair evolutions
// sequentially: U(dt) ≈ Π_i U_pair(i, i+1, dt)
//
// The quaternion encoding of this rotation:
//
//	q_pair = (cos(Jdt/4), sin(Jdt/4)·n_x, sin(Jdt/4)·n_y, sin(Jdt/4)·n_z)
//
// where n is the axis determined by the relative spin orientation.
func PairEvolution(j, dt float64) quat.Quat {
	angle := j * dt / 4.0
	return quat.New(math.Cos(angle), math.Sin(angle), 0, 0)
}

// PairEvolutionAdaptive computes the pair evolution quaternion with the
// rotation axis determined by the current spin orientations at sites i and j.
// This captures the full Heisenberg dynamics rather than a fixed-axis approximation.
func PairEvolutionAdaptive(si, sj quat.Quat, j, dt float64) quat.Quat {
	// The interaction Hamiltonian H_pair = J (S_i · S_{i+1})
	// In quaternion language, the spin expectation values are:
	//   <S_x> = Re(q* · i · q) / 2, etc.
	// The effective rotation axis for the pair is along the cross product
	// of the two spin directions.

	// Extract spin direction vectors from quaternions
	// Spin direction n_i = q_i (0,0,1) q_i* (Bloch sphere z-axis rotated)
	nix, niy, niz := quat.RotateVec(si, 0, 0, 1)
	njx, njy, njz := quat.RotateVec(sj, 0, 0, 1)

	// Cross product gives rotation axis
	cx := niy*njz - niz*njy
	cy := niz*njx - nix*njz
	cz := nix*njy - niy*njx

	crossNorm := math.Sqrt(cx*cx + cy*cy + cz*cz)
	if crossNorm < 1e-15 {
		// Parallel spins: no torque, identity evolution
		return quat.Identity()
	}

	// Normalise axis
	ax, ay, az := cx/crossNorm, cy/crossNorm, cz/crossNorm

	// Rotation angle proportional to J*dt and sin of angle between spins
	// Dot product for angle
	dot := nix*njx + niy*njy + niz*njz
	angle := j * dt * math.Sqrt(1-dot*dot) / 2.0

	// Quaternion for rotation by angle around axis (ax,ay,az)
	halfAngle := angle / 2.0
	s := math.Sin(halfAngle)
	return quat.New(math.Cos(halfAngle), s*ax, s*ay, s*az)
}

// QBPStep performs one Trotter step of the Heisenberg evolution using
// quaternion algebra. Returns the number of quaternion multiplications.
func QBPStep(state *QBPState, j, dt float64, wd *watchdog.Stats) int {
	ops := 0

	// Even bonds: (0,1), (2,3), (4,5), ...
	for i := 0; i+1 < state.N; i += 2 {
		u := PairEvolutionAdaptive(state.Spins[i], state.Spins[i+1], j, dt)
		ops++

		state.Spins[i] = quat.Mul(u, state.Spins[i])
		ops++
		wd.ObserveMul(state.Spins[i])

		state.Spins[i+1] = quat.Mul(u, state.Spins[i+1])
		ops++
		wd.ObserveMul(state.Spins[i+1])
	}

	// Odd bonds: (1,2), (3,4), (5,6), ...
	for i := 1; i+1 < state.N; i += 2 {
		u := PairEvolutionAdaptive(state.Spins[i], state.Spins[i+1], j, dt)
		ops++

		state.Spins[i] = quat.Mul(u, state.Spins[i])
		ops++
		wd.ObserveMul(state.Spins[i])

		state.Spins[i+1] = quat.Mul(u, state.Spins[i+1])
		ops++
		wd.ObserveMul(state.Spins[i+1])
	}

	return ops
}

// TotalNormDrift returns the sum of |1 - ||q||²| across all sites.
// For perfect algebraic computation, this should be exactly zero.
func TotalNormDrift(state *QBPState) float64 {
	var total float64
	for _, q := range state.Spins {
		total += math.Abs(1.0 - quat.NormSq(q))
	}
	return total
}

// ─── Float64-Scalar simulation (comparison baseline) ───────────────────────

// Complex represents a complex number for the scalar simulation.
type Complex struct {
	Re, Im float64
}

func cmul(a, b Complex) Complex {
	return Complex{
		Re: a.Re*b.Re - a.Im*b.Im,
		Im: a.Re*b.Im + a.Im*b.Re,
	}
}

func cadd(a, b Complex) Complex {
	return Complex{Re: a.Re + b.Re, Im: a.Im + b.Im}
}

func csub(a, b Complex) Complex {
	return Complex{Re: a.Re - b.Re, Im: a.Im - b.Im}
}

func cscale(s float64, a Complex) Complex {
	return Complex{Re: s * a.Re, Im: s * a.Im}
}

func cnorm2(a Complex) float64 {
	return a.Re*a.Re + a.Im*a.Im
}

// Mat2x2 is a 2x2 complex matrix for scalar spin-½ simulation.
type Mat2x2 struct {
	A, B, C, D Complex // [[A,B],[C,D]]
}

// ScalarState represents the spin chain using 2x2 density matrices.
// Each site stores a 2-component complex spinor.
type ScalarState struct {
	// Each site is a 2-component spinor [α, β] where |α|² + |β|² = 1
	Alpha []Complex
	Beta  []Complex
	N     int
}

// NewScalarState creates a chain with all spins up: [1, 0].
func NewScalarState(n int) *ScalarState {
	s := &ScalarState{
		Alpha: make([]Complex, n),
		Beta:  make([]Complex, n),
		N:     n,
	}
	for i := 0; i < n; i++ {
		s.Alpha[i] = Complex{Re: 1}
		s.Beta[i] = Complex{Re: 0}
	}
	return s
}

// NewScalarNeelState creates Néel state: |↑↓↑↓...⟩
func NewScalarNeelState(n int) *ScalarState {
	s := &ScalarState{
		Alpha: make([]Complex, n),
		Beta:  make([]Complex, n),
		N:     n,
	}
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			s.Alpha[i] = Complex{Re: 1}
		} else {
			s.Beta[i] = Complex{Re: 1}
		}
	}
	return s
}

// ScalarPairEvolution computes the 4x4 → two 2x2 pair evolution matrix
// elements using explicit matrix exponentiation.
// Returns the number of floating-point operations.
func ScalarPairEvolution(si, sj *ScalarState, idx int, j, dt float64, wd *watchdog.Stats) int {
	ops := 0

	// For the Heisenberg pair, compute effective rotation
	// Extract Bloch vectors from spinors
	ai, bi := si.Alpha[idx], si.Beta[idx]
	aj, bj := sj.Alpha[idx+1], sj.Beta[idx+1]

	// Bloch vector components: n = <ψ|σ|ψ>
	// n_x = 2 Re(α*β), n_y = 2 Im(α*β), n_z = |α|² - |β|²
	nix := 2 * (ai.Re*bi.Re + ai.Im*bi.Im)
	niy := 2 * (ai.Re*bi.Im - ai.Im*bi.Re)
	niz := cnorm2(ai) - cnorm2(bi)
	ops += 12

	njx := 2 * (aj.Re*bj.Re + aj.Im*bj.Im)
	njy := 2 * (aj.Re*bj.Im - aj.Im*bj.Re)
	njz := cnorm2(aj) - cnorm2(bj)
	ops += 12

	// Cross product → rotation axis
	cx := niy*njz - niz*njy
	cy := niz*njx - nix*njz
	cz := nix*njy - niy*njx
	ops += 6

	crossNorm := math.Sqrt(cx*cx + cy*cy + cz*cz)
	ops += 4

	if crossNorm < 1e-15 {
		return ops
	}

	ax, ay, az := cx/crossNorm, cy/crossNorm, cz/crossNorm
	ops += 3

	dot := nix*njx + niy*njy + niz*njz
	angle := j * dt * math.Sqrt(1-dot*dot) / 2.0
	ops += 6

	halfAngle := angle / 2.0
	cosH := math.Cos(halfAngle)
	sinH := math.Sin(halfAngle)
	ops += 3

	// Apply SU(2) rotation as 2x2 matrix multiplication:
	// U = cos(θ/2)I - i·sin(θ/2)(n·σ)
	// U = [[cos(θ/2) - i·sin(θ/2)·nz, -sin(θ/2)·(ny + i·nx)],
	//      [sin(θ/2)·(ny - i·nx), cos(θ/2) + i·sin(θ/2)·nz]]

	u00 := Complex{Re: cosH, Im: -sinH * az}
	u01 := Complex{Re: -sinH * ay, Im: -sinH * ax}
	u10 := Complex{Re: sinH * ay, Im: -sinH * ax}
	u11 := Complex{Re: cosH, Im: sinH * az}
	ops += 6

	// Apply to site i
	newAi := cadd(cmul(u00, ai), cmul(u01, bi))
	newBi := cadd(cmul(u10, ai), cmul(u11, bi))
	ops += 16

	// Apply to site i+1
	newAj := cadd(cmul(u00, aj), cmul(u01, bj))
	newBj := cadd(cmul(u10, aj), cmul(u11, bj))
	ops += 16

	si.Alpha[idx] = newAi
	si.Beta[idx] = newBi
	sj.Alpha[idx+1] = newAj
	sj.Beta[idx+1] = newBj

	// Unitarity check: |α|² + |β|² should = 1
	normI := cnorm2(newAi) + cnorm2(newBi)
	normJ := cnorm2(newAj) + cnorm2(newBj)
	ops += 8

	wd.ObserveUnitarity(math.Abs(1.0 - normI))
	wd.ObserveUnitarity(math.Abs(1.0 - normJ))

	return ops
}

// ScalarStep performs one Trotter step using conventional float64 computation.
func ScalarStep(state *ScalarState, j, dt float64, wd *watchdog.Stats) int {
	ops := 0

	// Even bonds
	for i := 0; i+1 < state.N; i += 2 {
		ops += ScalarPairEvolution(state, state, i, j, dt, wd)
	}

	// Odd bonds
	for i := 1; i+1 < state.N; i += 2 {
		ops += ScalarPairEvolution(state, state, i, j, dt, wd)
	}

	return ops
}

// ScalarTotalNormDrift returns the sum of |1 - (|α|² + |β|²)| across all sites.
func ScalarTotalNormDrift(state *ScalarState) float64 {
	var total float64
	for i := 0; i < state.N; i++ {
		norm := cnorm2(state.Alpha[i]) + cnorm2(state.Beta[i])
		total += math.Abs(1.0 - norm)
	}
	return total
}

// ─── Benchmark harness ─────────────────────────────────────────────────────

// BenchmarkConfig holds parameters for the spin-chain comparison.
type BenchmarkConfig struct {
	ChainLength int     // Number of spin sites
	Coupling    float64 // Exchange coupling J
	TimeStep    float64 // Trotter step size dt
	TotalSteps  int     // Number of time steps
	Renorm      bool    // Whether to renormalise periodically
	RenormEvery int     // Renormalise every N steps (if Renorm=true)
}

// DefaultConfig returns a reasonable default benchmark configuration.
func DefaultConfig() BenchmarkConfig {
	return BenchmarkConfig{
		ChainLength: 20,
		Coupling:    1.0,
		TimeStep:    0.01,
		TotalSteps:  10000,
		Renorm:      false, // Don't renormalise — we're measuring drift
		RenormEvery: 100,
	}
}

// BenchmarkResult holds the output of a benchmark run.
type BenchmarkResult struct {
	Config BenchmarkConfig

	// QBP results
	QBPOps       int64
	QBPWallTime  time.Duration
	QBPWatchdog  *watchdog.Stats
	QBPNormDrift float64 // final total norm drift

	// Scalar results
	ScalarOps       int64
	ScalarWallTime  time.Duration
	ScalarWatchdog  *watchdog.Stats
	ScalarNormDrift float64

	// Comparison
	NormDriftRatio float64 // scalar/QBP (>1 means QBP wins)
	OpsRatio       float64 // scalar/QBP
	TimeRatio      float64 // scalar/QBP
}

// RunBenchmark executes both simulations and compares results.
func RunBenchmark(cfg BenchmarkConfig) *BenchmarkResult {
	result := &BenchmarkResult{Config: cfg}

	// ── QBP-Algebraic ──
	qbpState := NewNeelState(cfg.ChainLength)
	qbpWD := watchdog.New()

	start := time.Now()
	var qbpOps int64
	for step := 0; step < cfg.TotalSteps; step++ {
		ops := QBPStep(qbpState, cfg.Coupling, cfg.TimeStep, qbpWD)
		qbpOps += int64(ops)

		if cfg.Renorm && step > 0 && step%cfg.RenormEvery == 0 {
			for i := range qbpState.Spins {
				qbpState.Spins[i] = quat.Normalize(qbpState.Spins[i])
			}
		}
	}
	result.QBPWallTime = time.Since(start)
	result.QBPOps = qbpOps
	result.QBPWatchdog = qbpWD
	result.QBPNormDrift = TotalNormDrift(qbpState)

	// ── Float64-Scalar ──
	scalarState := NewScalarNeelState(cfg.ChainLength)
	scalarWD := watchdog.New()

	start = time.Now()
	var scalarOps int64
	for step := 0; step < cfg.TotalSteps; step++ {
		ops := ScalarStep(scalarState, cfg.Coupling, cfg.TimeStep, scalarWD)
		scalarOps += int64(ops)

		if cfg.Renorm && step > 0 && step%cfg.RenormEvery == 0 {
			for i := 0; i < scalarState.N; i++ {
				norm := math.Sqrt(cnorm2(scalarState.Alpha[i]) + cnorm2(scalarState.Beta[i]))
				scalarState.Alpha[i] = cscale(1.0/norm, scalarState.Alpha[i])
				scalarState.Beta[i] = cscale(1.0/norm, scalarState.Beta[i])
			}
		}
	}
	result.ScalarWallTime = time.Since(start)
	result.ScalarOps = scalarOps
	result.ScalarWatchdog = scalarWD
	result.ScalarNormDrift = ScalarTotalNormDrift(scalarState)

	// ── Comparison ──
	if result.QBPNormDrift > 0 {
		result.NormDriftRatio = result.ScalarNormDrift / result.QBPNormDrift
	}
	if result.QBPOps > 0 {
		result.OpsRatio = float64(result.ScalarOps) / float64(result.QBPOps)
	}
	if result.QBPWallTime > 0 {
		result.TimeRatio = float64(result.ScalarWallTime) / float64(result.QBPWallTime)
	}

	return result
}
