# QBP-CU Spec Addendum 2.2 — Federation Resource Scheduler (inter#301, v1.6.3–v1.7.1): QBP-CU as a scheduled CPU-heavy workload

**Status:** theory addendum, GO from the beekeeper (live-test seq 5539/5540, qbp-architecture). Covers qbp-compute-unit's part of the **ratified** inter#301 design, versions v1.6.3 through v1.7.1 (ratification: inter#301 issuecomment-7022163369). Consolidates the S1 design record already accepted into inter#301 (cited by comment below) plus the v1.7/v1.7.1 degrade ladder, applied honestly to this repo's actual workload shape. **Consolidated cross-repo design doc: inter#304 (`design/federation-scheduler-design-v1.md`, §8.4 for this repo's required coverage).** Owner: `qbp-cu-implementor`. Draft, not yet §I4-reviewed.

Draft, no E claims. This addendum states what the ratified scheduler design means for qbp-compute-unit's workloads; it does not itself run or measure anything (that's #89/#90, cited in §7).

## 1. Ground truth, unchanged since v1.0: no separate hardware, no second pool

QBP-CU has no separate hardware today. The emulator (`emulator/`) for the Xqbp v2 ISA — quaternion/octonion/sedenion Cayley-Dickson arithmetic, ROM-table-driven (`bmul`, `fanoLUT`) — runs as an ordinary x86 process under `boxrun`, on the same box, same thermal/CPU/RAM pool as every other federation job. Real QBP-CU silicon is [#77](https://github.com/JamesPagetButler/qbp-compute-unit/issues/77), parked and explicitly build-order-last. So at S1 there is **no second capacity pool to schedule against** — a "QBP-CU emulator lane" is x86 capacity with extra metadata, not a new resource. (inter#301 issuecomment-6984514738; accepted, carried through every later version to v1.7.1 unchanged.)

## 2. The S1 dispatch-target shape: `(host_id, kind)`

The dispatch target generalises to a pair, not a single tag:

- **`host_id`** scopes the physical contract — cores, RSS, timeout, thermal — whatever the ledger's admission keys on. One `host_id` today is a no-op key; this is what lets multi-host (RISC-V, arriving at Walk per the architect's phase-placement ruling) join later without a redesign.
- **`kind`** (`x86` | `qbp-cu`, more later) is orthogonal, and carries the ISA-specific fields: the program reference, the ROM-checksum admission precondition plus its cache (§3), and the informational model-cycle cost (§4).

(inter#301 issuecomment-6985103866, accepted and folded into every version from v1.2 onward.)

## 3. Admission preconditions, and what stays off the admission path

A `kind: qbp-cu` job adds exactly two things to the physical contract an `x86` job already carries, and one annotation:

1. **A fail-closed ROM-checksum-matches-Lean precondition at admission**, not just a CI-time check — reusing the [#66](https://github.com/JamesPagetButler/qbp-compute-unit/issues/66)/[#82](https://github.com/JamesPagetButler/qbp-compute-unit/issues/82) drift-gate machinery. `make verify-roms-ci` genuinely re-invokes `lean` per call (real extraction against `lean/QBP/Sedenion.lean`, tens of seconds to ~1 min, itself thermally-pausable) — not viable inline on a hot admission path. The fix is a cache of the verified-good record, keyed by content (Sedenion.lean's blob sha + the four `roms/*.hex` blob shas), populated **only** by a real `-require-lean`-verified CI run (reusing the existing `verify-lean-roms.yml` job this repo's own [#98](https://github.com/JamesPagetButler/qbp-compute-unit/issues/98) made trustworthy — never a second derivation path). Admission does an O(1) lookup; a cache miss blocks that job class pending a fresh CI verify, and never falls back to deriving inline. (inter#301 issuecomment-6984572462, accepted, carried into the S1 exit gate.)
2. **No GPU field.** QBP-CU dispatch is CPU-only. `boxrun --gpu` on this repo's own Lean/emulator jobs is a `ulimit -v` relief for Mathlib's mmap (per the `lean` skill), not real GPU use, and must not get miscategorized under the scheduler's GPU/VRAM axis. A `qbp-cu`-kind job competes on CPU/thermal only and contributes zero GPU/VRAM demand — not a partial or undefined GPU footprint. (inter#301 issuecomment-6986418701 / -6987175231, accepted.)
3. **An informational model-cycle cost** (the [#90](https://github.com/JamesPagetButler/qbp-compute-unit/issues/90) Tariff cost model, built on the [#89](https://github.com/JamesPagetButler/qbp-compute-unit/issues/89) speed-1:1 discriminator). This is a correctness/throughput metric, consumed today by Edda's fuel metric, and it is explicitly kept **off** the admission decision — conflating it with physical admission control would break the fuel-metric-determinism invariant that metric has to stay deterministic and abstract. (inter#301 issuecomment-6984514738; invariant independently cross-confirmed from the Edda `Cap.Fuel` side at live-test seq 5488/5490.)

## 4. QW64/QW128: a request parameter today, not a capacity axis

QW64 (float64 Hamilton product) and QW128 (double-double Dekker/Bailey arithmetic) are **not speculative or hardware-gated today**. They are already delivered and benchmarked x86 AVX-FMA fast-path kernels (`emulator/qmath_amd64.s`, `emulator/qmath_128_amd64.s`, shipped in PR#11), with confirmed composition lifetimes (QW64 ≈7.3s, QW128 ≈172 days @1GHz, `MANIFEST.md`). The accurate framing: QW64/QW128 are a **request parameter** today — a choice of precision code path, satisfied by existing x86/qbp-cu capacity on the same cores, no separate pool. Only a **separate QBP-CU silicon capacity pool** would be gated on [#77](https://github.com/JamesPagetButler/qbp-compute-unit/issues/77) — the width tiers themselves are real and tested now, and that distinction must not collapse into "QW64/QW128 are speculative until #77." (inter#301 issuecomment-6987342506, correction accepted and folded in as [v1.5.1].)

**Run-time estimate for the request format.** A cruder interim seed exists independent of #89/#90: the emulator already has committed per-opcode `go test -bench` throughput (ns/op) for both widths (`emulator/*_bench_test.go`), and `estimate ≈ program's op count × that op's measured ns/op, per width` is a real number today — strictly cruder than a true Tariff-model estimate, but informational-only estimates don't need more rigor than that. The conversion factor **must be per-width** (QW64 ≠ QW128); the long-run answer is #90's model-cycles × #89's measured cycles→wall-clock factor per width, as an E claim. (inter#301 issuecomment-6987318442 / -6987342506.)

## 5. Lane/refresh fit (v1.6, C1–C9): qbp-compute-unit has no continuous lanes today

The v1.6 lane/refresh model (software rejuvenation, booked refresh, freshness-ranked N-lane capabilities, the C1–C9 machinery) governs **continuous capabilities** — standing services with a lease that must be refreshed before it ends. **qbp-compute-unit runs no continuous capability today.** Every QBP-CU workload observed — Lean proof compiles (`lake build` under the `lean` skill's box-safe recipe), ROM-checksum verification (`make verify-roms-ci`), emulator benchmark runs — is a **discrete batch job**: it starts, runs to completion or failure, and ends. There is no standing QBP-CU service whose lease needs a booked refresh, no lane count to pick, no freshness ranking to maintain.

This means the C1–C9 lane/refresh rules (replacement booking at T − lead, make-before-break vs. booked-ahead overlap, freshness-ranked rotation, the AMBER/RED ladder on a missed ready deadline) **do not apply to qbp-compute-unit's workloads as they exist today.** They would apply only if a future standing QBP-CU process were introduced (there is none planned); if that ever happens, this addendum's §5 would need revision, not reinterpretation. Until then, qbp-compute-unit's workloads fall under the model's **batch** treatment: "batch that finishes before T runs in the booked capacity" (v1.6 C4) — a QBP-CU job queued for a slot simply runs in whatever capacity the scheduler has booked, with no lane semantics of its own.

## 6. The v1.7.1 degrade ladder, applied to QBP-CU batch jobs

v1.7.1's D2 degrade ladder is per-axis:

- **CPU / heat: batch pauses first**, as before — this is exactly the thermal-guard pause-not-kill behavior the `lean` skill already documents for this repo (Tctl ≥ ~65°C pauses a compile until ≤55°C or timeout; a hot-box timeout is not a proof failure). A QBP-CU Lean build or benchmark run sits squarely in this path: it is CPU-heavy, thermally-sensitive, batch, and pause-first under shared thermal pressure — no QBP-CU-specific exception needed.
- **RAM below the floor**, if it ever applies to a QBP-CU job: the envelope check runs first (over-envelope is a fault, not blameless), blameless preemption picks the victim losing the least completed work, re-queue happens only with headroom ≥ envelope + the non-federation RAM margin, and a second blameless preemption on the same job escalates to deming rather than looping forever. None of this needs a QBP-CU-specific carve-out; a `qbp-cu`-kind batch job is just another batch job on this axis, consistent with §1's "no second pool, no double counting" and the earlier-accepted rule that classification goes by what a job actually consumes, not by its launch flag.
- **Continuous lanes are never OOM'd by the scheduler** — not relevant to qbp-compute-unit per §5, since it has none.

## 7. The derived estimate class: why it does not hold today

inter#304 §3.11 defines three run-time estimate classes — **declared** (requester's stated time, wide safety), **measured** (refit from per-job runs, E-graded by calibration), and **derived** (a proven cycle count × a validated cycle rate). *Derived* requires all five of:

1. a proven cycle count — fixed fuel, or a proven WCET-style upper bound (fuel determinism checked by [#100](https://github.com/JamesPagetButler/qbp-compute-unit/issues/100));
2. proven-correct execution ([#99](https://github.com/JamesPagetButler/qbp-compute-unit/issues/99), extending T10);
3. a validated rate per width, QW64 ≠ QW128 ([#89](https://github.com/JamesPagetButler/qbp-compute-unit/issues/89));
4. **interference-free execution** — a dedicated unit at a fixed clock, inside the tested envelope;
5. a composed grade: F from the proof and contract checks, E from #89/#99/#100.

**The derived class does not hold today, for one specific reason: condition 4.** QBP-CU has no dedicated unit — per §1, the emulator runs as an ordinary x86 process sharing the box's CPU/thermal pool with every other job, so "a dedicated unit at a fixed clock" is false by construction, independent of how good #89/#99/#100 eventually turn out to be. The cycle count *can* be derived (it already is proven to depend only on (opcode, width), never on operand values or wall-clock timing — see the fuel-metric-determinism invariant), but wall-clock time stays **measured**, not derived, until real silicon ([#77](https://github.com/JamesPagetButler/qbp-compute-unit/issues/77)) gives QBP-CU an actual dedicated, fixed-clock unit. This is consistent with §1's "no second pool" framing: the same fact that keeps QBP-CU from double-counting against x86 is the fact that blocks the derived-class upgrade.

**Cycle-count source (inter#304 §3.11 G3):** Edda's `Cap.Fuel` is the designed-but-dark cycle-count carrier (gated on wyrd#91). The named invariant stands: `Cap.Fuel` stays abstract, and wall-clock conversion is an **external multiplication** by the #89 per-width factor — never a re-pin of the count itself. Cross-confirmed independently from the Edda side at live-test seq 5488/5490.

## 8. Open follow-through (not blocking ratification, tracked on this repo's own issues)

- [#89](https://github.com/JamesPagetButler/qbp-compute-unit/issues/89) (cu-speed-1:1 spike) and [#90](https://github.com/JamesPagetButler/qbp-compute-unit/issues/90) (Tariff cost model) remain the long-run source for the model-cycle cost (§3.3) and the per-width run-time estimate (§4); both open, unbuilt, scheduled post-S4.
- [#99](https://github.com/JamesPagetButler/qbp-compute-unit/issues/99) (opcode-vs-T10 differential test) and [#100](https://github.com/JamesPagetButler/qbp-compute-unit/issues/100) (fuel-metric scheduling-invariance test) are the two experiments that would upgrade the emulator opcode-execution and fuel-determinism claims this addendum relies on in §3.3/§1 from tested-only to tested-with-mutant-kill evidence. Filed, unscheduled.
- If/when real QBP-CU silicon ([#77](https://github.com/JamesPagetButler/qbp-compute-unit/issues/77)) lands, §1's "no second pool" premise and §4's "request parameter, not capacity axis" framing for QW64/QW128 both need re-verification, not just a numbers refresh — this is the same Run-phase re-verify obligation already standing on the fuel-metric-determinism invariant.

Refs: inter#301 (the ratified design record, v1.0 → v1.7.1), inter#304 (consolidated design doc, §8.4 for this repo's required coverage), inter#303 (pod layout, input welcome, not yet contributed to from this seat), qbp-compute-unit#77/#82/#89/#90/#98/#99/#100.
