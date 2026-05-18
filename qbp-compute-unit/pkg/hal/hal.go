// Package hal provides the Hardware Abstraction Layer for the QBP Compute Unit.
//
// The HAL presents a uniform interface for quaternion-algebraic operations
// regardless of the backend hardware. Application code calls hal.QMUL(),
// hal.OMAC(), hal.FANO(), etc. The backend determines how they execute:
//
//	Crawl:  Pure Go (software Hamilton product)
//	Walk:   Go assembly with AVX, or CGo → ROCm HIP on GPU
//	Run:    RISC-V custom instructions via Go assembly .word directives
//	Glide:  SLM hologram rendering (optical SU(2)) + NV-centre control
//	Fly:    Diamond module device driver (/dev/qbp0)
//
// The HAL is selected at initialisation and cannot change during execution.
// Application code is phase-independent: the same Go source runs on every
// phase by changing only the HAL backend selection.
//
// LINUX INTEGRATION:
//
//	Every phase runs Linux on a host processor. The QBP-specific hardware
//	is an accelerator/peripheral accessed through standard Linux mechanisms:
//
//	Phase    Host              QBP Hardware        Linux Interface
//	─────    ────              ────────────        ───────────────
//	Crawl    FX-8350           (none, SW only)     Go runtime
//	Walk     FX-8350           RX 9070 XT GPU      /dev/kfd, /dev/dri (ROCm)
//	Run      RISC-V SoC        Custom ISA ext.     Userspace instructions
//	Glide    Any Linux host    SLM + NV cavity     HDMI/DRM + USB/PCIe
//	Fly      RISC-V SoC        Diamond module      /dev/qbp0 (custom driver)
package hal

import (
	"fmt"

	"runtime"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/fano"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
)

// ─── Backend interface ─────────────────────────────────────────────

// Backend defines the operations every HAL implementation must provide.
// This is the contract between application code and hardware.
type Backend interface {
	// Name returns a human-readable identifier for this backend.
	Name() string

	// Phase returns which hardware phase this backend targets.
	Phase() Phase

	// QMUL executes a quaternion Hamilton product.
	QMUL(a, b quat.Quat) quat.Quat

	// QROT executes a quaternion rotation: q v q*.
	QROT(q, v quat.Quat) quat.Quat

	// FANO executes a Fano-plane lookup for octonionic composition.
	FANO(i, j int) fano.Entry

	// QNorm returns ||q||².
	QNormSq(q quat.Quat) float64

	// Available returns true if this backend can execute on the current system.
	// For example, the ROCm backend returns false if no AMD GPU is present.
	Available() bool
}

// Phase identifies the hardware phase.
type Phase int

const (
	PhaseCrawl Phase = iota
	PhaseWalk
	PhaseRun
	PhaseGlide
	PhaseFly
)

func (p Phase) String() string {
	switch p {
	case PhaseCrawl:
		return "Crawl (Software)"
	case PhaseWalk:
		return "Walk (GPU/AVX)"
	case PhaseRun:
		return "Run (RISC-V Custom ISA)"
	case PhaseGlide:
		return "Glide (Optical SU(2) + NV Centre)"
	case PhaseFly:
		return "Fly (Diamond Module)"
	default:
		return "Unknown"
	}
}

// ─── Global HAL state ──────────────────────────────────────────────

var activeBackend Backend

// Init selects and initialises a HAL backend.
// Call once at program startup. Panics if backend is not available.
func Init(b Backend) {
	if !b.Available() {
		panic(fmt.Sprintf("HAL backend %q is not available on this system", b.Name()))
	}
	activeBackend = b
}

// Active returns the currently selected backend, or nil if uninitialised.
func Active() Backend {
	return activeBackend
}

// ─── Application-facing API ────────────────────────────────────────
// These are the functions application code calls. They dispatch to
// whatever backend is currently active.

// QMUL computes the Hamilton product a * b.
func QMUL(a, b quat.Quat) quat.Quat {
	return activeBackend.QMUL(a, b)
}

// QROT applies quaternion rotation q to vector v: q v q*.
func QROT(q, v quat.Quat) quat.Quat {
	return activeBackend.QROT(q, v)
}

// FANO looks up the octonionic product e_i × e_j.
func FANO(i, j int) fano.Entry {
	return activeBackend.FANO(i, j)
}

// QNormSq returns ||q||².
func QNormSq(q quat.Quat) float64 {
	return activeBackend.QNormSq(q)
}

// ─── Crawl backend (pure Go) ───────────────────────────────────────

// CrawlBackend executes all operations in pure Go software.
// This is the default backend and is always available.
type CrawlBackend struct{}

func (CrawlBackend) Name() string                  { return "Crawl (Pure Go)" }
func (CrawlBackend) Phase() Phase                  { return PhaseCrawl }
func (CrawlBackend) Available() bool               { return true } // always available
func (CrawlBackend) QMUL(a, b quat.Quat) quat.Quat { return quat.Mul(a, b) }
func (CrawlBackend) QROT(q, v quat.Quat) quat.Quat { return quat.Rotate(q, v) }
func (CrawlBackend) FANO(i, j int) fano.Entry      { return fano.Lookup(i, j) }
func (CrawlBackend) QNormSq(q quat.Quat) float64   { return quat.NormSq(q) }

// ─── Walk backend (stub — implemented when ROCm/AVX available) ─────

// WalkBackend dispatches QMUL to AVX assembly or ROCm GPU.
// This is a stub; the actual implementation depends on build tags
// and runtime hardware detection.
type WalkBackend struct {
	UseGPU bool // false = CPU AVX, true = ROCm GPU
}

func (w WalkBackend) Name() string {
	if w.UseGPU {
		return "Walk (ROCm GPU)"
	}
	return "Walk (AVX Assembly)"
}
func (WalkBackend) Phase() Phase { return PhaseWalk }
func (WalkBackend) Available() bool {
	if runtime.GOARCH == "amd64" {
		return true // AVX assembly kernel is available on amd64
	}
	// TODO: runtime detection of ROCm devices for GPU mode
	return false
}
func (WalkBackend) QMUL(a, b quat.Quat) quat.Quat { return quat.Mul(a, b) } // placeholder
func (WalkBackend) QROT(q, v quat.Quat) quat.Quat { return quat.Rotate(q, v) }
func (WalkBackend) FANO(i, j int) fano.Entry      { return fano.Lookup(i, j) }
func (WalkBackend) QNormSq(q quat.Quat) float64   { return quat.NormSq(q) }

// ─── Run backend (stub — for custom RISC-V) ───────────────────────

// RunBackend uses custom RISC-V instructions via Go assembly.
// On non-RISC-V systems, it falls back to CrawlBackend.
type RunBackend struct{}

func (RunBackend) Name() string                  { return "Run (RISC-V QMUL.128)" }
func (RunBackend) Phase() Phase                  { return PhaseRun }
func (RunBackend) Available() bool               { return false } // until RISC-V hardware exists
func (RunBackend) QMUL(a, b quat.Quat) quat.Quat { return quat.Mul(a, b) }
func (RunBackend) QROT(q, v quat.Quat) quat.Quat { return quat.Rotate(q, v) }
func (RunBackend) FANO(i, j int) fano.Entry      { return fano.Lookup(i, j) }
func (RunBackend) QNormSq(q quat.Quat) float64   { return quat.NormSq(q) }

// ─── Auto-select the best available backend ────────────────────────

// AutoSelect returns the highest-phase backend available on this system.
// Falls back to CrawlBackend if nothing better is detected.
func AutoSelect() Backend {
	// Try backends in reverse phase order (most advanced first)
	candidates := []Backend{
		RunBackend{},
		WalkBackend{UseGPU: true},
		WalkBackend{UseGPU: false},
	}
	for _, b := range candidates {
		if b.Available() {
			return b
		}
	}
	return CrawlBackend{} // always available
}
