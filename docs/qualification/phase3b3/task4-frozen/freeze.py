"""Run and retain the final immutable-source runtime qualification."""
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
from pathlib import Path
import hashlib
import json
import os
import subprocess
import sys
import time

repo = Path('/private/tmp/alan-phase3b3-runtime')
out = Path('/tmp/alan-phase3b3-runtime-frozen')
expected = sys.argv[1]
head = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=repo, text=True).strip()
assert head == expected, (head, expected)
assert not out.exists(), 'Use a new output root; never overwrite retained evidence'
out.mkdir()
env = os.environ.copy()
env.update(GOTOOLCHAIN='local', GOCACHE='/tmp/alan-phase3a-go-build',
           GOPROXY='off', GOSUMDB='off', STATICCHECK_CACHE='/tmp/alan-phase3b3-staticcheck',
           ALAN_PHASE3B3_OLD_READER='/tmp/alan-phase3b3-old-reader/alan')
inputs = subprocess.check_output(['git', 'ls-files', '--', '*.go', '*go.mod', '*go.sum'], cwd=repo, text=True).splitlines()
assert not subprocess.check_output(['git', 'diff', 'HEAD', '--', '*.go', '*go.mod', '*go.sum'], cwd=repo)
untracked = subprocess.check_output(['git', 'ls-files', '--others', '--exclude-standard'], cwd=repo, text=True).splitlines()
assert not [p for p in untracked if p.endswith('.go') or Path(p).name in ('go.mod', 'go.sum')]
source = {p: hashlib.sha256((repo / p).read_bytes()).hexdigest() for p in inputs}
metadata = {'SourceCommit': head, 'GoModuleFiles': source,
            'Go': subprocess.check_output(['go', 'version'], cwd=repo, env=env, text=True).strip(),
            'Staticcheck': subprocess.check_output(['staticcheck', '-version'], cwd=repo, env=env, text=True).strip(),
            'OldReaderSHA256': hashlib.sha256(Path(env['ALAN_PHASE3B3_OLD_READER']).read_bytes()).hexdigest(),
            'RunnerSHA256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest()}
(out / 'source.json').write_text(json.dumps(metadata, indent=2) + '\n')
pattern = '^TestPhase3B3(Reciprocal|CartesianOperations|GenericUnknown|ActualResultLaundering|SerialSerial|AppliedUnknownRevoked|RetainedMutations)'
tasks = [
    ('ordinary-module', ['go', 'test', '-json', '-count=1', './...'], {}),
    ('integration-core', ['go', 'test', '-json', '-tags=integration', '-count=1', './internal/executor', './runtime', './author', './examples/delegation'], {}),
    ('reciprocal-normal', ['go', 'test', '-json', '-tags=integration', '-count=1', './internal/durabletest', '-run', pattern], {'ALAN_PHASE3B3_EVIDENCE': str(out / 'normal')}),
    ('reciprocal-race', ['go', 'test', '-json', '-race', '-tags=integration', '-count=1', './internal/durabletest', '-run', pattern], {'ALAN_PHASE3B3_EVIDENCE': str(out / 'race')}),
    ('authority-race', ['go', 'test', '-json', '-race', '-tags=integration', '-count=1', './internal/executor', './runtime', './author', '-run', '^TestRequestAuthority'], {}),
    ('baseline-current', ['go', 'test', '-json', '-tags=integration', '-count=1', './internal/durabletest', '-run', '^TestPhase3B3Baseline'], {'ALAN_PHASE3B3_EVIDENCE': str(out / 'baseline-current')}),
    ('staticcheck', ['staticcheck', './...'], {}),
    ('staticcheck-integration', ['staticcheck', '-tags=integration', './...'], {}),
]

def run(task):
    name, command, additions = task
    local = env.copy()
    local.update(additions)
    start = datetime.now(timezone.utc).isoformat()
    tick = time.monotonic()
    log = out / (name + '.log')
    with log.open('wb') as stream:
        completed = subprocess.run(command, cwd=repo, env=local, stdout=stream, stderr=subprocess.STDOUT)
    record = {'Name': name, 'Command': command, 'Cwd': str(repo), 'SourceCommit': head,
              'Environment': {k: local[k] for k in ['GOTOOLCHAIN', 'GOCACHE', 'GOPROXY', 'GOSUMDB', 'STATICCHECK_CACHE', 'ALAN_PHASE3B3_OLD_READER']},
              'AdditionalEnvironment': additions, 'StartedUTC': start,
              'Seconds': time.monotonic() - tick, 'Exit': completed.returncode,
              'Log': log.name, 'SHA256': hashlib.sha256(log.read_bytes()).hexdigest()}
    if command[:2] == ['go', 'test']:
        events = []
        for line in log.read_text(errors='replace').splitlines():
            try:
                events.append(json.loads(line))
            except json.JSONDecodeError:
                pass
        record['Tests'] = {'PassEvents': sum(e.get('Action') == 'pass' and 'Test' in e for e in events),
                           'PackagePasses': [e['Package'] for e in events if e.get('Action') == 'pass' and 'Test' not in e],
                           'Failures': [e for e in events if e.get('Action') == 'fail'],
                           'Skips': [e for e in events if e.get('Action') == 'skip']}
    (out / (name + '-command.json')).write_text(json.dumps(record, indent=2) + '\n')
    print(json.dumps({'Name': name, 'Exit': completed.returncode, 'Seconds': record['Seconds']}), flush=True)
    return record

with ThreadPoolExecutor(max_workers=3) as pool:
    records = list(pool.map(run, tasks))
assert source == {p: hashlib.sha256((repo / p).read_bytes()).hexdigest() for p in inputs}, 'source changed during qualification'
(out / 'commands.json').write_text(json.dumps(records, indent=2) + '\n')
failures = [r['Name'] for r in records if r['Exit'] != 0]
(out / 'completion.json').write_text(json.dumps({'SourceStable': True, 'UnexpectedNonzero': failures, 'Commands': len(records)}, indent=2) + '\n')
print(json.dumps({'SourceStable': True, 'UnexpectedNonzero': failures}), flush=True)
sys.exit(bool(failures))
