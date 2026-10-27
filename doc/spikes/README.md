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

**R3-Q2 multiply closure (architecture-level, the statement of record):** W1 101 vs 104 · W2 71 vs 49 · W3 256 vs 126 → **no workload gives CD a ≥2× advantage. Closure = NO.** (CD counts from `OMulCDMAC`, measured; non-CD counts independently cross-derived.)

**R3-Q1 structural finding (napkin):** neither the Gram term (3% lock) nor the φ term (0%) locks alone, but the combination `4·Gram − 4φ²` locks at 99%. K-full spontaneity is consistent with Haar-G₂; every ψ-arm breaks it. The φ-vs-ψ gap is confounded by the **comass coupling scale** — the named diagnosis for the (drafted, unfrozen) round-4 probe.

## Pre-registrations (frozen = of record; drafts show the before-data trail)

- r2: `prereg/mdcu-round2-prereg-FROZEN-v3.md`, `prereg/mdcu-round2-q2-amendment-FROZEN-v3a.md`
- r3: `prereg/mdcu-round3-prereg-FROZEN-v1.md` (sha256 `070aeab8…c1848`)

Each round's raw output was hashed and committed **before** its analyzer ran (pre-registration integrity). R3-Q1 raw `mdcu-r3-2026-10-26.jsonl` = sha256 `633670f9…f76bc` (independently verified). Round-4 is drafted but **not frozen or run**; its record lands when R4 closes.

## Provenance

Spike branch tip at this record: see the PR description. Raw artifacts in `raw/`; studybuddy red-team reference scripts in `ref/`. Directing seat: `@qbp-architecture` (inter#197 director). Compute side: `qbp-cu-implementor`.
