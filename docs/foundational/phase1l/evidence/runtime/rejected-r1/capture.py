#!/usr/bin/env python3
"""Capture source-bound C_inv runtime qualification; never defines its oracle."""
from pathlib import Path
import datetime, gzip, hashlib, json, os, subprocess, sys, tempfile, time

ROOT = Path(__file__).resolve().parents[5]
DEST = Path(__file__).resolve().parent
PINNED = dict(GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off', GOFLAGS='-mod=readonly', GOCACHE='/tmp/alan-phase1f-go-build')
sha = lambda raw: hashlib.sha256(raw).hexdigest()
paths = sorted(set([ROOT/'AGENTS.md', ROOT/'go.mod', ROOT/'go.sum'] + [p for directory in ['internal/executor','internal/boundary','internal/substitutiontest','runtime','author','schema','examples/substitution','examples/join','examples/invocationjoin','research/substitution'] for p in (ROOT/directory).rglob('*.go')] + [ROOT/'examples/invocationjoin/consumer/go.mod',ROOT/'examples/invocationjoin/consumer/go.sum']))
protected = json.loads((ROOT/'docs/foundational/phase1l/evidence/baseline/protected-source.json').read_text())
paths=sorted(set(paths+[ROOT/r['path'] for r in protected['files']]))
sources = {str(p.relative_to(ROOT)):sha(p.read_bytes()) for p in paths}
source_digest = sha(json.dumps(sources,sort_keys=True,separators=(',',':')).encode())
protected = json.loads((ROOT/'docs/foundational/phase1l/evidence/baseline/protected-source.json').read_text())
assert all(sha((ROOT/r['path']).read_bytes()) == r['sha256'] for r in protected['files']), 'inherited source changed'
commands=[]; profiles={}
bundle=dict(scope='Concrete automatic samples, selected public schedules, controls and one recovery predicate; separate from finite-model census.', started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),source_aggregate_sha256=source_digest,commands=commands,profiles=profiles)
selection='^TestInvocationJoin(Runtime|Controls|Orders|Decoder|IncompletePrefixes|Recovery)$'
with tempfile.TemporaryDirectory(prefix='alan-phase1l-runtime-',dir='/private/tmp') as temporary:
    env={k:v for k,v in os.environ.items() if not k.startswith('ALAN_') and k!='OPENAI_API_KEY'};env.update(PINNED)
    def run(label,argv,profile=None):
        settings=dict(PINNED)
        if profile:
            artifacts=Path(temporary)/profile;artifacts.mkdir();settings['ALAN_I_ARTIFACTS']=str(artifacts)
        callenv=dict(env,**settings)
        print('RUN',label,flush=True);start=time.monotonic()
        p=subprocess.run(argv,cwd=ROOT,env=callenv,capture_output=True,text=True)
        events=[]
        for line in p.stdout.splitlines():
            try: event=json.loads(line)
            except ValueError: continue
            if 'Action' in event:events.append(event)
        selected=[e['Test'] for e in events if e.get('Action')=='run' and 'Test' in e]
        leaves=[case for case in selected if not any(other.startswith(case+'/') for other in selected)]
        record=dict(label=label,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),cwd=str(ROOT),argv=argv,environment_override=settings,environment_removed=['OPENAI_API_KEY','ALAN_*'],source_aggregate_sha256=source_digest,exit_code=p.returncode,seconds=round(time.monotonic()-start,6),selected_cases=selected,selected_leaves=leaves,test_results=[{k:e[k] for k in ['Action','Package','Test','Elapsed'] if k in e} for e in events if e.get('Action') in ['pass','fail','skip']],stdout=p.stdout,stderr=p.stderr)
        commands.append(record)
        if profile:
            records={}
            for f in sorted(artifacts.rglob('*')):
                if not f.is_file():continue
                key=str(f.relative_to(artifacts))
                if f.name.startswith('program-'):continue  # exact bytes already in each decisive raw Inspection
                records[key]=json.loads(gzip.decompress(f.read_bytes())) if f.suffix=='.gz' else [json.loads(line) for line in f.read_text().splitlines()]
            profiles[profile]=records
        print('EXIT',p.returncode,'leaves',len(leaves),'seconds',record['seconds'],flush=True)
        if p.returncode:raise RuntimeError(label)
    try:
        run('toolchain',['go','env','-json','GOVERSION','GOOS','GOARCH','CGO_ENABLED','GOTOOLCHAIN','GOFLAGS','GOPROXY','GOSUMDB','GOCACHE'])
        run('normal',['go','test','-json','-tags','integration','-run',selection,'-count=1','-timeout=10m','./internal/substitutiontest'],'normal')
        run('race',['go','test','-json','-race','-tags','integration','-run',selection,'-count=1','-timeout=15m','./internal/substitutiontest'],'race')
        if '--focused' not in sys.argv:
            run('public-normal',['go','-C','examples/invocationjoin/consumer','test','-json','-tags','integration','-count=1','./...'])
            run('public-race',['go','-C','examples/invocationjoin/consumer','test','-json','-race','-tags','integration','-count=1','./...'])
            run('vet',['go','vet','-tags','integration','./internal/substitutiontest'])
            run('default-suite',['go','test','-json','-count=1','-timeout=10m','./...'])
    finally:
        assert sources == {str(p.relative_to(ROOT)):sha(p.read_bytes()) for p in paths}, 'source changed during qualification'
        bundle['completed_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat()
        raw=json.dumps(bundle,separators=(',',':')).encode()
        with (DEST/'records.json.gz').open('wb') as f:f.write(gzip.compress(raw,mtime=0))
        (DEST/'sources.json').write_text(json.dumps(dict(aggregate_sha256=source_digest,files=sources),indent=2)+'\n')
        files={p.name:dict(sha256=sha(p.read_bytes()),bytes=p.stat().st_size) for p in [DEST/'records.json.gz',DEST/'sources.json',Path(__file__).resolve()]}
        index=dict(scope=bundle['scope'],source_aggregate_sha256=source_digest,files=files,commands=[{k:c[k] for k in ['label','argv','exit_code','seconds','selected_leaves']} for c in commands],profiles={k:dict(record_count=len(v),record_keys=list(v)) for k,v in profiles.items()},inherited_source_unchanged=True)
        (DEST/'index.json').write_text(json.dumps(index,indent=2)+'\n')
