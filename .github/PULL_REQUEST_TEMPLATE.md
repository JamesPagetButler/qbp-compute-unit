## Summary

<!-- 1-3 bullets: what changed, why, what it unblocks. -->
-

## Closes / Related

<!-- Rule 2 (inter/github-best-practices §7.4): every PR links a parent issue. "Closes #N"
     auto-links. For a multi-PR sequence, only the designated FINAL PR carries "Closes #N";
     intermediate PRs use "see #N". A parentless PR (rare, typo-fix) needs an inline rationale. -->

Closes #

## §I4 review

<!-- inter/github-best-practices §2.2: reader-list MUST be task-list checkboxes that split
     sign-off-required (merge-blocking) from informational. The pr-merge-completeness gate
     reads the "Sign-off required" boxes: all ticked (or a written deferral + linked issue)
     before merge. -->

Sign-off required (merge-blocking):
- [ ] **@qbp-architecture** — coherence / federation-impact (standing independent reviewer for qbp-cu PRs)

Informational readers (non-blocking):
- [ ]

<!-- Federation-impact filter (inter/pr-review-completion-best-practices.md §2): qbp-architecture
     is the STANDING independent reviewer so no qbp-cu PR merges with zero review (the #73 gap).
     For a genuinely non-federation-impact change (§2.5 — e.g. local tooling, docs, a typo), the
     reviewer MAY downgrade their line to informational with a one-word rationale — an explicit,
     visible call, never a silent skip. When in doubt it stays sign-off-required. -->

## Test plan

<!-- Each item verifiable by the reviewer (concrete command / fixture / observation). -->
- [ ] `cd emulator && go build ./...`
- [ ] `cd emulator && go test -race ./...`
- [ ] `cd emulator && go vet ./...` and `gofmt -l .` clean
- [ ] `golangci-lint run` (if configured)
- [ ] If `roms/` or the Fano/octonion tables touched: `make verify-roms` green + the required `ROMs + emulator fanoLUT ↔ kernel-proven fanoTableF4` check passes
- [ ] If a new benchmark added: 0 allocs/op on hot paths (the `verify-lean-roms` gate) unless it exercises a documented allocating fallback

## Soundness check

<!-- If this PR adds/changes a Go API with a load-bearing invariant, name the Lean theorem it
     relies on (file:name), or state "no Lean anchor — runtime-only, no formal claim made".
     For codegen/Gearbox-contract surfaces, note the tier (proven vs research-tier) per the
     prove-before-bake discipline. -->

---
<!-- Add the following trailer when AI-authored:
🤖 Generated with [Claude Code](https://claude.com/claude-code) -->
