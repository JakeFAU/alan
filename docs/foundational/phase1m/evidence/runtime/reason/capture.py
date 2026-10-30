#!/usr/bin/env python3
"""Capture one source-bound runtime check; never run historical capture scripts.

Usage: capture.py LABEL ARGV... . Labels are exclusive: no record is overwritten.
"""
from pathlib import Path
import datetime, gzip, hashlib, json, os, subprocess, sys, tempfile, time

ROOT = Path(__file__).resolve().parents[6]
DEST = Path(__file__).resolve().parent
sha = lambda data: hashlib.sha256(data).hexdigest()
PINNED = dict(GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off', GOFLAGS='-mod=readonly',
              CGO_ENABLED='1', GOCACHE='/private/tmp/alan-phase1l-go-build')

def sources():
    paths = subprocess.check_output(['rg', '--files', '-g', '*.go', '-g', 'go.mod', '-g', 'go.sum'],
                                    cwd=ROOT, text=True).splitlines()
    paths += ['AGENTS.md', str(Path(__file__).resolve().relative_to(ROOT))]
    return {p: sha((ROOT / p).read_bytes()) for p in sorted(set(paths))}

def main():
    label, *argv = sys.argv[1:]
    target = DEST / (label + '.json.gz')
    if target.exists():
        raise SystemExit('refusing to overwrite ' + str(target))
    before = sources()
    env = {k:v for k,v in os.environ.items() if not k.startswith('ALAN_') and k != 'OPENAI_API_KEY'}
    env.update(PINNED)
    with tempfile.TemporaryDirectory(prefix='alan-phase1m-' + label + '-', dir='/private/tmp') as temporary:
        env['ALAN_I_ARTIFACTS'] = temporary
        start = time.monotonic()
        command = subprocess.run(argv, cwd=ROOT, env=env, capture_output=True, text=True)
        events = []
        for line in command.stdout.splitlines():
            try: event = json.loads(line)
            except ValueError: continue
            if isinstance(event, dict): events.append(event)
        selected = [e['Test'] for e in events if e.get('Action') == 'run' and 'Test' in e]
        leaves = [t for t in selected if not any(u.startswith(t + '/') for u in selected)]
        records = {}
        for p in sorted(Path(temporary).rglob('*')):
            if not p.is_file() or p.name.startswith('program-'): continue
            raw = p.read_bytes()
            records[str(p.relative_to(temporary))] = (json.loads(gzip.decompress(raw)) if p.suffix == '.gz'
                                                    else [json.loads(line) for line in raw.splitlines()])
        snapshots = {p: (ROOT/p).read_text() for p in [
            'internal/substitutiontest/invocation_join_evidence_test.go',
            'internal/substitutiontest/consumer_policy_runtime_test.go',
            'internal/substitutiontest/consumer_policy_decoder_test.go',
            'internal/substitutiontest/consumer_reason_evidence_test.go',
            'internal/substitutiontest/consumer_reason_runtime_test.go',
            'internal/substitutiontest/consumer_reason_decoder_test.go',
        ]}
        record = dict(label=label, argv=argv, cwd=str(ROOT), utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            environment=dict(PINNED, ALAN_I_ARTIFACTS=temporary), removed_environment=['ALAN_*','OPENAI_API_KEY'],
            exit=command.returncode, elapsed_seconds=time.monotonic()-start, stdout=command.stdout, stderr=command.stderr,
            selected_leaves=leaves, test_results=[{k:e[k] for k in ['Action','Test','Package'] if k in e}
                for e in events if e.get('Action') in ['pass','fail','skip']],
            sources=before, source_aggregate_sha256=sha(json.dumps(before,sort_keys=True,separators=(',',':')).encode()),
            source_snapshots=snapshots, sources_unchanged=before==sources(), records=records)
        target.write_bytes(gzip.compress(json.dumps(record,separators=(',',':')).encode(),mtime=0))
        print(json.dumps({k:record[k] for k in ['label','exit','elapsed_seconds','selected_leaves','sources_unchanged']},indent=2))
        if command.returncode: print(command.stdout); print(command.stderr)
        if not record['sources_unchanged']: raise SystemExit('sources changed during command')
        raise SystemExit(command.returncode)

if __name__ == '__main__': main()
