# BMA Hardware Truth Experiments — Implementation Prompt

**For: CLI Opus (bma-systema repo)**
**Purpose: Write and run 10 hardware experiments, produce a structured report**
**Priority: Experiment 1 first, then 5, then the rest**

---

## Context

The BMA Red Team Review identified 10 experiments that must be run on Crawl
hardware before architectural decisions can be committed. Every number in
the current spec that isn't from the probe v3.0 report is an estimate.
These experiments replace estimates with measurements.

The most critical finding: the bilateral (two-context) architecture may or
may not be viable on 16GB VRAM depending on the ACTUAL KV cache size of a
GQA model on gfx1201. This is Experiment 1, and everything else depends on
its result.

**Hardware (confirmed by probe v3.0):**
- GPU: AMD RX 9070 XT (gfx1201), 16304 MB VRAM, ROCm 7.2.1
- CPU: FX-8350 Piledriver, 8 cores
- RAM: 31993 MB total, container limit 20GB
- Disk: Samsung 840 250GB SATA, 87GB free
- PCIe: 2.0 x16 (~8 GB/s)
- Container: Podman rootless

**Prerequisites:**
- llama.cpp built with ROCm/HIP support and installed (llama-server, llama-cli)
- A 7B-8B Q4_K_M GGUF model downloaded (e.g., Qwen2.5-7B-Instruct-Q4_K_M.gguf)
- rocm-smi available
- smartctl available (may need sudo)
- Beyond All Reason installed (for Experiment 5 — can be deferred if not available)
- curl, jq, bc available

---

## Deliverable

Write a single Go program: `cmd/bma-experiments/main.go`

The program:
1. Runs each experiment in sequence (or selectively via `--experiment N` flag)
2. Records all measurements with timestamps
3. Produces a structured JSON report: `experiments-report.json`
4. Produces a human-readable terminal summary (colored, like the probe)
5. Saves raw data in `experiments-report/raw/`

The report should follow the probe's format and be committable alongside it
as `doc/plan/crawl-experiments/`.

---

## Experiment Specifications

### Experiment 1: KV Cache Actual Size
**Priority: RUN FIRST — gates the bilateral architecture decision**

```
Procedure:
1. Record baseline VRAM (rocm-smi, no model loaded)
2. Start llama-server with target model, 8K context, full GPU offload
3. Wait for server ready (poll /health)
4. Record VRAM with model loaded but no context (VRAM_model)
5. Send a prompt that generates ~7000 tokens to fill the context
   - Use a simple generation: "Count from 1 to 10000, one number per line"
   - Or send a long prompt (~7K tokens of text)
6. Record VRAM with full KV cache (VRAM_full)
7. Calculate: KV_cache_size = VRAM_full - VRAM_model
8. Kill llama-server

Output fields:
{
  "experiment": 1,
  "name": "kv_cache_actual_size",
  "model": "<model filename>",
  "context_size": 8192,
  "vram_baseline_mb": N,
  "vram_model_only_mb": N,
  "vram_model_weights_mb": N,  // vram_model - vram_baseline
  "vram_full_context_mb": N,
  "kv_cache_size_mb": N,       // vram_full - vram_model
  "bilateral_viable_without_compression": true/false,  // kv_cache < 3500
  "bilateral_viable_with_turboq": true/false,           // kv_cache/6 * 2 + model fits
  "headroom_single_context_mb": N,  // 14800 - model_weights - kv_cache
  "headroom_bilateral_mb": N,       // 14800 - model_weights - 2*kv_cache
  "verdict": "BILATERAL_VIABLE | BILATERAL_NEEDS_COMPRESSION | UNILATERAL_ONLY"
}
```

### Experiment 2: PCIe Transfer Latency
**Priority: HIGH — gates swap strategy**

```
Procedure:
1. Start llama-server with model, 8K context
2. Fill context to ~7K tokens (same as Exp 1)
3. Time: POST /slots/0/save {"filename": "/tmp/bma_kv_test.bin"}
4. Record file size of /tmp/bma_kv_test.bin
5. Clear the slot: POST /slots/0/erase
6. Time: POST /slots/0/restore {"filename": "/tmp/bma_kv_test.bin"}
7. Repeat 5 times, record all timings
8. Clean up temp files

Output fields:
{
  "experiment": 2,
  "name": "pcie_transfer_latency",
  "kv_file_size_bytes": N,
  "save_times_ms": [N, N, N, N, N],
  "save_avg_ms": N,
  "restore_times_ms": [N, N, N, N, N],
  "restore_avg_ms": N,
  "effective_bandwidth_save_gbs": N,  // file_size / save_avg
  "effective_bandwidth_restore_gbs": N,
  "meets_100ms_target": false,
  "meets_500ms_target": true/false,
  "verdict": "SWAP_VIABLE | SWAP_MARGINAL | SWAP_TOO_SLOW"
}
```

### Experiment 3: KV Delta Size Across Turns
**Priority: MEDIUM — optimization target**

```
Procedure:
1. Start llama-server, fill context to ~4K tokens
2. Save KV cache: /slots/0/save → /tmp/kv_turn_0.bin
3. Send 10 conversation turns (short Q&A, ~100 tokens each)
4. Save KV cache: /slots/0/save → /tmp/kv_turn_10.bin
5. Compare files: size difference and byte-level diff
6. Also save after turns 1, 3, 5 for granular delta curve

Output fields:
{
  "experiment": 3,
  "name": "kv_delta_size",
  "kv_size_turn_0_bytes": N,
  "kv_size_turn_1_bytes": N,
  "kv_size_turn_3_bytes": N,
  "kv_size_turn_5_bytes": N,
  "kv_size_turn_10_bytes": N,
  "byte_diff_0_to_10": N,
  "delta_percent": N,  // byte_diff / total_size * 100
  "delta_sync_value": "HIGH | MEDIUM | LOW",
  "verdict": "DELTA_SYNC_WORTHWHILE | FULL_SWAP_SIMPLER"
}
```

### Experiment 4: Embedding Batch Throughput
**Priority: MEDIUM — calibrates sleep consolidation**

```
Procedure:
1. Start llama-server with model
2. Prepare 100 text chunks (~100 tokens each, varied content)
3. Record VRAM before batch
4. Time: single POST /embedding with all 100 chunks
5. Record VRAM during batch (poll rocm-smi)
6. Time: 100 individual POST /embedding calls sequentially
7. Calculate: speedup ratio, VRAM overhead, embeddings/second

Output fields:
{
  "experiment": 4,
  "name": "embedding_batch_throughput",
  "chunks": 100,
  "tokens_per_chunk": 100,
  "batch_time_ms": N,
  "sequential_time_ms": N,
  "speedup_ratio": N,
  "embeddings_per_second_batch": N,
  "embeddings_per_second_sequential": N,
  "vram_during_batch_mb": N,
  "vram_overhead_mb": N,
  "recommended_batch_size": N,
  "verdict": "BATCHING_CRITICAL | BATCHING_HELPFUL | BATCHING_MARGINAL"
}
```

### Experiment 5: Gaming Contention Profile
**Priority: HIGH — calibrates Possum State**

```
Procedure:
NOTE: This experiment requires manual coordination.
The program starts llama-server and begins sending periodic requests.
The operator manually starts BAR (Beyond All Reason) and plays for
at least 15 minutes, then exits the game.

1. Start llama-server with model
2. Record baseline inference latency (5 requests, no gaming)
3. Print: "START GAMING NOW. Press Enter when game is running."
4. Wait for operator input
5. Send inference requests every 30 seconds for 15 minutes
6. Record: latency, GPU temp, VRAM usage, GPU utilization for each
7. Print: "STOP GAMING NOW. Press Enter when game is closed."
8. Wait for operator input
9. Record recovery: 5 more requests post-gaming

If BAR is not available, use an alternative GPU stress test:
  rocm-smi --setperf high && glmark2 (or any GPU benchmark)

Output fields:
{
  "experiment": 5,
  "name": "gaming_contention_profile",
  "baseline_latency_ms": N,
  "gaming_samples": [
    {"time_s": N, "latency_ms": N, "gpu_temp_c": N, "vram_mb": N, "gpu_util_pct": N},
    ...
  ],
  "gaming_avg_latency_ms": N,
  "gaming_max_latency_ms": N,
  "gaming_min_latency_ms": N,
  "llama_crashed": false,
  "oom_occurred": false,
  "thermal_throttle_observed": false,
  "recovery_latency_ms": N,  // first post-gaming request
  "recovery_time_s": N,      // time to return to baseline latency
  "possum_trigger_recommended": true/false,
  "verdict": "GRACEFUL_DEGRADATION | POSSUM_REQUIRED | INCOMPATIBLE"
}
```

### Experiment 6: Shared-Trunk Sequence Branching
**Priority: HIGH if Experiment 1 shows KV < 3.5GB, otherwise SKIP**

```
Procedure:
1. Start llama-server with model, 8K context, 2 slots
2. Fill slot 0 with ~4K tokens of shared context
3. Record VRAM (VRAM_shared)
4. Use the llama.cpp sequence copy API to branch:
   POST /slots/0/save → then load into slot 1
   OR use the --slot-save-path mechanism
5. Generate different completions on each slot (divergent branches)
6. Record VRAM after branching (VRAM_branched)
7. Measure switch time between slots (alternate requests to slot 0 and 1)
8. Record: any errors, VRAM delta, switch latency

GATE: Only run if Experiment 1 shows headroom_bilateral_mb > 2000

Output fields:
{
  "experiment": 6,
  "name": "shared_trunk_branching",
  "skipped": false,
  "skip_reason": "",
  "shared_context_tokens": 4000,
  "vram_shared_mb": N,
  "vram_branched_mb": N,
  "vram_branch_delta_mb": N,
  "switch_times_ms": [N, N, N, N, N],
  "switch_avg_ms": N,
  "errors": [],
  "meets_100ms_target": true/false,
  "verdict": "SHARED_TRUNK_VIABLE | SHARED_TRUNK_UNSTABLE | SHARED_TRUNK_OOM"
}
```

### Experiment 7: Container GPU Overhead
**Priority: MEDIUM — validates Podman strategy**

```
Procedure:
1. Run a simplified version of Experiment 1 on BARE METAL
   (llama-server directly on host, not in container)
2. Record: VRAM usage, inference latency for 5 requests
3. Run the same test INSIDE the Podman container
   (llama-server inside container with --device /dev/kfd --device /dev/dri)
4. Compare all metrics

Output fields:
{
  "experiment": 7,
  "name": "container_gpu_overhead",
  "bare_metal": {
    "vram_model_mb": N,
    "vram_full_mb": N,
    "inference_latency_avg_ms": N
  },
  "container": {
    "vram_model_mb": N,
    "vram_full_mb": N,
    "inference_latency_avg_ms": N
  },
  "vram_overhead_mb": N,
  "vram_overhead_pct": N,
  "latency_overhead_ms": N,
  "latency_overhead_pct": N,
  "verdict": "NEGLIGIBLE | ACCEPTABLE | SIGNIFICANT"
}
```

### Experiment 8: Storage I/O Under Bind Mount
**Priority: MEDIUM — validates SATA strategy**

```
Procedure:
1. On host: dd write 100MB to /tmp, record speed
2. On host: dd read 100MB from /tmp, record speed
3. In container: dd write 100MB to /data (bind-mounted), record speed
4. In container: dd read 100MB from /data, record speed
5. On host: fio random 4K IOPS test on /tmp (if fio available)
6. In container: fio random 4K IOPS on /data (if fio available)

Output fields:
{
  "experiment": 8,
  "name": "storage_bind_mount_overhead",
  "host_sequential_write_mbs": N,
  "host_sequential_read_mbs": N,
  "container_sequential_write_mbs": N,
  "container_sequential_read_mbs": N,
  "write_overhead_pct": N,
  "read_overhead_pct": N,
  "host_random_4k_iops": N,       // null if fio unavailable
  "container_random_4k_iops": N,  // null if fio unavailable
  "verdict": "NEGLIGIBLE | ACCEPTABLE | SIGNIFICANT"
}
```

### Experiment 9: CURBy Reachability and Latency
**Priority: LOW — validates sleep randomness source**

```
Procedure:
1. Fetch https://random.colorado.edu/api/latest 10 times
2. Record response time, HTTP status, response body size
3. Parse random bits from response if available
4. Calculate: success rate, avg/min/max latency

Output fields:
{
  "experiment": 9,
  "name": "curby_reachability",
  "attempts": 10,
  "successes": N,
  "success_rate": N,
  "latencies_ms": [N, ...],
  "avg_latency_ms": N,
  "min_latency_ms": N,
  "max_latency_ms": N,
  "random_bits_received": N,
  "fallback_required": true/false,
  "verdict": "RELIABLE | INTERMITTENT | UNREACHABLE"
}
```

### Experiment 10: SMART Baseline
**Priority: MEDIUM — establishes write endurance budget**

```
Procedure:
1. Read SMART data from the boot/data drive
   (smartctl -a /dev/sda or appropriate device — may need sudo)
2. Extract: WearLevelingCount (or equivalent), TotalBytesWritten,
   ReallocatedSectors, PendingSectors, PowerOnHours
3. Calculate remaining TBW based on Samsung 840 specs (75 TBW typical)
4. Project MaxWritesPerCycle for 2-year survival at 2 cycles/day

NOTE: This experiment may require sudo. If run without sudo,
report what's available and flag what needs elevation.

Output fields:
{
  "experiment": 10,
  "name": "smart_baseline",
  "device": "/dev/sda",
  "needs_sudo": true/false,
  "smart_available": true/false,
  "wear_leveling_count": N,        // 0-100, null if unavailable
  "total_bytes_written_gb": N,     // null if unavailable
  "reallocated_sectors": N,
  "pending_sectors": N,
  "power_on_hours": N,
  "estimated_remaining_tbw_gb": N,
  "max_writes_per_cycle_gb": N,    // for 2-year, 2-cycles/day target
  "health_status": "HEALTHY | WORN | CRITICAL",
  "verdict": "ENDURANCE_OK | ENDURANCE_WATCH | REPLACE_SOON"
}
```

---

## Report Structure

The final `experiments-report.json` should look like:

```json
{
  "report": "bma-hardware-truth-experiments",
  "version": "1.0",
  "timestamp": "2026-04-06T...",
  "hostname": "pop-os",
  "probe_version": "3.0",
  "model_used": "<filename>",
  "experiments": [
    { ... experiment 1 ... },
    { ... experiment 2 ... },
    ...
  ],
  "summary": {
    "bilateral_decision": "SHARED_TRUNK | PCIE_SWAP | UNILATERAL",
    "bilateral_gate": "Experiment 1 KV cache = X MB",
    "possum_required": true/false,
    "container_overhead": "NEGLIGIBLE | ACCEPTABLE | SIGNIFICANT",
    "storage_health": "HEALTHY | WORN | CRITICAL",
    "curby_status": "RELIABLE | FALLBACK",
    "critical_findings": [
      "Finding 1...",
      "Finding 2...",
      ...
    ]
  },
  "next_steps": [
    "Step 1 based on results...",
    "Step 2...",
    ...
  ]
}
```

The terminal output should use the same colored format as the probe:
- ✓ green for passing/favorable results
- ✗ red for failures/blockers
- ! yellow for warnings/marginal results
- · blue for informational

---

## Implementation Notes

- The program should be runnable as `go run cmd/bma-experiments/main.go`
  or after building: `./bma-experiments`
- Flag `--experiment N` runs only experiment N
- Flag `--skip-interactive` skips Experiment 5 (gaming, requires manual input)
- Flag `--model PATH` specifies the GGUF model path
- Flag `--sudo` enables experiments that need root (Experiment 10)
- Each experiment should have a timeout (default 5 minutes, configurable)
- If llama-server is already running, detect and use it rather than starting a new instance
- If an experiment fails, record the error and continue to the next one
- The summary section should be generated AFTER all experiments complete,
  synthesizing the individual verdicts into architectural recommendations

---

## Gate Logic (in the summary)

```
IF exp1.kv_cache_size_mb < 3500 AND exp6.verdict == "SHARED_TRUNK_VIABLE":
    bilateral_decision = "SHARED_TRUNK"
ELIF exp1.kv_cache_size_mb < 3500 AND exp2.restore_avg_ms < 500:
    bilateral_decision = "PCIE_SWAP"
ELIF exp1.kv_cache_size_mb >= 3500 AND exp2.restore_avg_ms < 500:
    bilateral_decision = "PCIE_SWAP"  // but tight on VRAM
ELSE:
    bilateral_decision = "UNILATERAL"  // defer bilateral to Walk

IF exp5.gaming_max_latency_ms > 5000 OR exp5.llama_crashed:
    possum_required = true

IF exp7.latency_overhead_pct > 5:
    container_overhead = "SIGNIFICANT"
```

---

**Run Experiment 1 first. Its result determines whether Experiment 6 runs
and which bilateral strategy the summary recommends. The hardware dictates
the architecture.**
