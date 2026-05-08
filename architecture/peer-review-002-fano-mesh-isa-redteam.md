# Peer Review 002: Red Team Audit of QBP RISC-V ISA v2.0 (Fano Mesh Integration) Plan

**Date:** 2026-05-03
**Reviewer:** Claude (Red Team)
**Subject:** Gemini's "Implementation Plan: QBP RISC-V ISA v2.0 (Fano Mesh Integration)"
**Companion references:**
- [`Ref/RISC-V-Policies-and-Best-Practices.md`](../Ref/RISC-V-Policies-and-Best-Practices.md) — RISC-V conventions
- [`Ref/SiFive-Documentation-Patterns.md`](../Ref/SiFive-Documentation-Patterns.md) — vendor doc patterns
- [`spec/QBP-RISCV-ISA-Spec-v1.0.md`](../spec/QBP-RISCV-ISA-Spec-v1.0.md) — the spec under proposed revision
- [`doc/QBP-ISA-Refinement-Report.md`](../doc/QBP-ISA-Refinement-Report.md) — context document Gemini was given
- [`doc/Hardware-Strategy.md`](../doc/Hardware-Strategy.md) — Path 1/2/3 hardware roadmap

---

## 1. Executive Summary

**Verdict: Do not draft v2.0 as proposed.** Two of three refinements have category errors that cannot be fixed by editing; one is salvageable with significant rework. The proposal also re-opens v1.0 thirteen days after approval, mid-implementation, on the basis of unverifiable energy claims, and **directly violates several established RISC-V and vendor-extension conventions** documented in the companion references.

The conformant path forward is **not** v2.0. It is two new draft extensions — `Xqbpvcp` (interface) and `Xqbpmesh` (mesh-internal) — landing at v0.1, modeled on SiFive's `XSfvcp` + `Xsfmm*` pattern, while v1.0 remains stable. This delivers everything Gemini wants without re-opening a frozen spec.

---

## 2. Framing-Level Attacks (the plan's premises don't survive scrutiny)

### F1. The "discovery" is not a discovery

Gemini opens: *"The discovery of the Fano Cell Mesh Architecture fundamentally shifts how the RISC-V processor should interact with its execution units."*

The Fano Cell mesh has been implemented in `qbp-compute-unit/pkg/mesh/mesh.go` since at least the April 2026 Master Record:
- `FanoCell` (7 nodes, 7 hyperedges of 3) at line 149
- `Scheduler` over multiple cells at line 271
- Dynamic precision allocation already in place
- Watchdog-driven reallocation already documented in `pkg/watchdog/watchdog.go`

**This is a re-labeling of the existing software scheduler as ISA-visible hardware.** Hiding a re-labeling behind the language of "discovery" inflates the apparent novelty of the change and weakens the user's ability to assess whether reopening v1.0 is justified.

**Reinforced by RISC-V policy** (cf. `RISC-V-Policies-and-Best-Practices.md` §1): once a spec is Ratified, "no changes are allowed… must be addressed through a follow-on extension." The QBP v1.0 spec is in the Stable→Frozen band. Even if Gemini's proposal were genuinely novel, the procedural form would be a follow-on extension, not a v2.0 rewrite.

### F2. The strawman comparison

*"Rather than a static, power-hungry monolithic floating-point unit (FPU)… yielding massive energy savings compared to a monolithic 1024-bit FPU."*

Nobody ships a monolithic 1024-bit FPU. The realistic comparison is against AVX-512 (512-bit SIMD with per-lane clock gating, ubiquitous since 2017) or a vector unit with lane gating. Both already get the "dark silicon" benefit Gemini ascribes uniquely to the Fano mesh. The proposal is energy-positive against a baseline that doesn't exist.

### F3. Energy claims are unfalsifiable as written

The phrase "massive energy savings" appears three times. The plan provides:
- 0 joules, watts, or millijoules
- 0 transistor counts
- 0 reference to the 130nm OpenMPW process noted in `Hardware-Strategy.md`
- 0 power model, library cell choice, or activity factor
- 0 baseline measurement to subtract from

CTH discipline requires ρ_net validation, sediment, and confluences. None of these refinements has a confluence chain — they are uncalibrated speculation. Per the Refinement Report's tier table, these are Tier 3 at best (prediction awaiting experiment), but the plan presents them as ready for ISA freeze.

### F4. Wrong target hardware

The energy story applies only to **Hardware-Strategy Path 3** (custom RISC-V ASIC). The currently-active work is `issue_qw128.md` — QW128 fast-paths on the FX-8350 (Crawl) using AVX1/FMA3 double-double. **None of the three refinements help that workload.** Effort spent drafting v2.0 is effort not spent on the path-1.5/walk transition.

---

## 3. Per-Refinement Attacks

### Refinement 1 (`QSETWLI`) — salvageable, but currently architecturally wrong

| # | Defect | Severity |
|---|--------|----------|
| R1.1 | **Encoding ambiguity.** Plan says `QSETWLI rd, rs1` — but RVV's `vsetvli` encodes vtype as an immediate (I-type), not a register. Where do width and length live? `funct7`? Immediate? Two registers as in `vsetvl` (no immediate)? Unspec. (cf. `RISC-V-Policies-and-Best-Practices.md` §6.2) | High |
| R1.2 | **"2 nodes for QW128" is asserted, not derived.** The v1.0 cycle table (§2.1) says `QMUL.QW128` is 12 cycles on a single execution unit. Where does the 2-node figure come from? If from `pkg/mesh`, cite the function. | High |
| R1.3 | **Context-switch / preemption semantics are absent.** RVV solved this by putting state in CSRs (`vtype`/`vl`), so existing OS save-restore paths work transparently. Gemini's plan needs to inherit that solution, not invent a new one. (cf. `RISC-V-Policies-and-Best-Practices.md` §6.4) | Critical |
| R1.4 | **Concurrency model is absent.** Multi-hart QBP system: do harts share mesh allocation? Per-hart? Globally arbitrated? RVV's vector state is per-hart by spec — QBP must adopt the same model or justify divergence. | Critical |
| R1.5 | **Trap behavior on over-request is absent.** What happens if `QSETWLI` requests more nodes than free? Stall? Trap? Implementation-defined? RVV chose to set `vill` and trap subsequent dependent instructions — see R2 for why this matters. | High |
| R1.6 | **Wrong opcode space.** `QSETWLI` is configuration, not data-path. RISC-V idiom is **CSR-based** config (vsetvli reads/writes vtype). Putting QSETWLI in custom-0 violates the architectural separation between configuration state and data-path computation. (cf. `RISC-V-Policies-and-Best-Practices.md` §6.2) | Critical |
| R1.7 | **Backwards compatibility with v1.0 is unspec.** Existing software issues `QMUL.QW128` without a prior `QSETWLI` — is this reserved? Implementation-defined? Trap? Defaulted? Without an answer, every existing emulator binary becomes broken or non-conforming. | Critical |

**Verdict on R1:** Direction is correct (the BMA implementor memo explicitly says "setup overhead will absolutely dominate"), but the spec as written is *less* implementable than RVV's `vsetvli`, which it claims to imitate. The conformant form is **CSR-based, mirroring `vsetvli`**: define a `qmesh` CSR cluster, expose `qbp.setwli` as a pseudo-op that expands to a CSR write.

### Refinement 2 (`QYIELD`) — **reject; category error confirmed by RVV's vill model**

| # | Defect | Severity |
|---|--------|----------|
| R2.1 | **The DVFS analogy is false.** DVFS changes time-to-result, never the result. `QYIELD` changes the **value of the answer**. These are not analogous. | Critical |
| R2.2 | **Determinism violation.** Identical inputs may yield different results depending on whether the watchdog's threshold check fired. This is catastrophic for any reproducibility-required workload — including CTH itself, which is built on confluence theory: independent derivation paths must agree to high precision. **`QYIELD` can break the very framework that gives QBP its epistemic standing.** | Critical |
| R2.3 | **Lossy upgrade.** "If drift approaches the threshold, it spins them back up to QW128." But the QW64 intermediate state has already been computed at lower precision. You cannot recover lost mantissa bits by upgrading. A workflow that briefly drifted under threshold permanently returns lower-quality results — silently. | Critical |
| R2.4 | **Self-invalidating example.** Plan's example: "drift is 10⁻⁴⁰, tolerance 10⁻³⁰, downgrade to QW64." This is the **exact regime of constitutional verification** (Architecture v1.0 §3, BMA-Emulator-Integration §3) where downgrade is **forbidden**. The motivating example is the disallowed case. | Critical |
| R2.5 | **Conflict with existing watchdog semantics.** Architecture v1.0 §3.2 says the CTH Watchdog triggers a **Constitutional Audit interrupt** on norm drift > 10⁻³⁰. `QYIELD` proposes the same watchdog *silently downgrades* precision instead. These are contradictory roles for one hardware unit. | Critical |
| R2.6 | **Inverts RISC-V `vill`-trap semantics.** RVV's standard for "implementation cannot honor this configuration" is: set the `vill` bit and **trap** subsequent dependent instructions. RISC-V never allows hardware to autonomously change the precision or semantics of a computation. Gemini's QYIELD does exactly that. **The plan claims to imitate the RVV configuration model and then directly contradicts its core safety invariant.** (cf. `RISC-V-Policies-and-Best-Practices.md` §6.3, §6.5) | Critical |
| R2.7 | **Side-channel.** Auto-throttling exposes a timing/power side-channel correlated with data values that drove the threshold check. For a system intended to host BMA's governance and constitutional verification, this is a security regression. | High |
| R2.8 | **Audit-trail break.** BMA governance requires lineage. If the watchdog silently downgrades, the Judge Collective has no way to know which intermediate results were full-precision. This is incompatible with the prestige-bridge and judge-collective architecture. | High |
| R2.9 | **Who watches the watchdog?** The threshold check is itself a computation. Its precision must exceed the data precision, or the threshold itself drifts. The plan provides no answer. | Medium |
| R2.10 | **Information loss compounds.** If `QYIELD` fires twice in a chain (W128 → W64 → W32), each fire-back-up is bounded by the lowest-precision step. Long composition chains become precision-monotone-down. | High |

**Verdict on R2: Reject.** The instruction is not salvageable in its proposed form. It cannot be made deterministic without abandoning its core function (silent autonomous downgrade); cannot be made governance-safe without making it a privileged, opt-in, audited operation — at which point it stops resembling DVFS and becomes a "request-precision-change-and-record-it" syscall, not a hardware ISA primitive.

If the user wants something in this space, the right primitive is `qbp.yield.hint` — a **software-issued, governance-gated, audit-logged** precision-downgrade request that the runtime can honor or ignore. That is fundamentally a different design, and it does not require new hardware — only a new CSR field and an event log.

### Refinement 3 (28-lane QW8 SIMD packing) — math works, claim doesn't

| # | Defect | Severity |
|---|--------|----------|
| R3.1 | **"O(1) amortized instruction fetch energy for hypergraph pointer chasing"** is misleading. Pointer chasing is memory-bound; the dominant cost is cache miss, not decode. SIMD packing reduces compute energy per edge but not the gating cost. | Critical (because BMA is sizing budgets from this) |
| R3.2 | **28-port memory is implied but unstated.** 28 simultaneous pointer hops require 28-way banking, a 28-port L1, or accept serialization. Plan claims simultaneity; silicon would require either a banked SRAM or accept conflict-driven slowdown. | High |
| R3.3 | **Internal doc disagreement on QW8 lifetime.** Refinement-Report §1.4: QW8 algebraic life "< 1 op." BMA-Emulator-Integration §2.1: "~8 ops, microseconds." These are wildly different. If the autonomic 10Hz loop runs at QW8 with 140K QMULs/tick, which figure is binding? | High |
| R3.4 | **Register file unspec.** 28 lanes × 32 bits = 896 bits. v1.0 §3 binds quaternions to 4 contiguous F-registers. Where do the 28 lanes live — `qr_k` extended to 1024 bits? Multiple `qr` registers? RVV solved this with `vlmul` (register-grouping multiplier); QBP should explicitly adopt or deviate. (cf. `RISC-V-Policies-and-Best-Practices.md` §6.1) | High |
| R3.5 | **FANO ROM port contention.** v1.0 §5 hardwires the FANO ROM (98 bytes, 1-cycle lookup). 28-lane SIMD invoking FANO 28 times in parallel needs a 28-read-port ROM or accepts replication (28× area, defeats the energy story). | Critical |
| R3.6 | **TLB pressure ignored.** 28 simultaneous hops across a hypergraph hit 28 different pages in the worst case. Production graph is 100K nodes / 500K edges (Walk-Eval §3.1). TLB walks dominate at this scale. | High |

**Verdict on R3:** Salvageable as a *targeted* optimization for hypergraph traversal, but the energy framing is wrong (memory-bound), the silicon implications are unspec'd (memory ports, ROM ports), and the existing internal disagreement on QW8 lifetime must be resolved first. The conformant pattern is to use RVV's `vlmul`-style register grouping rather than inventing a new register-naming scheme.

---

## 4. Cross-Cutting / Systemic Attacks

### S1. ISA stability discipline — backed by RISC-V policy

v1.0 was approved on 2026-05-04 — today. `issue_qw128.md` is mid-implementation. Re-opening v1.0 thirteen days post-approval, on the basis of three under-specified refinements, sets a precedent that "approved" specs are continuously negotiable.

**Reinforced by industry cadence (cf. `SiFive-Documentation-Patterns.md` §5):** SiFive's flagship matrix extension `Xsfmm*` is currently at **v0.6**. They have shipped silicon, but have not yet committed to the stability promise of v1.0. The QBP path of v1.0 → v2.0 in 13 days **has no industry analog**.

**Reinforced by RISC-V policy (cf. `RISC-V-Policies-and-Best-Practices.md` §1):** "Ratified" forbids changes outright. Spec evolution happens through follow-on extensions.

The honest assessment: QBP's v1.0 label may have been premature. SiFive's pattern suggests current QBP maturity is closer to **v0.6 to v0.9** — spec-stable enough for software prototyping, but with v1.0 properly reserved for after Walk-phase ROCm/AVX validation.

### S2. Microarchitecture leaking into ISA — VCIX is the established remedy

Putting "mesh allocation" in user-visible ISA assumes every QBP processor is a Fano mesh. Hardware-Strategy explicitly identifies three paths:
- Path 1 (Threadripper / AVX-512) — not a mesh
- Path 2 (Sharp Butler 9900X / AVX-512) — not a mesh
- Path 3 (custom RISC-V ASIC, OpenMPW 130nm) — is a mesh

Two of three target backends would have to either emulate the mesh ops or treat them as advisory NOPs. The plan does not say which. This is the same mistake Itanium made with explicit-bundling: locking microarchitecture into ISA.

**The remedy is established by SiFive's VCIX pattern (cf. `SiFive-Documentation-Patterns.md` §6):** the scalar-to-coprocessor *interface* and the coprocessor-internal *instructions* live in **separate extensions**. SiFive's `XSfvcp` (v1.1.0, ratified) defines dispatch; `Xsfmm*` (v0.6, draft) defines the matrix coprocessor's instructions. They evolve at independent rates.

**The conformant decomposition for QBP:**

| Concern | Where it lives |
|---------|----------------|
| Scalar core dispatches operands to mesh | `Xqbpvcp` (modeled on `XSfvcp`) |
| Mesh allocation, width selection, watchdog config | `qmesh` CSR cluster (modeled on `vtype`/`vl`) |
| Mesh-internal instructions (28-lane QW8 packing, etc.) | `Xqbpmesh` |
| Quaternion algebra | `Xqbpquat` |
| Octonion algebra | `Xqbpoct` |
| Quantum error correction | `Xqbpqec` |
| Wide memory ops | `Xqbpmem` |

This decomposition delivers everything Gemini wants **without modifying the v1.0 spec.** Each new extension can land at v0.1 and mature independently.

### S3. CTH-tier mislabeling

Per the Refinement Report's epistemic table:
- Existing quaternion core: **T1** (Hurwitz-proven, Lean-verified)
- `QSETWLI`: **T2** (architectural model, not yet measured)
- `QYIELD` w/ auto-downgrade: **T3** (unvalidated prediction; arguably untestable as specified)
- 28-lane QW8 SIMD: **T2** with a T3 sub-claim (the energy figure)

The plan does not tier-label. v1.0 §2.1 already has an "Epistemic Tier" column; the omission lets readers conflate proven and predicted instructions.

### S4. Audit-trail / governance hostility

QBP is an HE programme; BMA is the governance host; CTH is the verification framework. All three demand reproducibility, lineage, and verifiability. `QYIELD` (R2) directly attacks reproducibility. `QSETWLI` (R1) needs context-switch semantics that respect governance ring transitions. Neither is addressed.

### S5. Versioning inflation

v1.0 → v2.0 for one new architectural concept (mesh-aware) plus two refinements is a major-version inflation. The semantic-versioning convention in instruction sets is: **major = backward-incompatible change**. None of these refinements is backward-incompatible *if* R1.7 is resolved properly. **v1.1 is the honest version number** — and even that should arguably be a new draft extension at v0.1 rather than a base-spec bump.

---

## 5. New Findings From RISC-V / SiFive Conventions Research

These findings are derived from the companion reference docs and were not present in the initial review.

### NF1. **Mnemonic prefix non-conformance — blocks upstream toolchain integration**

The RISC-V vendor extension policy (cf. `RISC-V-Policies-and-Best-Practices.md` §4) is explicit:

> Every vendor instruction's mnemonic must carry a vendor prefix that is at least 2 characters long, lowercase, and ends with a period.

Registered vendor prefixes include `sf.` (SiFive), `th.` (T-HEAD), `vt.` (Ventana), `qc.` (Qualcomm), `cv.` (CORE-V), `nds.` (Andes), `smt.` (SpacemiT), `mips.`, `aif.`, `wch.`.

**Every QBP instruction in the current v1.0 spec violates this rule.** `QMUL.w`, `QROT.w`, `OMAC.w`, `FANO`, `QNORM.w`, etc. all lack a vendor prefix.

Conformant mnemonics: `qbp.qmul.w`, `qbp.qrot.w`, `qbp.omac.w`, `qbp.fano`, `qbp.qnorm.w`.

**Severity: Critical.** Until QBP adopts the vendor-prefix convention and registers the `qbp.` prefix with the Toolchain SIG, **no upstream LLVM or GCC will accept it**. This is a Walk-phase blocker for the BMA integration path that depends on stock toolchain support.

This finding applies to v1.0 as well as Gemini's proposed v2.0. The proper response is a v1.1 cleanup, not a v2.0 expansion.

### NF2. **Extension-granularity decomposition is the productive path forward**

(See S2 above for the table.) This is the architectural alternative that makes Gemini's mesh work tractable without touching v1.0. It matches SiFive's split of `XSfvcp` (interface, ratified v1.1.0) from `Xsfmm*` (instructions, draft v0.6) — the most directly relevant industry precedent.

### NF3. **Missing extension X-name**

Per the RISC-V naming convention (cf. `RISC-V-Policies-and-Best-Practices.md` §3), vendor extensions are named `X<vendor><name>`. The QBP spec currently calls itself "QBP RISC-V ISA Spec" without committing to an X-name. This makes the spec uncitable in any conformant ISA string.

Recommended cluster: **`Xqbp*`** as the vendor prefix; specific extensions per the §S2 decomposition.

### NF4. **Trap-behavior section missing across all v1.0 instructions**

RISC-V specs include "Exception Behavior" or "Trap Behavior" subsections per instruction. The QBP v1.0 spec has none. This is acceptable for an emulator-only artifact, but it must be added before any silicon path.

### NF5. **No revision history**

SiFive vendor extension specs include revision history. The QBP v1.0 spec has none. Every vendor extension should carry a "Changes from previous version" appendix from v0.1 forward.

---

## 6. What Is Missing Entirely from Gemini's Plan

| # | Missing piece | Why it matters |
|---|--------------|----------------|
| M1 | Save/restore semantics for mesh allocation state | OS context-switch correctness (RVV solved by CSR-based state — Gemini should inherit) |
| M2 | Privilege-level model (M/S/U mode interactions) | Whether user code can disable QYIELD; whether supervisor must consent to mesh ops |
| M3 | Trap/exception behavior | What happens when allocation fails or mesh is busy |
| M4 | Performance counters | The whole pitch is "energy savings." Without counters, savings are unmeasurable on real silicon. |
| M5 | Backwards compatibility statement vs v1.0 | Whether existing emulator binaries continue to run |
| M6 | Resolution of v1.0 §7 open risks | Hessian code orthogonality, Octonion associativity — both still open |
| M7 | Resolution of Refinement Report §4 Q3, Q6 | Q-extension vs 4×D registers, OGROUP for octonion grouping |
| M8 | Backend applicability matrix | Which instructions are real vs advisory NOPs on Threadripper / ROCm / ASIC |
| M9 | Power model | Even back-of-envelope. Without it, "energy savings" is a marketing phrase. |
| M10 | Test plan | RVV's `vsetvli` semantics took multiple revisions to settle; how will we know `QSETWLI` is correct? |
| M11 | Walkthrough | Plan promises one in §Verification, but does not deliver. |
| M12 | Mnemonic-prefix conformance plan | NF1 above — blocker for upstream toolchain |
| M13 | X-extension name registration | NF3 above — required for any conformant ISA string |
| M14 | Trap-behavior section per instruction | NF4 — required by RISC-V spec convention |
| M15 | Revision-history appendix | NF5 — required by vendor doc convention |

---

## 7. Strategic-Level Concerns

### ST1. Opportunity cost

The team's actual gating work is QW128 fast-path on FX-8350 (`issue_qw128.md`). Drafting v2.0 spec now competes for scarce attention with the Crawl-phase wall-clock benchmark that the MANIFEST flags as PENDING. The MANIFEST's pending list:

1. Port benchmark to FX-8350
2. Write AVX assembly kernel
3. ROCm port
4. Lean 4 verification
5. Integrate with BMA hypergraph
6. Cleanup

None of those is helped by ISA v2.0. Several are blocked or slowed by it.

### ST2. Premature optimization for unbuilt silicon

Path 3 (custom ASIC) is the Run phase per Hardware-Strategy. The current phase is **Crawl** (FX-8350 software). Tuning the ISA for an ASIC two phases away, before the emulator at the current phase is benchmarked, is **textbook premature optimization**.

### ST3. Process violation — Claude-Gemini protocol bypass

This proposal arrived as a complete plan with "User Review Required" flagging that drafting begins on approval. The Claude-Gemini protocol calls for distillation and synthesis with both views. Here Gemini bypassed Claude's red-team role and presented to the beekeeper directly. A fair process would have routed this through red team first.

### ST4. Industry-cadence violation

(cf. `SiFive-Documentation-Patterns.md` §5.) SiFive — the canonical RISC-V vendor — keeps flagship vendor extensions at v0.x until silicon validation. v1.0 → v2.0 in 13 days is unprecedented in the ecosystem.

---

## 8. Recommended Action

| Refinement | Recommendation |
|------------|---------------|
| R1 (`QSETWLI`) | **Hold for rework.** Direction is correct; specification is below the bar of `vsetvli` it imitates. Re-cast as **CSR-based** in a new `Xqbpmesh` v0.1 draft. |
| R2 (`QYIELD`) | **Reject.** Category error confirmed by RVV's `vill`-trap semantics. If a precision-downgrade primitive is wanted, redesign as a software-requested, governance-gated, audit-logged operation in a separate `Xqbpyield` extension. |
| R3 (28-lane QW8 SIMD) | **Hold.** Resolve QW8 lifetime disagreement first. Then revisit using RVV's `vlmul` register-grouping pattern. |
| Mesh-aware execution model framing (§1 of plan) | **Accept, but re-target.** Document the existing `pkg/mesh` design as an architectural addendum, then create draft `Xqbpvcp` v0.1 (interface) and `Xqbpmesh` v0.1 (mesh-internal). Modeled on SiFive `XSfvcp` + `Xsfmm*`. **Do not modify v1.0.** |

Net result: there is no v2.0 to draft. There may eventually be a v1.1 of the existing base spec that adds vendor-prefix conformance (NF1) and the missing trap-behavior / revision-history sections (NF4, NF5). All mesh-related work belongs in new draft extensions.

---

## 9. Falsifiable Asks for the Next Iteration

If Gemini wants to bring this back, these are the artifacts that would make it reviewable:

1. **A power model.** Even a 30-line spreadsheet: gate count × activity factor × switching energy at 130nm OpenMPW. Compare against AVX-512 baseline and against Path-3 ASIC without the mesh ISA.
2. **A concrete encoding** for the mesh-config primitive: bit-by-bit field assignment, immediate vs register, **CSR vs custom-0 decision with rationale referencing RVV's `vsetvli`**.
3. **A context-switch correctness proof sketch** for the mesh state, modeled on RVV's vtype/vl save-restore.
4. **A determinism / reproducibility statement.** If results are nondeterministic when YIELD fires, the instruction is rejected on governance grounds. If they are deterministic, the *because* needs to be spelled out.
5. **An audit-log mechanism** for any hardware that changes precision autonomously.
6. **A backend applicability matrix** for Path 1 / Path 2 / Path 3.
7. **A QW8-lifetime resolution memo** reconciling Refinement-Report §1.4 with BMA-Emulator-Integration §2.1.
8. **A version-bump justification.** If the answer to "is this backward-incompatible?" is no, the version is v1.1 — and even then, mesh-related work belongs in a new draft extension.
9. **A vendor-prefix conformance plan.** Adopt `qbp.` as the registered prefix; commit to a v1.1 cleanup pass renaming all v1.0 mnemonics. (cf. `RISC-V-Policies-and-Best-Practices.md` §4)
10. **An extension-decomposition proposal** following SiFive's `XSfvcp` / `Xsfmm*` split. (cf. `SiFive-Documentation-Patterns.md` §6)
11. **An explicit X-vendor naming commitment** — `Xqbpvcp`, `Xqbpmesh`, `Xqbpquat`, etc. (cf. `RISC-V-Policies-and-Best-Practices.md` §3)

---

## 10. Recommended Next Moves for the Beekeeper

1. **Send this review back to Gemini** with the §9 falsifiable-ask list. Do not authorize drafting until those artifacts arrive.
2. **Re-affirm v1.0 freeze** for the duration of `issue_qw128.md` implementation, to protect Crawl-phase delivery.
3. **Open an ADR (architecture decision record)** in `architecture/` capturing the v1.0 freeze, the rationale for postponing v2.0 until power-model evidence exists, and the commitment to follow RISC-V vendor-extension conventions going forward.
4. **Flag the process deviation** (cf. ST3): future ISA proposals should route through red team before reaching the beekeeper for approval, per the Claude-Gemini protocol.
5. **Schedule a v1.1 cleanup pass** to address the mnemonic-prefix non-conformance (NF1), missing trap behavior sections (NF4), and missing revision history (NF5). This is independent of any mesh work.

---

## 11. References

- [`Ref/RISC-V-Policies-and-Best-Practices.md`](../Ref/RISC-V-Policies-and-Best-Practices.md) — RISC-V maturity workflow, naming convention, vendor-extension policy, Vector-extension configuration template, custom opcode-space rules.
- [`Ref/SiFive-Documentation-Patterns.md`](../Ref/SiFive-Documentation-Patterns.md) — SiFive doc taxonomy, Xsf* extension catalog, versioning maturity pattern, VCIX coprocessor-interface model.
- [`spec/QBP-RISCV-ISA-Spec-v1.0.md`](../spec/QBP-RISCV-ISA-Spec-v1.0.md) — the spec under proposed revision.
- [`doc/QBP-ISA-Refinement-Report.md`](../doc/QBP-ISA-Refinement-Report.md) — context document Gemini was given.
- [`doc/Hardware-Strategy.md`](../doc/Hardware-Strategy.md) — Crawl/Walk/Run/Glide hardware roadmap.
- [`doc/BMA-Emulator-Integration.md`](../doc/BMA-Emulator-Integration.md) — BMA cognitive-mode-to-precision mapping.
- [`doc/memo_to_bma_implementor.md`](../doc/memo_to_bma_implementor.md) — Gemini's own statement that "setup overhead will absolutely dominate" — the motivation for `QSETWLI` direction.
- [`spec/QBP-Compute-Unit-Architecture-v1.0.md`](../spec/QBP-Compute-Unit-Architecture-v1.0.md) — CTH Watchdog → Constitutional Audit interrupt; conflicts with QYIELD silent-downgrade semantics.
- `qbp-compute-unit/pkg/mesh/mesh.go` — existing FanoCell + Scheduler implementation that Gemini's plan reframes as a "discovery."

---

*Status: RECORDED | Companion: peer-review-001.md (April 2026) | Reference: PROGRESS_LOG.md*
