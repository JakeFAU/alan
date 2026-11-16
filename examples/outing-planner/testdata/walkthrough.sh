#!/bin/sh
set -eu

cd "$(dirname "$0")/../../.."
fixtures="$PWD/examples/outing-planner/testdata"
work=$(mktemp -d)
server_pid=
stop_server() {
    if [ -n "$server_pid" ]; then
        pid=$server_pid
        server_pid=
        kill -TERM "$pid" 2>/dev/null || true
        wait "$pid"
    fi
}
trap stop_server EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

printf 'Walkthrough files: %s\n' "$work"
go build -o "$work/outing-planner" ./examples/outing-planner
go test -tags integration -c -o "$work/fixture-server" ./examples/outing-planner
ALAN_OUTING_WALKTHROUGH_FIXTURE="$fixtures/fixture.json" \
    ALAN_OUTING_WALKTHROUGH_READY="$work/endpoint" \
    "$work/fixture-server" -test.run '^TestOutingWalkthroughFixtureServer$' -test.v -test.timeout=10m \
    >"$work/fixture.log" 2>&1 &
server_pid=$!
for attempt in $(seq 1 100); do
    [ -s "$work/endpoint" ] && break
    kill -0 "$server_pid" 2>/dev/null || break
    sleep 0.1
done
if [ ! -s "$work/endpoint" ]; then
    cat "$work/fixture.log" >&2
    exit 1
fi

python3 -I - "$work" "$fixtures" <<'PY'
import json
import sys
from pathlib import Path

work, fixtures = map(Path, sys.argv[1:])
config = json.loads((fixtures / "configuration.json").read_text())
config["endpoint"] = (work / "endpoint").read_text().strip()
config["reference_time"] = json.loads((fixtures / "fixture.json").read_text())["reference_time"]
for name, codes in [("recommended", [0, 1, 2, 3]), ("no-match", [])]:
    config["allowed_weather_codes"] = codes
    (work / f"{name}.json").write_text(json.dumps(config) + "\n")
PY

planner="$work/outing-planner"
for name in recommended no-match; do
    db="$work/$name.db"
    "$planner" start --db "$db" --config "$work/$name.json" --json >"$work/$name-start.json"
    "$planner" run --db "$db" --json >"$work/$name-run.json"
    "$planner" resume --db "$db" --json >"$work/$name-resume.json"
    "$planner" status --db "$db"
    "$planner" inspect --db "$db" >"$work/$name-inspect.json"
    "$planner" replay --db "$db" --json >"$work/$name-replay.json"
done
stop_server
cat "$work/fixture.log"
"$planner" check-replacement --config "$fixtures/configuration.json" --fixture "$fixtures/fixture.json" \
    --weather assess-first --output-dir "$work/check-assess-first" >"$work/check-assess-first.json"
"$planner" check-replacement --config "$fixtures/configuration.json" --fixture "$fixtures/fixture.json" \
    --weather confirm-first --output-dir "$work/check-confirm-first" >"$work/check-confirm-first.json"
"$planner" inspect --db "$work/check-confirm-first/candidate.db" >"$work/blocked-inspect.json"
"$planner" replay --db "$work/check-confirm-first/candidate.db" --json >"$work/blocked-replay.json"
python3 -I - "$work" <<'PY'
import json
import sys
from pathlib import Path

work = Path(sys.argv[1])

def read(name):
    return json.loads((work / f"{name}.json").read_text())

for name, outcome in [("recommended", "recommended"), ("no-match", "no_match")]:
    assert read(f"{name}-start")["outcome"] == "pending"
    for command in ("run", "resume", "inspect", "replay"):
        report = read(f"{name}-{command}")
        assert report["outcome"] == outcome
        assert report["decision_selected_at"]["Run"]["Subject"] == "weather"
    assert read(f"{name}-replay")["replay"]["Namespace"] == report["namespace"]

assess = read("check-assess-first")
assert assess["outcome"] == "recommended"
assert assess["baseline"]["fixture_requests"] == assess["candidate"]["fixture_requests"] == 2
confirm = read("check-confirm-first")
assert confirm["outcome"] == "circular_dependency"
assert confirm["baseline"]["fixture_requests"] == 2
assert confirm["candidate"]["fixture_requests"] == 0
assert confirm["candidate"]["reopened"]["outcome"] == "circular_dependency"
assert read("blocked-inspect")["outcome"] == read("blocked-replay")["outcome"] == "pending"
print("Verified recommended, no_match, assess-first control, and confirm-first reciprocal wait.")
PY
