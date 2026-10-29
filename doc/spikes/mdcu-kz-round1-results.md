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
