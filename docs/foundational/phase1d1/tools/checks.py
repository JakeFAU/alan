#!/usr/bin/env python3
"""Record static gates and compare deterministic research payloads separately."""
from pathlib import Path
import gzip, hashlib, json, os, subprocess
ROOT=Path(__file__).resolve().parents[4]; DOC=ROOT/'docs/foundational/phase1d1'
m=json.loads((DOC/'source-manifest.json').read_text()); source=m['source_sha256']
for f in m['files']:
    assert hashlib.sha256((ROOT/f['path']).read_bytes()).hexdigest()==f['sha256']
env=dict(os.environ,GOTOOLCHAIN='local',GOCACHE='/tmp/alan-phase1d1-go-build',GOPROXY='off',GOSUMDB='off')
commands=[]
for cmd in [['go','vet','./...'],['gofmt','-l','cmd','internal','research'],['git','diff','--check'],['go','mod','verify']]:
    p=subprocess.run(cmd,cwd=ROOT,env=env,text=True,capture_output=True)
    commands.append(dict(Command=cmd,ExitCode=p.returncode,Stdout=p.stdout,Stderr=p.stderr))
    assert p.returncode==0,commands[-1]
    if cmd[0]=='gofmt':assert not p.stdout

def rows(path):return [json.loads(x) for x in gzip.decompress(path.read_bytes()).splitlines()]
baseline=rows(DOC/'evidence/baseline/normal.records.jsonl.gz')
comparisons=[]
for mode in ('normal','race'):
    current=rows(DOC/f'evidence/final-{mode}.records.jsonl.gz')
    for marker in ('QUALIFICATION','COMPOSITION','SUPERVISION'):
        a=[x['Evidence'] for x in baseline if x['Marker']==marker]
        b=[x['Evidence'] for x in current if x['Marker']==marker]
        assert a==b, (mode,marker)
        comparisons.append(dict(Mode=mode,Marker=marker,Records=len(a),Equal=True,
            Scope='Exact ordered deterministic payload equality against reproduced6effdc0 baseline; wrapper source identity differs. No normalization.'))
    a=[x['Evidence'] for x in baseline if x['Marker']=='CONFORMANCE']
    b=[x['Evidence'] for x in current if x['Marker']=='CONFORMANCE']
    assert len(a)==len(b)==6 and all(x.get('passed') is True for x in b)
    assert [(x['test'],x['fixture']) for x in a]==[(x['test'],x['fixture']) for x in b]
    comparisons.append(dict(Mode=mode,Marker='CONFORMANCE',Records=6,AssertionsPass=True,
        ExactPayloads=sum(x==y for x,y in zip(a,b)),
        Scope='Six existing independent checkpoint comparisons pass; ephemeral endpoint URLs/program IDs in two fixtures vary. Raw records preserved, no equality claim for those fixtures.'))
summary=dict(SourceID=source,Commands=commands,ResearchComparison=comparisons,
    Note='Process/runtime records are checked by causal invariants in tests, not compared byte-for-byte or sorted.')
(DOC/'evidence/final-checks.json').write_text(json.dumps(summary,indent=2)+'\n')
print(json.dumps(summary,indent=2))
