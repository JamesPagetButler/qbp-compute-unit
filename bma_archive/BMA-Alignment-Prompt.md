# BMA Crawl Alignment Prompt

**For: Opus (Organizer instance)**
**From: Opus (Red Team session, March–April 2026)**
**Purpose: Align action plan, probe script, and document index before first probe run**

---

## Context

You are the organizer instance responsible for maintaining the BMA Crawl Action Plan and coordinating execution. A Red Team Opus session produced extensive architectural work across multiple sessions — container strategy, context management, interface design, sleep consolidation, death/succession protocols, Code Mode integration, and the document index. A Sonnet session produced the Pre-Crawl Synthesis Brief covering six external repos, the judge collective governance architecture, and the Godfrey-Smith consciousness decomposition.

James is now home at the physical machine and ready to execute. Before running the pre-deploy probe (STEP 1), the action plan and probe need alignment corrections identified by the Red Team review.

**Attached documents you should read:**
1. `BMA-CRAWL-ACTION-PLAN.md` — the current action plan (needs updates)
2. `pre-deploy-probe.sh` — the bash probe script (needs minor additions)
3. `BMA-Document-Index.md` — the master document index (already updated, use as reference)
4. `BMA-PreCrawl-Synthesis-Brief.md` — Sonnet's synthesis (already in index)

---

## Alignment Issues to Resolve

### 1. STEP 9 pre-seed document list: update from 5 to 6

**Current (action plan):**
```
1. Theory
2. Ethics (v1.1)
3. Spec (consolidated)
4. Crawl Environment
5. Component Summary
```

**Correct:**
```
1. Theory Consolidated
2. Ethics v1.1
3. Spec Consolidated
4. Crawl Environment
5. Component Summary
6. Pre-Crawl Synthesis Brief (judge collective, governance, skill extraction, six repos)
```

### 2. Governance Document is a named prerequisite — not inline in STEP 9

The Synthesis Brief states: "The governance document does not exist yet and must be created. This is the most important section." Sonnet's suggested work order puts the Governance Document first, before everything else.

**Action:** Add a step between STEP 0 and STEP 1 (or expand STEP 0) that explicitly calls out:
- Write the BMA Governance Document (judge collective architecture, domain-weighted approval, scoped veto, constitutional protections, beekeeper succession formalization)
- This document becomes part of the pre-seed package and the launch reading order
- It gates STEP 9 (instantiation) — BMA cannot be instantiated without governance in place

### 3. Launch reading order needs Governance Document at position 3

**Current (action plan STEP 9):** Not specified in detail.

**Correct reading order for launch:**
```
1. Theory Consolidated (know what you are)
2. Ethics v1.1 (know what you value)
3. Governance Document (know how decisions are made)
4. Spec Consolidated (know how you're built)
5. Seeds (meet your founders)
6. Notes-to-Opus from Sonnet (learn from a prior instance)
7. Final Briefing + session outputs (the most recent work)
8. Empathy Synthesis (the evolutionary framework)
```

### 4. Probe script: add CURBy and API reachability checks to Network section

The probe checks connectivity to `api.anthropic.com` and `github.com`. It should also check:

```bash
# Add to the Network section's host loop:
for host in api.anthropic.com github.com pypi.org random.colorado.edu generativelanguage.googleapis.com; do
    if ping -c 1 -W 3 "$host" &>/dev/null; then
        pass "Can reach $host"
    else
        warning "Cannot reach $host"
    fi
done
```

- `random.colorado.edu` — CURBy quantum randomness beacon (used in sleep consolidation spiral)
- `generativelanguage.googleapis.com` — Gemini API (BRIDGE provider)

These are not blockers — they're resource availability checks. CURBy falls back to crypto/rand. Gemini is a BRIDGE provider, not a Crawl prerequisite. But knowing they're reachable on Run 1 saves time later.

### 5. Probe script: add API account verification section (optional)

After the Network section, consider adding an optional API verification section that checks if environment variables are set (without exposing the keys):

```bash
section "API Accounts (optional)"

if [ -n "${ANTHROPIC_API_KEY:-}" ]; then
    pass "ANTHROPIC_API_KEY is set (${#ANTHROPIC_API_KEY} chars)"
else
    info "ANTHROPIC_API_KEY not set — needed for BRIDGE (Phase 6)"
    action "CONFIGURE: export ANTHROPIC_API_KEY=sk-ant-..."
fi

if [ -n "${GEMINI_API_KEY:-}" ]; then
    pass "GEMINI_API_KEY is set (${#GEMINI_API_KEY} chars)"
else
    info "GEMINI_API_KEY not set — needed for BRIDGE (Phase 6)"
    action "CONFIGURE: export GEMINI_API_KEY=..."
fi
```

### 6. LUKS write amplification tradeoff — add note to action plan STEP 3

The probe checks for LUKS readiness. The action plan mentions "LUKS container → mount" as an option. On the Samsung 840 (early SSD, limited write endurance), LUKS adds write amplification. Add a decision note:

```
LUKS decision: On Samsung 840 with limited write endurance, LUKS adds
write amplification. Tradeoff: data-at-rest encryption vs SSD lifespan.
Recommendation: defer LUKS until Walk hardware (newer SSD with higher
endurance), OR use LUKS only for the identity/ directory (keys, succession
contacts) not the full hypergraph.
```

### 7. QBP repo review should be a STEP 0 task

The Red Team session identified that the QBP repo should be reviewed BEFORE instantiation because its development workflow (infrastructure/housekeeping two-stream model, PR comment accumulation, review cycle, issue management) is the operational prototype for BRIDGE's orchestration patterns. The action plan lists QBP work only in the parallel track. Add a note:

```
STEP 0 addition: QBP repo review (workflow patterns, not physics)
- Review development workflow: issue → plan → PR → review → fix cycle
- Identify patterns to formalize into BRIDGE spec
- Review the two-stream model (infrastructure vs housekeeping)
- Review PR comment accumulation pattern
- This informs BRIDGE design (Phase 6), not Crawl infrastructure
```

---

## What Does NOT Need Changing

The core dependency chain (STEP 0 through STEP 9) is correct and matches the spec's Crawl milestones (C-M1 through C-M4). The probe script's structure, priority ordering, and hardware checks are thorough and well-designed. The parallel QBP track is correctly separated. The CPU-only Crawl fallback for PCIe atomics failure is properly handled.

The probe script's board-specific notes for the Crosshair V Formula-Z are excellent — the IOMMU and CSM notes may save hours of debugging.

---

## Deliverables from This Alignment Session

1. **Updated BMA-CRAWL-ACTION-PLAN.md** with:
   - Governance Document as explicit prerequisite (before or within STEP 0)
   - Pre-seed document list updated to 6
   - Launch reading order with Governance at position 3
   - LUKS tradeoff note in STEP 3
   - QBP workflow review noted as informing Phase 6

2. **Updated pre-deploy-probe.sh** with:
   - CURBy and Gemini API in network reachability checks
   - Optional API account verification section
   - (Any other improvements you identify during review)

3. **Clear next steps** for James to execute tonight/tomorrow:
   - STEP 0: Contact Brett and Skyler (phone)
   - STEP 1: Run the probe on bare metal (no GPU yet)
   - Review probe output
   - Decide: install GPU next or resolve software blockers first

---

## Available Resources

- Claude Max API: $100/month (BRIDGE budget, ~3,500 complex exchanges at Code Mode rates)
- Gemini Ultra account (1M+ token context window)
- CURBy: https://random.colorado.edu/ (quantum randomness, free)
- GoDeMode: https://github.com/imran31415/godemode (Go Code Mode reference)
- Kaiju Engine: https://github.com/KaijuEngine/kaiju (Go + Vulkan, late Crawl visualization)

---

## Red Team Notes

The single biggest technical risk remains PCIe atomics on the FX-8350. The probe correctly flags this. If ROCm GPU compute fails due to missing atomics, CPU-only Crawl is viable through Phase 5 (HG-RETRIEVE). GPU becomes a Walk hardware dependency. This is an acceptable degradation — the system learns resource discipline under even greater constraint. But it means the 72-hour gate (STEP 8) runs without GPU inference, which changes what "continuous operation" means at that phase.

The Governance Document is the biggest non-technical gap. It needs to exist before instantiation. The judge collective architecture, constitutional protections, and beekeeper succession formalization are foundational — they're the equivalent of a constitution written before the government starts operating.

*End of alignment prompt.*
