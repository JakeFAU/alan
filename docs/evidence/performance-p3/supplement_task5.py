#!/usr/bin/env python3
"""Fixed 16/32/64 mechanism coverage supplement; never primary timing."""
import argparse
import json
from pathlib import Path
import sys
from diagnostics import Campaign
from extract_task5 import readiness
from custody import sha,write
from validation import require,validate_transport_proof,TRANSPORT

HERE=Path(__file__).resolve().parent


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action',choices=('freeze','run','extract'));parser.add_argument('root',type=Path)
    parser.add_argument('--input',type=Path);parser.add_argument('--output',type=Path)
    args=parser.parse_args();root=args.root.resolve();path=root/'coverage-supplement.json'
    if args.action=='freeze':
        c=Campaign(root)
        rows=[{'variant':v,'bound':n,'name':f'counter-b{n}-{v}'} for n in (16,32,64) for v in ('A','B')]
        commands=[[str(root/('counter-'+r['variant'])/'driver.bin'),'-bound',str(r['bound']),'-dir',str(root/(r['name']+'.store')),
                   '-sample',r['name'],'-source-label',c.primary['variants'][r['variant']]['commit'],'-mode=direct','-prefix=true'] for r in rows]
        plan={'version':'alan/performance-p3/coverage-supplement-v1',
              'rationale':'Controller-requested completion of the seven-bound mechanism matrix; fixed one A/B row per missing bound; no adaptive primary/control sampling or timing inference',
              'original_protocol_sha256':sha(root/'protocol.json'),'built_inputs_sha256':sha(root/'built-inputs.json'),
              'rows':rows,'commands':commands,'cwd':str(root),'environment':c.plan['environment'],
              'tools':{str(p):sha(p) for p in (Path(__file__).resolve(),HERE/'diagnostics.py',HERE/'extract_task5.py')}}
        write(path,plan);(root/'frozen-tools'/'supplement_task5.py').write_bytes(Path(__file__).read_bytes())
        print(sha(path));return
    plan=json.loads(path.read_text());require({p:sha(p) for p in plan['tools']}==plan['tools'],'supplement tool drift')
    require(sha(root/'protocol.json')==plan['original_protocol_sha256'] and sha(root/'built-inputs.json')==plan['built_inputs_sha256'],'original diagnostic inputs changed')
    c=Campaign(root)
    if args.action=='run':
        write(root/'supplement-start.json',{'protocol_sha256':sha(path),'rows':plan['rows']})
        for row,command in zip(plan['rows'],plan['commands']):c.run(row['name'],command,root)
        c.finish('counter-supplement');return
    result=json.loads(args.input.read_text())
    result['coverage_supplement']={'path':str(path),'sha256':sha(path),'classification':plan['rationale']}
    for n in (16,32,64):
        pair={}
        for v in ('A','B'):
            row=next(r for r in plan['rows'] if r['bound']==n and r['variant']==v)
            gate=root/'gates'/row['name'];raw=gate/'stdout.log';r=json.loads(raw.read_text())
            require(json.loads((gate/'outcome.json').read_text())['exit']==0 and r['invariants_ok'] and not r.get('error'),'failed supplemental row')
            require(r['sample']==row['name'] and r['bound']==n and r['source_label']==c.primary['variants'][v]['commit'],'supplement row identity')
            require(r['instrumented'] and not r['profiled'] and r['transport']==TRANSPORT and r['server_contacts']==[],'supplement configuration')
            require(all(r[k]==n for k in ('completed','selected','spent')) and r['protected_approvals']==n-1 and r['reductions']==4*n-1,'supplement logical work')
            require(len(r['contacts'])==len(r['evidence'])==len(r['prefixes'])==n,'supplement cardinality')
            require(all(p['position']==i and p['contacts']==i and p['server_contacts']==0 and p['cut']['completed']==i for i,p in enumerate(r['prefixes'],1)),'supplement prefixes')
            validate_transport_proof(r,n)
            counters=next(s for s in r['stages'] if s['name']=='execute_window')['counter_deltas'];ready=readiness(counters)
            loads=ready['eligible_visits'] if v=='A' else ready['distinct_per_call_sum']
            require(all(counters[k]==loads for k in ('p3.load.success','p3.decode.calls','p3.validate.calls')),'supplement load mechanism')
            require(counters['p3.binding.calls']==ready['eligible_visits'] and counters['work.reduce.selectReady']==counters['work.reduce.Commit']==4*n-1,'supplement remaining work')
            require(counters['p3.observation.attempts']==counters['p3.observation.selected']==2*n-1,'supplement observations')
            pair[v]={'report_sha256':sha(raw),'readiness':ready,'loads':loads,
                     'counters':{k:v for k,v in counters.items() if k.startswith(('p3.','work.','fn.','sql.total.','sql.attribution.')) and not k.startswith(('p3.distribution.','p3.program_id.'))}}
        require(pair['A']['readiness']==pair['B']['readiness'],'supplement visit distribution differs')
        eliminated=pair['A']['loads']-pair['B']['loads'];sql_a=pair['A']['counters']['sql.total.executions'];sql_b=pair['B']['counters']['sql.total.executions']
        require(sql_a-sql_b==2*eliminated,'supplement SQL mechanism')
        result['mechanism'][str(n)]={'variants':pair,'eliminated_loads':eliminated,'baseline_sql':sql_a,'candidate_sql':sql_b,'eliminated_sql':sql_a-sql_b}
    require(set(result['mechanism'])=={'1','8','16','32','64','128','256'},'incomplete seven-bound mechanism matrix')
    write(args.output,result)


if __name__=='__main__':main()
