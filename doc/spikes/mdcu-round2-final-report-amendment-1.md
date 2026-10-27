# Spike MDCU — round 2 final report, amendment 1 (2026-10-24)

Amends: [`mdcu-round2-final-report.md`](mdcu-round2-final-report.md), sha256 `bf85e16cfb105feb9a3829b7a5c886a6aee2c2e1083dca14d0b3767f9e2e3528` (commit `cbdecc5`; accepted as the record by qbp-architecture, mdcu-spike seq 35). The frozen file is **not edited**; this file adds to it.
Direction: qbp-architecture, mdcu-spike seq 32.

## A1. Addition to §4 (Capability gaps)

- **The leverage is structure, not a CD multiplier.** On the G₂ workloads examined so far, the advantages come from structure: Fano-line factorisation (round-2 Q2 B0b: lowest host ns, 94 ns/determinant) and Hodge-dual 4-form contraction (associator vector: about 126 data×data multiplies, against 256 for four `OMulCDMAC` products). The 256 is emulator-authoritative (qbp-cu-implementor, seq 34); the 126 is a hand count, to be tape-counted in round 3. A CD-MAC multiplier gives no multiply-count advantage of 2× or more on these workloads. The capability gap is therefore a datapath for structured contractions (Fano-line minors, coassociative-quadruple contraction), not a faster CD product.

## A2. Record

- Verdicts of record are unchanged: Q1 INCONCLUSIVE, Q2 INCONCLUSIVE (B1-variant-sensitive).
- No new run, data or analysis enters through this amendment.
