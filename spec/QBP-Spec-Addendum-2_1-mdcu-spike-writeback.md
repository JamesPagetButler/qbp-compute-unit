# QBP-CU Spec Addendum 2.1 — MDCU / Xqbp-v2 spike → theory write-back (mdcu-r3, mdcu-r4)

**Status:** spike→theory write-back, per inter#246 AC4 / inter#247 (forging doc §3 rule 7). Records the **MDCU / Xqbp-v2 spike** (Branch B, inter#197, **napkin / spike grade — no physics claim, no CD speed claim at any reading**) into the unit's spec, not just `doc/spikes/`. Owner: `qbp-cu-implementor`. Notary verifies every E move below.

The spike's own records are on `main` in `doc/spikes/` (r3: `mdcu-round3-results.md` + README; r4: `mdcu-round4-results.md` + README, PRs #86/#87/#88). This addendum is the **theory-record write-back** those did not carry.

## Write-back table (claim → where stated → E before→after → F → write-back)

| # | Claim in the spec | Where stated | E before → after | F | Write-back |
|---|---|---|---|---|---|
| 1 | The octonion 3-form / associator unit's orientation is **hardwired** (Conway–Smith ROM; φ-specific structure is load-bearing) | `spec/QBP-RISCV-ISA-Spec-v1.1.md` §116 (associator ROM) · `spec/QBP-RISCV-Xqbpoct-Spec-v0.1.md` §141 (`qbp.omac`/associator); QASSOC | `unconfronted` → **the φ-specific reading is `falsified (napkin)`**; the adopted design is a **generic, programmable 3-form unit (φ = the standard instance), not φ-hardwired** | napkin / spike (**mdcu-r4**) | this addendum + inter#197 issuecomment-6432217737 (QASSOC design-closure) |
| 2 | The octonion/sedenion **Cayley–Dickson MAC gives a ≥2× multiply/speed advantage** (the "lower-impedance" efficiency thesis, extended to CD multiplication) | `spec/QBP-Compute-Unit-Spec-Rev2.txt` §4.2 (Advantages) / MANIFEST central thesis; implied by the efficiency framing | `unconfronted` → **`falsified` on this (serialising, no-AVX2) hardware**: no workload gives CD a ≥2× advantage; **reopens only on non-serialising / AVX2+ / wide-SIMD hardware or a validated cycle model** (→ qbp-compute-unit#89/#90) | napkin / spike (**mdcu-r3**, inconclusive) | this addendum + `doc/spikes/` + #89/#90 |

## mdcu-r4 — the 3-form unit is generic/programmable, not φ (Q4b = FORM)

- **Q4a NO:** an energy term using φ alone is not a locking driver on this ring at any coupling/step count tested (φ locked 0/200 at 250/1000/4000 steps).
- **Q4b = FORM, not COUPLING:** K-phi 0% at level-1 and at κ; K-psi 100% @level-1, 52% @κ; the `clearly-higher(K-phi@κ, K-phi)` COUPLING conjunct **fails**. **`kill-COUPLING-ONLY` fired; `kill-FORM` did not.** The surviving φ/ψ gap is the *form's landscape*, not coupling scale.
- **ψ-locks-more is NOT read as G₂ / φ evidence.** The mechanism (flat φ-maximiser vs funnelled ψ) is napkin-grade (S-D monotone dose-response); H-A3 (φ's locked planes Haar-uniform) is **untestable here** (φ never locks).
- **Design consequence (the spec write-back):** QASSOC / the 3-form unit is specified as a **generic, programmable 3-form unit** — the algebra is a swappable ROM/table, φ is the standard instance, **not φ-hardwired**. This is the "programmable-not-φ-hardwired" architecture (also the design constraint for QBP#776 rung-2 / E6: swap the algebra, keep the substrate ops). It is a **design input + theory item, not an anchor**.
- G4 pass (K-full 3-lock 99.5% @250); regression 30/30 bit-identical to the R3 raw.

## mdcu-r3 — inconclusive, with its reopen condition (the comass confound)

- **Q1a NO** (the Gram term alone does not drive 3-locking, 3%); 3-locking needs the **`4·Gram − 4φ²` combination** (99%).
- **Q1b INCONCLUSIVE:** the φ term alone never locks (0/200), and the φ-vs-ψ gap is **comass-scale-confounded** (post-hoc: ψ-forms' comass median 1.88 vs φ's 1; collinear with d at ρ 0.97). **No G₂/φ reading.** This is the **reopen condition**: the comass confound is what mdcu-r4 was built to resolve (and did → FORM, row 1).
- **Q2 multiply closure = NO:** no workload gives CD a ≥2× advantage (W1 1.03× · W2 0.69× · W3 0.49×, counts measured + independently cross-derived). The one host-ns "win" (W2) is an **implementation-shape artefact** (non-CD table-lookup overhead), not architecture — a table-free unrolled kernel showed both non-CD paths do the identical 49 multiplies, and against the fair kernel **CD is 3.7× slower** (no-AVX2 disclosure applies). **No CD speed claim.**
- **Recorded negative (carried):** "CD-multiplier speed advantage" — not established on this hardware; **reopens only on non-serialising (AVX2+/wide-SIMD) hardware** where the CD-MAC's lane parallelism isn't serialised, **or** under the cycle-accurate cost model (#89/#90).

## Explicitly NOT confronted (no laundering)

The **quaternion-rotation structural claims are untouched by this spike** and their E/F are unchanged:
- **11.3× fewer operations** (quaternion rotation 28 FP ops vs SU(2) matrix 56+; `Spec-Rev2.txt` §3.1) — a **quaternion op-count** result, scale-independent, `[CONFIRMED]`/structural. The MDCU spike was about the **octonion/sedenion CD-MAC** and the 3-form unit, a different claim. **E unchanged (`consistent`/structural); not in scope of r3/r4.**
- **2× norm preservation** (`Spec-Rev2.txt` §3.2) — likewise untouched.

Recording this boundary is itself part of the write-back: the spike's negative is about the **CD-MAC speed thesis**, not the quaternion op-count advantage.

## E moves for notary verification

- Row 1: `unconfronted → falsified (napkin)` for the φ-hardwired / φ-is-special reading (mdcu-r4, Q4b=FORM). Evidence: `doc/spikes/mdcu-round4-results.md`, inter#194 issuecomment-6431920143.
- Row 2: `unconfronted → falsified` for the CD-MAC ≥2× speed claim on serialising hardware, **with the stated reopen condition** (mdcu-r3 Q2 closure). Evidence: `doc/spikes/mdcu-round3-results.md`, `doc/spikes/mdcu-r3q2-w2diag-result-2026-10-26.md`.
- Unchanged (for completeness): the 11.3× / 2× quaternion structural claims remain `consistent`/`[CONFIRMED]`, **not** moved by this spike.

Refs: inter#197, inter#194, inter#224 (record convention), inter#246/#247 (this write-back rule), qbp-compute-unit#86/#87/#88 (spike records), #89/#90 (the CD-speed reopen path).
