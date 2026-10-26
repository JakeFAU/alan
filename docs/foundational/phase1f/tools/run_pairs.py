#!/usr/bin/env python3
"""Run exactly one frozen, counterbalanced pair per question, retaining failures."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import time

ROOT=Path(__file__).resolve().parents[4]
DOC=ROOT/'docs/foundational/phase1f'

def digest(p):return hashlib.sha256(Path(p).read_bytes()).hexdigest()
def verify(manifest):
    for f in manifest['files']:
        raw=(ROOT/f['path']).read_bytes()
        assert len(raw)==f['bytes'] and hashlib.sha256(raw).hexdigest()==f['sha256'],f['path']

def main():
    parser=argparse.ArgumentParser()
    for name in ('binary','inputs','output','ledger','source-manifest','study'):
        parser.add_argument('--'+name,required=True)
    parser.add_argument('--credential-file')
    args=parser.parse_args()
    source=json.loads(Path(args.source_manifest).read_text());verify(source)
    source_id=source.get('source_sha256',source.get('files_sha256'))
    assert source_id
    inputs=json.loads(Path(args.inputs).read_text())
    assert len(inputs['cases'])==4
    assert len({c['id'] for c in inputs['cases']})==4
    for case in inputs['cases']:
        assert set(case)=={'id','question'} and isinstance(case['question'],str)
        assert re.fullmatch('[a-z]+-[0-9]+',case['id'])
    assert re.fullmatch('[a-z0-9-]+',args.study)
    out=Path(args.output).resolve();out.mkdir(parents=True,exist_ok=False)
    config=dict(study=args.study,model='gpt-6.1-sol',max_output_tokens=4096,
        max_response_bytes=65536,request_timeout='120s',max_revisions=1,
        topology='nested',baseline='simple evidence.Provision with shared reviewed helper prompt',
        application_source_sha256=source_id,binary_sha256=digest(args.binary),
        input_sha256=digest(args.inputs),corpus_freeze=digest(DOC/'evaluation/freeze-manifest.json'),
        case_order=[],run_plan_sha256=digest(DOC/'evaluation-run-plan.md'))
    for n,case in enumerate(inputs['cases']):
        for system in (('baseline','investigation') if n%2==0 else ('investigation','baseline')):
            config['case_order'].append(dict(case_id=case['id'],system=system))
    (out/'configuration.json').write_text(json.dumps(config,indent=2)+'\n')
    for item in config['case_order']:
        verify(source)
        case=next(c for c in inputs['cases'] if c['id']==item['case_id'])
        system=item['system'];runid=args.study+'-'+case['id']+'-'+system
        work=Path(tempfile.mkdtemp(prefix='alan-phase1f-'+runid+'-'))
        db=work/'run.sqlite';target=out/(case['id']+'-'+system)
        env={k:v for k,v in os.environ.items() if k not in ('OPENAI_API_KEY','ALAN_BARRIER')}
        cmd=[args.binary,'init','--db',str(db),'--question',case['question'],'--model','gpt-6.1-sol',
             '--corpus-manifest',str(DOC/'evaluation/corpus/manifest.json'),
             '--max-output-tokens','4096','--max-response-bytes','65536','--request-timeout','120s']
        cmd+=['--baseline'] if system=='baseline' else ['--nested','--max-revisions','1']
        start=time.monotonic()
        proc=subprocess.run(cmd,cwd=ROOT,env=env,capture_output=True,text=True)
        init=dict(command=cmd,exit_code=proc.returncode,elapsed_seconds=round(time.monotonic()-start,3),
                  stdout=proc.stdout,stderr=proc.stderr,case_id=case['id'],system=system,source=source_id)
        if proc.returncode:
            target.mkdir();(target/'init-command.json').write_text(json.dumps(init,indent=2)+'\n')
            raise RuntimeError('init failed; original outcome retained')
        live=[sys.executable,str(DOC/'tools/live_control.py'),'--binary',args.binary,'--db',str(db),
              '--output',str(target),'--ledger',args.ledger,'--run-id',runid]
        if args.credential_file:live+=['--credential-file',args.credential_file]
        elif 'OPENAI_API_KEY' in os.environ:env['OPENAI_API_KEY']=os.environ['OPENAI_API_KEY']
        print('START '+runid,flush=True)
        proc=subprocess.run(live,cwd=ROOT,env=env,capture_output=True,text=True)
        (target/'init-command.json').write_text(json.dumps(init,indent=2)+'\n')
        (target/'controller.stdout').write_text(proc.stdout);(target/'controller.stderr').write_text(proc.stderr)
        print(json.dumps(dict(run=runid,exit_code=proc.returncode,
            summary=json.loads((target/'summary.json').read_text()) if (target/'summary.json').exists() else None)),flush=True)
        if proc.returncode:raise RuntimeError('controller stopped; retained outcomes, no rerun')
        view=json.loads((target/'inspect.json').read_text());result=view['Result'] or {}
        def field(k):return result.get(k,{}).get('Text')
        print(json.dumps(dict(run=runid,disposition=field('disposition'),
            answer=field('answer'),revision_count=result.get('revision_count',{}).get('Number'))),flush=True)
    verify(source)
    print('PAIRS COMPLETE',flush=True)

if __name__=='__main__':main()
