#!/usr/bin/env bash
# R4 runner wrapper (mdcu). Run from emulator/ . Two steps, in this order:
#   scripts/run-r4.sh probe <probe-out.jsonl>
#   scripts/run-r4.sh grid  <probe-in.jsonl> <raw-out.jsonl>
# Refuses to start if deming's pause file exists. Records loadavg at start and end
# to a sidecar <out>.meta.txt (the JSONL itself is written only by the driver).
# Needs a prebuilt ./bin/mdcu-r4 (go build -o bin/mdcu-r4 ./cmd/mdcu-r4).
set -u
PAUSE="$HOME/.federation-watcher/box-pause/mdcu"
BIN=./bin/mdcu-r4
MODE="${1:-}"
if [ ! -x "$BIN" ]; then echo "missing $BIN (build it first)" >&2; exit 9; fi
if [ -e "$PAUSE" ]; then echo "PAUSED: $PAUSE exists" >&2; exit 4; fi

case "$MODE" in
  probe)
    PROBE_OUT="${2:?usage: run-r4.sh probe <probe-out.jsonl>}"
    META="$PROBE_OUT.meta.txt"
    ARGS=(-run-probe -out "$PROBE_OUT")
    ;;
  grid)
    PROBE_IN="${2:?usage: run-r4.sh grid <probe-in.jsonl> <raw-out.jsonl>}"
    RAW_OUT="${3:?usage: run-r4.sh grid <probe-in.jsonl> <raw-out.jsonl>}"
    META="$RAW_OUT.meta.txt"
    ARGS=(-run-grid -probe-file "$PROBE_IN" -out "$RAW_OUT")
    ;;
  *) echo "usage: run-r4.sh probe <out> | grid <probe-in> <raw-out>" >&2; exit 2 ;;
esac

{
  echo "# R4 $MODE meta"
  echo "# started: $(date -Is)"
  echo "# load_start: $(cut -d' ' -f1-3 /proc/loadavg)"
  echo "# cpu: $(grep -m1 'model name' /proc/cpuinfo | cut -d: -f2- | sed 's/^ //')"
  echo "# go: $(go version)"
  echo "# binary_sha256: $(sha256sum "$BIN" | cut -d' ' -f1)"
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
