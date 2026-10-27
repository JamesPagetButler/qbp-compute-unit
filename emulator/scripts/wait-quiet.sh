#!/usr/bin/env bash
# Quiet-window waiter for R3-Q2 (frozen prereg 3.3: 1-min loadavg <= 1.0 at start).
# Zero LLM tokens while waiting; exits (waking the session) when the load has been
# <= 1.0 for 3 consecutive 20 s samples, or after MAX_WAIT_S (exit 3).
# Usage (from ~/Documents/QBP-CU-mdcu/emulator): bash scripts/wait-quiet.sh
set -u
MAX_WAIT_S="${MAX_WAIT_S:-14400}"
CEIL="${LOAD_CEIL:-1.0}"
start=$(date +%s); ok=0
while :; do
  l1=$(cut -d' ' -f1 /proc/loadavg)
  if awk -v a="$l1" -v c="$CEIL" 'BEGIN{exit !(a<=c)}'; then ok=$((ok+1)); else ok=0; fi
  if [ "$ok" -ge 3 ]; then echo "QUIET load1=$l1 at $(date -Is)"; exit 0; fi
  if [ $(( $(date +%s) - start )) -ge "$MAX_WAIT_S" ]; then echo "TIMEOUT load1=$l1 at $(date -Is)"; exit 3; fi
  sleep 20
done
