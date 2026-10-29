#!/usr/bin/env bash
# mdcu-kz runner wrapper. Run from emulator/ . Two steps, in this order, each only after
# the run go (amend-1 signed, harness reviewed and pushed, deming box-safety, arch go):
#   scripts/run-kz.sh probe <probe-out.jsonl>
#   scripts/run-kz.sh grid  <probe-in.jsonl> <raw-out.jsonl> <amend-1.md>
# Refuses to start if deming's pause file exists or if the unit tests (go test -run TestKZ)
# fail. Records loadavg and the binary and amend sha256 to a sidecar <out>.meta.txt (the
# JSONL itself is written only by the driver). Compute is pinned: GOMAXPROCS=1, nice 19,
# core 7. Needs a prebuilt ./bin/mdcu-kz (go build -o bin/mdcu-kz ./cmd/mdcu-kz).
# After the grid: post the raw sha256 BEFORE any analysis, then
#   bin/mdcu-kz -analyze <raw-out.jsonl>
set -u
PAUSE="$HOME/.federation-watcher/box-pause/mdcu"
BIN=./bin/mdcu-kz
MODE="${1:-}"
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
  *) echo "usage: run-kz.sh probe <out> | grid <probe-in> <raw-out> <amend-1.md>" >&2; exit 2 ;;
esac

if ! go test ./mdcu -run TestKZ >/dev/null 2>&1; then echo "unit tests (go test ./mdcu -run TestKZ) FAIL: no run" >&2; exit 10; fi

{
  echo "# mdcu-kz $MODE meta"
  echo "# started: $(date -Is)"
  echo "# load_start: $(cut -d' ' -f1-3 /proc/loadavg)"
  echo "# cpu: $(grep -m1 'model name' /proc/cpuinfo | cut -d: -f2- | sed 's/^ //')"
  echo "# go: $(go version)"
  echo "# binary_sha256: $(sha256sum "$BIN" | cut -d' ' -f1)"
  [ "$MODE" = grid ] && echo "# amend_sha256: $AMEND_SHA"
  echo "# unit tests: go test ./mdcu -run TestKZ passed"
} > "$META"

GOMAXPROCS=1 nice -n 19 taskset -c 7 "$BIN" "${ARGS[@]}"
RC=$?

{
  echo "# exit: $RC"
  echo "# load_end: $(cut -d' ' -f1-3 /proc/loadavg)"
  echo "# finished: $(date -Is)"
} >> "$META"
cat "$META"
exit $RC
