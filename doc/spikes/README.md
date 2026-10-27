# MDCU / Xqbp v2 spike — record (rounds 1–3)

Durable, doc-only record of the **multi-dimensional compute-unit (MDCU / Xqbp v2)** spike on `qbp-compute-unit`. This is **spike/napkin-grade evidence, no physics claim** (beekeeper ruling 2026-10-23); it is **Branch B** on inter#197.

**The spike CODE is not here, by design** (spike rule 1): it lives only on the branch `spike/mdcu-xqbp-v2` (`emulator/mdcu/`, `emulator/cmd/mdcu-*`), which is *not* merged to `main`. The frozen `emulator/v0.1.0-rc1` Gearbox API is untouched (ADR-002). This directory is the results, the pre-registrations, and the raw evidence only.

## Headline

Across three rounds, **no round produced an MDCU advantage under its pre-registered rules.** The recurring finding: where a non-CD path wins, it wins by **structure** (Fano-line factorisation, Hodge-dual / exterior-form contraction), **not** by a Cayley-Dickson multiplier.

| Round | Question | Verdict |
|---|---|---|
| r1 (`mdcu-xqbp-v2-spike.md`) | A0–A6 correctness/perf ladder | A0/A1*/A2a/A3/A4/A5 confirmed; A1 only as a **symmetry reduction**, not CD-MAC evidence (red-team RT1–RT5) |
| r2 (`mdcu-round2-final-report.md` + amendment-1) | Q1 octonion-specific locking? · Q2 CD 2× multiply advantage? | **Both INCONCLUSIVE** — no MDCU advantage; the Fano-factored non-CD path is fastest on host |
| r3 (`mdcu-round3-results.md`, `mdcu-r3q2-result-2026-10-26.md`) | Q1 four-arm (Gram/φ/ψ) · Q2 tape-counted closure + host-ns | **Q1a NO, Q1b INCONCLUSIVE (φ-only never locks, comass-confounded) → Row 5**; **Q2 closure NO** (no 2× anywhere), benchstat YES on W2 alone → YES by host-ns tiebreak, but W2 is an **implementation-shape artifact** (non-CD table-lookup overhead), not architecture — **no CD speed claim** |
| r4 (`mdcu-round4-results.md`) | Q4a φ-lock w/ more steps? · Q4b gap = form or coupling? · S-A…S-D | **Q4a NO** (φ 0/200 at 250/1000/4000); **Q4b = FORM** (K-phi 0% at level 1 & κ; K-psi 100%@1 / 52%@κ; COUPLING conjunct fails) — ψ-locks-more **not** read as G₂ evidence; flat-vs-funnelled mechanism napkin-grade (S-D monotone dose-response); kill-COUPLING-ONLY fired; G4 pass, regression 30/30 bit-identical |

**R3-Q2 multiply closure (architecture-level, the statement of record):** W1 101 vs 104 · W2 71 vs 49 · W3 256 vs 126 → **no workload gives CD a ≥2× advantage. Closure = NO.** (CD counts from `OMulCDMAC`, measured; non-CD counts independently cross-derived.)

**R3-Q1 structural finding (napkin):** neither the Gram term (3% lock) nor the φ term (0%) locks alone, but the combination `4·Gram − 4φ²` locks at 99%. K-full spontaneity is consistent with Haar-G₂; every ψ-arm breaks it. The φ-vs-ψ gap is confounded by the **comass coupling scale** — the named diagnosis for the (drafted, unfrozen) round-4 probe.

## Pre-registrations (frozen = of record; drafts show the before-data trail)

- r2: `prereg/mdcu-round2-prereg-FROZEN-v3.md`, `prereg/mdcu-round2-q2-amendment-FROZEN-v3a.md`
- r3: `prereg/mdcu-round3-prereg-FROZEN-v1.md` (sha256 `070aeab8…c1848`)

Each round's raw output was hashed and committed **before** its analyzer ran (pre-registration integrity). R3-Q1 raw `mdcu-r3-2026-10-26.jsonl` = sha256 `633670f9…f76bc` (independently verified). Round-4 is drafted but **not frozen or run**; its record lands when R4 closes.

## Provenance

Spike branch tip at this record: see the PR description. Raw artifacts in `raw/`; studybuddy red-team reference scripts in `ref/`. Directing seat: `@qbp-architecture` (inter#197 director). Compute side: `qbp-cu-implementor`.

## Record-convention block (inter#224 items 4–6)

**(4) Claims, sorted.**
- *Grounded:* Q1a = no (the Gram term alone does not drive 3-locking, 3%); 3-locking needs the **`4·Gram − 4φ²` combination** (99% vs 3%/0% for the parts); the R3-Q2 **multiply closure = NO** (no workload gives CD a ≥2× advantage: W1 1.03× · W2 0.69× · W3 0.49×, counts measured + independently cross-derived).
- *Unverified:* any G₂- or φ-specific reading. Q1b is **inconclusive** — φ-only never locks (0/200) and the φ-vs-ψ gap is **comass-scale-confounded** (post-hoc: ψ-forms' comass median 1.88 vs φ's 1; collinear with d at ρ 0.97). Napkin, no claim.
- *Mislabeled-risk:* the R3-Q2 **host-ns tiebreak "YES"**. It rests on **W2 alone**, where CD does *more* multiplies (71 vs 49) yet is 2.7× faster — arithmetic cannot produce that, so it reads as an **implementation-shape artefact** (non-CD table-lookup overhead), not an architectural CD win. A table-free unrolled-W2 post-hoc diagnostic **landed and explained the gap** (`raw/mdcu-r3q2-w2diag-result-2026-10-26.md`, post-hoc, no verdict change): both non-CD kernels do the **identical 49 multiplies** (independently cross-checked), so the 10× (23081→2304 ns) is **table-lookup overhead, not arithmetic**; against the fair unrolled kernel **CD is 3.7× slower** (no-AVX2 disclosure applies). The R3-Q2 W2 tiebreak "YES" is therefore recorded as an **implementation-shape artefact**. **No CD speed claim.**

**(5) Fired planning rows → dispositions.**
- R3 planning-consequence table: inter#194 issuecomment-6391302019.
- Q1 "inconclusive/mixed" → **`mdcu-r4`** (inter#224 shelf row; R4 table inter#194 issuecomment-6431920143).
- Q2 "closure vs benchstat disagree" → **no speed claim**, plus the W2 post-hoc diagnostic (mdcu-spike seq 89).
- Joint **QASSOC design-closure** → inter#197 issuecomment-6432217737: QASSOC stays a **correctness / tier-gated** op (prove-before-bake tie + S5-oracle role), **no performance justification**; φ-hardwired-vs-programmable-3-form deferred to `mdcu-r4`.
- **Recorded negative:** "CD-multiplier speed advantage" — not established on this hardware; **reopens only on non-serialising (AVX2+/wide-SIMD) hardware** where the CD-MAC's lane parallelism isn't serialised.

**(6) Lock.** There is **no CI lock on `main` for this spike**, by design. The spike's regression gate — the K-full signed-energy runner being **bit-identical to `RunRingGen`** — lives with the spike code on the **unmerged** branch `spike/mdcu-xqbp-v2` (spike rule 1: no spike code on `main`). This is a **known gap, not an omission**: the record is doc-only, so nothing on `main` is under a drift-gate for it; the gate travels with the code if/when the direction is promoted out of spike.

---

## Record-convention block — r4 (inter#224 items 4–6)

**(4) Claims, sorted.**
- *Grounded:* G4 pass (K-full 3-lock 99.5% @250); **Q4a NO** (φ 0/200 at 250/1000/4000); **Q4b = FORM** (K-phi 0% at level 1 & κ; K-psi 100%@level-1, 52%@κ; the `clearly-higher(K-phi@κ, K-phi)` COUPLING conjunct **fails**); kill-COUPLING-ONLY fired, kill-FORM did not; regression **30/30 bit-identical** to the R3 raw (`633670f9…`, pre-grid inside the driver); independently rate-recomputed by the director (arch) — all match.
- *Unverified:* the flat-maximiser (φ) vs funnelled (ψ) **mechanism** — supported at **napkin grade** (S-D gives a monotone dose-response: ψ@comass-1 locks 0% at s≤0.01 rising to ~100% at s=0.05, → φ's 0% as ψ→φ). Candidate mechanisms (qbp-oppenheimer, **hypotheses with kills, not adopted**): **M1** degenerate G₂/SO(4) maximiser family — *kill:* the Hessian of −φ² at an associative frame has a zero-mode count ≠ 8 (dim G₂/SO(4)); **M2** coupling window (ψ lock 100%→72.5%→52% as level-1→native→κ; K-full@κ 2-collapses) — *kill:* a K-full coupling ladder monotone in coupling refutes the window; **M3** metric mismatch (S-B@κ ρ = −0.687, CI excludes 0) — not separable from M1 in this design. **H-A3 (φ's locked planes Haar-uniform) is UNTESTABLE here** — φ never locks, so the Haar half of the mechanism stays unconfirmed. **S-C (verbatim rewording, oppenheimer seq 135 / arch seq 137):** "(a) does not hold at κ: no term 3-locks and K-full at κ mostly 2-collapses — R3's finding (a) is coupling-specific."
- *Mislabeled-risk:* none new. The Tier-C **crossed-ruling episode** (commit `2fabe67` implemented an extrapolation Tier-C decision per a *retracted* ruling) was **caught in implementor review + reverted** (`3622166`, tree-neutral); the run used the frozen **2.5 s** rule (code byte-identical to the GO'd `d2c29d1`). Disclosed in results §8.

**(5) Fired planning rows → dispositions** (R4 planning table inter#194 issuecomment-6431920143).
- **FORM row:** no φ claim; the surviving φ/ψ gap is the *form's landscape*, not coupling scale; **ψ-locks-more is not G₂ evidence**. **QASSOC design-closure deferred point 3 RESOLVED:** a **generic, programmable 3-form unit (φ the standard instance); no φ-hardwired unit** (arch, recorded on inter#197). The mechanism becomes a **design input to QBP#776 rung-2 (E6)** + an oppenheimer theory item — **not an anchor**.
- **Q4a row:** an energy term using φ alone is not a locking driver on this ring at any coupling/step count tested.
- **mdcu line:** S-D settles the metric question at napkin grade → **no further φ-vs-ψ round**; next on the line is `mdcu-kz`.
- **Recorded negatives** carry over from r3 (CD-multiplier speed advantage; reopens only on non-serialising hardware).

**(6) Lock & load.** Same as r3: **no `main` CI lock** (spike rule 1); the K-full↔`RunRingGen` + the 3000–3009 bit-identity regression gate live with the code on the unmerged spike branch — known gap. **End-load note:** R4's FROZEN-v1 carries **no load rule** (R3-Q2's ≤1.5 was host-ns-only); R4 verdicts are lock-rates from *deterministic* dynamics (30/30 regression bit-identical), so they're **load-insensitive**. The end-load 1.64 is disclosed, note-only (arch seq 133).
