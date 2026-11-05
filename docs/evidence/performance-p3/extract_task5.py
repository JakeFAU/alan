#!/usr/bin/env python3
"""Extract bounded P3 diagnostics without treating instrumentation as primary timing."""
import argparse
import json
from pathlib import Path
import re
import statistics
import sys
sys.path.insert(0,str(Path(__file__).resolve().parent/'campaign'))
from custody import sha,write
from validation import validate_transport_proof,validate_report,TRANSPORT,require


def summary(values):
    return {'values':values,'median':statistics.median(values),'min':min(values),'max':max(values)}


def readiness(c):
    hist={k:v for k,v in c.items() if k.startswith('p3.distribution.')}
    calls=visits=distinct=selected=errors=0
    for key,count in hist.items():
        m=re.fullmatch(r'p3.distribution.(waiting|selected|error).visits_(\d+).distinct_(\d+)',key)
        require(m is not None and type(count) is int and count>=0,'invalid readiness histogram')
        v,d=map(int,m.groups()[1:]);require(0<=d<=v,'impossible distinct/visit count')
        calls+=count;visits+=count*v;distinct+=count*d;selected+=count*(m[1]=='selected');errors+=count*(m[1]=='error')
    require(calls==c['p3.calls'] and visits==c['p3.eligible_visits'] and distinct==c['p3.distinct_programs'],'readiness totals detached from histogram')
    ids={key.split('.')[2]:v for key,v in c.items() if key.startswith('p3.program_id.') and key.endswith('.visits')}
    require(sum(ids.values())==visits and c['p3.reusable_visits']==visits-distinct,'program ID visits mismatch')
    require(errors==0 and not c.get('p3.load.failure',0),'unexpected failed selector/load')
    return {'calls':calls,'eligible_visits':visits,'distinct_per_call_sum':distinct,'repeated_visits':visits-distinct,
            'selected':selected,'errors':errors,'program_id_visits':ids,'distribution':hist}


def main():
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('primary',type=Path);p.add_argument('diagnostics',type=Path);p.add_argument('output',type=Path);a=p.parse_args()
    primary=json.loads(a.primary.read_text());plan=json.loads((a.diagnostics/'protocol.json').read_text());binding=json.loads(Path(plan['primary_binding']).read_text())
    result={'scope':'primary pairs plus separately identified diagnostic evidence; no timing subtraction',
            'primary_analysis':{'path':str(a.primary),'sha256':sha(a.primary)},'primary':{k:v for k,v in primary.items() if k not in ('bounds','artifacts')},
            'bounds':[{k:v for k,v in b.items() if k not in ('prefixes','prefix_residuals')} for b in primary['bounds']],
            'mechanism':{},'controls':{},'profile':{}}
    for n in (1,8,128,256):
        pair={}
        for v in ('A','B'):
            row=next(r for r in plan['counter_rows'] if r['variant']==v and r['bound']==n)
            gate=a.diagnostics/'gates'/row['name'];path=gate/'stdout.log';r=json.loads(path.read_text())
            require(json.loads((gate/'outcome.json').read_text())['exit']==0 and r['invariants_ok'] and not r.get('error'),'failed counter row')
            require(r['instrumented'] and not r['profiled'] and r['transport']==TRANSPORT and r['server_contacts']==[],'wrong counter configuration')
            require(r['bound']==n and r['sample']==row['name'] and r['source_label']==binding['variants'][v]['commit'],'counter identity')
            require(all(r[k]==n for k in ('completed','selected','spent')) and r['protected_approvals']==n-1 and r['reductions']==4*n-1,'counter logical work mismatch')
            require(len(r['contacts'])==len(r['evidence'])==len(r['prefixes'])==n,'counter cardinality mismatch')
            require(all(p['position']==i and p['contacts']==i and p['server_contacts']==0 and p['cut']['completed']==i for i,p in enumerate(r['prefixes'],1)),'counter prefix work mismatch')
            validate_transport_proof(r,n)
            c=next(s for s in r['stages'] if s['name']=='execute_window')['counter_deltas'];ready=readiness(c)
            loads=ready['eligible_visits'] if v=='A' else ready['distinct_per_call_sum']
            require(all(c[k]==loads for k in ('p3.load.success','p3.decode.calls','p3.validate.calls')),'unexpected program load mechanism')
            require(c['p3.binding.calls']==ready['eligible_visits'] and c['work.reduce.selectReady']==c['work.reduce.Commit']==4*n-1,'remaining binding/reduction work changed')
            require(c['p3.observation.attempts']==c['p3.observation.selected']==2*n-1,'observation work changed')
            pair[v]={'report_sha256':sha(path),'readiness':ready,'loads':loads,
                     'counters':{k:v for k,v in c.items() if k.startswith(('p3.','work.','fn.','sql.total.','sql.attribution.')) and not k.startswith(('p3.distribution.','p3.program_id.'))}}
            del r,c
        require(pair['A']['readiness']==pair['B']['readiness'],'visited/distinct/selection distributions differ')
        eliminated=pair['A']['loads']-pair['B']['loads'];sql_a=pair['A']['counters']['sql.total.executions'];sql_b=pair['B']['counters']['sql.total.executions']
        require(sql_a-sql_b==2*eliminated,'unexplained SQL change')
        result['mechanism'][str(n)]={'variants':pair,'eliminated_loads':eliminated,'baseline_sql':sql_a,'candidate_sql':sql_b,'eliminated_sql':sql_a-sql_b}
    counts=[]
    for line in (a.diagnostics/'gates/control-counts/stdout.log').read_text().splitlines():
        if 'P3_CONTROL_COUNTER ' in line:counts.append(json.loads(line.split('P3_CONTROL_COUNTER ',1)[1]))
    require(len(counts)==3,'missing/extra counter controls')
    for case in counts:
        distinct=32 if case['case']=='distinct_waiting' else 4
        reductions=1 if case['case']=='shared_late_ready' else 0
        for v in ('baseline','candidate'):
            c=case[v];ids={k[3:]:v for k,v in c.items() if k.startswith('id:')}
            require(len(ids)==distinct and sum(ids.values())==c['eligible_visits']==32 and c['distinct_programs']==distinct,'actual control visit distribution mismatch')
            require(c['bindings']==32 and c.get('reductions',0)==reductions and c['loads']==(32 if v=='baseline' else distinct),'control mechanism mismatch')
        require({k:v for k,v in case['baseline'].items() if k.startswith('id:')}=={k:v for k,v in case['candidate'].items() if k.startswith('id:')},'control visited IDs differ')
        measured={v:[] for v in ('baseline','candidate')}
        for row in plan['controls']:
            if row['case']!=case['case']:continue
            gate=a.diagnostics/'gates'/row['name']
            lines=[line.split('P3_CONTROL_TIMING ',1)[1] for line in (gate/'stdout.log').read_text().splitlines() if 'P3_CONTROL_TIMING ' in line]
            require(len(lines)==1 and json.loads((gate/'outcome.json').read_text())['exit']==0,'control process failed')
            data=json.loads(lines[0]);require(data['case']==case['case'] and data['implementation']==row['implementation'] and data['iterations']==20 and data['excluded_calls']==1,'control protocol drift')
            measured[row['implementation']].append(data)
        require(all(len(v)==6 for v in measured.values()),'missing control pair')
        result['controls'][case['case']]={'counter_evidence':case,'samples':measured,
            'batch_ns':{v:summary([x['wall_ns'] for x in values]) for v,values in measured.items()},
            'batch_allocated_bytes':{v:summary([x['allocated_bytes'] for x in values]) for v,values in measured.items()},
            'paired_candidate_over_baseline':[c['wall_ns']/b['wall_ns'] for b,c in zip(measured['baseline'],measured['candidate'])]}
    profile_path=a.diagnostics/'gates'/plan['profile']['name']/'stdout.log';r=json.loads(profile_path.read_text())
    runtime=dict(binding['runtime'],profiled=True)
    validate_report(r,{'name':plan['profile']['name'],'bound':128,'source':binding['variants']['B']['commit'],'expected_runtime':runtime})
    result['profile']={'report_sha256':sha(profile_path),'cpu_sha256':sha(a.diagnostics/'candidate-b128.cpu'),'allocation_sha256':sha(a.diagnostics/'candidate-b128.alloc'),
                       'classification':'separate profiled candidate B128; no primary timing use'}
    write(a.output,result)


if __name__=='__main__':main()
