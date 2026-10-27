#!/usr/bin/env bash
# R3-Q2 host-ns benchmark runner (mdcu). Run from emulator/ .
# Refuses to start if deming's pause file exists. Records loadavg at start and end.
set -u
PAUSE="$HOME/.federation-watcher/box-pause/mdcu"
OUT="${1:-../doc/spikes/raw/mdcu-r3q2-hostns-2026-10-26.txt}"
if [ -e "$PAUSE" ]; then echo "PAUSED: $PAUSE exists" >&2; exit 4; fi
{
  echo "# R3-Q2 host-ns raw output"
  echo "# started: $(date -Is)"
  echo "# load_start: $(cut -d' ' -f1-3 /proc/loadavg)"
  echo "# cpu: $(grep -m1 'model name' /proc/cpuinfo | cut -d: -f2- | sed 's/^ //')"
  echo "# go: $(go version)"
  GOMAXPROCS=1 nice -n 19 taskset -c 7 timeout 600 ./mdcu.test -test.run '^$' -test.bench 'BenchmarkR3Q2' -test.count 10 -test.benchtime 200ms
  echo "# exit: $?"
  echo "# load_end: $(cut -d' ' -f1-3 /proc/loadavg)"
  echo "# finished: $(date -Is)"
} > "$OUT" 2>&1
tail -n 4 "$OUT"
