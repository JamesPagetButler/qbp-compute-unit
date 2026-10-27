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
echo '{"type":"probe","seeds":[1,2],"steps":1000,"arm":"K-phi","per_run_s":[0.1,0.1],"mean_s":0.1,"tierc_seeds":200}' > bad1.jsonl
expect 8 "a probe on the wrong seeds is refused"          -- -run-grid -probe-file bad1.jsonl -out x.jsonl
echo '{"type":"probe","seeds":[9960,9961,9962,9963,9964,9965,9966,9967,9968,9969],"steps":1000,"arm":"K-phi","per_run_s":[0.9,0.9,0.9,0.9,0.9,0.9,0.9,0.9,0.9,0.9],"mean_s":0.9,"tierc_seeds":200}' > bad2.jsonl
expect 8 "a probe decision that contradicts its timings is refused" -- -run-grid -probe-file bad2.jsonl -out x.jsonl
expect 8 "a missing probe file is refused"                -- -run-grid -probe-file nope.jsonl -out x.jsonl
[ -e x.jsonl ] && { echo "FAIL: a refused run created its output file"; fail=1; }
exit $fail
