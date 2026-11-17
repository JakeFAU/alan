#!/bin/sh
# A local, frozen two-park example. Optional argument retains the outputs there.
set -eu
cd "$(dirname "$0")/../../.."
work=${1:-$(mktemp -d)}
mkdir -p "$work"
work=$(cd "$work" && pwd)
binary=$(mktemp -d)/outing-planner
trap 'rm -f "$binary"; rmdir "$(dirname "$binary")"' EXIT
printf 'Retaining two-park evidence in %s\n' "$work"
go build -o "$binary" ./examples/outing-planner
"$binary" check-replacement --config examples/outing-planner/testdata/configuration.json \
  --fixture examples/outing-planner/testdata/fixture.json --weather delegated \
  --output-dir "$work/accepted" > "$work/accepted-cli.json"
"$binary" check-replacement --config examples/outing-planner/testdata/configuration.json \
  --fixture examples/outing-planner/testdata/fixture.json --weather delegated --helper-admission 0 \
  --output-dir "$work/refused" > "$work/refused-cli.json"
for name in accepted refused; do
    pair="$work/$name"
    for side in baseline candidate; do
        python3 -I - "$pair/$side.db" <<'PYBEFORE'
import sys,hashlib
from pathlib import Path
p=Path(sys.argv[1]);p.with_suffix('.db.before-sha').write_text(hashlib.sha256(p.read_bytes()).hexdigest())
PYBEFORE
        "$binary" inspect --db "$pair/$side.db" > "$pair/$side-inspect.json"
        python3 -I -B examples/outing-planner/model/test_evidence.py capture \
          "$pair/$side.db" "$pair/$side-captured.db"
        python3 -I - "$pair/$side.db" <<'PYAFTER'
import sys,hashlib,json
from pathlib import Path
p=Path(sys.argv[1]);before=p.with_suffix('.db.before-sha').read_text()
assert hashlib.sha256(p.read_bytes()).hexdigest()==before
p.with_suffix('.db.read-provenance.json').write_text(json.dumps({'original_main_sha256':before,'original_main_unchanged':True,'method':'closed CLI Store, public read-only inspect then mode=ro transactional backup; only reader sidecars created'})+'\n')
p.with_suffix('.db.before-sha').unlink()
PYAFTER
    done
    python3 -I -B examples/outing-planner/model/run.py compare \
      --baseline "$pair/baseline-captured.db" --candidate "$pair/candidate-captured.db" \
      --configuration "$pair/derived-configuration.json" --fixture "$pair/fixture.json" \
      --manifest "$pair/deployment-manifest.json" --replacement "$pair/replacement.json" \
      --output "$pair/independent-comparison.json"
    "$binary" status --db "$pair/candidate.db"
    "$binary" inspect --db "$pair/candidate.db" > "$pair/inspect.json"
    "$binary" replay --db "$pair/candidate.db" --json > "$pair/replay.json"
done
python3 -I - "$work" <<'PY'
import json,sys
from pathlib import Path
root=Path(sys.argv[1])
accepted=json.loads((root/'accepted/replacement.json').read_bytes())
refused=json.loads((root/'refused/replacement.json').read_bytes())
assert accepted['delegated_comparison']['positive_replacement']
assert not refused['delegated_comparison']['positive_replacement']
assert refused['candidate']['fixture_requests']==0
print('Cedar and Hemlock were assessed through one helper. The completed pair agrees on its choice; delegation pays for one invocation and one created run. Refusing the helper records helper_refused with zero candidate GETs.')
PY
