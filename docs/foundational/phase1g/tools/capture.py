#!/usr/bin/env python3
"""Retain an individual development/auxiliary command without replacing earlier attempts."""
from pathlib import Path
import datetime, gzip, hashlib, json, os, platform, subprocess, sys, time
ROOT=Path(__file__).resolve().parents[4]
name=sys.argv[1]; command=sys.argv[2:]
OUT=ROOT/'docs/foundational/phase1g/evidence/task4';OUT.mkdir(exist_ok=True)
assert not (OUT/(name+'.command.json')).exists(), 'refuse to replace evidence'
env=dict(os.environ);env.pop('OPENAI_API_KEY',None)
env.update(GOTOOLCHAIN='local',GOPROXY='off',GOSUMDB='off',GOCACHE='/tmp/alan-phase1f-go-build',GOFLAGS='-mod=readonly')
def source():
 paths=sorted(p for p in ROOT.rglob('*') if p.is_file() and (p.suffix=='.go' or p.name in ('go.mod','go.sum')) and '.git' not in p.parts)
 return {str(p.relative_to(ROOT)):hashlib.sha256(p.read_bytes()).hexdigest() for p in paths}
before=source();start=time.monotonic();started=datetime.datetime.now(datetime.timezone.utc).isoformat()
p=subprocess.run(command,cwd=ROOT,env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
after=source();raw=p.stdout;err=p.stderr
(OUT/(name+'.stdout.gz')).write_bytes(gzip.compress(raw,mtime=0));(OUT/(name+'.stderr.gz')).write_bytes(gzip.compress(err,mtime=0))
meta=dict(command=command,directory=str(ROOT),started=started,seconds=round(time.monotonic()-start,3),exit=p.returncode,platform=platform.platform(),environment={k:env[k] for k in ('GOTOOLCHAIN','GOPROXY','GOSUMDB','GOCACHE','GOFLAGS')},source_before=before,source_after=after,source_unchanged=before==after,stdout_sha256=hashlib.sha256(raw).hexdigest(),stderr_sha256=hashlib.sha256(err).hexdigest())
(OUT/(name+'.command.json')).write_text(json.dumps(meta,indent=2)+'\n');print(json.dumps({k:v for k,v in meta.items() if not k.startswith('source_')},indent=2));print(raw.decode()[-2000:]);print(err.decode()[-2000:]);sys.exit(p.returncode)
