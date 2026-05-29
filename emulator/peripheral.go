package emulator

import (
	"context"
	"log/slog"
	"sync/atomic"
)

// peripheralState holds the runtime state for the peripheral goroutine.
// Lifecycle: allocated by OnSeam or StartPeripheral; goroutine started
// by StartPeripheral; goroutine stopped by StopPeripheral or ctx cancel.
// All fields except callback (atomic) are written before the goroutine
// starts or are themselves atomic; no additional lock is needed inside run.
type peripheralState struct {
	operands chan [2]QW8                     // buffered; capacity 256 per §2.1 design
	stop     chan struct{}                   // closed by StopPeripheral to signal exit
	done     chan struct{}                   // closed when goroutine exits
	running  atomic.Bool                     // true while goroutine is executing
	seamID   atomic.Uint64                   // monotonic per-Gearbox seam counter
	callback atomic.Pointer[func(SeamEvent)] // hot-swappable via OnSeam
}

// defaultSeamThreshold is the L∞ residue threshold at QW8 precision.
// Matches the A18 §9 Q2 compile-time default K=10 at Crawl phase.
const defaultSeamThreshold int32 = 10

// OnSeam registers a callback for Seam-detection events from the
// peripheral register. The callback runs on the peripheral goroutine;
// long-running work must be dispatched to a caller-owned goroutine to
// avoid blocking the peripheral scan throughput.
//
// Calling OnSeam(nil) clears the callback. Multiple registrations
// replace the previous one atomically; the peripheral loop observes the
// new callback on its next operand-pair scan.
//
// May be called before or after StartPeripheral. Thread-safe.
func (g *Gearbox) OnSeam(cb func(SeamEvent)) {
	g.mu.Lock()
	if g.peripheral == nil {
		// Lazily allocate so the callback can be stored before
		// StartPeripheral is called.
		g.peripheral = newPeripheralState()
	}
	ps := g.peripheral
	g.mu.Unlock()
	ps.callback.Store(cbPtr(cb))
}

// StartPeripheral spawns the peripheral-register goroutine. Runs until
// StopPeripheral is called or ctx is cancelled. The goroutine continuously
// scans operand pairs supplied via SubmitPeripheral and emits SeamEvent
// on the registered OnSeam callback when a Seam is detected.
//
// Idempotent: calling StartPeripheral when the peripheral goroutine is
// already running returns nil without spawning a second goroutine.
// Always runs at QW8 precision (peripheral default).
func (g *Gearbox) StartPeripheral(ctx context.Context) error {
	g.mu.Lock()
	if g.peripheral == nil {
		g.peripheral = newPeripheralState()
	}
	ps := g.peripheral
	g.mu.Unlock()

	if !ps.running.CompareAndSwap(false, true) {
		return nil // already running
	}

	go func() {
		ps.run(ctx, g)
		ps.running.Store(false)
		close(ps.done)
	}()
	return nil
}

// StopPeripheral signals the peripheral goroutine to drain and exit.
// Blocks until the goroutine has exited. Safe to call when the peripheral
// is not running (no-op).
func (g *Gearbox) StopPeripheral() {
	g.mu.Lock()
	ps := g.peripheral
	g.mu.Unlock()
	if ps == nil || !ps.running.Load() {
		return
	}
	select {
	case <-ps.stop:
		// already closed — goroutine may still be draining
	default:
		close(ps.stop)
	}
	<-ps.done
}

// SubmitPeripheral hands an operand pair (q, v) to the peripheral
// register for Seam-detection scanning. Non-blocking: returns false if
// the operand channel is full (capacity 256) or the peripheral is not
// running. Submissions made before StartPeripheral or after StopPeripheral
// are dropped.
func (g *Gearbox) SubmitPeripheral(q, v [4]int8) bool {
	g.mu.RLock()
	ps := g.peripheral
	g.mu.RUnlock()
	if ps == nil || !ps.running.Load() {
		return false
	}
	pair := [2]QW8{QW8(q), QW8(v)}
	select {
	case ps.operands <- pair:
		return true
	default:
		return false
	}
}

// run is the peripheral goroutine body. Reads operand pairs from the
// operands channel, runs DetectSeam8, and fires the registered callback
// on detection. Exits when ctx is cancelled or stop is closed.
func (ps *peripheralState) run(ctx context.Context, g *Gearbox) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ps.stop:
			return
		case pair, ok := <-ps.operands:
			if !ok {
				return
			}
			ps.scanPair(g, pair[0], pair[1])
		}
	}
}

// scanPair runs Seam detection on a single operand pair and fires the
// callback if a Seam is detected. Runs on the peripheral goroutine.
func (ps *peripheralState) scanPair(g *Gearbox, q, v QW8) {
	cbp := ps.callback.Load()
	if cbp == nil {
		return // no callback registered; discard
	}

	// DetectSeam8 acquires mu.RLock internally.
	residueI32, detected := g.DetectSeam8(q, v, defaultSeamThreshold)
	if !detected {
		return
	}

	seamID := ps.seamID.Add(1)

	// Acquire the canonical cpu cycle; fall back to seamID when no CPU
	// is wired (test-fixture path). slog.Debug marks the fallback so
	// test telemetry doesn't silently look like production telemetry.
	var cycle uint64
	if cpuCycle := gearboxCPU(g); cpuCycle != 0 {
		cycle = cpuCycle
	} else {
		slog.Debug("peripheral: cpu nil — using internal cycle counter")
		cycle = seamID
	}

	residue := float32(residueI32)
	threshold := float32(defaultSeamThreshold)
	var magnitude float32
	if threshold > 0 {
		magnitude = residue / threshold
		if magnitude > 1.0 {
			magnitude = 1.0
		}
	}

	ev := SeamEvent{
		Q:             q,
		V:             v,
		Residue:       residue,
		Threshold:     threshold,
		PrecisionTier: W8,
		Cycle:         cycle,
		SeamID:        seamID,
		Magnitude:     magnitude,
		// Locale and DetectionContext zero by default; BMA harness
		// (OnSeamDispatcher) populates Locale from working-tree context.
	}
	(*cbp)(ev)
}

// gearboxCPU returns the canonical accelerator cycle from the CPU
// associated with this Gearbox. Returns 0 if no CPU is wired.
//
// Crawl-phase stub: no CPU pointer on Gearbox yet. Walk-α: CPU pointer
// injection tracked as follow-up to this PR.
func gearboxCPU(_ *Gearbox) uint64 {
	return 0
}

// newPeripheralState allocates a fresh peripheralState with unbuffered
// lifecycle channels and the operands channel at capacity 256.
func newPeripheralState() *peripheralState {
	return &peripheralState{
		operands: make(chan [2]QW8, 256),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// cbPtr returns a *func(SeamEvent) for cb, or nil if cb is nil.
func cbPtr(cb func(SeamEvent)) *func(SeamEvent) {
	if cb == nil {
		return nil
	}
	return &cb
}
