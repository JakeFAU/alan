#!/usr/bin/env python3
"""Exclusive Task4 command/source/raw-record capture. Never rewrites old evidence."""
from pathlib import Path
import datetime, gzip, hashlib, io, json, os, subprocess, sys, tarfile, tempfile, time

DEST = Path(__file__).resolve().parent
ROOT = DEST
while not (ROOT / 'go.mod').exists():
    ROOT = ROOT.parent
sha = lambda b: hashlib.sha256(b).hexdigest()

def sources():
    files = subprocess.check_output(['rg', '--files', '-g', '*.go', '-g', 'go.mod', '-g', 'go.sum'], cwd=ROOT, text=True).splitlines()
    files += [str(p.relative_to(ROOT)) for p in DEST.glob('*.py')]
    files += ['AGENTS.md', 'docs/foundational/phase1n/01-contract.md', 'docs/foundational/phase1n/01-representation.md']
    return {p: sha((ROOT/p).read_bytes()) for p in sorted(set(files))}

def main():
    label, *argv = sys.argv[1:]
    out = DEST / label
    out.mkdir()
    before = sources()
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode='w') as archive:
        for p in before:
            raw = (ROOT/p).read_bytes()
            info = tarfile.TarInfo(p); info.size = len(raw); info.mode = 0o644; info.mtime = 0
            archive.addfile(info, io.BytesIO(raw))
    snapshot = gzip.compress(buf.getvalue(), mtime=0)
    (out/'sources.tar.gz').write_bytes(snapshot)
    env = {k:v for k,v in os.environ.items() if not k.startswith('ALAN_') and k != 'OPENAI_API_KEY'}
    pinned = dict(GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off', GOFLAGS='-mod=readonly', CGO_ENABLED='1', GOCACHE='/private/tmp/alan-phase1n-runtime-go-build')
    env.update(pinned)
    toolchain = subprocess.run(['/opt/homebrew/bin/go', 'version'], cwd=ROOT, env=env, capture_output=True, text=True)
    effective = subprocess.run(['/opt/homebrew/bin/go', 'env', '-json', 'GOVERSION', 'GOOS', 'GOARCH', 'CGO_ENABLED', 'GOTOOLCHAIN', 'GOPROXY', 'GOSUMDB', 'GOFLAGS', 'GOCACHE'], cwd=ROOT, env=env, capture_output=True, text=True)
    with tempfile.TemporaryDirectory(prefix='alan-phase1n-runtime-', dir='/private/tmp') as tmp:
        env['ALAN_I_ARTIFACTS'] = tmp
        start = time.monotonic()
        command = subprocess.run(argv, cwd=ROOT, env=env, capture_output=True, text=True)
        events = []
        for line in command.stdout.splitlines():
            try: e = json.loads(line)
            except ValueError: continue
            if isinstance(e, dict): events.append(e)
        runs = [e['Test'] for e in events if e.get('Action') == 'run' and 'Test' in e]
        leaves = [x for x in runs if not any(y.startswith(x+'/') for y in runs)]
        records = {}
        for p in sorted(Path(tmp).rglob('*')):
            if not p.is_file(): continue
            raw = p.read_bytes()
            if p.suffix == '.gz': value = json.loads(gzip.decompress(raw))
            elif p.suffix == '.jsonl': value = [json.loads(line) for line in raw.splitlines()]
            else: value = json.loads(raw)
            records[str(p.relative_to(tmp))] = value
        changed = [p for p,h in before.items() if sha((ROOT/p).read_bytes()) != h]
        record = dict(label=label, argv=argv, cwd=str(ROOT), utc=datetime.datetime.now(datetime.timezone.utc).isoformat(), environment=dict(pinned, ALAN_I_ARTIFACTS=tmp), removed_environment=['ALAN_*','OPENAI_API_KEY'], toolchain=toolchain.stdout, effective_environment=json.loads(effective.stdout), exit=command.returncode, elapsed_seconds=time.monotonic()-start, stdout=command.stdout, stderr=command.stderr, selected_leaves=leaves, test_results=[{k:e[k] for k in ['Action','Test','Package'] if k in e} for e in events if e.get('Action') in ['pass','fail','skip']], sources=before, source_snapshot_sha256=sha(snapshot), changed_during_run=changed, records=records)
        (out/'command.json.gz').write_bytes(gzip.compress((json.dumps(record,sort_keys=True,indent=2)+'\n').encode(),mtime=0))
        print(json.dumps(dict(label=label,exit=command.returncode,leaves=len(leaves),records=len(records),changed=changed)))
        if command.returncode: print(command.stdout[-14000:]); print(command.stderr[-4000:])
        sys.exit(command.returncode or bool(changed))

if __name__ == '__main__': main()
