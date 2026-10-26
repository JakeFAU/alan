#!/usr/bin/env python3
"""Record static gates and compare deterministic research payloads separately."""
from pathlib import Path
import hashlib, json, os, subprocess, sys
from comparison import compare_research, rows
ROOT=Path(__file__).resolve().parents[4]; DOC=ROOT/'docs/foundational/phase1e'
m=json.loads((DOC/'source-manifest.json').read_text()); source=m['source_sha256']
for f in m['files']:
    assert hashlib.sha256((ROOT/f['path']).read_bytes()).hexdigest()==f['sha256']
env=dict(os.environ,GOTOOLCHAIN='local',PYTHONDONTWRITEBYTECODE='1')
for k,v in dict(GOCACHE='/tmp/alan-phase1e-go-build',GOPROXY='off',GOSUMDB='off').items():env.setdefault(k,v)
commands=[]
for cmd in [[sys.executable,'-m','unittest','discover','-s','docs/foundational/phase1e/tools','-p','test_comparison.py','-v'],['go','vet','./...'],['go','vet','-tags','integration','./...'],['gofmt','-l','author','schema','runtime','effects','adapters','cmd','internal','research','examples'],['go','mod','verify']]:
    p=subprocess.run(cmd,cwd=ROOT,env=env,text=True,capture_output=True)
    commands.append(dict(Command=cmd,ExitCode=p.returncode,Stdout=p.stdout,Stderr=p.stderr))
    assert p.returncode==0,commands[-1]
    if cmd[0]=='gofmt':assert not p.stdout

baseline=rows(DOC/'evidence/baseline/final-normal.records.jsonl.gz')
comparisons=[]
for mode in ('normal','race'):
    current=rows(DOC/f'evidence/final-{mode}.records.jsonl.gz')
    events=rows(DOC/f'evidence/final-{mode}.go-test.jsonl.gz')
    comparisons.extend(compare_research(baseline,current,ROOT,source,mode,events))
summary=dict(SourceID=source,Commands=commands,ResearchComparison=comparisons,
    Note='Process/runtime records are checked by causal invariants in tests, not compared byte-for-byte or sorted.')
(DOC/'evidence/final-checks.json').write_text(json.dumps(summary,indent=2)+'\n')
print(json.dumps(summary,indent=2))
