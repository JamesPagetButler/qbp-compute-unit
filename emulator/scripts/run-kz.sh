#!/usr/bin/env bash
# mdcu-kz runner wrapper. Run from emulator/ . Two steps, in this order, each only after
# the run go (amend-1 signed, harness reviewed and pushed, deming box-safety, arch go):
#   scripts/run-kz.sh probe <probe-out.jsonl>
#   scripts/run-kz.sh grid  <probe-in.jsonl> <raw-out.jsonl> <amend-1.md>
# qbp-kz-g2 round 1 (FROZEN-v1 §2 addition 1, §5 load discipline), one tau_Q per process,
# run sequentially, each only after the director's run go:
#   scripts/run-kz.sh 0b <tau> <raw-out.jsonl> <frozen.md>
#   (KZG2_RERUN_AMEND=<amend-N.md> selects the reserved re-run sub-block 5940-5979)
# qbp-kz-g2 round 3 (r3 FROZEN-v1 §2 row 7, §5; sha256 132f6a42…1a623), one (L, t) point per
# process, 27 processes run one at a time (L = 8 → 12 → 16, t ascending; a sequential shell
# loop over the pairs is allowed; never a parallel launch), each only after the director's go:
#   scripts/run-kz.sh eq <L> <t> <raw-out.jsonl> <frozen.md>
#   (KZG2_RERUN_AMEND=<amend-N.md> selects the reserved re-run sub-block 6120-6239)
# Refuses to start if deming's pause file exists or if the unit tests (go test -run TestKZ)
# fail. Records loadavg and the binary and amend sha256 to a sidecar <out>.meta.txt (the
# JSONL itself is written only by the driver). Compute is pinned: GOMAXPROCS=1, nice 19,
# core 7. Needs a prebuilt ./bin/mdcu-kz (go build -o bin/mdcu-kz ./cmd/mdcu-kz).
# Box lease (0b and eq; deming's contract, mdcu-spike seq 368): after the go-test and the
# quiet-load wait and immediately before the pinned driver run, flock -n on
# ~/.federation-watcher/state/box-lease.lock; BUSY → back to the quiet-load loop, never start;
# on acquire one holder line in ~/.federation-watcher/state/box-lease.holder, removed before
# the release; the driver inherits the lease fd so the lease lasts exactly as long as the
# driver runs (release implicit on its exit; the wrapper then closes the fd); every subshell
# the wrapper opens closes the fd. The box-pause file overrides everything.
# After the grid: post the raw sha256 BEFORE any analysis, then
#   bin/mdcu-kz -analyze <raw-out.jsonl>
# After the four 0b rungs: post the four raw sha256s TOGETHER before any gap is read, then
#   bin/mdcu-kz -analyze-0b <tau1600.jsonl> <tau3200.jsonl> <tau6400.jsonl> <tau12800.jsonl>
# After the 27 eq points: post the 27 raw sha256s TOGETHER before any histogram is read, then
#   bin/mdcu-kz -analyze-eq <27 raw files> [-plot-out plot.json]
# Exit codes of this wrapper: 4 pause pre-check · 8 missing input · 9 no binary · 10 unit
# tests failed · 11 quiet-load / lease wait timed out · otherwise the driver's exit code.
# (r3 §5: wrapper 4/8/9/10/11, a BUSY lease, and driver 4/8 or a pause at start are no-runs,
# not kills; only an abort after the first plane-step is a kill.)
set -u
PAUSE="$HOME/.federation-watcher/box-pause/mdcu"
LEASE_LOCK="$HOME/.federation-watcher/state/box-lease.lock"
LEASE_HOLDER="$HOME/.federation-watcher/state/box-lease.holder"
LEASE_SEAT="${KZ_LEASE_SEAT:-qbp-cu-implementor}"
LEASE_FD=""
LEASE_PURPOSE=""
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
    LEASE_PURPOSE="qbp-kz-g2-r1:0b:tau$TAU"
    RERUN_SHA=""
    if [ -n "${KZG2_RERUN_AMEND:-}" ]; then
      [ -f "$KZG2_RERUN_AMEND" ] || { echo "missing re-run amendment $KZG2_RERUN_AMEND" >&2; exit 8; }
      RERUN_SHA="$(sha256sum "$KZG2_RERUN_AMEND" | cut -d' ' -f1)"
      ARGS+=(-rerun-amend-sha "$RERUN_SHA")
    fi
    ;;
  eq)
    L="${2:?usage: run-kz.sh eq <L> <t> <raw-out.jsonl> <frozen.md>}"
    TIDX="${3:?usage: run-kz.sh eq <L> <t> <raw-out.jsonl> <frozen.md>}"
    RAW_OUT="${4:?usage: run-kz.sh eq <L> <t> <raw-out.jsonl> <frozen.md>}"
    FROZEN="${5:?usage: run-kz.sh eq <L> <t> <raw-out.jsonl> <frozen.md>}"
    [ -f "$FROZEN" ] || { echo "missing frozen pre-registration $FROZEN" >&2; exit 8; }
    FROZEN_SHA="$(sha256sum "$FROZEN" | cut -d' ' -f1)"
    META="$RAW_OUT.meta.txt"
    STDOUT_FILE="$RAW_OUT.stdout.txt"
    # r3 §5 output cap 10 MB (driver default 50); no -tc; seeds derived from (L, t) in the driver
    ARGS=(-run-eq -L "$L" -t "$TIDX" -prereg-sha "$FROZEN_SHA" -max-out-mb 10 -out "$RAW_OUT")
    LEASE_PURPOSE="qbp-kz-g2-r3:eq:L$L:t$TIDX"
    RERUN_SHA=""
    if [ -n "${KZG2_RERUN_AMEND:-}" ]; then
      [ -f "$KZG2_RERUN_AMEND" ] || { echo "missing re-run amendment $KZG2_RERUN_AMEND" >&2; exit 8; }
      RERUN_SHA="$(sha256sum "$KZG2_RERUN_AMEND" | cut -d' ' -f1)"
      ARGS+=(-rerun-amend-sha "$RERUN_SHA")
    fi
    ;;
  *) echo "usage: run-kz.sh probe <out> | grid <probe-in> <raw-out> <amend-1.md> | 0b <tau> <raw-out> <frozen.md> | eq <L> <t> <raw-out> <frozen.md>" >&2; exit 2 ;;
esac

# unit tests run BEFORE the quiet-load wait (FROZEN-v1 §5: round 1's pre-step test raised load_start)
if ! go test ./mdcu -run TestKZ >/dev/null 2>&1; then echo "unit tests (go test ./mdcu -run TestKZ) FAIL: no run" >&2; exit 10; fi

if [ "$MODE" = 0b ] || [ "$MODE" = eq ]; then
  # FROZEN-v1 §5 load discipline: 1-min loadavg <= 1.0 on three consecutive 20 s samples, then
  # the box lease; BUSY -> back to the quiet-load loop (never start); the whole wait is bounded
  # by MAX_WAIT_S (default 4 h) -> exit 11 (a no-run).
  DEADLINE=$(( $(date +%s) + ${MAX_WAIT_S:-14400} ))
  while :; do
    REMAIN=$(( DEADLINE - $(date +%s) ))
    [ "$REMAIN" -gt 0 ] || { echo "quiet-load / lease wait timed out: no run" >&2; exit 11; }
    LOAD_CEIL=1.0 MAX_WAIT_S="$REMAIN" bash "$(dirname "$0")/wait-quiet.sh" || { echo "quiet-load wait timed out: no run" >&2; exit 11; }
    if [ -e "$PAUSE" ]; then echo "PAUSED: $PAUSE exists (after the quiet wait)" >&2; exit 4; fi
    mkdir -p "$(dirname "$LEASE_LOCK")"
    exec {LEASE_FD}>"$LEASE_LOCK"
    if flock -n "$LEASE_FD"; then break; fi
    exec {LEASE_FD}>&-
    LEASE_FD=""
    echo "box lease BUSY ($(cat "$LEASE_HOLDER" 2>/dev/null || echo 'holder unknown')): back to the quiet-load loop at $(date -Is)" >&2
    sleep 60
  done
  echo "holder=$LEASE_SEAT purpose=$LEASE_PURPOSE pid=$$ since=$(date -Is)" > "$LEASE_HOLDER"
  # the holder line is removed on every exit path; the lock itself is released when the last
  # holder of the fd (the wrapper, or the driver it hands the fd to) exits
  trap 'rm -f "$LEASE_HOLDER"' EXIT
fi

# close_lease_fd: every subshell the wrapper opens drops the lease fd (it must not outlive the run)
close_lease_fd() { if [ -n "$LEASE_FD" ]; then exec {LEASE_FD}>&-; fi; }

START_S=$(date +%s)
(
  close_lease_fd
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
  fi
  if [ "$MODE" = eq ]; then
    echo "# L: $L"
    echo "# t_index: $TIDX"
    echo "# max_out_mb: 10"
  fi
  if [ "$MODE" = 0b ] || [ "$MODE" = eq ]; then
    echo "# frozen_file: $FROZEN"
    echo "# prereg_sha256: $FROZEN_SHA"
    echo "# rerun_amend_sha256: ${RERUN_SHA:-none}"
    echo "# pause_file: $PAUSE (absent at start)"
    echo "# box_lease: $LEASE_LOCK acquired; holder=$LEASE_SEAT purpose=$LEASE_PURPOSE pid=$$"
    echo "# pinned: GOMAXPROCS=1 nice -n 19 taskset -c 7"
  fi
  echo "# unit tests: go test ./mdcu -run TestKZ passed"
) > "$META"

if [ "$MODE" = eq ]; then
  # the driver's terminal output goes to a file first (plan + the blind end line + the raw sha)
  # and is shown after the run; the driver inherits the lease fd (see the header)
  GOMAXPROCS=1 nice -n 19 taskset -c 7 "$BIN" "${ARGS[@]}" > "$STDOUT_FILE"
  RC=$?
  cat "$STDOUT_FILE"
else
  GOMAXPROCS=1 nice -n 19 taskset -c 7 "$BIN" "${ARGS[@]}"
  RC=$?
fi

# release: holder line first, then the fd (the lock drops with the last fd holder)
if [ -n "$LEASE_FD" ]; then rm -f "$LEASE_HOLDER"; exec {LEASE_FD}>&-; LEASE_FD=""; fi

(
  echo "# exit: $RC"
  echo "# wall_s: $(( $(date +%s) - START_S ))"
  echo "# load_end: $(cut -d' ' -f1-3 /proc/loadavg)"
  echo "# pause_file_present_at_end: $([ -e "$PAUSE" ] && echo yes || echo no)"
  echo "# finished: $(date -Is)"
  if { [ "$MODE" = 0b ] || [ "$MODE" = eq ]; } && [ -f "$RAW_OUT" ]; then
    (cd "$(dirname "$RAW_OUT")" && sha256sum "$(basename "$RAW_OUT")" > "$(basename "$RAW_OUT").sha256")
    echo "# raw_sha256: $(cut -d' ' -f1 "$RAW_OUT.sha256")  (written to $RAW_OUT.sha256)"
  fi
  if [ "$MODE" = eq ] && [ -f "$STDOUT_FILE" ]; then
    # §7.1: the driver's end line carries wall, steps, ms/step, peak Sys and the exit code only
    echo "# driver_end: $(grep -m1 -E '^eq L=[0-9]+ t=[0-9]+ (done|ABORT):' "$STDOUT_FILE" || echo 'none')"
  fi
) >> "$META"
cat "$META"
exit $RC
