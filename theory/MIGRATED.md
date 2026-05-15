# BMA Theory Addenda — Migrated

**Date:** 2026-05-14 (initial migration) → 2026-05-15 (recovery + canonical-location update to inter/theory/)
**Authorized by:** Beekeeper (James Paget Butler)

The BMA Theory Addendum series (A11.0 — A17.0) has been moved to its canonical federation-canonical home at **`~/Documents/inter/theory/`** per the hybrid placement decision (federation-canonical theory + cross-tenant-invariant spec live in `inter/`; BMA-specific operational addenda stay in `BMA/spec/`).

- **New canonical path:** `~/Documents/inter/theory/`
- **GitHub:** federation-canonical (no single-repo home; referenced by every federation tenant)

(Migration history: initial migration on 2026-05-14 landed the files in `~/Documents/BMA/doc/theory/`. Later that same day the beekeeper consolidated to `~/Documents/BMA/theory/`. On 2026-05-15, after the phantom-artifact / git-reset incident, files were placed at the federation-canonical home `~/Documents/inter/theory/` per the hybrid placement decision in the same session. The path above is the final canonical home.)

## Files migrated

| Addendum | Canonical location |
|---|---|
| `BMA-Theory-Addendum-11_0-Topological-Cognition.md` | `~/Documents/inter/theory/` |
| `BMA-Theory-Addendum-12_0-Prestige-Bridge.md` | `~/Documents/inter/theory/` |
| `BMA-Theory-Addendum-13_0-Cognitive-Worktrees.md` | `~/Documents/inter/theory/` |
| `BMA-Theory-Addendum-14_0-Topological-Git.md` | `~/Documents/inter/theory/` |
| `BMA-Theory-Addendum-15_0-Reciprocal-Focus.md` | `~/Documents/inter/theory/` |
| `BMA-Theory-Addendum-16_0-Cognitive-Honing.md` | `~/Documents/inter/theory/` |
| `BMA-Theory-Addendum-17_0-Proactive-Curiosity.md` | `~/Documents/inter/theory/` |

The originals in `~/Documents/QBP-Compute-Unit/theory/` are **preserved as historical record** (not deleted) to honor the authoring lineage (James Paget Butler + Gemini-CLI, in QBP-CU worktree period). Forward citation should use the new canonical path under `~/Documents/inter/theory/`. The QBP-CU originals will be retired in a future cleanup once the inter/ location has accumulated sufficient cross-references.

## Why federation-canonical (inter/) rather than BMA-canonical

The A11–A24 series describes algebraic cognitive theory whose claims are federation-wide (cross-tenant invariants). The hybrid placement decision on 2026-05-15 placed these in `inter/theory/` because:
- They are referenced by every federation tenant (not just BMA)
- Restructuring tenant agents to their own root directories (planned for next sprint) would have made BMA-rooted theory hard for other tenants to consume
- The "BMA Theory Addendum" series name is preserved as authorship attribution; location is federation-canonical

## Continuing the series

Post-A17 addenda are authored directly in `~/Documents/inter/theory/`. Current state of the numbering (as of 2026-05-15):

- **A18.0 Hypergraph Access Pattern** (canonical living theory; co-authored James + Gemini + Opus + qbp-cu-implementor 2026-05-07) — stays at `~/Documents/BMA/theory/hypergraph-inference/` since it predated the inter/ canonical decision; reference paths are preserved
- **A19.0 — reserved** for Gemini-led Stance-Algorithm coupling table per A18 §9 Q1+Q3=C invitation flow
- **A20.0 Pentagon Pod Cognitive Frame** (companion to Spec Addendum 9.1) → `repo-bma-systema-issue-#163`
- **A21.0 Federation Knowledge-Sovereignty Frame** (companion to Spec Addendum 9.2 — Federation Lean Promotion Protocol) → `repo-bma-systema-issue-#164`
- **A22.0 Cross-Tenant Autonomic Translation Layer** → `repo-bma-systema-issue-#165`
- **A23.0 Research-Aid Frame** (companion to Spec Addendum 9.4) → `repo-bma-systema-issue-#166`
- **A24.0 Hardware-Boundary Semantics** (companion to Spec Addendum 9.5) → `repo-bma-systema-issue-#167`

A20 and A21 were authored as A18.0 / A19.0 on 2026-05-14 by qbp-architecture, then renumbered the same day after the beekeeper directed consolidation surfaced the collision with the canonical A18 and the reserved A19 slot.

## Authorship note

The migrated A11.0 — A17.0 are credited to **James Paget Butler (Beekeeper) & Gemini CLI (Architect)** — that authorship is preserved in the files themselves. Migration does not re-attribute authorship.

A20.0 — A24.0 are credited to **James Paget Butler (Beekeeper) & Claude Opus 4.7 (qbp-architecture)**, with Gemini-3-Pro credited as review co-author on A22 / A23 / A24 (the addenda that responded to Gemini's review).

## Recovery note

A20.0 — A24.0 + Spec Addenda 9.{1,2,4,5} were authored on 2026-05-14 but a concurrent agent's `git reset` operation in the shared `~/Documents/BMA/` working tree at 2026-05-14 21:47:31 wiped them as untracked files. On 2026-05-15 they were reconstructed from the session transcript and placed at the final canonical locations (theory → inter/theory/; spec → inter/spec/ for federation-canonical, BMA/spec/ for BMA-specific). See `repo-bma-systema-issue-#168` (to be filed) for the incident record + lessons.

---

*BMA Theory Addenda migration record | 2026-05-14 (initial) → 2026-05-15 (canonical location update)*
