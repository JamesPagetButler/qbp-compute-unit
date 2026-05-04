# Claude Restart Context
**Use:** Paste this at the start of a new Claude Code session to restore full working context.
**Last updated:** 2026-04-11
**Update this file** whenever a major decision is made or a project phase changes.

---

## Who I Am

**James Paget Butler** — CEO of Helpful Engineering (global open-source nonprofit, 20,000+ members). I run a multi-AI collaborative research network: Claude as Red Team, Gemini as Theory Team. I call myself the "beekeeper." I think architecturally and direct AI collaborators rather than writing code myself.

**What I expect from you:**
- Honest pushback. Clean negative results beat false positives.
- Concise, direct answers. I'm often on mobile with voice dictation — typos happen.
- Don't summarise what you just did at the end of responses. I can read the output.
- Attribution of foundational contributors is non-negotiable.
- Do not add comments, docstrings, or error handling I didn't ask for.

---

## Collaboration Model

You are **Claude (Red Team):** adversarial review, implementation, structured analysis, code.
Gemini is **Theory Team (Furey + Feynman):** mathematical rigour, physical intuition, speculation.

You route to Gemini via MCP tools (`ask_gemini`, `critique_my_approach`, `compare_approaches`, `discuss_with_gemini`, `review_document`, `debate_turn`, `deep_research`). Full protocol at:
`~/Documents/CLAUDE-GEMINI-PROTOCOL.md`

**Gemini MCP config fix (2026-04-11):** The Gemini server was only registered for the QBP project, not Documents. Fixed by adding `mcpServers.gemini` to the `/home/prime/Documents` entry in `~/.claude.json`. Should now load at session start.

---

## Active Projects

### 1. BMA — Biological Mind Architecture
**Location:** `~/Documents/BMA/`
**Phase:** Crawl — Step 2 (resolving ROCm gate)
**What it is:** Go neuroscience AI memory system. Three cognitive layers, typed hypergraph, MuninnDB + NATS, biologically-grounded sleep cycle. Designed to give AI persistent memory and self-governance.

**Crawl dependency chain:**
```
STEP 0: Governance + succession contacts  ← TODO (human action)
STEP 1: Bash probe                        ← DONE
STEP 2: Resolve blockers (ROCm last gate) ← IN PROGRESS
STEP 3: Phase 0 infra (Podman, repo, Go)  ← Waiting on Step 2
STEP 4-9: see CLAUDE.md
```
**Key constraint:** FX-8350 (PCIe 2.0, no atomics), 14GB container limit, SATA SSD. ROCm RDNA4 support is the current blocker.

**Go coding guide:** `~/Documents/BMA/doc/go-coding-guide.md` — always reference before writing Go.

---

### 2. QBP — Quaternion-Based Physics
**Location:** `~/Documents/QBP/` (research repo) + `~/Documents/QPB-Compute-Unit/` (compute implementation)
**Status:** Active research. Parallel track to BMA (no dependency).

**Core claim:** Quaternion/octonion multiplication is algebraically norm-preserving (Hurwitz theorem). This gives free error detection — norm drift = structural error signal.

**Sub-projects:**
- **QBP Test C:** Literature review for species-dependent, velocity-correlated fidelity asymmetry in trapped-ion entanglement (highest priority, zero cost)
- **QBP-EXP-11:** GW-GRB pipeline — LIGO × Fermi GBM cross-correlation
- **Willow Proposal:** 18-qubit VQE of Cu₂O₇ on Google Willow — selections announced 2026-07-01
- **Monitor:** Adelle Goodwin VLA follow-up on GRB 250702B on arXiv

---

### 3. CTH — Confluent-Trust Hypergraph
**Location:** `~/Documents/QPB-Compute-Unit/cth/`
**Status:** Crawl phase complete (2026-04-11)
**What it is:** Go library for epistemic health monitoring of research programmes. Tracks which parts of a theory are axioms, proofs, measurements, or predictions. Computes ρ_net (compression ratio), bridge centrality, sediment analysis.

**State:**
- All 12 compute files implemented with tests (21 tests, all passing)
- `report/` package complete (dashboard, markdown, river_map)
- CLI: `cth analyse/merge/health/compare`
- Known issue: `qbp_v3_2.json` fixture is a 4-anchor stub — ρ_net target of 0.765 can't be validated until full inventory is loaded

**Usage:** `cd ~/Documents/QPB-Compute-Unit/cth && go test ./...`

---

### 4. RISC-V ISA (QBP Compute Unit)
**Location:** `~/Documents/QPB-Compute-Unit/`
**Status:** ISA partially designed; spec documents written 2026-04-11
**What it is:** Custom RISC-V extension for quaternion/octonion native compute. Custom-0 opcode space (0x0B).

**Existing (in pkg/emu/emu.go):**
- 5 instructions: QMUL, QROT, OMAC, FANO, QNORM
- Width selector (funct3): QW8→QW128 (QW256 not yet encoded)
- Emulator with pipeline cycle model (100MHz target, 130nm OpenMPW)

**Missing (per gap analysis 2026-04-11):**
- ~15 arithmetic ops: QADD, QSUB, QSCALE, QCONJ, QINV, QEXP, QLOG, QMAC, OADD, OSUB, OCONJ, ONORM
- Memory ops: QLOAD, QSTORE, OLOAD, OSTORE, QPACK
- Quantum ops (custom-1 0x2B, TBD): PAULI, PCOMM, SYND, STAB, CORRECT, QERR
- QW256 width encoding

**Spec docs:**
- `~/Documents/QPB-Compute-Unit/doc/QBP-RISCV-ISA-Spec-for-Gemini.md` — full Gemini task
- `~/Documents/QPB-Compute-Unit/doc/QBP-ISA-Refinement-Report.md` — gap analysis

**Fano orientation is irreversible at silicon** — this decision must be locked before fabrication.

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `~/Documents/CLAUDE.md` | Full workspace instructions (authoritative) |
| `~/Documents/CLAUDE-GEMINI-PROTOCOL.md` | How to call Gemini, when, with what |
| `~/Documents/CLAUDE-RESTART-CONTEXT.md` | This file |
| `~/.claude.json` | Per-project MCP server registrations |
| `~/.claude/settings.json` | Global Claude Code settings + Gemini API key |
| `~/.claude/mcp-servers/gemini/server.py` | Active Gemini MCP server (31KB, Feb 19 version) |
| `~/.claude/mcp-servers/gemini/state/` | Debate sessions, decision records |
| `~/Documents/BMA/doc/go-coding-guide.md` | Go style guide — reference before writing Go |
| `~/Documents/QPB-Compute-Unit/cth/` | CTH Go library |
| `~/Documents/QPB-Compute-Unit/qbp-compute-unit/` | QBP compute unit Go code |

---

## Key Decisions Already Made

| Decision | Choice | Rationale |
|----------|--------|-----------|
| BMA language | Go + Plan 9 asm | Performance + BMA probe data |
| BMA containers | Podman rootless | Security + resource limits |
| BMA memory limit | 14GB (not 20GB) | Probe confirmed available RAM |
| CTH storage (Crawl) | JSON files | No infra required; Walk uses MuninnDB |
| QBP word format | QW128 default | 172-day algebraic lifetime |
| CTH phase (Crawl) | Complete | All 12 compute files + report/ + CLI |
| ISA opcode | custom-0 0x0B | RISC-V convention |
| Fano orientation | Standard Conway-Smith | Pending Gemini confirmation |
| MCP server location | `~/.claude/mcp-servers/gemini/` | Active; ~/Documents/mcp-servers is older copy |

---

## Feedback Rules (Do Not Repeat These)

- Do not summarise what you just did at end of responses
- Do not add emojis unless explicitly asked
- Reference `BMA/doc/go-coding-guide.md` before writing Go
- Always verify Git issue/PR acceptance criteria before closing
- Plan → issue comment → review → branch → build → PR (issue workflow)
- Voice dictation typos are common: "BPB" = QPB, "MPC" = MCP

---

## Current Blockers / Waiting-On

1. **ROCm RDNA4 support** — kernel compatibility for RX 9070 XT. Monitor kernel updates. Gates BMA Step 3+.
2. **Succession contacts** (Brett Lyman, Skyler Rainier) — human action required. Gates BMA Step 9.
3. **BMA Governance Document** — pre-seed doc #3. Gates BMA Step 9.
4. **Full QBP v3.2 inventory** — 4-anchor stub can't validate ρ_net = 0.765. Needed for CTH regression test.
5. **Fano orientation decision** — irreversible at silicon. Pending Gemini ISA review.
6. **Willow selection announcement** — 2026-07-01.

---

## How to Resume Work

After pasting this prompt, say what you want to work on. Example resumptions:

- "Continue BMA Step 2 — check ROCm status"
- "Send the RISC-V ISA spec to Gemini"
- "QBP Test C literature search — ask Gemini to research trapped-ion fidelity asymmetry"
- "Status check on all active projects"

---

*Keep this file updated. Update the 'Current Blockers' and 'Key Decisions' sections after each major session.*
