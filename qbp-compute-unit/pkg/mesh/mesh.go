// Package mesh implements the QBP Compute Unit's dynamic precision scheduler.
//
// CORE PRINCIPLE: Precision is a property of the TASK, not the HARDWARE.
// Each physical node in the mesh operates at a native width (e.g., QW64).
// Tasks that require higher precision are assigned multiple nodes that
// work as a coordinated group, jointly implementing wider quaternion words.
// Tasks that require lower precision pack multiple independent operations
// into a single node via SIMD.
//
// The mesh topology replicates the Fano plane at each scale: 7 nodes,
// 7 hyperedges, each connecting 3 nodes. This structure determines how
// node groups are formed and how they communicate.
//
// SCHEDULING MODEL:
//
//	Task declares: required composition depth (N operations)
//	               and tolerance (max acceptable norm drift)
//	Scheduler computes: required width from drift-per-op formula
//	                    → number of nodes per group
//	Mesh allocates: node group from available capacity
//	                routes through Fano topology
//	Watchdog monitors: actual drift during execution
//	                   can trigger reallocation if drift exceeds prediction
//
// EXAMPLE (7-node Fano cell, each node native QW64):
//
//	┌─────────────────────────────────────────────────────┐
//	│ Allocation A: 7 × QW64 tasks (max throughput)       │
//	│   [n1] [n2] [n3] [n4] [n5] [n6] [n7]               │
//	│   Each node: independent QW64 task                  │
//	│   Throughput: 7 tasks/cycle                         │
//	│                                                     │
//	│ Allocation B: 3 × QW128 + 1 × QW64 (physics mix)   │
//	│   [n1+n2] [n3+n4] [n5+n6] [n7]                     │
//	│   Three pairs: coordinated QW128                    │
//	│   One solo: QW64 housekeeping                       │
//	│   Throughput: 4 tasks/cycle (3 high-precision)      │
//	│                                                     │
//	│ Allocation C: 1 × QW256 + 3 × QW64 (max precision) │
//	│   [n1+n2+n3+n4] [n5] [n6] [n7]                     │
//	│   One quad: QW256 physics task                      │
//	│   Three solos: QW64 monitoring/housekeeping         │
//	│   Throughput: 4 tasks/cycle (1 ultra-high-precision) │
//	│                                                     │
//	│ Allocation D: 28 × QW8 (hypergraph traversal burst) │
//	│   Each node: 4 × QW8 via SIMD packing              │
//	│   Throughput: 28 tasks/cycle                        │
//	└─────────────────────────────────────────────────────┘
package mesh

import (
	"fmt"
	"math"
	"sync"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/qword"
)

// ─── Task ──────────────────────────────────────────────────────────

// Task represents a unit of QBP computation with a declared precision
// requirement. The scheduler inspects these fields to determine how
// many mesh nodes to allocate.
type Task struct {
	ID string

	// Declared by the task creator:
	CompositionDepth int64   // estimated number of chained QMUL operations
	DriftTolerance   float64 // max acceptable |1 - ||q||²| at completion

	// Computed by the scheduler:
	RequiredWidth qword.Width // minimum width to meet tolerance at declared depth
	NodesNeeded   int         // number of native-width nodes to form the group
	Allocated     bool        // whether the scheduler has assigned nodes

	// Runtime (updated by watchdog):
	ActualDepth  int64   // operations completed so far
	ActualDrift  float64 // current measured drift
	NeedsRealloc bool    // watchdog signals that drift exceeds prediction
}

// RequiredWidthFor computes the minimum quaternion word width needed to
// sustain the given composition depth within the given drift tolerance.
//
// Based on empirical data: drift_per_op ≈ ε² × 28 where ε is machine
// epsilon for the component type, and 28 is the FP operation count in
// one Hamilton product.
//
// Returns the smallest standard width that satisfies:
//
//	depth × drift_per_op < tolerance
func RequiredWidthFor(depth int64, tolerance float64) qword.Width {
	widths := []qword.Width{
		qword.W8, qword.W16, qword.W32,
		qword.W64, qword.W128, qword.W256,
	}

	for _, w := range widths {
		maxDepth := qword.MaxCompositionDepth(w, tolerance)
		if maxDepth >= depth {
			return w
		}
	}
	return qword.W256 // if nothing else suffices
}

// NodesForWidth returns how many native-width nodes are needed to
// implement a task at the required width.
//
// If the required width is smaller than or equal to native, it's 1
// (or fractional via SIMD packing, represented as 1 here).
// If larger, it's the ratio rounded up.
func NodesForWidth(required, native qword.Width) int {
	if required <= native {
		return 1
	}
	// Each doubling of width requires 2× nodes for the carry chain
	ratio := int(required) / int(native)
	return ratio
}

// SIMDPackingFactor returns how many independent tasks at the required
// width can be packed into a single native-width node via SIMD.
// Only applicable when required < native.
func SIMDPackingFactor(required, native qword.Width) int {
	if required >= native {
		return 1
	}
	return int(native) / int(required)
}

// ─── Node ──────────────────────────────────────────────────────────

// Node represents a single physical compute node in the mesh.
// Each node has a native precision (determined by hardware) and can be
// assigned to a task group.
type Node struct {
	ID          int
	NativeWidth qword.Width
	AssignedTo  string // task ID, or "" if free
	GroupRole   int    // position within a multi-node group (0 = lead)
}

// ─── Fano Cell ─────────────────────────────────────────────────────

// FanoCell represents one 7-node unit of the mesh topology.
// The Fano plane structure (7 nodes, 7 hyperedges of 3) determines
// communication patterns within the cell.
type FanoCell struct {
	mu    sync.Mutex
	Nodes [7]*Node
	ID    int

	// Fano lines: each triple of node indices that share a hyperedge.
	// These determine which nodes can form efficient groups.
	Lines [7][3]int
}

// NewFanoCell creates a 7-node cell with the standard Fano topology.
func NewFanoCell(cellID int, nativeWidth qword.Width) *FanoCell {
	c := &FanoCell{
		ID: cellID,
		// Standard Fano lines (0-indexed node IDs within cell)
		Lines: [7][3]int{
			{0, 1, 2}, // line 1
			{0, 3, 4}, // line 2
			{0, 6, 5}, // line 3
			{1, 3, 5}, // line 4
			{1, 4, 6}, // line 5
			{2, 3, 6}, // line 6
			{2, 5, 4}, // line 7
		},
	}
	for i := 0; i < 7; i++ {
		c.Nodes[i] = &Node{
			ID:          cellID*7 + i,
			NativeWidth: nativeWidth,
		}
	}
	return c
}

// FreeNodes returns the number of unassigned nodes.
func (c *FanoCell) FreeNodes() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	count := 0
	for _, n := range c.Nodes {
		if n.AssignedTo == "" {
			count++
		}
	}
	return count
}

// AllocateGroup assigns n free nodes to a task, preferring nodes that
// share a Fano line (for efficient intra-group communication).
// Returns the allocated node IDs, or nil if insufficient capacity.
func (c *FanoCell) AllocateGroup(taskID string, needed int) []int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if needed > 7 {
		return nil // single cell can't satisfy
	}

	// First, try to find a Fano line whose nodes are all free
	// (optimal for 2-3 node groups)
	if needed <= 3 {
		for _, line := range c.Lines {
			allFree := true
			for _, idx := range line {
				if c.Nodes[idx].AssignedTo != "" {
					allFree = false
					break
				}
			}
			if allFree {
				ids := make([]int, needed)
				for i := 0; i < needed; i++ {
					c.Nodes[line[i]].AssignedTo = taskID
					c.Nodes[line[i]].GroupRole = i
					ids[i] = c.Nodes[line[i]].ID
				}
				return ids
			}
		}
	}

	// Fallback: allocate any free nodes
	ids := make([]int, 0, needed)
	role := 0
	for _, n := range c.Nodes {
		if n.AssignedTo == "" && len(ids) < needed {
			n.AssignedTo = taskID
			n.GroupRole = role
			ids = append(ids, n.ID)
			role++
		}
	}
	if len(ids) < needed {
		// Rollback
		for _, n := range c.Nodes {
			if n.AssignedTo == taskID {
				n.AssignedTo = ""
				n.GroupRole = 0
			}
		}
		return nil
	}
	return ids
}

// Release frees all nodes assigned to a task.
func (c *FanoCell) Release(taskID string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	freed := 0
	for _, n := range c.Nodes {
		if n.AssignedTo == taskID {
			n.AssignedTo = ""
			n.GroupRole = 0
			freed++
		}
	}
	return freed
}

// ─── Scheduler ─────────────────────────────────────────────────────

// Scheduler manages task-to-node allocation across one or more Fano cells.
type Scheduler struct {
	mu          sync.Mutex
	Cells       []*FanoCell
	NativeWidth qword.Width
	Tasks       map[string]*Task
}

// NewScheduler creates a scheduler managing the given number of Fano cells.
func NewScheduler(numCells int, nativeWidth qword.Width) *Scheduler {
	s := &Scheduler{
		NativeWidth: nativeWidth,
		Tasks:       make(map[string]*Task),
		Cells:       make([]*FanoCell, numCells),
	}
	for i := 0; i < numCells; i++ {
		s.Cells[i] = NewFanoCell(i, nativeWidth)
	}
	return s
}

// Submit accepts a task, computes its precision requirement, and attempts
// to allocate mesh nodes. Returns an error if insufficient capacity.
func (s *Scheduler) Submit(t *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Compute required width from task's declared depth and tolerance
	t.RequiredWidth = RequiredWidthFor(t.CompositionDepth, t.DriftTolerance)
	t.NodesNeeded = NodesForWidth(t.RequiredWidth, s.NativeWidth)

	// Find a cell with enough free nodes
	for _, cell := range s.Cells {
		ids := cell.AllocateGroup(t.ID, t.NodesNeeded)
		if ids != nil {
			t.Allocated = true
			s.Tasks[t.ID] = t
			return nil
		}
	}

	return fmt.Errorf("insufficient capacity: task %s needs %d nodes (width %s), none available",
		t.ID, t.NodesNeeded, t.RequiredWidth)
}

// Complete marks a task as finished and releases its nodes.
func (s *Scheduler) Complete(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, cell := range s.Cells {
		cell.Release(taskID)
	}
	delete(s.Tasks, taskID)
}

// Status returns a summary of current mesh utilisation.
func (s *Scheduler) Status() MeshStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := MeshStatus{
		TotalNodes:  len(s.Cells) * 7,
		NativeWidth: s.NativeWidth,
	}
	for _, cell := range s.Cells {
		st.FreeNodes += cell.FreeNodes()
	}
	st.UsedNodes = st.TotalNodes - st.FreeNodes
	st.ActiveTasks = len(s.Tasks)

	// Compute effective throughput
	// Each free node can run 1 native-width task per cycle
	// Each SIMD-packed node can run multiple sub-native tasks
	st.Utilisation = float64(st.UsedNodes) / float64(st.TotalNodes)

	return st
}

// MeshStatus reports current mesh utilisation.
type MeshStatus struct {
	TotalNodes  int
	FreeNodes   int
	UsedNodes   int
	ActiveTasks int
	NativeWidth qword.Width
	Utilisation float64 // 0.0 to 1.0
}

func (ms MeshStatus) String() string {
	return fmt.Sprintf(
		"Mesh: %d/%d nodes used (%.0f%% util), %d active tasks, native %s",
		ms.UsedNodes, ms.TotalNodes, ms.Utilisation*100,
		ms.ActiveTasks, ms.NativeWidth,
	)
}

// ─── Utility: estimate nodes needed for common scenarios ───────────

// EstimateAllocation returns what the scheduler would allocate for a
// given workload without actually allocating. Useful for capacity planning.
func EstimateAllocation(depth int64, tolerance float64, nativeWidth qword.Width) (qword.Width, int, string) {
	w := RequiredWidthFor(depth, tolerance)
	n := NodesForWidth(w, nativeWidth)
	simd := SIMDPackingFactor(w, nativeWidth)

	desc := fmt.Sprintf("depth=%d tol=%.1e → %s (%d nodes",
		depth, tolerance, w, n)
	if simd > 1 {
		desc += fmt.Sprintf(", %d× SIMD packing", simd)
	}
	desc += ")"

	maxLife := qword.MaxCompositionDepth(w, tolerance)
	if maxLife > depth {
		headroom := float64(maxLife) / float64(depth)
		desc += fmt.Sprintf(" [%.0f× headroom]", headroom)
	}

	return w, n, desc
}

// ─── Watchdog integration ──────────────────────────────────────────

// CheckReallocation inspects a running task's actual drift and determines
// whether it needs more nodes (higher precision) or could release nodes
// (drift lower than predicted).
func CheckReallocation(t *Task) ReallocationAdvice {
	if t.ActualDepth == 0 {
		return ReallocationAdvice{Action: "none", Reason: "no data yet"}
	}

	actualDriftRate := t.ActualDrift / float64(t.ActualDepth)
	remainingDepth := t.CompositionDepth - t.ActualDepth
	projectedFinalDrift := t.ActualDrift + actualDriftRate*float64(remainingDepth)

	if projectedFinalDrift > t.DriftTolerance*1.5 {
		// Drift exceeding budget — need more precision
		newWidth := RequiredWidthFor(t.CompositionDepth, t.DriftTolerance/2)
		return ReallocationAdvice{
			Action:       "upgrade",
			Reason:       fmt.Sprintf("projected drift %.2e exceeds tolerance %.2e", projectedFinalDrift, t.DriftTolerance),
			CurrentWidth: t.RequiredWidth,
			AdvisedWidth: newWidth,
		}
	}

	if projectedFinalDrift < t.DriftTolerance*0.01 && t.RequiredWidth > qword.W8 {
		// Drift well under budget — could release nodes
		// Find the smallest width that still covers the projected drift
		for _, w := range []qword.Width{qword.W8, qword.W16, qword.W32, qword.W64, qword.W128} {
			maxD := qword.MaxCompositionDepth(w, t.DriftTolerance)
			if maxD >= t.CompositionDepth {
				if w < t.RequiredWidth {
					return ReallocationAdvice{
						Action:       "downgrade",
						Reason:       fmt.Sprintf("projected drift %.2e well under tolerance %.2e", projectedFinalDrift, t.DriftTolerance),
						CurrentWidth: t.RequiredWidth,
						AdvisedWidth: w,
					}
				}
				break
			}
		}
	}

	return ReallocationAdvice{
		Action: "none",
		Reason: fmt.Sprintf("on track: projected drift %.2e within tolerance %.2e",
			projectedFinalDrift, t.DriftTolerance),
	}
}

// ReallocationAdvice tells the scheduler whether a task needs more or fewer nodes.
type ReallocationAdvice struct {
	Action       string // "none", "upgrade", "downgrade"
	Reason       string
	CurrentWidth qword.Width
	AdvisedWidth qword.Width
}

func (ra ReallocationAdvice) String() string {
	if ra.Action == "none" {
		return fmt.Sprintf("[%s] %s", ra.Action, ra.Reason)
	}
	return fmt.Sprintf("[%s] %s → %s: %s", ra.Action, ra.CurrentWidth, ra.AdvisedWidth, ra.Reason)
}

// ─── Helper ────────────────────────────────────────────────────────

func max(a, b float64) float64 {
	return math.Max(a, b)
}
