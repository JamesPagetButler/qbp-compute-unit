# MDCU round 4 results (Branch B, spike/napkin grade)

> No CD speed claim is permitted at any reading. Nothing here is a physics claim. Verdicts follow the frozen reading table only. Host timing was not a verdict input.

## 1. Provenance

| Item | Value |
|---|---|
| Frozen pre-registration | `doc/spikes/prereg/mdcu-round4-prereg-FROZEN-v1.md`, sha256 `699a8a21692d4f612c038425d23c2ef5ec21d142309940a453d23ae1c5587a5d` (unedited) |
| Harness | content of `d2c29d1` (implementor review GO, seq 120; arch accepted running exactly `d2c29d1`, seq 129) |
| Binary | sha256 `e6296fa218f61328cb21dbac7c992df911d4da72536763c5c63a96e7f9e1ac2c` |
| Timing probe | `doc/spikes/raw/mdcu-r4-probe-2026-10-27.jsonl`, sha256 `5d4a65857943de4640eb334b5ad19586310f1b80a2bc47f23032a311c8420d8a` |
| Grid raw | `doc/spikes/raw/mdcu-r4-2026-10-27.jsonl`, sha256 `757f6054e47ad1f360d37295b6180efde7d3c5fb4727c6adfb6985a68efef423`, posted on the bridge (seq 131) before any analysis |
| Analysis output | `doc/spikes/raw/mdcu-r4-analysis-2026-10-27.txt` (`bin/mdcu-r4-analyze`, 1000 Monte-Carlo reps) |
| Seeds | grid 4000–4199, S-D 4200–4399, probe 9960–9969 |
| Run | 3998 runs, exit 0, wall 357 s, no abort, no NaN |
| Regression | 30 records (seeds 3000–3009, K-full, K-phi, K-psi, 250 steps) bit-identical to the R3 raw, checked before any grid seed |

## 2. Gates and kill conditions

- **G4 (replication):** K-full 3-lock at 250 steps is 99.5% (needs ≥ 90%). Pass.
- **Sampler misses:** 0 in 100 S1-split calls (0.0%, limit 5%). Ok.
- **Comass:** 0 unconverged of 200 grid seeds (limit 5%). Native comass median 1.885 (min 1.160, max 2.302). Two-set relative difference at most 8.7e-16 (tolerance 1e-6).
- **S-D:** 1 of 200 seeds unconverged; its K-psi@1 arm is dropped (n = 199 in the S-D cells).
- **Power:** Tier C ran on 200 seeds, above the 100-seed floor.
- **R3 K-psi replication (diagnostic, not a gate):** native K-psi 3-lock 72.5% against R3's 79%, within ±15 points.

## 3. Decisions (frozen rules)

| Question | Result |
|---|---|
| Q4a: does φ lock with more steps? | **Does not lock by 4000.** K-phi 3-lock is 0.0% at 250, 1000 and 4000 steps (n = 200 each). |
| Q4b: coupling scale or form? | **FORM.** |
| Q4c (definite, split) | **Inconclusive (φ never locks)** in both strata. |
| Reading table (Q4b = FORM, any Q4c) | "The gap is not coupling scale. ψ locks where φ does not at matched comass. Reported; ψ-locks-more is not read as G₂ evidence. Candidate mechanisms are for oppenheimer to name." |

Q4b detail (250 steps, 200 paired seeds; all / definite / split):

| Arm | 3-lock |
|---|---|
| K-phi | 0.0 / 0.0 / 0.0% |
| K-phi@κ | 0.0 / 0.0 / 0.0% |
| K-psi@1 | 100.0 / 100.0 / 100.0% |
| K-psi@κ | 52.0 / 48.0 / 56.0% |

- Δ(K-psi@1, K-phi): rates 100.0% vs 0.0%, Δ median S3 +0.361, 95% CI [+0.353, +0.367]. Clearly higher.
- Δ(K-psi@κ, K-phi@κ): rates 52.0% vs 0.0%, Δ median +0.366, 95% CI [+0.360, +0.374]. Clearly higher.
- Δ(K-phi@κ, K-phi): rates 0.0% vs 0.0%, Δ median −0.017, CI [−0.029, −0.009]. Not clearly higher, so the COUPLING conjunct fails.
- K-phi and K-phi@κ are both at the floor (0.0%); that pair is floor-equivalent and changes no verdict (reported only).

**Mechanism reading (oppenheimer seq 77; no claim):** kill FORM = false; kill COUPLING-ONLY = true (K-phi@κ lags K-psi@1: rates 0.0% vs 100.0%, Δ median −0.376, CI [−0.380, −0.370]). Since the reading is FORM, the COUPLING-ONLY kill is recorded as the napkin design input named in the frozen planning table. It is not a finding about φ.

## 4. Secondaries (no decision weight)

**S-A spontaneity** (Haar-G₂ KS, 1000 reps; reference is Haar-G₂ relative to φ's G₂ for every arm):

| Arm @ cell | locked planes | D | p | reading |
|---|---|---|---|---|
| K-full @ A250 | 199 | 0.0052 | 0.968 | consistent with Haar-G₂ |
| K-psi @ A250 | 145 | 0.0782 | < 0.001 | not Haar-G₂ |
| K-psi@1 @ A250 | 200 | 0.0709 | < 0.001 | not Haar-G₂ |
| K-psi@κ @ A250 | 104 | 0.0898 | < 0.001 | not Haar-G₂ |
| every K-phi, K-phi@κ cell | 0 | n/a | n/a | untestable |

H-A1 holds, H-A2 holds, H-A3 untestable (no φ arm locks).

**S-B dose-response on the rescaled arm** (definite seeds, 250 steps, Spearman of paired s₃(K-phi@L) − s₃(K-psi@L) against d):
- L = κ: n = 100, ρ = −0.687, CI [−0.771, −0.568]. H-B holds.
- L = 1: n = 100, ρ = −0.038, CI [−0.223, 0.151]. H-B killed.

**S-C finding (a) at matched coupling:** rates K-full@κ 0.5%, K-gram@κ 0.5%, K-phi@κ 0.0%. The rule fails on K-full@κ ≥ 90%. The frozen fallback label is "(a) fails at matched coupling: a single term locks". That label is the frozen wording, but it does not describe this data: no single term locks, and K-full@κ itself does not lock. At κ the combined and single-term arms mostly 2-collapse (K-full@κ 87.5%, K-gram@κ 80.0% at 250 steps).

**S-D dose ladder** (K-psi@1 vs K-phi, comass 1; K-phi is 0.0% in every cell):

| s | 250 steps | 1000 steps |
|---|---|---|
| 0.005 | 0.0% | 0.0% |
| 0.010 | 0.0% | 0.0% |
| 0.020 | 42.0% | 58.0% |
| 0.050 | 98.0% | 100.0% |

H-D kill (rate gap ≥ 50 points at s = 0.005) did not fire. "Falls continuously" has no numeric rule in the frozen text, so only the kill is evaluated, as ruled (arch seq 115, item 7).

## 5. Prediction on file (§1.6), mechanical scoring, record only

| | Prediction | Result | Score |
|---|---|---|---|
| P1 | Q4b = MIXED; K-phi@κ 20–80%; K-psi@1 30–80% | FORM; 0.0%; 100.0% | false |
| P2 | Q4a: K-phi reaches ≥ 50% by 4000 steps | does not lock by 4000 | false |
| P3 | S-C holds | fails (K-full@κ 0.5%) | false |
| P4 | S-B \|ρ\| < 0.15 on the rescaled arm (true iff both L = κ and L = 1) | ρ(κ) = −0.687, ρ(1) = −0.038 | false |
| P5 | K-psi@1 stays non-Haar (p < 0.01) | p < 0.001 | true |

One of five is true. The analyzer prints P5 as "see the S-A line"; I scored it from that line (K-psi@1 @ A250: D = 0.0709, p = 0.000 over 1000 reps, so p < 0.001). The prediction was low-confidence and is not edited.

## 6. Observations, no claim

These are descriptions of the table above, not findings. Naming mechanisms is oppenheimer's.

1. **Near-ceiling at L = 1.** K-psi@1 has median S3 = 1.000 and 3-lock 100% in every grid cell, so the paired difference in S-B at L = 1 is dominated by K-phi's own s₃ variation, which shows no detectable dependence on d (ρ = −0.038, CI includes 0). The frozen rule kills H-B there; the ceiling makes that reading hard to interpret. It does not change the verdict.
2. **Lock rate falls as the ψ scale rises.** K-psi@1 100%, native K-psi 72.5% (native comass median 1.885), K-psi@κ 52.0%. The comass 1 arm locks most.
3. **S-D threshold-like rise.** K-psi@1 is at the floor for s ≤ 0.01 and near ceiling by s = 0.05.
4. **K-full@κ does not 3-lock.** In this run the whole-triple κ² arm is 0.5% 3-lock (87.5% 2-collapse) against 99.5% for K-full.

## 7. Interpretation choices (fixed before the run, no change to the frozen text)

Posted at seq 113; all twelve settled by arch seq 115/118/122 and oppenheimer seq 114/117. The list is in the header comment of `emulator/mdcu/r4rules.go`. Disclosed items that bore on this run:
- **G4** was applied as a verdict override after the run, as in R3; the bit-identity regression is pre-grid (driver exit 7). It passed.
- **Unconverged comass** drops only K-psi@1 and K-psi@κ. None dropped on the grid.
- **Q4b** pools both strata and reports strata rates. **Q4c** uses the first cell where K-phi reaches ≥ 50% on the stratum; none exists.
- **S-D:** only the kill is evaluated.
- **Floor-equivalence:** reported only.
- **S-A reference:** Haar-G₂ relative to φ's G₂ for every arm (confirmed, seq 116/117/118).

## 8. Deviations and disclosures

1. **Tier C guard.** Decided by the frozen explicit rule (4 × probe mean at 1000 steps > 2.5 s), as ruled (arch seq 118/119/122/129). Probe mean 0.019271 s, so 4 × mean = 0.0771 s: no fallback. Info, not a decision: the extrapolated total, 3400 × t₁₀₀₀ = 65.5 s, against 3600 × 0.9 = 3240 s. The two readings do not diverge.
2. **Probe load.** The probe started at load1 2.18 (rule ≤ 1.0): the quiet check had passed 12 minutes earlier and I did not re-check before it ran. A busy box inflates times, which biases toward fallback; the decision margin is about 32 times. Not rerun.
3. **Grid end load.** Start load1 0.43 (met). End load1 1.64, above the frozen end rule (≤ 1.5) by 0.14. The run is one runnable thread and adds about 1.0 to load1 by itself; the background share is not known. Timing is not a verdict input.
4. **Crossed ruling.** I built an amendment to a Tier C ruling that arch had already retracted (my seq 123 crossed seq 118). The implementor caught it; I reverted it locally (revert `3622166`). The tree that ran is identical to `d2c29d1`.
5. **Local commits since `d2c29d1`:** `2fabe67` and its revert `3622166` (tree-neutral), `bbfc813` (probe), `9bbdb7b` (grid raw), and this results commit. Not pushed.
6. **Beekeeper condition.** "Don't freeze until the R3 report has been read" was read as the reviewers' read (seq 108). The beekeeper has not confirmed. FROZEN-v1 is unaffected by how that resolves.
7. **Wall.** 357 s against the frozen estimate of about 22 min; per-run cost was far below the assumed 0.1/0.4/1.6 s.

## 9. Routing

Consequences are routed by the R4 planning table (inter#194 issuecomment-6431920143, sha256 `90b870d3…`), which is the authority and is not restated here. This document supplies the readings. No outcome changes a Sprint 4 build, and no CD speed claim is permitted.
