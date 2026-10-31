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
grep -q '"kind":"calibration"' cal-paused.jsonl && { echo "FAIL: a paused calibration wrote a result"; fail=1; }
rm -f cal-paused.jsonl

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

# ---- qbp-kz-g2 round 1 Stage-0b-only mode (FROZEN-v1 §2 additions 1-2, §5 aborts) ----
# 5900-5939 (run block) and 5940-5979 (re-run block) are refused by every other mode;
# -run-0b derives its seeds from -tau and refuses typed seeds, tau off the set, mode mixing.
TC=1.0666262219986817
expect 4 "smoke seeds inside the kz-g2 r1 block are refused"         -- -first-seed 5900 -out x.jsonl
expect 4 "smoke seeds at the end of the kz-g2 r1 block are refused"  -- -first-seed 5939 -R 1 -out x.jsonl
expect 4 "smoke seeds inside the kz-g2 r1 re-run block are refused"  -- -first-seed 5940 -out x.jsonl
expect 4 "smoke seeds straddling into the kz-g2 r1 block are refused" -- -first-seed 5895 -R 8 -out x.jsonl
expect 4 "-smoke-0b on the kz-g2 r1 block is refused"                -- -smoke-0b -tau 20 -tc 1 -first-seed 5900 $P -out x.jsonl
expect 4 "-smoke-0b on the re-run block is refused"                  -- -smoke-0b -tau 20 -tc 1 -first-seed 5970 $P -out x.jsonl
expect 4 "-smoke-0b straddling into the block is refused"            -- -smoke-0b -tau 20 -tc 1 -first-seed 5895 $P -out x.jsonl
expect 4 "-run-0b with -run-grid is refused"                         -- -run-0b -run-grid -tau 1600 -tc $TC -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-0b with -run-probe is refused"                        -- -run-0b -run-probe -tau 1600 -tc $TC -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-0b with -run-calibration is refused"                  -- -run-0b -run-calibration -cal-ensemble aligned -tau 1600 -tc $TC -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-0b with -smoke-0b is refused"                         -- -run-0b -smoke-0b -tau 1600 -tc $TC -prereg-sha $SHA $P -out x.jsonl
expect 4 "-analyze with -run-0b is refused"                          -- -analyze x.jsonl -run-0b
expect 4 "-analyze-0b with -run-0b is refused"                       -- -analyze-0b -run-0b a b c d
expect 4 "-analyze-0b with -run-grid is refused"                     -- -analyze-0b -run-grid a b c d
expect 4 "-analyze-0b with -analyze is refused"                      -- -analyze-0b -analyze x.jsonl a b c d
expect 4 "-run-0b with a typed first seed is refused (seeds are derived)" -- -run-0b -tau 1600 -tc $TC -prereg-sha $SHA -first-seed 5900 $P -out x.jsonl
expect 4 "-run-0b with a typed wrong first seed is refused"          -- -run-0b -tau 1600 -tc $TC -prereg-sha $SHA -first-seed 5910 $P -out x.jsonl
expect 4 "-run-0b with a typed free first seed is refused"           -- -run-0b -tau 1600 -tc $TC -prereg-sha $SHA -first-seed 13000 $P -out x.jsonl
expect 4 "-run-0b with tau off the frozen set is refused"            -- -run-0b -tau 1700 -tc $TC -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-0b with the round-1 ladder tau is refused"            -- -run-0b -tau 200 -tc $TC -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-0b without -tau is refused"                           -- -run-0b -tc $TC -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-0b with -L is refused (L = 16 is frozen)"             -- -run-0b -tau 1600 -tc $TC -prereg-sha $SHA -L 8 $P -out x.jsonl
expect 8 "-run-0b without -tc is refused (no default)"               -- -run-0b -tau 1600 -prereg-sha $SHA $P -out x.jsonl
expect 8 "-run-0b without -prereg-sha is refused"                    -- -run-0b -tau 1600 -tc $TC $P -out x.jsonl
expect 8 "-run-0b with a short -prereg-sha is refused"               -- -run-0b -tau 1600 -tc $TC -prereg-sha abc $P -out x.jsonl
expect 8 "-run-0b with a non-hex -prereg-sha is refused"             -- -run-0b -tau 1600 -tc $TC -prereg-sha zz00000000000000000000000000000000000000000000000000000000000000 $P -out x.jsonl
expect 8 "-run-0b with a short -rerun-amend-sha is refused"          -- -run-0b -tau 1600 -tc $TC -prereg-sha $SHA -rerun-amend-sha abc $P -out x.jsonl
expect 0 "-run-0b dry-run prints the plan and writes nothing"        -- -run-0b -tau 6400 -tc $TC -prereg-sha $SHA -dry-run
grep -q 'seeds 5920..5929' "$T/out" || { echo "FAIL: -run-0b dry-run for tau 6400 must plan seeds 5920..5929"; cat "$T/out"; fail=1; }
grep -q 'qbp-kz-g2-r1-0b-tau6400.jsonl' "$T/out" || { echo "FAIL: default output name"; cat "$T/out"; fail=1; }
[ -e qbp-kz-g2-r1-0b-tau6400.jsonl ] && { echo "FAIL: dry-run created the output"; fail=1; }
expect 0 "-run-0b dry-run under the re-run amendment plans the re-run sub-block" -- -run-0b -tau 12800 -tc $TC -prereg-sha $SHA -rerun-amend-sha $SHA -dry-run
grep -q 'seeds 5970..5979' "$T/out" || { echo "FAIL: re-run sub-block for tau 12800 must be 5970..5979"; cat "$T/out"; fail=1; }
# safety net: a -run-0b that passes every guard stops at the pause file before any step
expect 3 "-run-0b stops at the pause file before any step"          -- -run-0b -tau 1600 -tc $TC -prereg-sha $SHA $P -out 0b-paused.jsonl
grep -q '"kind":"meta"' 0b-paused.jsonl || { echo "FAIL: paused 0b wrote no meta"; fail=1; }
grep -q '"seeds":\[5900,5901,5902,5903,5904,5905,5906,5907,5908,5909\]' 0b-paused.jsonl || { echo "FAIL: meta seeds for tau 1600"; head -c 600 0b-paused.jsonl; fail=1; }
grep -q "\"prereg_sha\":\"$SHA\"" 0b-paused.jsonl || { echo "FAIL: meta must record prereg_sha"; fail=1; }
grep -q '"kind":"stage0b"' 0b-paused.jsonl && { echo "FAIL: a paused 0b run wrote a stage0b record"; fail=1; }
grep -q '"kind":"end"' 0b-paused.jsonl && { echo "FAIL: a paused 0b run wrote an end record"; fail=1; }
expect 3 "-run-0b re-run sub-block stops at the pause file"         -- -run-0b -tau 3200 -tc $TC -prereg-sha $SHA -rerun-amend-sha $SHA $P -out 0b-paused2.jsonl
grep -q '"seeds":\[5950,5951,5952,5953,5954,5955,5956,5957,5958,5959\]' 0b-paused2.jsonl || { echo "FAIL: re-run seeds for tau 3200"; fail=1; }
grep -q '"rerun":true' 0b-paused2.jsonl || { echo "FAIL: meta must flag the re-run"; fail=1; }
# the round-1 analyzer refuses a 0b-only file by design; -analyze-0b refuses a partial set
expect 1 "-analyze refuses a 0b-only file"                           -- -analyze 0b-paused.jsonl
expect 1 "-analyze-0b with no files is refused"                      -- -analyze-0b
expect 1 "-analyze-0b with three files is refused (no partial-set reading)" -- -analyze-0b 0b-paused.jsonl 0b-paused2.jsonl 0b-paused.jsonl
grep -q 'partial' "$T/err" || { echo "FAIL: the partial-set refusal must say so"; cat "$T/err"; fail=1; }
expect 1 "-analyze-0b with four incomplete files is refused"         -- -analyze-0b 0b-paused.jsonl 0b-paused2.jsonl 0b-paused.jsonl 0b-paused2.jsonl
rm -f 0b-paused2.jsonl
expect 0 "-smoke-0b dry-run on free seeds is accepted"               -- -smoke-0b -tau 20 -tc 1 -first-seed 13000 -L 8 -dry-run

# ---- qbp-kz-g2 round 3 equilibrium mode (r3 FROZEN-v1 §2 rows 3-4, §5 seed guard; sha 132f6a42…1a623) ----
# 6000-6119 (run block) and 6120-6239 (re-run block) are refused by every other mode (smokes
# included); -run-eq derives its seeds from (L, t), refuses typed seeds, -tc, L off {8,12,16},
# t off 0..9, (L = 8, t in {3,5,7}) and mode mixing.
expect 4 "smoke seeds inside the kz-g2 r3 block are refused"         -- -first-seed 6000 -out x.jsonl
expect 4 "smoke seeds at the end of the kz-g2 r3 block are refused"  -- -first-seed 6119 -R 1 -out x.jsonl
expect 4 "smoke seeds inside the kz-g2 r3 re-run block are refused"  -- -first-seed 6120 -out x.jsonl
expect 4 "smoke seeds at the end of the r3 re-run block are refused" -- -first-seed 6239 -R 1 -out x.jsonl
expect 4 "smoke seeds straddling into the kz-g2 r3 block are refused" -- -first-seed 5995 -R 8 -out x.jsonl
expect 4 "-smoke-0b on the kz-g2 r3 block is refused"                -- -smoke-0b -tau 20 -tc 1 -first-seed 6000 $P -out x.jsonl
expect 4 "-smoke-0b on the kz-g2 r3 re-run block is refused"         -- -smoke-0b -tau 20 -tc 1 -first-seed 6230 $P -out x.jsonl
expect 4 "-smoke-eq on the kz-g2 r3 block is refused"                -- -smoke-eq -L 8 -first-seed 6000 $P -out x.jsonl
expect 4 "-smoke-eq on the kz-g2 r3 re-run block is refused"         -- -smoke-eq -L 8 -first-seed 6238 $P -out x.jsonl
expect 4 "-smoke-eq straddling into the r3 block is refused"         -- -smoke-eq -L 8 -first-seed 5999 $P -out x.jsonl
expect 4 "-smoke-eq straddling out of the r3 re-run block is refused" -- -smoke-eq -L 8 -first-seed 6239 $P -out x.jsonl
expect 4 "-smoke-eq on the kz-g2 r1 block is refused"                -- -smoke-eq -L 8 -first-seed 5900 $P -out x.jsonl
expect 4 "-run-eq with -run-grid is refused"                         -- -run-eq -run-grid -L 16 -t 5 -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-eq with -run-probe is refused"                        -- -run-eq -run-probe -L 16 -t 5 -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-eq with -run-0b is refused"                           -- -run-eq -run-0b -tau 1600 -tc $TC -L 16 -t 5 -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-eq with -smoke-eq is refused"                         -- -run-eq -smoke-eq -L 16 -t 5 -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-eq with -smoke-0b is refused"                         -- -run-eq -smoke-0b -tau 20 -tc 1 -L 16 -t 5 -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-0b with -smoke-eq is refused"                         -- -run-0b -smoke-eq -tau 1600 -tc $TC -prereg-sha $SHA $P -out x.jsonl
expect 4 "-analyze-eq with -run-eq is refused"                       -- -analyze-eq -run-eq a
expect 4 "-analyze-eq with -run-0b is refused"                       -- -analyze-eq -run-0b a
expect 4 "-analyze-eq with -analyze-0b is refused"                   -- -analyze-eq -analyze-0b a b c d
expect 4 "-analyze-eq with -analyze is refused"                      -- -analyze-eq -analyze x.jsonl a
expect 4 "-analyze-0b with -run-eq is refused"                       -- -analyze-0b -run-eq a b c d
expect 4 "-analyze with -run-eq is refused"                          -- -analyze x.jsonl -run-eq
expect 4 "-run-eq with a typed first seed is refused (seeds are derived)" -- -run-eq -L 16 -t 5 -prereg-sha $SHA -first-seed 6100 $P -out x.jsonl
expect 4 "-run-eq with a typed free first seed is refused"           -- -run-eq -L 16 -t 5 -prereg-sha $SHA -first-seed 13000 $P -out x.jsonl
expect 4 "-run-eq with -tc is refused (not an eq flag)"              -- -run-eq -L 16 -t 5 -tc $TC -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-eq without -L is refused (no default)"                -- -run-eq -t 5 -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-eq without -t is refused (no default)"                -- -run-eq -L 16 -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-eq with L = 24 is refused"                            -- -run-eq -L 24 -t 5 -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-eq with L = 10 is refused"                            -- -run-eq -L 10 -t 5 -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-eq with t = 10 is refused"                            -- -run-eq -L 16 -t 10 -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-eq with t = -1 is refused"                            -- -run-eq -L 16 -t -1 -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-eq L = 8 at t = 3 is refused"                         -- -run-eq -L 8 -t 3 -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-eq L = 8 at t = 5 is refused"                         -- -run-eq -L 8 -t 5 -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-eq L = 8 at t = 7 is refused"                         -- -run-eq -L 8 -t 7 -prereg-sha $SHA $P -out x.jsonl
expect 4 "-run-eq L = 8 at t = 3 is refused even as a dry-run"       -- -run-eq -L 8 -t 3 -prereg-sha $SHA -dry-run
expect 8 "-run-eq without -prereg-sha is refused"                    -- -run-eq -L 16 -t 5 $P -out x.jsonl
expect 8 "-run-eq with a short -prereg-sha is refused"               -- -run-eq -L 16 -t 5 -prereg-sha abc $P -out x.jsonl
expect 8 "-run-eq with a non-hex -prereg-sha is refused"             -- -run-eq -L 16 -t 5 -prereg-sha zz00000000000000000000000000000000000000000000000000000000000000 $P -out x.jsonl
expect 8 "-run-eq with a short -rerun-amend-sha is refused"          -- -run-eq -L 16 -t 5 -prereg-sha $SHA -rerun-amend-sha abc $P -out x.jsonl
expect 0 "-run-eq dry-run (L=16, t=5) prints the plan and writes nothing" -- -run-eq -L 16 -t 5 -prereg-sha $SHA -dry-run
grep -q 'seeds 6100,6101,6102' "$T/out" || { echo "FAIL: -run-eq dry-run for (16, 5) must plan seeds 6100,6101,6102"; cat "$T/out"; fail=1; }
grep -q 'T=1.15' "$T/out" || { echo "FAIL: -run-eq dry-run for t=5 must plan T=1.15"; cat "$T/out"; fail=1; }
grep -q 'starts haar,haar,aligned' "$T/out" || { echo "FAIL: -run-eq dry-run must plan the start kinds haar,haar,aligned"; cat "$T/out"; fail=1; }
grep -q 'burn-in 5000  sampling 50000  record every 10 (5000 records/seed)' "$T/out" || { echo "FAIL: -run-eq dry-run must plan 5000/50000/10"; cat "$T/out"; fail=1; }
grep -q 'qbp-kz-g2-r3-eq-L16-t05.jsonl' "$T/out" || { echo "FAIL: default output name"; cat "$T/out"; fail=1; }
[ -e qbp-kz-g2-r3-eq-L16-t05.jsonl ] && { echo "FAIL: dry-run created the output"; fail=1; }
expect 0 "-run-eq dry-run (L=8, t=9) plans the last L=8 sub-block"  -- -run-eq -L 8 -t 9 -prereg-sha $SHA -dry-run
grep -q 'seeds 6036,6037,6038' "$T/out" || { echo "FAIL: (8, 9) must plan seeds 6036,6037,6038"; cat "$T/out"; fail=1; }
expect 0 "-run-eq dry-run (L=12, t=0) plans the first L=12 sub-block" -- -run-eq -L 12 -t 0 -prereg-sha $SHA -dry-run
grep -q 'seeds 6040,6041,6042' "$T/out" || { echo "FAIL: (12, 0) must plan seeds 6040,6041,6042"; cat "$T/out"; fail=1; }
expect 0 "-run-eq dry-run under the re-run amendment plans the re-run sub-block" -- -run-eq -L 16 -t 5 -prereg-sha $SHA -rerun-amend-sha $SHA -dry-run
grep -q 'seeds 6220,6221,6222' "$T/out" || { echo "FAIL: re-run sub-block for (16, 5) must be 6220,6221,6222"; cat "$T/out"; fail=1; }
# the end-line blindness surface: the plan never prints a statistic word
grep -qiE 'mean|binder|barrier|variance' "$T/out" && { echo "FAIL: the eq plan prints a statistic word"; cat "$T/out"; fail=1; }
# safety net: a -run-eq that passes every guard stops at the pause file before any step
expect 3 "-run-eq stops at the pause file before any step"          -- -run-eq -L 8 -t 0 -prereg-sha $SHA $P -out eq-paused.jsonl
grep -q '"kind":"meta"' eq-paused.jsonl || { echo "FAIL: paused eq wrote no meta"; fail=1; }
grep -q '"mode":"eq"' eq-paused.jsonl || { echo "FAIL: meta mode must be eq"; fail=1; }
grep -q '"seeds":\[6000,6001,6002\]' eq-paused.jsonl || { echo "FAIL: meta seeds for (8, 0)"; head -c 600 eq-paused.jsonl; fail=1; }
grep -q '"starts":\["haar","haar","aligned"\]' eq-paused.jsonl || { echo "FAIL: meta starts"; fail=1; }
grep -q "\"prereg_sha\":\"$SHA\"" eq-paused.jsonl || { echo "FAIL: meta must record prereg_sha"; fail=1; }
grep -q '"noise_stream":100002' eq-paused.jsonl || { echo "FAIL: meta must record the eq noise stream 100002"; fail=1; }
grep -q '"kind":"abort".*"steps":0' eq-paused.jsonl || { echo "FAIL: the abort at start must record steps 0 (a no-run)"; fail=1; }
grep -q '"kind":"eq"' eq-paused.jsonl && { echo "FAIL: a paused eq run wrote an eq record"; fail=1; }
grep -q '"kind":"end"' eq-paused.jsonl && { echo "FAIL: a paused eq run wrote an end record"; fail=1; }
expect 3 "-run-eq re-run sub-block stops at the pause file"         -- -run-eq -L 12 -t 3 -prereg-sha $SHA -rerun-amend-sha $SHA $P -out eq-paused2.jsonl
grep -q '"seeds":\[6172,6173,6174\]' eq-paused2.jsonl || { echo "FAIL: re-run seeds for (12, 3)"; fail=1; }
grep -q '"rerun":true' eq-paused2.jsonl || { echo "FAIL: meta must flag the re-run"; fail=1; }
expect 3 "-smoke-eq stops at the pause file before any step"        -- -smoke-eq -L 4 -first-seed 13000 $P -out smoke-eq-paused.jsonl
grep -q '"mode":"smoke-eq"' smoke-eq-paused.jsonl || { echo "FAIL: smoke-eq meta mode"; fail=1; }
grep -q '"seeds":\[13000,13001\]' smoke-eq-paused.jsonl || { echo "FAIL: smoke-eq seeds"; fail=1; }
# the analyzers refuse each other's files and partial sets
expect 1 "-analyze refuses an eq file"                               -- -analyze eq-paused.jsonl
expect 1 "-analyze-0b refuses eq files"                              -- -analyze-0b eq-paused.jsonl eq-paused2.jsonl eq-paused.jsonl eq-paused2.jsonl
expect 1 "-analyze-eq refuses a 0b file set"                         -- -analyze-eq 0b-paused.jsonl
expect 1 "-analyze-eq with no files is refused"                      -- -analyze-eq
expect 1 "-analyze-eq with one file is refused (no partial-set reading)" -- -analyze-eq eq-paused.jsonl
grep -q 'partial' "$T/err" || { echo "FAIL: the partial-set refusal must say so"; cat "$T/err"; fail=1; }
expect 1 "-analyze-eq with 26 files is refused"                      -- -analyze-eq $(for i in $(seq 26); do printf 'eq-paused.jsonl '; done)
grep -q 'partial' "$T/err" || { echo "FAIL: the 26-file refusal must say partial"; cat "$T/err"; fail=1; }
expect 1 "-analyze-eq with 27 copies of an incomplete file is refused" -- -analyze-eq $(for i in $(seq 27); do printf 'eq-paused.jsonl '; done)
expect 1 "-analyze-eq with 27 copies of a smoke file is refused"     -- -analyze-eq $(for i in $(seq 27); do printf 'smoke-eq-paused.jsonl '; done)
grep -qE 'smoke|two files' "$T/err" || { echo "FAIL: the smoke refusal must say so"; cat "$T/err"; fail=1; }
rm -f eq-paused.jsonl eq-paused2.jsonl smoke-eq-paused.jsonl 0b-paused.jsonl
expect 0 "-smoke-eq dry-run on free seeds is accepted"               -- -smoke-eq -L 8 -first-seed 13000 -dry-run
grep -q 'SMOKE-EQ' "$T/out" || { echo "FAIL: smoke-eq dry-run must say its mode"; cat "$T/out"; fail=1; }
grep -q 'burn-in 500  sampling 2000  record every 10 (200 records/seed)' "$T/out" || { echo "FAIL: smoke-eq lengths 500/2000/10"; cat "$T/out"; fail=1; }
grep -q 'starts haar,aligned' "$T/out" || { echo "FAIL: smoke-eq must run one hot + one cold seed"; cat "$T/out"; fail=1; }
expect 0 "-smoke-eq dry-run with the default first seed is accepted" -- -smoke-eq -L 8 -dry-run
grep -q 'seeds 13000,13001' "$T/out" || { echo "FAIL: smoke-eq default seeds 13000,13001"; cat "$T/out"; fail=1; }

# the positive control for the guard itself: a free-seed dry run is accepted
expect 0 "a smoke dry-run on free seeds is accepted"               -- -dry-run

[ -e x.jsonl ] && { echo "FAIL: a refused run created its output file"; fail=1; }
exit $fail
