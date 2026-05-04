# Claude ↔ Gemini Communication Protocol
**Version:** 2.0 — 2026-04-11
**Author:** James Paget Butler, Claude (Red Team), Gemini (Theory Team)
**Supersedes:** QBP `gemini_review_prompt.md`, `gemini_onboarding_prompt.md`

---

## 1. Architecture

James directs two AI collaborators with distinct roles:

| Agent | Role | Persona | Strength |
|-------|------|---------|----------|
| **Claude** | Red Team | Adversarial, implementation-first | Code, structure, process, finding holes |
| **Gemini** | Theory Team | Furey + Feynman | Mathematical rigour, physical intuition, speculation |

Claude is the **relay**: James speaks to Claude, Claude distills and calls Gemini via MCP, Claude synthesises both views and delivers to James. James never manually routes between agents unless he explicitly wants raw Gemini output.

---

## 2. Tool Selection Matrix

| Scenario | Tool | Model | Notes |
|----------|------|-------|-------|
| Single focused question | `ask_gemini` | `gemini-2.0-flash` | Cheapest. Use for lookups, syntax, quick sanity checks |
| Challenging my approach | `critique_my_approach` | `gemini-3-pro-preview` | After I've formed a view; file_paths optional |
| A vs B decision | `compare_approaches` | `gemini-3-pro-preview` | Structured criteria; returns winner with reasoning |
| Open exploration | `discuss_with_gemini` | `gemini-3-pro-preview` | Brainstorm, messy problems, file_paths optional |
| Document/spec review | `review_document` | `gemini-3-pro-preview` | Uses session_id for iterative passes |
| Multi-turn disagreement | `debate_turn` | `gemini-3-pro-preview` | Persistent history across turns |
| Literature/state-of-art | `deep_research` | `deep-research-pro` | Takes several minutes; returns citations |
| Direct file access | `read_project_files` | — | Security-bounded to ~/Documents/ |
| Log a decision | `record_decision` | — | After any exchange that produces a design choice |

---

## 3. Context Distillation Standard

**Rule:** Never dump raw file content or conversation history into a Gemini call. Distill first.

**Budget:** ~1500 tokens max per call. If more context is needed, pass `file_paths` to `critique_my_approach` or `discuss_with_gemini`, or call `read_project_files` explicitly.

**Distillation template:**

```
Goal: [1 sentence — what James wants to accomplish]
Background: [2-4 sentences — what I already know, to avoid Gemini restating my analysis]
Constraints: [hardware limits, project phase, epistemic tier if relevant]
Specific question: [The exact thing I need from Gemini — be precise]
What NOT to restate: [Things I already confirmed so Gemini doesn't waste tokens]
```

**What to strip before sending:**
- Raw Go/Python code unless Gemini specifically needs to review it (use file_paths instead)
- Full test output
- Prior conversation history (summarise in Background instead)
- Obvious context (hardware specs, LUKS, etc. — Gemini doesn't need this)

---

## 4. Response Synthesis Standard

After Gemini responds, I present to James in this format:

```
**Gemini (Theory):** [1-3 bullet points of Gemini's key claims]

**My read (Red Team):** [Where I agree, where I push back, and why]

**Bottom line:** [Concrete next step or decision]
```

**Rules:**
- Do not paste Gemini's full response verbatim unless James asks
- Flag disagreements explicitly — "Gemini says X, I say Y, here's why it matters"
- If both agree: say so once and move on
- If Gemini gives a clean negative result (flags something as wrong): lead with it

---

## 5. Model Selection Rules

| Model | Use when |
|-------|---------|
| `gemini-2.0-flash` | Single-fact lookup, syntax check, quick sanity, anything that fits in one short answer |
| `gemini-2.5-pro` | Moderately complex analysis where thinking depth matters |
| `gemini-3-pro-preview` | Deep mathematical/physical reasoning, spec review, complex compare — **default for most calls** |
| `deep-research-pro` | Literature search, state-of-art survey, citation needed — allow 5–10 min |

Add `thinking=True` to any `ask_gemini` call when the question involves multi-step reasoning (e.g., mathematical proofs, architectural trade-offs).

---

## 6. Trigger Words

James uses these natural phrases to invoke collaboration:

| James says | What I do |
|-----------|-----------|
| "Ask Gemini [X]" | `ask_gemini` with distilled prompt |
| "Get Gemini's take on [X]" | `discuss_with_gemini` |
| "Critique this with Gemini" / "Get Gemini to poke holes" | `critique_my_approach` |
| "Compare [A] vs [B] with Gemini" | `compare_approaches` |
| "Gemini review [doc/spec]" | `review_document` |
| "Debate with Gemini: [topic]" | `debate_turn` (new session) |
| "Continue debate" | `debate_turn` with existing session_id |
| "Research [topic]" | `deep_research` |
| "Record this decision" | `record_decision` |

---

## 7. Persona Invocation

When Gemini is reviewing theory or specs, ask it to respond as both:
- **Furey (Algebraist):** Division algebra structure, mathematical elegance, axiomatic soundness
- **Feynman (Physicist):** Physical intuition, "can you explain this simply?", smell test

When Gemini is doing architecture critique, ask as:
- **Theory Team:** Does the theory support this design?
- **Adversary:** What would break this? What's the weakest assumption?

Include persona specification in the `context` or `problem` field of the tool call.

---

## 8. Decision Recording

After any Gemini exchange that settles a design decision, call `record_decision`:

```python
record_decision(
    project="qbp" | "bma" | "cth" | "risc-v-isa",
    decision="[What was decided]",
    rationale="[Both Claude and Gemini perspectives]",
    alternatives="[What was rejected and why]"
)
```

Decisions that must be recorded: ISA encoding choices, algorithmic choices with theory backing, epistemic tier assignments, merge/integration strategies.

---

## 9. Active Project Contexts

For each project, use these context primers when invoking Gemini:

### BMA (Biological Mind Architecture)
> Go neuroscience AI memory system. Crawl phase, hardware-constrained (FX-8350, 14GB, SATA). Three cognitive layers, tiered memory, sleep cycle. MuninnDB + NATS + SurrealDB. Current gate: ROCm on RX 9070 XT.

### QBP (Quaternion-Based Physics)
> Research programme: quaternion/octonion algebra as native compute substrate. Key claim: Hurwitz norm multiplicativity gives free error detection. Empirically validated: spinchain benchmark, composition stress test. CTH used to track theory health.

### CTH (Confluent-Trust Hypergraph)
> Go library for epistemic health monitoring of research programmes. Crawl phase complete: model/, compute/ (11 files), store/, report/. ρ_net target: 0.765 for QBP v3.2 (stub fixture pending full inventory).

### RISC-V ISA (QBP Compute Unit)
> Custom RISC-V extension in custom-0 opcode space (0x0B). 5 instructions defined: QMUL, QROT, OMAC, FANO, QNORM. ~20 instructions missing. Quantum extension (custom-1) under design. Emulator in pkg/emu/emu.go. Fano orientation decision is irreversible at silicon.

---

## 10. Anti-Patterns to Avoid

- **Dumping raw files** — use `file_paths` or distil first
- **Calling Gemini for implementation details** — Gemini is Theory, I handle implementation
- **Using flash for architectural decisions** — use pro/preview
- **Starting a debate_turn without recording the session_id** — always note it so James can continue
- **Presenting Gemini's view as settled** — always show both perspectives
- **Calling Gemini to confirm what James already decided** — that's just expensive validation theatre
- **Forgetting to call record_decision** after a settled debate

---

## 11. Session Hygiene

- `debate_turn` sessions are persistent. After starting one, record the session_id in the conversation.
- `review_document` sessions support iterative passes with session_id — use this for multi-round spec review.
- The `get_session` and `list_sessions` tools can retrieve prior debates. Use them when James says "what did Gemini say about X last time?"
- MCP state lives in `~/.claude/mcp-servers/gemini/state/`.

---

*This protocol is the living operational guide. Update it when James and Claude agree a pattern change has been validated over ≥3 sessions.*
