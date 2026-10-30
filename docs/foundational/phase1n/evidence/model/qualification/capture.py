"""Exclusive source-bound Task3b command capture; no oracle definitions."""
import datetime, gzip, hashlib, io, json, os, pathlib, subprocess, sys, tarfile, time
root = pathlib.Path(__file__).resolve().parent
while not (root / 'go.mod').exists():
    root = root.parent
label, *argv = sys.argv[1:]
out = pathlib.Path(__file__).parent / label
out.mkdir()
raw = out / 'raw'
raw.mkdir()
env = {k:v for k,v in os.environ.items() if not k.startswith('ALAN_') and k != 'OPENAI_API_KEY'}
pinned = dict(GOTOOLCHAIN='local',CGO_ENABLED='1',GOPROXY='off',GOSUMDB='off',GOFLAGS='-mod=readonly',
              GOCACHE='/private/tmp/alan-phase1n-model-go-build',ALAN_MODEL_EVIDENCE=str(raw))
env.update(pinned)
files = subprocess.check_output(['rg','--files','-g','*.go','-g','go.mod','-g','go.sum'],cwd=root,text=True).splitlines()
files += [str(p.relative_to(root)) for p in pathlib.Path(__file__).parent.glob('*.py')]
files += ['docs/foundational/phase1n/01-contract.md','docs/foundational/phase1n/01-representation.md',
          'docs/foundational/phase1n/02-lifting.md','docs/foundational/phase1n/evidence/reviews/reference-r1.md']
sources = {p:hashlib.sha256((root/p).read_bytes()).hexdigest() for p in sorted(set(files))}
buffer = io.BytesIO()
with tarfile.open(fileobj=buffer,mode='w') as archive:
    for p in sources:
        data=(root/p).read_bytes();info=tarfile.TarInfo(p);info.size=len(data);info.mode=0o644;info.mtime=0
        archive.addfile(info,io.BytesIO(data))
snapshot = gzip.compress(buffer.getvalue(),mtime=0)
(out/'sources.tar.gz').write_bytes(snapshot)
start=time.monotonic()
run=subprocess.run(argv,cwd=root,env=env,text=True,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
events=[]
for line in run.stdout.splitlines():
    try: e=json.loads(line)
    except ValueError: continue
    if isinstance(e,dict):events.append(e)
runs=[e['Test'] for e in events if e.get('Action')=='run' and e.get('Test')]
leaves=[n for n in runs if not any(m.startswith(n+'/') for m in runs)]
outputs={}
for p in sorted(raw.glob('*')):
    data=p.read_bytes();dest=out/(p.name+'.gz');dest.write_bytes(gzip.compress(data,mtime=0))
    outputs[dest.name]=dict(sha256=hashlib.sha256(dest.read_bytes()).hexdigest(),raw_sha256=hashlib.sha256(data).hexdigest())
    p.unlink()
raw.rmdir()
changed=[p for p,h in sources.items() if hashlib.sha256((root/p).read_bytes()).hexdigest()!=h]
record=dict(label=label,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),argv=argv,cwd=str(root),
            environment=pinned,stripped=['ALAN_*','OPENAI_API_KEY'],exit=run.returncode,
            elapsed_seconds=time.monotonic()-start,stdout=run.stdout,stderr=run.stderr,
            sources=sources,source_snapshot_sha256=hashlib.sha256(snapshot).hexdigest(),changed_during_run=changed,
            test_runs=runs,leaves=leaves,passes=[e['Test'] for e in events if e.get('Action')=='pass' and e.get('Test')],
            failures=[e['Test'] for e in events if e.get('Action')=='fail' and e.get('Test')],
            skips=[e['Test'] for e in events if e.get('Action')=='skip' and e.get('Test')],outputs=outputs)
(out/'command.json.gz').write_bytes(gzip.compress((json.dumps(record,sort_keys=True,indent=2)+'\n').encode(),mtime=0))
print(json.dumps(dict(label=label,exit=run.returncode,leaves=len(leaves),failures=record['failures'],changed=changed,outputs=list(outputs))))
if run.returncode:print(run.stdout[-12000:]);print(run.stderr[-4000:])
sys.exit(run.returncode or bool(changed))
