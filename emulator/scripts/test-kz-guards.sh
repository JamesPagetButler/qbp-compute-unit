#!/usr/bin/env bash
# Guard tests for the mdcu-kz driver (frozen mdcu-kz prereg + amend-1). Run from emulator/ .
# Every case must be REFUSED before any dynamics run: no frozen kz seed (main 5000-5199,
# null 5200-5399, reserved 5400-5599, Stage 0 5600-5629, saturation 5630-5649, probe
# 9930-9949) and no seed used by an earlier round is touched, nothing is written except
# to a temp dir, and no probe or grid mode is ever started.
#
# Safety net: every -run-grid case passes a pause file that exists. If a guard were ever
# to accept a bad input, the run would stop at its "before start" check with exit 3
# instead of touching a seed, and the case would report FAIL (exit 3, want 8).
set -u
BIN="${BIN:-bin/mdcu-kz}"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
fail=0
touch "$T/pause"
expect() { # expect <exit code> <description> -- <args...>
  local want="$1" desc="$2"; shift 3
  "$BIN" "$@" >"$T/out" 2>"$T/err"; local got=$?
  if [ "$got" -ne "$want" ]; then echo "FAIL: $desc: exit $got, want $want"; sed 's/^/   /' "$T/err" | head -3; fail=1; else echo "ok:   $desc (exit $got)"; fi
}
cd "$T" || exit 1
B="$OLDPWD/$BIN"; BIN="$B"
SHA=0000000000000000000000000000000000000000000000000000000000000000
P="-pause-file $T/pause"

# seed and mode guards (smoke mode: each refused before anything runs)
expect 4 "smoke seeds inside the main range are refused"           -- -first-seed 5000 -out x.jsonl
expect 4 "smoke seeds straddling main and null are refused"        -- -first-seed 5190 -R 8 -out x.jsonl
expect 4 "smoke seeds inside the null range are refused"           -- -first-seed 5300 -out x.jsonl
expect 4 "smoke seeds inside the reserved split-G2 range are refused" -- -first-seed 5400 -out x.jsonl
expect 4 "smoke seeds inside Stage 0 are refused"                  -- -first-seed 5600 -out x.jsonl
expect 4 "smoke seeds inside the saturation range are refused"     -- -first-seed 5630 -out x.jsonl
expect 4 "smoke seeds inside the run tilt block are refused"       -- -first-seed 5700 -out x.jsonl
expect 4 "smoke seeds inside the calibration block are refused"    -- -first-seed 5860 -out x.jsonl
expect 4 "smoke seeds straddling into the tilt block are refused"  -- -first-seed 5640 -out x.jsonl
expect 4 "smoke seeds inside the probe range are refused"          -- -first-seed 9930 -out x.jsonl
expect 4 "smoke seeds used by R3 are refused"                      -- -first-seed 3000 -out x.jsonl
expect 4 "smoke seeds used by R4 are refused"                      -- -first-seed 4000 -out x.jsonl
expect 4 "smoke seeds in the 9100-9929 used block are refused"     -- -first-seed 9900 -out x.jsonl
expect 4 "smoke seeds in the 9950-9979 used block are refused"     -- -first-seed 9950 -out x.jsonl
expect 4 "-run-grid with -run-probe is refused"                    -- -run-grid -run-probe -out x.jsonl
expect 4 "-analyze with -run-grid is refused"                      -- -analyze x.jsonl -run-grid
expect 4 "-analyze with -run-probe is refused"                     -- -analyze x.jsonl -run-probe
expect 4 "-run-calibration with -run-grid is refused"              -- -run-calibration -run-grid -cal-ensemble aligned -out x.jsonl
expect 4 "-run-calibration with -run-probe is refused"             -- -run-calibration -run-probe -cal-ensemble aligned -out x.jsonl
expect 4 "-analyze with -run-calibration is refused"               -- -analyze x.jsonl -run-calibration
expect 8 "-run-calibration without an ensemble is refused"         -- -run-calibration -out x.jsonl
expect 8 "-run-calibration with an unknown ensemble is refused"    -- -run-calibration -cal-ensemble gaussian -out x.jsonl
expect 3 "-run-calibration stops at the pause file, writing no result" -- -run-calibration -cal-ensemble aligned $P -out cal-paused.jsonl
expect 3 "-run-calibration (smooth) stops at the pause file, writing no result" -- -run-calibration -cal-ensemble smooth $P -out cal-paused2.jsonl
grep -q '"kind":"calibration"' cal-paused.jsonl && { echo "FAIL: a paused calibration wrote a result"; fail=1; }
grep -q '"kind":"calibration"' cal-paused2.jsonl 2>/dev/null && { echo "FAIL: a paused smooth calibration wrote a result"; fail=1; }
rm -f cal-paused.jsonl cal-paused2.jsonl

# grid guards (all with the safety-net pause file)
expect 8 "-run-grid without a probe file is refused"               -- -run-grid $P -amend-sha $SHA -out x.jsonl
expect 8 "-run-grid without -amend-sha is refused"                 -- -run-grid $P -probe-file nope.jsonl -out x.jsonl
expect 8 "-run-grid with a short -amend-sha is refused"            -- -run-grid $P -probe-file nope.jsonl -amend-sha abc -out x.jsonl
expect 8 "a missing probe file is refused"                         -- -run-grid $P -probe-file nope.jsonl -amend-sha $SHA -out x.jsonl

SEEDS='"eta_seeds":[9930,9931,9932,9933,9934],"step16_seeds":[9935,9936,9937,9938,9939],"step24_seeds":[9940,9941,9942,9943,9944],"det_seeds":[9945,9946,9947,9948,9949]'
ARR1='[1e-4,1e-4,1e-4,1e-4,1e-4]'; ARR2='[3e-4,3e-4,3e-4,3e-4,3e-4]'; ARR3='[1e-3,1e-3,1e-3,1e-3,1e-3]'; ARR4='[3e-3,3e-3,3e-3,3e-3,3e-3]'
TIMES="\"ts16\":$ARR1,\"ts24\":$ARR2,\"td16\":$ARR3,\"td24\":$ARR4"
TIMING='"timing":{"TS16":1e-4,"TD16":1e-3,"TS24":3e-4,"TD24":3e-3}'
GOODDEC='"decision":{"Run":true,"L":24,"R":200,"WallSeconds":0,"LowPower":false}'
mk() { echo "{\"kind\":\"probe\",\"eta\":$1,\"eta_ok\":$2,$SEEDS,$TIMES,$3,$4}" > "$5"; }

echo '{"kind":"probe","eta":0.1,"eta_ok":true,"eta_seeds":[1,2,3,4,5]}' > bad1.jsonl
expect 8 "a probe on the wrong seeds is refused"                   -- -run-grid $P -probe-file bad1.jsonl -amend-sha $SHA -out x.jsonl
mk 0.1 true "$TIMING" '"decision":{"Run":true,"L":16,"R":25,"WallSeconds":0,"LowPower":true}' bad2.jsonl
expect 8 "a probe decision that contradicts its timings is refused" -- -run-grid $P -probe-file bad2.jsonl -amend-sha $SHA -out x.jsonl
mk 0.123 true "$TIMING" "$GOODDEC" bad3.jsonl
expect 8 "a probe with eta off the ladder is refused"              -- -run-grid $P -probe-file bad3.jsonl -amend-sha $SHA -out x.jsonl
mk 0.1 false "$TIMING" "$GOODDEC" bad4.jsonl
expect 8 "a probe whose descent test failed is refused"            -- -run-grid $P -probe-file bad4.jsonl -amend-sha $SHA -out x.jsonl
mk 0.1 true '"timing":{"TS16":9e-4,"TD16":1e-3,"TS24":3e-4,"TD24":3e-3}' "$GOODDEC" bad5.jsonl
expect 8 "timings that are not the medians of the arrays are refused" -- -run-grid $P -probe-file bad5.jsonl -amend-sha $SHA -out x.jsonl
echo "{\"kind\":\"probe\",\"eta\":0.1,\"eta_ok\":true,$SEEDS,\"ts16\":[1,1,1,1,1],\"ts24\":[1,1,1,1,1],\"td16\":$ARR3,\"td24\":$ARR4,\"timing\":{\"TS16\":1,\"TD16\":1e-3,\"TS24\":1,\"TD24\":3e-3},\"decision\":{\"Run\":false,\"L\":0,\"R\":0,\"WallSeconds\":0,\"LowPower\":false}}" > bad6.jsonl
expect 8 "a probe whose rule says the round does not run is refused" -- -run-grid $P -probe-file bad6.jsonl -amend-sha $SHA -out x.jsonl
echo '{"kind":"end"}' > bad7.jsonl
expect 8 "a probe file with no probe record is refused"             -- -run-grid $P -probe-file bad7.jsonl -amend-sha $SHA -out x.jsonl

# analyze refuses partial data
echo '{"kind":"meta","L":8,"R":2,"ladder":[25,50]}' > partial.jsonl
expect 1 "-analyze of a file with no end record is refused"        -- -analyze partial.jsonl
expect 1 "-analyze of a missing file is refused"                   -- -analyze nope.jsonl

# the positive control for the guard itself: a free-seed dry run is accepted
expect 0 "a smoke dry-run on free seeds is accepted"               -- -dry-run

[ -e x.jsonl ] && { echo "FAIL: a refused run created its output file"; fail=1; }
exit $fail
