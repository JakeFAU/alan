"""Record one source-bound command; use a fresh output directory to reproduce."""
import datetime, hashlib, json, os, pathlib, subprocess, sys, time
root = pathlib.Path(__file__).resolve().parent
while not (root / 'go.mod').is_file():
    root = root.parent
out, label, evidence, *argv = sys.argv[1:]
out = pathlib.Path(out).resolve()
out.mkdir(parents=True, exist_ok=True)
target = out / (label + '.json')
if target.exists():
    raise SystemExit('refusing to overwrite command record: ' + str(target))
env = {k: v for k, v in os.environ.items() if not k.startswith('ALAN_') and k != 'OPENAI_API_KEY'}
pinned = dict(GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off', GOFLAGS='-mod=readonly', CGO_ENABLED='1', GOCACHE='/private/tmp/alan-phase1m-model-go-build')
if evidence != '-':
    pathlib.Path(evidence).mkdir(parents=True, exist_ok=True)
    pinned['ALAN_MODEL_EVIDENCE'] = str(pathlib.Path(evidence).resolve())
env.update(pinned)
files = subprocess.check_output(['rg','--files','-g','*.go','-g','go.mod','-g','go.sum'],cwd=root,text=True).splitlines()
sources = {p: hashlib.sha256((root/p).read_bytes()).hexdigest() for p in sorted(files)}
extra = {p: hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest() for p in argv if pathlib.Path(p).is_file() and p.endswith(('.go','.py'))}
start=time.monotonic()
p=subprocess.run(argv,cwd=root,env=env,text=True,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
events=[]
for line in p.stdout.splitlines():
    try: event=json.loads(line)
    except ValueError: continue
    if isinstance(event,dict): events.append(event)
runs=[e['Test'] for e in events if e.get('Action')=='run' and 'Test' in e]
leaves=[n for n in runs if not any(m.startswith(n+'/') for m in runs)]
record=dict(label=label,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),cwd=str(root),argv=argv,environment=pinned,removed_environment=['OPENAI_API_KEY','ALAN_*'],exit=p.returncode,elapsed_seconds=time.monotonic()-start,stdout=p.stdout,stderr=p.stderr,source_sha256=sources,extra_source_sha256=extra,test_runs=runs,selected_leaves=leaves,passes=[e['Test'] for e in events if e.get('Action')=='pass' and 'Test' in e],failures=[e['Test'] for e in events if e.get('Action')=='fail' and 'Test' in e],skips=[e['Test'] for e in events if e.get('Action')=='skip' and 'Test' in e])
with target.open('x') as f: json.dump(record,f,indent=2); f.write('\n')
print(json.dumps({k:record[k] for k in ['label','exit','elapsed_seconds','selected_leaves','failures','skips']},indent=2))
for e in events:
    if 'Output' in e and ('states=' in e['Output'] or 'pairs=' in e['Output']): print(e['Output'].rstrip())
if p.returncode: print(p.stdout); print(p.stderr)
sys.exit(p.returncode)
