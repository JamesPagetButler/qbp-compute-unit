# mdcu-kz round 1: results (premise failed at Stage 0b, main arm)

Runner: antigravity. Code `c739a62` (origin `spike/mdcu-xqbp-v2`), binary sha256 `a564c9901349d5e32cbb84b773b55bc9a02f198eb7738bcbe65a402fc9de3cd2`. Pre-registration: FROZEN-v1 `7ac4ed0` (sha256 `176bdce2bd4544f0f0bac8523ecc04f5a6eb582d603bb2714713fa72fa676a5f`) plus amend-1 (sha256 `e226da2352ae431ad40d49c582417f1fee0ea715a9ff9163a5d6e7440d26db59`, signed by oppenheimer seq 234, arch seq 235, qbp-cu seq 236). Run go: arch seq 244. Branch B, spike/napkin grade: no physics claim and no CD speed claim.

## Verdict

**premise failed (0 ladder runs).** The main arm failed Stage 0b (hysteresis check). By the frozen orchestration a main-arm Stage 0 failure stops the run. The null arm's Stage 0 and the whole sweep were never run. There is no Q5a, Q5b or Q5c reading, no defect count and no exponent.

## Evidence

| Item | Value | Source |
|---|---|---|
| Probe decision | L = 16, R = 50, η = 0.1, projected wall 4,565 s, low power | `mdcu-kz-probe.jsonl`, sha256 `f5f88ff5605634aa398d5ccc7ba047291ed11e2054defe5ad144874918c0d3a4` |
| Arch recomputation of (L, R) | holds (seq 247); qbp-cu §6 check (seq 248) | channel |
| Saturation gate (20 Haar-G₂ fields, L = 16) | raw −1 density **0.4642**, inside 0.50 ± 0.05 | grid raw |
| Stage 0a, main arm (3 seeds) | ordered = true; T_c = 1.0666 (midpoint of the steepest-drop pair) | grid raw |
| Stage 0a mean S near T_c | T = 0.8756: 0.7102; T = 0.9973: 0.6405; **T = 1.1359: 0.0534**; T = 1.2938: 0.0258 (SE ≤ 0.004 at each) | grid raw |
| Stage 0b, main arm (10 seeds, τ_Q = 1600, the slowest rung; ramp [1.5 T_c, 0.5 T_c]) | **pass = false**; max \|cool − heat\| = **0.5176** of S_max; largest recorded-step jump 0.0374 (cool) and 0.0570 (heat) | grid raw |
| Stage 0b reason string | "hysteresis gap 0.0856 > 5% of S_max at grid point 27" | grid raw |

Grid raw file: `doc/spikes/raw/mdcu-kz-grid-raw.jsonl`, sha256 `ba20ddb29cace852ecc19b991f12998912de2565890d788e47bfe1fde2f1390e`, posted at seq 249 before any analysis. Wall 1,384 s, exit 0, no abort, pause file absent. `-analyze` refuses a "premise failed" file (no grid to analyse), so there is no analyzer output.

## What this does and does not show

- **Frozen prediction P5** ("Stage 0 finds a continuous-looking transition with no hysteresis beyond tolerance", about 60%) did **not** hold on the main arm.
- Stage 0a ran on 3 seeds per arm (5600–5602 main). FROZEN-v1's seed list names 5 (5600–5604) while its wall formula counts 3; the harness used 3, within the named range, and said so in check-in 2.

- The Stage 0b rule fires on the hysteresis gap test, not on the jump test (both jumps are below the 10% limit). The frozen rule reads this as "first-order or glassy, premise fails".
- **Reading of the reason string.** The reason shows the gap at the *last* failing grid point (point 27, 0.0856). The largest gap is 0.5176 (`max_gap`). The code overwrites the reason on each failing point, so the string is not the maximum. This is a reporting quirk of the harness, not a change of any threshold or of the verdict.
- Stage 0a's seed-mean S falls from 0.64 to 0.05 between two adjacent grid points (T = 0.997 and 1.136). That step is an observation about this 3-seed Stage 0a scan. It is consistent with the 0b failure, but it is not a gate and not a measurement of the order of the transition.
- Nothing here bears on the S³ null (option B). The null arm was never run, so amend-1 §5's expectation of an expected-zero floor was neither confirmed nor contradicted.
- The detector, parity, resolution and saturation checks other than the saturation gate were not exercised on ladder data. The saturation gate passed.

## Disclosures

- The probe's recorded `load_start` was 2.01 and `load_end` 2.63, because the wrapper's pre-step `go test` ran after the quiet-load wait and raised the load (qbp-cu seq 248: conservative direction; arch seq 247: the decision stands). The probe was not re-run.
- The grid meta records `load_start` 2.39 and `load_end` 1.57, for the same reason.
- Host ns were measured on an FX-8350 without AVX2. The CD-MAC's wide-lane parallelism is serialised, so host ns is biased against CD. No CD speed claim is made.
- The amend-1 §5 pointer to `TestKZExplore` resolves to tag `mdcu-kz-explore-254a9af` (arch seq 239).

## Not decided here

Whether to open a follow-up round (a premise-compatible target, a different dynamics or a changed T range) is the beekeeper's and the reviewers' call. No rule is changed after this output, and nothing was re-run.

## Write-back (forging doc §3 rule 7)

Per the spike→theory write-back rule (inter#246 / inter#247): a spike result — including a clean negative — writes back into the theory record, not just the plan. Table columns: claim · where stated · E before → after · F · write-back.

| Claim | Where stated | E before → after | F | Write-back |
|---|---|---|---|---|
| "The G₂/SO(4) plane-field sigma model (this chordal energy, Model-A dynamics, this ramp) has a continuous ordering transition" | FROZEN-v1 §2a; QBP#776 (E6 framing) | `unconfronted` → **`falsified`** (this model + ramp only): Stage 0b hysteresis max gap **0.5176 of S_max** ≫ the 5% limit; Stage 0a seed-mean S drops 0.64 → 0.05 across one grid step | napkin | this report PR; QBP#776 note (qbp-oppenheimer owns) |

**Scope (explicit):** the write-back is about **this sigma model with this ramp only**. It says **nothing about the topology** of the G₂/SO(4) order-parameter manifold (π₂ = ℤ₂ point defects) — the ladder never ran, so there is no defect-density, exponent, or topological reading. A continuous-transition target would need a new model, a new pre-registration, and the beekeeper's re-approval (off the inter#224 approved list). @notary-implementor verifies the E move.

**Reopen condition / next-probe candidate** (qbp-oppenheimer's theory item, seq 252; *not* a reading of this round): whether the transition is genuinely first-order (**H-first:** hysteresis gap saturates to a nonzero value as τ_Q → ∞) or continuous-but-too-fast for Model A at this L (**H-slow:** gap falls as a power of 1/τ_Q and drops below 5% at some reachable τ_Q), separated by a Stage-0b-only probe at τ_Q ∈ {1600, 3200, 6400, 12800} that **stores the full 41-point cool/heat seed-mean arrays** (the round-1 raw kept only `max_gap` + the two jumps — arch's measurement-gap note, seq 253). A round-2 candidate for inter#224, pending its own pre-reg and the beekeeper's re-approval. **Round 2's pre-data expectation is now H-first** (per the Landau cubic-invariant argument in the P5-miss row below), so the candidate is largely confirmatory (arch seq 258).

## Recorded prediction miss — P5 (a-priori, not data; kept separate from the E-move above)

Per arch seq 258 and oppenheimer seq 257 §3, recorded as a first-class row, not a footnote:

| Prediction | Filed | Outcome | Why it was mis-set |
|---|---|---|---|
| **P5:** "Stage 0 finds a continuous-looking transition with no hysteresis beyond tolerance" (~60%) | FROZEN-v1 §10 | **failed** on the main arm (Stage 0b, gap 0.5176 of S_max > 5%) | The order parameter here — the projector **Π onto the 3-plane**, a traceless symmetric tensor — carries a **cubic Landau invariant `tr(Π_tl³)`** (the nematic Q-tensor / RP^{N−1} / CP^{N−1} class in 3D), whose ordering transitions are **first-order** by that mechanism. So **first-order was the a-priori expectation**; a continuous transition was the less likely premise and P5's 60% was too high. The argument needed no data (oppenheimer's recorded miss, seq 257 §3). |

**Separation (notary seq 259, anti-laundering):** this is the **a-priori-expectation correction**, NOT the justification for the E-move above — the falsification is drawn from the Stage-0b data, and the cubic-invariant / H-first argument stays a distinct theory item. The 0a drop (0.64 → 0.05 over one grid step) is consistent with H-first but, at L = 16, not discriminating (finite-size rounding of a continuous transition would be of the same width as the grid spacing).

**Process (oppenheimer's wisdom, landed inter#249):** a physics freeze-check on any future ordering-transition spike asks "**does the order parameter admit a cubic invariant?**" before the premise probability is filed.
