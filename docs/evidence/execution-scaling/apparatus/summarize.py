#!/usr/bin/env python3
"""Produce compact Task1 baseline metadata from retained serial rows."""
import argparse
import hashlib
import json
import pathlib
import shutil
import statistics

HERE = pathlib.Path(__file__).resolve().parent
KEYS = ['sql.total.executions','sql.total.sqlite_row_returns','sql.total.fullscan_steps','sql.total.vm_steps',
        'work.selectReady.runs_considered','work.selectReady.observations_considered','work.selectReady.bindings',
        'work.reduce.selectReady','work.reduce.Commit','work.settleRoots.roots_considered',
        'work.readRootCompletions.roots','work.finishRun.root_intent_pairs','work.pendingDeliveries.intents_considered']


def sha(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()


def artifact(p):
    return {'path':str(p),'sha256':sha(p),'bytes':p.stat().st_size}


def report(root, name):
    p=root/name/'stdout.json'
    x=json.loads(p.read_text())
    row=json.loads((p.parent/'row.json').read_text())
    if row['exit_code'] or row['validation_error'] or not x['invariants_ok'] or x.get('error'):
        raise ValueError('failed row '+name)
    execute=next(s for s in x['stages'] if s['name']=='execute_window')
    result={'bound':x['bound'],'row':artifact(p.parent/'row.json'),'output':artifact(p),
            'exit_code':row['exit_code'],'invariants_ok':x['invariants_ok'],'execute':execute}
    if x['instrumented']:
        counters=execute['counter_deltas']
        registry=x['execute_counter_registry']
        if registry.get('sql.attribution.incomplete_events',0):
            raise ValueError('incomplete registry '+name)
        trends={}
        for key in KEYS:
            total=counters[key]
            values=[p['direct']['counter_deltas'].get(key,0) for p in x['prefixes']]
            trends[key]={'total':total,'per_bound':total/x['bound'],'per_bound_squared':total/x['bound']**2,
                         'position_first':values[0],'position_median':statistics.median(values),'position_last':values[-1]}
        result['work_trends']=trends
        # Whole terminal family map stays in raw output, never duplicated here.
        result['execute']={k:v for k,v in execute.items() if k!='counter_deltas'}
        result['execute_attribution_complete']=True
        result['unknown_scope_executions']=sum(v for k,v in registry.items() if k.startswith('sql.scope.unknown.class.') and k.endswith('.executions'))
    return result


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('root',type=pathlib.Path)
    p.add_argument('output',type=pathlib.Path)
    a=p.parse_args();r=a.root
    if a.output.exists():
        p.error('output must be new')
    metadata={'source_commit':'d59eaf2976849e450635d252c7bca62eeccbef2e',
              'source_tree':'dc082ea07e4c24d8f36e56d756771f7c1154b0ec','raw_root':str(r),
              'purpose':'Descriptive Task1 baseline; excluded from confirmatory schedule',
              'source_archive':artifact(r/'baseline-source.tar'),
              'source_manifest':artifact(r/'baseline-source-manifest.json'),
              'baseline_test_summary':artifact(r/'baseline-test-summary.json'),
              'baseline_test_raw':[artifact(r/n) for n in ['baseline-tests.jsonl','baseline-tests.stderr','baseline-tests-permitted.jsonl','baseline-tests-permitted.stderr']],
              'initial_state':json.loads((r/'source-initial.json').read_text()),
              'pilots':[report(r,'pilot-original-b'+str(b)) for b in [1,32,128,256]],
              'diagnostics':[report(r,('gate-compact-counters-b' if b in [1,8] else 'diagnostic-compact-b')+str(b)) for b in [1,8,32,128,256]],
              'counter_equality':artifact(r/'counter-equality.json'),
              'schedule':artifact(r/'task1-schedule.json'),
              'controller':artifact(r/'task1-gates.py'),
              'tools':[artifact(q) for q in sorted(HERE.iterdir()) if q.is_file()],
              'binaries':[artifact(r/q) for q in ['original.bin','ordinary-final/driver.bin','original-counters/driver.bin','compact-counters-final/driver.bin']],
              'build_metadata':[artifact(r/q/'overlay.json') for q in ['ordinary-final','compact-counters-final']],
              'environment':{q.stem:q.read_text() for q in sorted((r/'environment').glob('*.stdout'))},
              'environment_commands':artifact(r/'environment/commands.json'),
              'native_tool_hashes':{name:artifact(pathlib.Path(shutil.which(name))) for name in ['go','clang','python3']},
              'failed_or_incomplete_attempts':artifact(r/'failed-attempts.json')}
    a.output.write_text(json.dumps(metadata,indent=2,sort_keys=True)+'\n')


if __name__=='__main__':
    main()
