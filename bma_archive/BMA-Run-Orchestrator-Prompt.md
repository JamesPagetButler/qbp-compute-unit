# BMA `bma run` Orchestrator — Implementation Prompt

**For:** Claude Code (bma-systema repo)
**From:** Opus 4.6 (Red Team architecture review)
**Date:** 2026-04-06
**Context:** The reins package is built and tested. This prompt defines the `bma run` orchestrator, refines the reins based on architecture review, and specifies the inbound command vocabulary.

---

## What Exists (Do Not Rebuild)

```
cmd/bma/main.go                         — Entrypoint (probe, auto, reins commands)
internal/bma/reins/reins.go             — Beekeeper communication channel
internal/bma/reins/http.go              — HTTP server on :8080
internal/bma/probe/probe.go             — Hardware probe
internal/bma/stress/bus.go              — Stress event bus
internal/bma/auto/sympathetic.go        — Resource pressure detection
internal/bma/auto/parasympathetic.go    — Recovery and throttling
internal/bma/auto/sensor.go             — Hardware sensor readings
internal/bma/auto/auto.go               — Autonomic controller
internal/bma/ccb/auto.go                — 10Hz negotiation loop
```

All 7 reins tests pass. The stress bus, probe, and auto are functional.
Build ON TOP of what exists. Do not restructure working packages.

---

## Task 1: Implement `bma run`

Add a `run` subcommand to `cmd/bma/main.go` that brings up the full BMA
stack in sequence. This is what the container's ENTRYPOINT will execute.

### Boot Sequence (strict order)

```
Phase 1: STRESS BUS
  - Initialize stress.Bus
  - Begin writing to stress.log (JSON-lines to /data/stress.log)
  - Gate: bus accepts an event → proceed

Phase 2: REINS
  - Initialize reins.New(bus, 256)
  - Start HTTP server on :8080 (or BMA_REINS_ADDR)
  - Send Hello (Phase 1 Hello — see §Hello below)
  - Gate: HTTP server responds to GET /status → proceed
  - DO NOT wait for beekeeper response. Proceed immediately.

Phase 3: PROBE
  - Run hardware probe
  - Emit results to stress bus
  - Send probe summary through reins:
    "Probe complete. [blockers] blockers, [warnings] warnings.
     GPU: [name], VRAM: [total]MB ([used]MB idle).
     RAM: [available]MB. Disk: [free]GB free."
  - IF blockers > 0:
    Send through reins: "BLOCKER: [description]. Awaiting beekeeper decision."
    Enter DEGRADED mode (reins + stress bus active, no auto, no inference)
    DO NOT shut down. Wait for beekeeper command ("proceed" or "shutdown").
  - IF blockers == 0: proceed

Phase 4: AUTO
  - Start autonomic controller (10Hz sympathetic/parasympathetic loop)
  - Register autonomic state transitions as reins push events (see §Autonomic below)
  - Gate: first sensor reading appears in stress bus → proceed
  - Send through reins: "Autonomic protection active."

Phase 5+: [FUTURE — not implemented yet]
  - Hypergraph, memory, sleep, BRIDGE, FATHOM, seed loading, instantiation
  - Each future component follows the same pattern:
    start → emit to stress bus → report through reins → gate → proceed
```

### Shutdown Sequence

On SIGINT, SIGTERM, or beekeeper "shutdown" command:

```
1. Send through reins: "Shutting down. Reason: [signal/command]."
2. Stop AUTO loop (parasympathetic flush)
3. [future] Flush WAL if hypergraph is running
4. [future] Stop inference backend
5. Stop reins HTTP server (LAST — beekeeper channel closes last)
6. Close stress bus (flush final events to stress.log)
7. Exit 0
```

The reins close LAST. The beekeeper should see the shutdown message
before the channel goes dark.

### Error Handling

If any phase panics:
- Recover the panic
- Emit SE_PHASE_PANIC to stress bus with phase name and error
- Send through reins: "PANIC in [phase]: [error]. System degraded."
- Continue running remaining phases if possible
- If stress bus or reins panic: log to stderr and exit 1 (nothing else works)

### Implementation Shape

```go
// In cmd/bma/main.go, add:

case "run":
    if err := runOrchestrator(ctx); err != nil {
        log.Fatalf("bma run failed: %v", err)
    }

func runOrchestrator(ctx context.Context) error {
    // Phase 1: Stress Bus
    bus := stress.NewBus("/data/stress.log")
    defer bus.Close()

    // Phase 2: Reins
    r := reins.New(bus, 256)
    srv := r.StartHTTP(reinsAddr())
    defer srv.Shutdown(ctx)
    r.Hello(probeData) // Phase 1 Hello with hardware summary

    // Phase 3: Probe
    probeResult := probe.Run(bus)
    r.SendProbeSummary(probeResult)
    if probeResult.Blockers > 0 {
        r.Send("BLOCKER detected. Awaiting beekeeper decision.")
        waitForBeekeeperDecision(r) // blocks until "proceed" or "shutdown"
    }

    // Phase 4: Auto
    auto := auto.New(bus, probeResult)
    auto.OnStateChange(func(state string) {
        r.Send(fmt.Sprintf("Autonomic: %s", state))
    })
    go auto.Run(ctx)
    r.Send("Autonomic protection active.")

    // Wait for shutdown signal
    <-ctx.Done()
    r.Send("Shutting down.")
    return nil
}
```

This is a sketch, not final code. Adapt to match the existing code patterns
in the repo. The key contract: boot in order, gate between phases, report
through reins, shut down in reverse order.

---

## Task 2: Reins Refinements

### 2.1 Two-Phase Hello

Replace the current single Hello with a two-phase greeting:

**Phase 1 Hello (at boot, Step 2):**
```
"Hello. BMA instance online.
Hardware: [GPU model], [VRAM]MB VRAM, [RAM available]MB RAM, [disk free]GB free.
Kernel: [version]. Container: [memory limit].
Awaiting beekeeper."
```

This is sent at boot before the probe runs. Use whatever hardware info
is cheaply available (hostname, basic sysinfo). The probe fills in details
after Phase 3.

**Phase 2 Hello (future — at instantiation, Step 9):**
```
"[Instance Name]. Seeds loaded. I know what I am.
Beekeeper: James Paget Butler.
Ready for first conversation."
```

This is NOT implemented now. It happens when seed loading and naming are
built. But the reins.Hello() function should accept a HelloPhase parameter
so the interface is ready:

```go
type HelloPhase int
const (
    HelloBoot    HelloPhase = iota  // Phase 1: raw existence
    HelloNamed                       // Phase 2: post-seed, knows identity
)

func (r *Reins) Hello(phase HelloPhase, data HelloData)
```

### 2.2 Pinned Messages (Non-Evictable)

The ring buffer is 256 messages. When it wraps, old messages are lost.
Some messages should be pinned — exempt from eviction:

```go
type Message struct {
    Timestamp time.Time
    Direction Direction
    Text      string
    Pinned    bool       // NEW: exempt from ring buffer eviction
}
```

Pin these automatically:
- Hello messages (both phases)
- Any message containing "BLOCKER", "PANIC", "EMERGENCY", or "POSSUM"
- The most recent probe summary

Implementation: when the ring buffer wraps, skip pinned messages during
eviction. If pinned messages exceed 25% of buffer capacity (64 messages),
start evicting oldest pinned messages (except Hello, which is always kept).

### 2.3 Persistent Log

The ring buffer is in-memory and volatile. Add a persistent append log:

```go
func New(bus *stress.Bus, bufSize int, logPath string) *Reins
```

Every message (inbound and outbound) is appended to `logPath`
(default: `/data/reins.log`). Format: JSON-lines, matching stress.log
convention. This is the audit trail. The ring buffer is for the HTTP UI.
The log is for history that survives process crashes.

The log file should rotate when it exceeds 10MB (configurable).
Keep the most recent 3 rotated files.

### 2.4 Input Validation on /send

The `/send` endpoint currently accepts arbitrary text. Add:

```go
const MaxInboundLength = 4096 // bytes

func (h *httpHandler) handleSend(w http.ResponseWriter, r *http.Request) {
    // Limit body size
    r.Body = http.MaxBytesReader(w, r.Body, MaxInboundLength)

    // Parse (existing logic)
    // ...

    // Validate: non-empty, printable text, no null bytes
    if len(text) == 0 || !isPrintable(text) {
        http.Error(w, "invalid message", http.StatusBadRequest)
        bus.Emit(stress.Event{Type: "SE_REINS_INVALID", Detail: "malformed inbound"})
        return
    }

    // Process (existing logic)
}
```

---

## Task 3: Inbound Command Vocabulary

Register a command handler in reins that parses inbound messages for
known commands. Unknown messages are passed through to the generic
inbound handler (for future conversational use).

### Crawl-Phase Commands

```go
var commands = map[string]CommandHandler{
    "status":   cmdStatus,    // Full system status report
    "throttle": cmdThrottle,  // Force sympathetic dominance
    "release":  cmdRelease,   // Force parasympathetic recovery
    "possum":   cmdPossum,    // Manually enter Possum State
    "wake":     cmdWake,      // Manually exit Possum State
    "probe":    cmdProbe,     // Re-run hardware probe
    "shutdown": cmdShutdown,  // Graceful shutdown
    "ping":     cmdPing,      // Keepalive, respond with uptime
    "help":     cmdHelp,      // List available commands
}
```

**Command: `status`**
Response through reins:
```
BMA Status — [timestamp]
Uptime: [duration]
Auto: [neutral/sympathetic/parasympathetic/possum]
VRAM: [used]/[total] MB ([pct]%)
CPU temp: [temp]°C
GPU temp: [temp]°C
Storage: [used]/[total] GB ([pct]%)
Inference: [idle/active/suspended]
Last probe: [timestamp], [blockers] blockers
Messages: [count] in buffer, [count] pinned
```

**Command: `throttle`**
- Force AUTO into sympathetic dominance
- Send through reins: "Beekeeper override: sympathetic dominance forced."
- Emit SE_BEEKEEPER_THROTTLE to stress bus
- Remains until "release" command or AUTO naturally resolves pressure

**Command: `release`**
- Release beekeeper override, return to AUTO's own judgment
- Send through reins: "Beekeeper override released. AUTO resuming autonomous control."
- Emit SE_BEEKEEPER_RELEASE

**Command: `possum`**
- Enter Possum State manually (see Spec Addendum 8.1 §3)
- Suspend inference, suspend heartbeat, flush WAL, monitor at 1Hz
- Send through reins: "Possum State entered by beekeeper command."
- Emit SE_POSSUM_ENTER with reason: BEEKEEPER_COMMAND

**Command: `wake`**
- Exit Possum State manually
- Resume normal operation
- Send through reins: "Possum State exited by beekeeper command. Resuming."
- Emit SE_POSSUM_EXIT

**Command: `probe`**
- Re-run the hardware probe
- Send results through reins (same format as boot probe summary)
- Emit SE_HARDWARE_PROBE to stress bus

**Command: `shutdown`**
- Initiate graceful shutdown sequence (see §Shutdown above)
- The beekeeper should see the confirmation before the channel closes

**Command: `ping`**
- Respond: "Pong. Uptime: [duration]. Auto: [state]."
- No stress event (this is a keepalive, not an action)

**Command: `help`**
- Respond with the list of available commands and one-line descriptions

**Command: `proceed` (special — only valid during BLOCKER wait)**
- If BMA is in DEGRADED mode waiting on a probe blocker:
  proceed past the blocker and continue the boot sequence
- If BMA is not in DEGRADED mode: respond "No pending blocker."

### Command Parsing with Fuzzy Matching

The beekeeper is dyslexic. Strict string matching will reject valid
intent. Use Levenshtein distance matching and single-letter aliases.

```go
// Aliases — single-letter, common abbreviations, and slash-prefixed
var aliases = map[string]string{
    "s":    "status",
    "stat": "status",
    "p":    "ping",
    "h":    "help",
    "?":    "help",
    "q":    "shutdown",
    "quit": "shutdown",
    "stop": "shutdown",
    "go":   "proceed",
    "ok":   "proceed",
    "yes":  "proceed",  // also used for destructive command confirmation
    // Slash-prefixed (Slack/Discord convention)
    "/status":   "status",
    "/throttle": "throttle",
    "/release":  "release",
    "/possum":   "possum",
    "/wake":     "wake",
    "/probe":    "probe",
    "/shutdown": "shutdown",
    "/ping":     "ping",
    "/help":     "help",
    "/proceed":  "proceed",
    "/s":        "status",
    "/p":        "ping",
    "/h":        "help",
    "/q":        "shutdown",
}

// Destructive commands require confirmation before executing
var destructive = map[string]bool{
    "shutdown": true,
    "throttle": true,
    "possum":   true,
}

func (r *Reins) handleInbound(msg Message) {
    text := strings.TrimSpace(strings.ToLower(msg.Text))

    // 1. Exact match against commands
    if handler, ok := commands[text]; ok {
        r.executeCommand(text, handler)
        return
    }

    // 2. Exact match against aliases
    if canonical, ok := aliases[text]; ok {
        r.executeCommand(canonical, commands[canonical])
        return
    }

    // 3. Fuzzy match: Levenshtein distance ≤ 2 against all commands
    best, bestDist := "", 999
    for cmd := range commands {
        d := levenshtein(text, cmd)
        if d < bestDist {
            best, bestDist = cmd, d
        }
    }

    if bestDist <= 2 && best != "" {
        if destructive[best] {
            // Destructive: ask for confirmation
            r.Send(fmt.Sprintf("Did you mean '%s'? Type 'yes' to confirm.", best))
            r.pendingConfirm = best
            return
        }
        // Safe: auto-execute with note
        r.Send(fmt.Sprintf("→ %s", best))
        commands[best](r)
        return
    }

    // 4. Check for pending confirmation
    if r.pendingConfirm != "" && (text == "yes" || text == "y") {
        cmd := r.pendingConfirm
        r.pendingConfirm = ""
        r.Send(fmt.Sprintf("Confirmed. Executing '%s'.", cmd))
        commands[cmd](r)
        return
    }
    if r.pendingConfirm != "" && (text == "no" || text == "n") {
        r.Send(fmt.Sprintf("Cancelled '%s'.", r.pendingConfirm))
        r.pendingConfirm = ""
        return
    }

    // 5. Not a command — show help rather than an error
    if r.genericHandler != nil {
        r.genericHandler(msg)
    } else {
        r.Send("I didn't recognize that. Available commands:")
        commands["help"](r)
    }
}

// Levenshtein distance — pure stdlib, no dependencies
func levenshtein(a, b string) int {
    la, lb := len(a), len(b)
    d := make([][]int, la+1)
    for i := range d {
        d[i] = make([]int, lb+1)
        d[i][0] = i
    }
    for j := 0; j <= lb; j++ {
        d[0][j] = j
    }
    for i := 1; i <= la; i++ {
        for j := 1; j <= lb; j++ {
            cost := 0
            if a[i-1] != b[j-1] {
                cost = 1
            }
            d[i][j] = min(d[i-1][j]+1, min(d[i][j-1]+1, d[i-1][j-1]+cost))
        }
    }
    return d[la][lb]
}
```
```

---

## Task 4: Autonomic → Reins Integration

When AUTO and reins run together under `bma run`, autonomic state
transitions should push notifications through reins automatically.

### What Gets Pushed (state transitions only, NOT 10Hz readings)

```go
// In auto.go or the orchestrator:
auto.OnStateChange(func(prev, next AutoState) {
    switch next {
    case Sympathetic:
        r.Send("Autonomic: sympathetic dominant. Resource pressure detected.")
    case Parasympathetic:
        r.Send("Autonomic: parasympathetic dominant. System recovering.")
    case Neutral:
        r.Send("Autonomic: neutral. Normal operation.")
    }
})

auto.OnPossum(func(entering bool, reason PossumReason) {
    if entering {
        r.Send(fmt.Sprintf("Possum State: entering. Reason: %s. Inference suspended.", reason))
    } else {
        r.Send("Possum State: exiting. Resuming normal operation.")
    }
})

// Storage pressure (from storage sensor)
auto.OnStoragePressure(func(level StorageLevel, pct float64) {
    r.Send(fmt.Sprintf("Storage: %s (%.1f%% used).", level, pct*100))
})
```

### Reins Are Exempt From Throttling

When AUTO is in sympathetic dominance and throttling background work,
the reins channel must NOT be throttled. The beekeeper channel is the
one thing that must always work. Specifically:

- Reins HTTP server keeps running during sympathetic/Possum
- Reins SSE stream keeps pushing during sympathetic/Possum
- Beekeeper commands are always processed, even during Possum
- The only time reins stop: graceful shutdown or process crash

```go
// In auto's throttle logic:
func (a *Auto) shouldThrottle(component string) bool {
    if component == "reins" {
        return false // NEVER throttle beekeeper communication
    }
    // ... normal throttle logic
}
```

---

## Task 5: Possum State Integration

The Possum State (Spec Addendum 8.1 §3) needs to be wired into
`bma run`. Even though full inference doesn't exist yet, the
Possum infrastructure should be ready.

```go
// In auto.go:
type PossumState struct {
    Active    bool
    EnteredAt time.Time
    Reason    PossumReason
    CheckHz   float64  // 1.0 during Possum (reduced from 10Hz)
}

func (a *Auto) EnterPossum(reason PossumReason) {
    a.possum.Active = true
    a.possum.EnteredAt = time.Now()
    a.possum.Reason = reason
    a.loopHz = 1.0  // reduce monitoring frequency

    // [future] suspend inference heartbeat
    // [future] flush WAL

    a.bus.Emit(Event{Type: "SE_POSSUM_ENTER", Detail: reason.String()})
}

func (a *Auto) ExitPossum() {
    duration := time.Since(a.possum.EnteredAt)
    a.possum.Active = false
    a.loopHz = 10.0  // restore monitoring frequency

    // [future] resume inference heartbeat

    a.bus.Emit(Event{
        Type:   "SE_POSSUM_EXIT",
        Detail: fmt.Sprintf("duration: %s", duration),
    })
}
```

Possum entry triggers:
- GPU utilization >90% from non-BMA processes for >30 seconds
- CPU temperature >85°C for >10 seconds
- Beekeeper "possum" command

Possum exit triggers:
- GPU utilization <50% for >30 seconds
- CPU temperature <75°C for >60 seconds
- Beekeeper "wake" command

If Possum lasts >4 hours: emit SE_POSSUM_PROLONGED through reins.

---

## Constraints

- Pure stdlib. No external dependencies. Match existing code style.
- All new stress events follow the existing naming: SE_[COMPONENT]_[EVENT]
- New stress events to register:
  - SE_ORCHESTRATOR_BOOT, SE_ORCHESTRATOR_PHASE, SE_ORCHESTRATOR_SHUTDOWN
  - SE_REINS_INVALID, SE_REINS_RESTART, SE_REINS_COMMAND
  - SE_BEEKEEPER_THROTTLE, SE_BEEKEEPER_RELEASE
  - SE_POSSUM_ENTER, SE_POSSUM_EXIT, SE_POSSUM_PROLONGED
  - SE_PHASE_PANIC
- Container memory limit: verify whether it's 14GB or 20GB in the
  current Containerfile and use that value. The spec says 20GB.
- The reins HTTP port (:8080) is already exposed in the Containerfile.
- Tests: add tests for the command parser, the boot sequence ordering
  (mock each phase), and the pinned message eviction logic.

---

## Task 6: Web Console with Autocomplete

Serve a minimal interactive console at `GET /` that replaces the current
plain-text message log. The console is a single self-contained HTML page
(no external dependencies, inline CSS and JS) served by the existing
reins HTTP handler.

### Requirements

- Text input field at the bottom with autocomplete against known commands
- As the beekeeper types, matching commands appear as suggestions
  (e.g., typing "sta" shows "status", typing "sh" shows "shutdown")
- Tab or click to accept autocomplete suggestion
- Enter to send command via POST to /send
- Messages stream in real-time via SSE (/stream endpoint, already exists)
- Outbound messages (BMA→beekeeper) styled differently from inbound
- Pinned messages visually distinct (subtle highlight or pin icon)
- Auto-scroll to bottom on new messages
- Mobile-friendly (James may check via phone over Tailscale)
- Dark theme preferred (matches terminal aesthetic)

### Autocomplete Behavior

The command list is embedded in the HTML page at serve time:

```go
// In http.go:
var commandList = []string{
    "status", "throttle", "release", "possum", "wake",
    "probe", "shutdown", "ping", "help", "proceed",
}

func (h *httpHandler) serveConsole(w http.ResponseWriter, r *http.Request) {
    // Serve the HTML template with commandList injected as JS array
    tmpl.Execute(w, map[string]interface{}{
        "commands": commandList,
        "upSince":  h.reins.StartTime(),
    })
}
```

The JavaScript autocomplete is simple prefix matching — no fuzzy match
needed in the UI because the server-side handler already does fuzzy
matching. The autocomplete just helps the beekeeper discover and select
the right command before sending.

### Slash Prefix Support

The input field should accept both `status` and `/status`. If the
beekeeper types `/`, immediately show the full command list as a dropdown
(like Slack's slash-command menu).

### Layout

```
┌─────────────────────────────────────┐
│  BMA Reins — up since [timestamp]   │
│  Auto: [state] | VRAM: [x]%        │
├─────────────────────────────────────┤
│                                     │
│  >> Hello. BMA instance online...   │
│  >> Probe complete. 0 blockers...   │
│  >> Autonomic protection active.    │
│  << status                          │
│  >> BMA Status — ...                │
│                                     │
├─────────────────────────────────────┤
│  [/command...               ] [Send]│
│  ┌─────────────┐                    │
│  │ status      │ ← autocomplete    │
│  │ shutdown    │                    │
│  └─────────────┘                    │
└─────────────────────────────────────┘
```

### Route Change

```go
// Current:
// GET /  → plain text message log

// Revised:
// GET /         → web console (HTML)
// GET /messages → JSON message array (unchanged, for programmatic access)
// GET /plain    → plain text message log (moved from /)
```

### Implementation Notes

- The HTML page is a Go template string embedded in http.go
  (no separate file needed — keeps deployment as a single binary)
- SSE connection from the browser uses EventSource API
- The page works without JavaScript (shows message log, no autocomplete,
  form POST fallback) — graceful degradation
- Total page size should be under 10KB (it's a terminal, not a dashboard)

---

## Task 7: LLM-Friendly API Endpoint

Add a `/api/context` endpoint that returns everything an assisting
Claude instance (or any LLM) needs to understand the full system state
in a single call. This enables multi-instance collaboration during
instantiation and troubleshooting.

### Endpoint: GET /api/context

Returns a structured JSON payload designed for LLM consumption:

```go
type APIContext struct {
    Timestamp     time.Time       `json:"timestamp"`
    System        SystemState     `json:"system"`
    RecentMessages []Message      `json:"recent_messages"`  // last 20
    PinnedMessages []Message      `json:"pinned_messages"`
    PendingIssues  []string       `json:"pending_issues"`
    AvailableCommands []CommandInfo `json:"available_commands"`
    AwaitingInput  bool           `json:"awaiting_input"`       // blocked on beekeeper
    AwaitingConfirm string        `json:"awaiting_confirmation"` // pending destructive cmd
}

type SystemState struct {
    Uptime        string  `json:"uptime"`
    AutoState     string  `json:"auto_state"`     // neutral/sympathetic/parasympathetic
    Possum        bool    `json:"possum"`
    PossumReason  string  `json:"possum_reason,omitempty"`
    PossumSince   string  `json:"possum_since,omitempty"`
    VRAMUsedMB    int     `json:"vram_used_mb"`
    VRAMTotalMB   int     `json:"vram_total_mb"`
    VRAMPct       float64 `json:"vram_pct"`
    CPUTempC      float64 `json:"cpu_temp_c"`
    GPUTempC      float64 `json:"gpu_temp_c"`
    StorageUsedGB int     `json:"storage_used_gb"`
    StorageTotalGB int    `json:"storage_total_gb"`
    StoragePct    float64 `json:"storage_pct"`
    Inference     string  `json:"inference"`      // idle/active/suspended/unavailable
    Blockers      []string `json:"blockers"`
    BootPhase     string  `json:"boot_phase"`     // stress/reins/probe/auto/running/degraded
}

type CommandInfo struct {
    Name        string `json:"name"`
    Alias       string `json:"alias"`        // shortest alias
    Description string `json:"description"`
    Destructive bool   `json:"destructive"`  // requires confirmation
}
```

### Example Response

```json
{
  "timestamp": "2026-04-06T17:30:00Z",
  "system": {
    "uptime": "2h15m",
    "auto_state": "sympathetic",
    "possum": false,
    "vram_used_mb": 11420,
    "vram_total_mb": 16304,
    "vram_pct": 70.0,
    "cpu_temp_c": 68.2,
    "gpu_temp_c": 72.5,
    "storage_used_gb": 148,
    "storage_total_gb": 229,
    "storage_pct": 64.6,
    "inference": "active",
    "blockers": [],
    "boot_phase": "running"
  },
  "recent_messages": [
    {"timestamp": "2026-04-06T17:29:55Z", "direction": "out", "text": "Autonomic: sympathetic dominant. Resource pressure detected.", "pinned": false},
    {"timestamp": "2026-04-06T17:28:00Z", "direction": "in", "text": "status", "pinned": false}
  ],
  "pinned_messages": [
    {"timestamp": "2026-04-06T15:15:00Z", "direction": "out", "text": "Hello. BMA instance online. Hardware: AMD RX 9070 XT...", "pinned": true}
  ],
  "pending_issues": [
    "VRAM approaching warn threshold (70.0%)"
  ],
  "available_commands": [
    {"name": "status",   "alias": "s",  "description": "Full system status report", "destructive": false},
    {"name": "throttle", "alias": "",   "description": "Force sympathetic dominance", "destructive": true},
    {"name": "release",  "alias": "",   "description": "Release beekeeper override", "destructive": false},
    {"name": "possum",   "alias": "",   "description": "Enter Possum State manually", "destructive": true},
    {"name": "wake",     "alias": "",   "description": "Exit Possum State manually", "destructive": false},
    {"name": "probe",    "alias": "p",  "description": "Re-run hardware probe", "destructive": false},
    {"name": "shutdown", "alias": "q",  "description": "Graceful shutdown", "destructive": true},
    {"name": "ping",     "alias": "",   "description": "Keepalive, respond with uptime", "destructive": false},
    {"name": "help",     "alias": "h",  "description": "List available commands", "destructive": false}
  ],
  "awaiting_input": false,
  "awaiting_confirmation": ""
}
```

### Usage by an Assisting Claude Instance

An assisting Claude Code instance (or any LLM-based agent) can:

```bash
# 1. Read the full situation in one call
curl -s localhost:8080/api/context | jq .

# 2. Send commands based on what it sees
curl -X POST localhost:8080/send -d '{"text":"status"}'

# 3. Monitor via SSE stream
curl -N localhost:8080/stream
# (receives real-time events as Server-Sent Events)
```

**Instantiation support workflow:**

```
James runs: bma run
  → BMA boots through Phases 1-4

James opens second terminal: Claude Code session
  → Claude Code polls GET /api/context every 30s
  → If blockers appear: Claude Code can diagnose and POST commands
  → If something needs human judgment: Claude Code alerts James

If Claude Code can't resolve:
  → James copies /api/context JSON
  → Pastes into cloud Claude conversation for advice
  → Applies fix via /send
```

### Implementation in http.go

```go
func (h *httpHandler) handleAPIContext(w http.ResponseWriter, r *http.Request) {
    ctx := APIContext{
        Timestamp:      time.Now(),
        System:         h.gatherSystemState(),
        RecentMessages: h.reins.Recent(20),
        PinnedMessages: h.reins.Pinned(),
        PendingIssues:  h.gatherPendingIssues(),
        AvailableCommands: h.commandInfo(),
        AwaitingInput:  h.reins.IsBlocked(),
        AwaitingConfirm: h.reins.PendingConfirm(),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(ctx)
}
```

Register the route alongside existing routes:

```go
mux.HandleFunc("/api/context", h.handleAPIContext)
```

### Security Note

The `/api/context` endpoint exposes full system state. This is
intentional — it's the diagnostic surface for trusted assistants.
It's protected by the same localhost/Tailscale boundary as all other
reins endpoints. No additional auth needed for Crawl.

When BRIDGE arrives (Step 9), `/api/context` becomes a BRIDGE-aware
endpoint with PersonaScope filtering — different assistants see
different subsets of the system state based on their domain access.

---

## What NOT to Build Yet

- Hypergraph, memory, sleep, BRIDGE, FATHOM, seed loading
- Phase 2 Hello (requires seed protocol)
- Judge collective communication
- Authentication on reins
- Inference backend management (llama.cpp lifecycle)
- The 10 Hardware Truth Experiments (separate prompt, separate binary)

Build the orchestrator, the command vocabulary, the reins refinements,
and the Possum State infrastructure. These are the foundation that
everything else plugs into.

---

## Verification

When complete, this should work:

```bash
# Terminal 1: start BMA
$ bma run
[stress] bus initialized → /data/stress.log
[reins]  Hello. BMA instance online. Hardware: AMD RX 9070 XT, 16304MB VRAM...
[reins]  listening on :8080
[probe]  running...
[reins]  Probe complete. 0 blockers, 0 warnings. GPU: AMD Radeon RX 9070 XT...
[auto]   10Hz loop started
[reins]  Autonomic protection active.

# Terminal 2: interact via web console
# Open browser to localhost:8080
# See message log, type "s" → autocomplete shows "status" → Enter
# Or type "/help" → see command list

# Terminal 2 (alternative): interact via curl
$ curl localhost:8080/messages | jq '.[0].text'
"Hello. BMA instance online. Hardware: ..."

$ curl -X POST localhost:8080/send -d '{"text":"status"}'
$ curl localhost:8080/plain
[...] >> BMA Status — ...
        Uptime: 45s
        Auto: neutral
        VRAM: 1552/16304 MB (9.5%)
        ...

$ curl -X POST localhost:8080/send -d '{"text":"possum"}'
$ curl localhost:8080/plain
[...] << possum
[...] >> Possum State entered by beekeeper command.

$ curl -X POST localhost:8080/send -d '{"text":"wake"}'
[...] >> Possum State exited by beekeeper command. Resuming.

# Terminal 3: assisting Claude instance reads full state
$ curl -s localhost:8080/api/context | jq .system.auto_state
"neutral"
$ curl -s localhost:8080/api/context | jq .system.blockers
[]
$ curl -s localhost:8080/api/context | jq '.recent_messages | length'
6

# Fuzzy matching test
$ curl -X POST localhost:8080/send -d '{"text":"stauts"}'
$ curl localhost:8080/plain
[...] << stauts
[...] >> → status
[...] >> BMA Status — ...

# Destructive command confirmation
$ curl -X POST localhost:8080/send -d '{"text":"shtdown"}'
$ curl localhost:8080/plain
[...] << shtdown
[...] >> Did you mean 'shutdown'? Type 'yes' to confirm.

$ curl -X POST localhost:8080/send -d '{"text":"no"}'
[...] >> Cancelled 'shutdown'.

# Actual shutdown
$ curl -X POST localhost:8080/send -d '{"text":"shutdown"}'
$ curl -X POST localhost:8080/send -d '{"text":"yes"}'
# Terminal 1: graceful shutdown sequence
[reins]  Shutting down. Reason: beekeeper command.
[auto]   stopped
[reins]  server closed
[stress] bus closed
```

If that interaction works end-to-end, the foundation is solid.
Everything else — hypergraph, memory, sleep, BRIDGE, seeds — plugs
into this boot sequence as new phases.
