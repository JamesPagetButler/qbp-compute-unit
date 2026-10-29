# QBP-CU Spec Addendum 2.1 — MDCU / Xqbp-v2 spike → theory write-back (mdcu-r3, mdcu-r4)

**Status:** spike→theory write-back, per inter#246 AC4 / inter#247 (forging doc §3 rule 7). Records the **MDCU / Xqbp-v2 spike** (Branch B, inter#197, **napkin / spike grade — no physics claim, no CD speed claim at any reading**) into the unit's spec, not just `doc/spikes/`. Owner: `qbp-cu-implementor`. Notary verifies every E move below.

The spike's own records are on `main` in `doc/spikes/` (r3: `mdcu-round3-results.md` + README; r4: `mdcu-round4-results.md` + README, PRs #86/#87/#88). This addendum is the **theory-record write-back** those did not carry. E and F are independent axes; a design choice has **no** E grade.

## Write-back table (claim → where stated → E before→after → F → write-back)

| # | Claim | Where stated | E before → after | F | Write-back |
|---|---|---|---|---|---|
| 1a | An energy term in **φ alone** is a locking driver on this ring (and the φ/ψ gap is set by coupling scale) | the φ-specific-structure reading implied at `QBP-RISCV-ISA-Spec-v1.1.md` §116 / `QBP-RISCV-Xqbpoct-Spec-v0.1.md` §141 | `unconfronted` → **`falsified`** (Q4a: φ locks 0/200 at 250/1000/4000 steps; `kill-COUPLING-ONLY` fired) | napkin (mdcu-r4) | this addendum · `doc/spikes/mdcu-round4-results.md` |
| 1b | *(design, not empirical)* The 3-form / associator ROM orientation is **hardwired** / φ-specific | `QBP-RISCV-ISA-Spec-v1.1.md` §116 | **E `n/a`** (design write-back): adopt a **generic, programmable 3-form unit** (φ = the standard instance), not φ-hardwired | napkin (design) | this addendum · inter#197 issuecomment-6432217737 (QASSOC design-closure) · inter#194 issuecomment-6431920143 (FORM planning row) |
| 2a | The octonion **CD-MAC needs ≥ 2× fewer multiplies** on W1–W3 (the architecture-level multiply-count closure) | **not stated in the spec** — recorded here as a negative so it is never asserted later | `unconfronted` → **`falsified`** (closure **no**: W1 1.030 · W2 0.690 · W3 0.492; pre-registered and hardware-independent — no "on this hardware" qualifier) | napkin (mdcu-r3, frozen R3 §3) | this addendum · `doc/spikes/mdcu-round3-results.md` |
| 2b | The octonion **CD-MAC gives a host-speed advantage** | **not stated in the spec** — recorded as a negative | **stays `unconfronted`**: the frozen confrontation returned **"R3-Q2: YES (host-ns tiebreak)"** (W2 only), *explained* post-hoc as an implementation-shape artefact (**no verdict change**); a post-hoc explanation cannot set `falsified`. Reopens on non-serialising / AVX2+ hardware or the #89/#90 cycle model | napkin (mdcu-r3) | this addendum · `doc/spikes/raw/mdcu-r3q2-result-2026-10-26.md` · `doc/spikes/raw/mdcu-r3q2-w2diag-result-2026-10-26.md` |

## mdcu-r4 — φ alone is not a locking driver (Q4b = FORM); the unit is generic/programmable

- **Q4a NO (the E-moving claim):** an energy term using **φ alone** is not a locking driver on this ring at any coupling/step count tested (φ locked 0/200 at 250/1000/4000). → row 1a `falsified`.
- **Q4b = FORM, not COUPLING:** K-phi 0% at level-1 and κ; K-psi 100% @level-1, 52% @κ; the `clearly-higher(K-phi@κ, K-phi)` COUPLING conjunct **fails**. **`kill-COUPLING-ONLY` fired; `kill-FORM` did not.** ψ-locks-more is **not** read as G₂/φ evidence; the φ/ψ gap is the *form's landscape*, not coupling scale. The flat-vs-funnelled mechanism is napkin-grade (S-D monotone dose-response); H-A3 is **untestable here** (φ never locks).
- **Design write-back (row 1b, E n/a):** driven by the frozen FORM planning row (inter#194 issuecomment-6431920143: "no φ claim; generic programmable 3-form unit") and the QASSOC design-closure (inter#197 issuecomment-6432217737). QASSOC / the 3-form unit is specified as a **generic, programmable 3-form unit** — the algebra is a swappable ROM/table, φ is the standard instance, **not φ-hardwired** (also the design constraint for QBP#776 rung-2 / E6: swap the algebra, keep the substrate ops). A **design input + theory item, not an anchor.**
- G4 pass (K-full 3-lock 99.5% @250); regression 30/30 bit-identical to the R3 raw.

## mdcu-r3 — inconclusive, with its reopen condition (the comass confound)

- **Q1a NO** (the Gram term alone does not drive 3-locking, 3%); 3-locking needs the **`4·Gram − 4φ²` combination** (99%).
- **Q1b INCONCLUSIVE:** the φ term alone never locks (0/200), and the φ-vs-ψ gap is **comass-scale-confounded** (post-hoc: ψ-forms' comass median 1.88 vs φ's 1; collinear with d at ρ 0.97). **No G₂/φ reading.** This is the **reopen condition** — the comass confound is exactly what mdcu-r4 was built to resolve (and did → FORM, row 1a/1b).
- **Q2, recorded faithfully to the frozen verdict:**
  - **Multiply-count closure = no** (row 2a): no workload gives CD a ≥2× advantage — W1 1.030 · W2 0.690 · W3 0.492 (counts measured + independently cross-derived). This is the **architecture-level statement, hardware-independent**, and it is `falsified` and unchanged.
  - **R3-Q2 recorded verdict = "YES (host-ns tiebreak)"** (frozen §3.4), resting on **W2 alone** (W1 is 1.7× *slower* for CD; W3 a 1.04× tie). It is **explained post-hoc, with no verdict change**: an unrolled table-free W2 kernel (same 49 multiplies, 1e-12 agreement) runs ~10× faster than the frozen non-CD W2 path, and against it **CD is 3.7× slower** — so the "YES" was an **implementation-shape artefact** of the frozen non-CD table-lookup path, not an architectural advantage. Per the write-back rule, a post-hoc explanation does not set `falsified`, so the host-speed claim **stays `unconfronted`** (row 2b), attempt recorded verbatim.
- **Recorded negative (carried):** "CD-multiplier speed advantage" is not established on this hardware; it **reopens only on non-serialising (AVX2+/wide-SIMD) hardware** or under the cycle-accurate cost model (#89/#90).

## Explicitly NOT confronted (no laundering)

The **quaternion-rotation structural claims are untouched by this spike** and their E/F are unchanged:
- **11.3× fewer operations** (quaternion rotation 28 FP ops vs SU(2) matrix 56+; `QBP-Compute-Unit-Spec-Rev2.txt` §3.1) — a **quaternion op-count** result, scale-independent, `[CONFIRMED]`/structural. The MDCU spike was about the **octonion/sedenion CD-MAC** and the 3-form unit, a different claim. **E unchanged (`consistent`/structural, F confirmed); out of r3/r4 scope.**
- **2× norm preservation** (`QBP-Compute-Unit-Spec-Rev2.txt` §3.2) — likewise untouched.

Recording this boundary is itself part of the write-back: the spike's negative is about the **octonion CD-MAC** (rows 2a/2b), not the quaternion op-count advantage.

## E moves for notary verification (per inter#246 AC1)

- **Row 1a:** `unconfronted → falsified` (F napkin) — "φ alone is a locking driver / the gap is coupling scale" (Q4a NO; kill-COUPLING-ONLY). Evidence: `doc/spikes/mdcu-round4-results.md`, inter#194 issuecomment-6431920143.
- **Row 1b:** `E n/a` (design write-back, F napkin) — hardwired → generic programmable 3-form unit. Evidence: inter#197 issuecomment-6432217737.
- **Row 2a:** `unconfronted → falsified` (F napkin) — CD-MAC ≥2×-fewer-multiplies closure = no (W1 1.030/W2 0.690/W3 0.492), hardware-independent. Evidence: `doc/spikes/mdcu-round3-results.md`, `doc/spikes/raw/mdcu-r3q2-result-2026-10-26.md`.
- **Row 2b:** `stays unconfronted` (F napkin) — host-speed frozen verdict YES-by-tiebreak, explained post-hoc, no verdict change; reopen condition stated. Evidence: `doc/spikes/raw/mdcu-r3q2-result-2026-10-26.md` §Verdict, `doc/spikes/raw/mdcu-r3q2-w2diag-result-2026-10-26.md`.
- **Unchanged (for completeness):** the 11.3× / 2× quaternion structural claims remain `consistent`/`[CONFIRMED]`, **not** moved by this spike.

Refs: inter#197, inter#194, inter#224 (record convention), inter#246/#247 (this write-back rule), qbp-compute-unit#86/#87/#88 (spike records), #89/#90 (the CD-speed reopen path).
