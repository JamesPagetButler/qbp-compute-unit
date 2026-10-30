#!/usr/bin/env bash
# mdcu-kz runner wrapper. Run from emulator/ . Two steps, in this order, each only after
# the run go (amend-1 signed, harness reviewed and pushed, deming box-safety, arch go):
#   scripts/run-kz.sh probe <probe-out.jsonl>
#   scripts/run-kz.sh grid  <probe-in.jsonl> <raw-out.jsonl> <amend-1.md>
# qbp-kz-g2 round 1 (FROZEN-v1 §2 addition 1, §5 load discipline), one tau_Q per process,
# run sequentially, each only after the director's run go:
#   scripts/run-kz.sh 0b <tau> <raw-out.jsonl> <frozen.md>
#   (KZG2_RERUN_AMEND=<amend-N.md> selects the reserved re-run sub-block 5940-5979)
# Refuses to start if deming's pause file exists or if the unit tests (go test -run TestKZ)
# fail. Records loadavg and the binary and amend sha256 to a sidecar <out>.meta.txt (the
# JSONL itself is written only by the driver). Compute is pinned: GOMAXPROCS=1, nice 19,
# core 7. Needs a prebuilt ./bin/mdcu-kz (go build -o bin/mdcu-kz ./cmd/mdcu-kz).
# After the grid: post the raw sha256 BEFORE any analysis, then
#   bin/mdcu-kz -analyze <raw-out.jsonl>
# After the four 0b rungs: post the four raw sha256s TOGETHER before any gap is read, then
#   bin/mdcu-kz -analyze-0b <tau1600.jsonl> <tau3200.jsonl> <tau6400.jsonl> <tau12800.jsonl>
# Exit codes of this wrapper: 4 pause pre-check · 8 missing input · 9 no binary · 10 unit
# tests failed · 11 quiet-load wait timed out · otherwise the driver's exit code.
set -u
PAUSE="$HOME/.federation-watcher/box-pause/mdcu"
BIN=./bin/mdcu-kz
MODE="${1:-}"
# FROZEN-v1 §2 "T_c is reused, not re-derived": the full-precision literal typed into -tc.
KZG2_TC="1.0666262219986817"
if [ ! -x "$BIN" ]; then echo "missing $BIN (build it first)" >&2; exit 9; fi
if [ -e "$PAUSE" ]; then echo "PAUSED: $PAUSE exists" >&2; exit 4; fi

case "$MODE" in
  probe)
    PROBE_OUT="${2:?usage: run-kz.sh probe <probe-out.jsonl>}"
    META="$PROBE_OUT.meta.txt"
    ARGS=(-run-probe -out "$PROBE_OUT")
    ;;
  grid)
    PROBE_IN="${2:?usage: run-kz.sh grid <probe-in.jsonl> <raw-out.jsonl> <amend-1.md>}"
    RAW_OUT="${3:?usage: run-kz.sh grid <probe-in.jsonl> <raw-out.jsonl> <amend-1.md>}"
    AMEND="${4:?usage: run-kz.sh grid <probe-in.jsonl> <raw-out.jsonl> <amend-1.md>}"
    [ -f "$AMEND" ] || { echo "missing amendment file $AMEND" >&2; exit 8; }
    AMEND_SHA="$(sha256sum "$AMEND" | cut -d' ' -f1)"
    META="$RAW_OUT.meta.txt"
    ARGS=(-run-grid -probe-file "$PROBE_IN" -amend-sha "$AMEND_SHA" -out "$RAW_OUT")
    ;;
  0b)
    TAU="${2:?usage: run-kz.sh 0b <tau> <raw-out.jsonl> <frozen.md>}"
    RAW_OUT="${3:?usage: run-kz.sh 0b <tau> <raw-out.jsonl> <frozen.md>}"
    FROZEN="${4:?usage: run-kz.sh 0b <tau> <raw-out.jsonl> <frozen.md>}"
    [ -f "$FROZEN" ] || { echo "missing frozen pre-registration $FROZEN" >&2; exit 8; }
    FROZEN_SHA="$(sha256sum "$FROZEN" | cut -d' ' -f1)"
    META="$RAW_OUT.meta.txt"
    ARGS=(-run-0b -tau "$TAU" -tc "$KZG2_TC" -prereg-sha "$FROZEN_SHA" -out "$RAW_OUT")
    RERUN_SHA=""
    if [ -n "${KZG2_RERUN_AMEND:-}" ]; then
      [ -f "$KZG2_RERUN_AMEND" ] || { echo "missing re-run amendment $KZG2_RERUN_AMEND" >&2; exit 8; }
      RERUN_SHA="$(sha256sum "$KZG2_RERUN_AMEND" | cut -d' ' -f1)"
      ARGS+=(-rerun-amend-sha "$RERUN_SHA")
    fi
    ;;
  *) echo "usage: run-kz.sh probe <out> | grid <probe-in> <raw-out> <amend-1.md> | 0b <tau> <raw-out> <frozen.md>" >&2; exit 2 ;;
esac

# unit tests run BEFORE the quiet-load wait (FROZEN-v1 §5: round 1's pre-step test raised load_start)
if ! go test ./mdcu -run TestKZ >/dev/null 2>&1; then echo "unit tests (go test ./mdcu -run TestKZ) FAIL: no run" >&2; exit 10; fi

if [ "$MODE" = 0b ]; then
  # FROZEN-v1 §5 load discipline: 1-min loadavg <= 1.0 on three consecutive 20 s samples
  LOAD_CEIL=1.0 MAX_WAIT_S="${MAX_WAIT_S:-14400}" bash "$(dirname "$0")/wait-quiet.sh" || { echo "quiet-load wait timed out: no run" >&2; exit 11; }
  if [ -e "$PAUSE" ]; then echo "PAUSED: $PAUSE exists (after the quiet wait)" >&2; exit 4; fi
fi

START_S=$(date +%s)
{
  echo "# mdcu-kz $MODE meta"
  echo "# started: $(date -Is)"
  echo "# load_start: $(cut -d' ' -f1-3 /proc/loadavg)"
  echo "# cpu: $(grep -m1 'model name' /proc/cpuinfo | cut -d: -f2- | sed 's/^ //')"
  echo "# go: $(go version)"
  echo "# binary_sha256: $(sha256sum "$BIN" | cut -d' ' -f1)"
  echo "# code_sha: $(git rev-parse HEAD 2>/dev/null || echo unknown)  dirty_files: $(git status --porcelain 2>/dev/null | wc -l)"
  [ "$MODE" = grid ] && echo "# amend_sha256: $AMEND_SHA"
  if [ "$MODE" = 0b ]; then
    echo "# tau_Q: $TAU"
    echo "# tc: $KZG2_TC"
    echo "# frozen_file: $FROZEN"
    echo "# prereg_sha256: $FROZEN_SHA"
    echo "# rerun_amend_sha256: ${RERUN_SHA:-none}"
    echo "# pause_file: $PAUSE (absent at start)"
    echo "# pinned: GOMAXPROCS=1 nice -n 19 taskset -c 7"
  fi
  echo "# unit tests: go test ./mdcu -run TestKZ passed"
} > "$META"

GOMAXPROCS=1 nice -n 19 taskset -c 7 "$BIN" "${ARGS[@]}"
RC=$?

{
  echo "# exit: $RC"
  echo "# wall_s: $(( $(date +%s) - START_S ))"
  echo "# load_end: $(cut -d' ' -f1-3 /proc/loadavg)"
  echo "# pause_file_present_at_end: $([ -e "$PAUSE" ] && echo yes || echo no)"
  echo "# finished: $(date -Is)"
  if [ "$MODE" = 0b ] && [ -f "$RAW_OUT" ]; then
    (cd "$(dirname "$RAW_OUT")" && sha256sum "$(basename "$RAW_OUT")" > "$(basename "$RAW_OUT").sha256")
    echo "# raw_sha256: $(cut -d' ' -f1 "$RAW_OUT.sha256")  (written to $RAW_OUT.sha256)"
  fi
} >> "$META"
cat "$META"
exit $RC
