#!/usr/bin/env bash
# Guard tests for the R4 driver (frozen R4 prereg §2/§3). Run from emulator/ .
# Every case must be REFUSED before any dynamics run: the frozen seed ranges
# (grid 4000-4199, S-D 4200-4399, probe 9960-9969) are not touched, nothing is
# written except to a temp dir, and no grid/probe mode is ever started.
set -u
BIN="${BIN:-bin/mdcu-r4}"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
fail=0
expect() { # expect <exit code> <description> -- <args...>
  local want="$1" desc="$2"; shift 3
  "$BIN" "$@" >"$T/out" 2>"$T/err"; local got=$?
  if [ "$got" -ne "$want" ]; then echo "FAIL: $desc: exit $got, want $want"; sed 's/^/   /' "$T/err" | head -3; fail=1; else echo "ok:   $desc (exit $got)"; fi
}
cd "$T" || exit 1
B="$OLDPWD/$BIN"; BIN="$B"
expect 4 "smoke seeds inside the grid are refused"        -- -first-seed 4000 -seeds 2 -sd-seeds 0 -out x.jsonl
expect 4 "smoke seeds inside the S-D range are refused"   -- -first-seed 4198 -seeds 4 -sd-seeds 0 -out x.jsonl
expect 4 "smoke seeds inside the probe range are refused" -- -first-seed 9965 -seeds 2 -sd-seeds 0 -out x.jsonl
expect 4 "-run-grid with -run-probe is refused"           -- -run-grid -run-probe -out x.jsonl
expect 8 "-run-grid without a probe file is refused"      -- -run-grid -out x.jsonl
# Probe fixtures in the arch-ruled format (bridge seq 115): per-step-count timings, the
# predicted total wall, and the Tier C decision, all of which the driver recomputes.
SEEDS='[9960,9961,9962,9963,9964,9965,9966,9967,9968,9969]'
ten() { local v="$1" s="" i; for i in 1 2 3 4 5 6 7 8 9 10; do s="$s${s:+,}$v"; done; echo "[$s]"; }
probe() { # probe <t250> <t1000> <t4000> <predicted> <tierc> <seeds-json>
  printf '{"type":"probe","arm":"K-phi","seeds":%s,"per_run_s":{"250":%s,"1000":%s,"4000":%s},"mean_s":{"250":%s,"1000":%s,"4000":%s},"predicted_wall_s":%s,"tierc_seeds":%s}\n' \
    "$6" "$(ten "$1")" "$(ten "$2")" "$(ten "$3")" "$1" "$2" "$3" "$4" "$5"
}
probe 0.1 0.1 0.1 1 200 '[1,2]' > bad1.jsonl
expect 8 "a probe on the wrong seeds is refused"          -- -run-grid -probe-file bad1.jsonl -out x.jsonl
# 0.9 s per run at every step count: predicted 3600 s > 3240 s, so the rule gives 100, not 200.
probe 0.9 0.9 0.9 3600 200 "$SEEDS" > bad2.jsonl
expect 8 "a probe decision that contradicts its timings is refused" -- -run-grid -probe-file bad2.jsonl -out x.jsonl
# Costs 0.1/0.4/1.6 predict 1360 s; a recorded 999 s does not follow from the timings.
probe 0.1 0.4 1.6 999 200 "$SEEDS" > bad3.jsonl
expect 8 "a probe whose predicted wall contradicts its timings is refused" -- -run-grid -probe-file bad3.jsonl -out x.jsonl
expect 8 "a missing probe file is refused"                -- -run-grid -probe-file nope.jsonl -out x.jsonl
# A well-formed probe is accepted (dry run: nothing executes, no frozen seed is touched).
probe 0.1 0.4 1.6 1360 200 "$SEEDS" > good.jsonl
expect 0 "a consistent probe is accepted (dry run)"       -- -run-grid -dry-run -probe-file good.jsonl -out x.jsonl
[ -e x.jsonl ] && { echo "FAIL: a refused run created its output file"; fail=1; }
exit $fail
