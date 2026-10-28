#!/usr/bin/env python3
"""Task2-only command recorder; never touches the Phase1G source manifests."""
import datetime, hashlib, json, os, pathlib, subprocess, sys, time
root=pathlib.Path(__file__).resolve().parents[5]
label=sys.argv[1]
args=sys.argv[2:]
if args and args[0]=='--':args=args[1:]
if not args:raise SystemExit('command required')
evidence=pathlib.Path(__file__).resolve().parent
sources=set(subprocess.check_output(['git','ls-files','*.go','go.mod','go.sum'],cwd=root,text=True).splitlines())
for p in (root/'examples/adaptive').rglob('*'):
 if p.is_file():sources.add(str(p.relative_to(root)))
for p in (root/'docs/foundational/phase1h/evidence/fixtures').iterdir():
 if p.is_file():sources.add(str(p.relative_to(root)))
manifest={p:hashlib.sha256((root/p).read_bytes()).hexdigest() for p in sorted(sources)}
identity=hashlib.sha256(json.dumps(manifest,sort_keys=True,separators=(',',':')).encode()).hexdigest()
(evidence/(label+'.sources.json')).write_text(json.dumps({'sha256':identity,'files':manifest},indent=2)+'\n')
env=dict(os.environ)
env.pop('OPENAI_API_KEY',None)
env.update(GOTOOLCHAIN='local',GOPROXY='off',GOSUMDB='off',GOFLAGS='-mod=readonly',GOCACHE='/tmp/alan-phase1f-go-build')
record={'argv':args,'cwd':str(root),'source_sha256':identity,'head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip(),'env':{k:env[k] for k in ['GOTOOLCHAIN','GOPROXY','GOSUMDB','GOFLAGS','GOCACHE']},'OPENAI_API_KEY':'removed','started':datetime.datetime.now(datetime.timezone.utc).isoformat()}
start=time.monotonic()
with (evidence/(label+'.log')).open('wb') as out:
 completed=subprocess.run(args,cwd=root,env=env,stdout=out,stderr=subprocess.STDOUT)
record.update(exit_code=completed.returncode,seconds=time.monotonic()-start,finished=datetime.datetime.now(datetime.timezone.utc).isoformat(),log_sha256=hashlib.sha256((evidence/(label+'.log')).read_bytes()).hexdigest())
(evidence/(label+'.command.json')).write_text(json.dumps(record,indent=2)+'\n')
print(json.dumps(record))
raise SystemExit(completed.returncode)
