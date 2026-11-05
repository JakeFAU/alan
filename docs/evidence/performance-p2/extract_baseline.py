#!/usr/bin/env python3
"""Pure extraction/arithmetic on reviewed compact P2 summaries; no raw DB/report access."""
import csv
import argparse
import hashlib
import json
from pathlib import Path
import statistics
import sys

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--inputs', required=True, type=Path, help='Immutable original P2 scratch inputs')
parser.add_argument('--source', required=True, type=Path, help='Read-only reviewed source export')
parser.add_argument('--out', required=True, type=Path, help='New directory for JSON/CSV outputs')
parser.add_argument('--aux-out', required=True, type=Path, help='New directory for preview/hash outputs')
args = parser.parse_args()
scratch, repo = args.inputs.resolve(), args.source.resolve()
destination, auxiliary = args.out.resolve(), args.aux_out.resolve()
# Preflight every destination before any write. Source/input directories are
# read-only; output and auxiliary records must be explicitly separate and new.
for target in (destination, auxiliary):
    if target.exists():
        raise FileExistsError('output directory already exists: ' + str(target))
    if target.is_relative_to(scratch) or target.is_relative_to(repo):
        raise ValueError('outputs must be outside immutable inputs and source: ' + str(target))
if destination == auxiliary or destination.is_relative_to(auxiliary) or auxiliary.is_relative_to(destination):
    raise ValueError('output and auxiliary directories must be distinct and non-nested')
def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()
def read(path):
    return json.loads(Path(path).read_text())
def save(path, value):
    with Path(path).open('x') as out:
        json.dump(value, out, separators=(',', ':'), sort_keys=True, allow_nan=False)
        out.write('\n')

metrics = ('completed_prepares','invocation_attempts','step_calls','executions','sqlite_row_returns',
           'fullscan_steps','vm_steps','step_errors','bind_errors')
bounds = (1,8,16,32,64,128,256)
inputs, summary_inputs, summaries = {}, [], {}
for mode, expected in [('timing',29),('diagnostic',12)]:
    folder = scratch / ('baseline-' + mode + '-summary-initial')
    index = read(folder/'index.json')
    inputs[str((folder/'index.json').relative_to(scratch))] = sha(folder/'index.json')
    assert len(index['samples']) == expected
    assert index['schedule_sha256'] == sha(scratch/('baseline-' + mode + '-schedule.json'))
    for item in index['samples']:
        path = folder/item['summary']
        assert sha(path) == item['summary_sha256']
        row = read(path)
        assert row['input_sha256'] == item['input_sha256']
        assert row['sample'] == item['sample']
        summaries[row['sample']] = row
        summary_inputs.append(dict(sample=row['sample'],path=str(path.relative_to(scratch)),
                                   sha256=sha(path),input_sha256=row['input_sha256'],
                                   validation_scope=row['validation_scope'],
                                   binary_sha256=row['schedule_row']['expected_binary_sha256']))
assert len(summaries) == 41

# These are existing small records/text/manifests, never original DBs or large
# raw counter reports. Record the exact source identities for the report.
small_sources = [
 'baseline-record.json','baseline-source-before.json','baseline-before-measurements.json',
 'baseline-after-diagnostics-source.json','baseline-timing-verification.json',
 'baseline-timing-schedule.json','baseline-diagnostic-schedule.json',
 'baseline-measurement-plan.json','baseline-measurement-results/commands.json',
 'baseline-diagnostic-plan.json','baseline-diagnostic-command-results/commands.json',
 'task2b-build-plan.json','task2b-build-results/commands.json',
 'counter-round1/generation.json','counter-round1/build.sha256','counter-round1/build-info.txt',
 'counter-round1/build-platform.txt','counter-round1/toolchain.txt',
 'coarse-initial/timing-generation.json','coarse-initial/alan-original-manifest.json',
 'coarse-initial/alan-modified-manifest.json','coarse-initial/driver-original-manifest.json',
 'coarse-initial/driver-modified-manifest.json',
 'profile-tool-identities.json','baseline-profile-analysis/commands.json',
 'reviews/profile-attribution-supplement/commands-results.json',
 'reviews/source-cost-model.md','reviews/profile-attribution-initial.md',
 'reviews/task3-analysis-review.md','reviews/task3-analysis-data-review.md',
 'reviews/task3-analysis-data-verification.json','user-request.json',
]
for path in small_sources:
    inputs[path] = sha(scratch/path)
for folder in ('baseline-profile-analysis','reviews/profile-attribution-supplement'):
    for path in sorted((scratch/folder).glob('*.stdout')):
        inputs[str(path.relative_to(scratch))] = sha(path)
for path in ('docs/evidence/performance-p2/contract.md','docs/evidence/performance-p2/summarize_diagnostics.py',
             'docs/evidence/performance-p2/driver/main.go','docs/evidence/performance-p2/instrument/README.md',
             'internal/executor/composition_inspect.go','internal/executor/composition_facts.go'):
    inputs['repo:'+path] = sha(repo/path)
before = read(scratch/'baseline-before-measurements.json')['source_files']
after = read(scratch/'baseline-after-diagnostics-source.json')
assert before == after and len(before) == 427
profile_commands = read(scratch/'baseline-profile-analysis/commands.json')
supplement = read(scratch/'reviews/profile-attribution-supplement/commands-results.json')
assert len(profile_commands) == 32 and all(c['exit'] == 0 for c in profile_commands)
assert len(supplement) == 3 and all(c['exit_code'] == 0 for c in supplement)

def small_row(row):
    return dict(sample=row['sample'],bound=row['declaration_bound'],counts=row['counts'],
                execute_resources=row['execute']['resources'],process=row['process'],
                prefix_residual=row['prefix_residual']['resources'] if row['prefix_residual'] else None)
ordinary = [dict(small_row(summaries['ordinary-b'+str(b)]),
                 stages_by_name=summaries['ordinary-b'+str(b)]['stages_by_name']) for b in bounds]
counter = []
prefix_checks = []
marginals = []
for b in bounds:
    t, q = summaries['ordinary-b'+str(b)], summaries['counters-b'+str(b)]
    assert q['run_attribution']['attribution_complete'] is True
    assert len(t['prefixes']) == len(q['prefixes']) == b
    for k,(tp,qp) in enumerate(zip(t['prefixes'],q['prefixes']),1):
        assert tp['position'] == qp['position'] == k
        assert tp['declaration_bound'] == qp['declaration_bound'] == b
        expected = dict(direct=b*k,cumulative=b*k*(k+1)//2)
        for label in expected:
            pairs = qp[label]['function_work']['work.finishRun.root_intent_pairs']
            membership = qp[label]['sql']['finishRun_membership_families']
            assert len(membership)==1
            sql_count = membership[0]['metrics']['executions']
            assert pairs == sql_count == expected[label]
        tv, cv = tp['direct']['resources'], tp['cumulative']['resources']
        qd, qc = qp['direct']['sql']['totals'], qp['cumulative']['sql']['totals']
        marginals.append(dict(ordinary_sample=t['sample'],counter_sample=q['sample'],fixed_B=b,k=k,
                             ordinary_adjacent_wall_ns=tv['wall_ns'],ordinary_cumulative_wall_ns=cv['wall_ns'],
                             ordinary_adjacent_alloc_bytes=tv['total_alloc_bytes'],ordinary_cumulative_alloc_bytes=cv['total_alloc_bytes'],
                             ordinary_adjacent_user_cpu_ns=tv['user_cpu_ns'],ordinary_adjacent_system_cpu_ns=tv['system_cpu_ns'],
                             ordinary_adjacent_mallocs=tv['mallocs'],ordinary_adjacent_gc_count=tv['gc_count'],
                             counter_adjacent_executions=qd['executions'],counter_cumulative_executions=qc['executions'],
                             counter_adjacent_membership_executions=expected['direct'],counter_cumulative_membership_executions=expected['cumulative'],
                             **{'counter_adjacent_'+metric:qd[metric] for metric in metrics if metric!='executions'}))
    expected_total=b*b*(b+1)//2
    assert q['execute']['function_work']['work.finishRun.root_intent_pairs'] == expected_total
    assert q['execute']['sql']['finishRun_membership_families'][0]['metrics']['executions'] == expected_total
    prefix_checks.append(dict(bound=b,prefixes_checked=b,each_direct_matches_B_k=True,
                              each_cumulative_matches_B_k_kplus1_over2=True,total=expected_total))
    counter.append(dict(sample=q['sample'],bound=b,sql=q['execute']['sql'],function_work=q['execute']['function_work'],
                        attribution=q['run_attribution'],
                        stage_sql_totals={name:g['sql_totals'] for name,g in q['stages_by_name'].items()}))

bridge = []
for b in (32,256):
    rows=[small_row(summaries['direct-bridge-b'+str(b)+'-s'+str(s)]) for s in (1,2,3)]
    walls=[r['execute_resources']['wall_ns'] for r in rows]
    bridge.append(dict(bound=b,samples=rows,median_wall_ns=statistics.median(walls),min_wall_ns=min(walls),max_wall_ns=max(walls)))
calibration=[]
for b in (1,8,32,256):
    for pair in (1,2):
        on,off = [summaries['prefix-calibration-b'+str(b)+'-pair'+str(pair)+'-'+flag] for flag in ('on','off')]
        a,z=on['execute']['resources'],off['execute']['resources']
        calibration.append(dict(bound=b,pair=pair,order='on/off' if pair==1 else 'off/on',
                                on=small_row(on),off=small_row(off),
                                on_minus_off_wall_ns=a['wall_ns']-z['wall_ns'],on_over_off=a['wall_ns']/z['wall_ns'],
                                on_minus_off_alloc_bytes=a['total_alloc_bytes']-z['total_alloc_bytes']))
coarse=[]
for b in (128,256):
    row=summaries['coarse-b'+str(b)]
    coarse.append(dict(small_row(row),boundaries=row['execute']['coarse'],stages_by_name=row['stages_by_name']))
profiles=[dict(small_row(summaries['cpu-alloc-b'+str(b)]),
               stages_by_name=summaries['cpu-alloc-b'+str(b)]['stages_by_name']) for b in (32,128)]
normalizations=[]
for t,q in zip(ordinary,counter):
    b=t['bound']; ns=t['execute_resources']['wall_ns']; a=t['execute_resources']['total_alloc_bytes']; sql=q['sql']['totals']['executions']
    normalizations.append(dict(bound=b,wall_ns_over_B=ns/b,wall_ns_over_B2=ns/b**2,wall_ns_over_B3=ns/b**3,
                               alloc_bytes_over_B2=a/b**2,executions_over_B2=sql/b**2,executions_over_B3=sql/b**3))
load_values=[value for row in summaries.values() for field in ('load_before','load_after') for value in row['process'][field]]
data=dict(version='alan/performance-p2/baseline-attribution-v1',normalization='none',
          identities=dict(base='76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315',contract='34409c56a37a5f5917913ac56a4807479f145375',
                          workload='9d1209a8069a52fbcf71f2176061eef66d6b9fe8',probe='14ea2effa91ad0a1f101dcf59fd400717508dc4f',
                          analysis='0fd0f2fd78e23f86320b25befb3b650db840e305',source_files_unchanged=427),
          extraction=dict(script='task3-report-fix2/extract-v3.py',sha256=sha(__file__),
                          command=['python3',str(Path(__file__).resolve()),'--inputs',str(scratch),
                                   '--source',str(repo),'--out',str(destination),'--aux-out',str(auxiliary)]),
          source_input_sha256=inputs,summary_inputs=summary_inputs,
          ordinary_single_samples=ordinary,counter_diagnostics=counter,direct_contextual_bridge=bridge,
          prefix_calibration=calibration,coarse_diagnostics=coarse,profile_driver_intervals=profiles,
          prefix_recurrence_checks=prefix_checks,descriptive_normalizations=normalizations,
          shared_host_load_observed_min=min(load_values),shared_host_load_observed_max=max(load_values),
          notes=['Timing/SQL columns are different processes; no heavy-counter timing/allocation enters ordinary claims.',
                 'No statistical fit, candidate mechanism, overhead subtraction, or speedup prediction.',
                 'Input hashes refer to immutable external artifacts; no raw counter report or database was opened by this extraction.'])
preview=dict(ordinary=[dict(bound=t['bound'],wall_s=t['execute_resources']['wall_ns']/1e9,
                           cpu_s=(t['execute_resources']['user_cpu_ns']+t['execute_resources']['system_cpu_ns'])/1e9,
                           alloc_GB=t['execute_resources']['total_alloc_bytes']/1e9,mallocs=t['execute_resources']['mallocs'],
                           gc=t['execute_resources']['gc_count'],gc_pause_s=t['execute_resources']['gc_pause_ns']/1e9,
                           gc_cpu_s=t['execute_resources']['runtime_metric_deltas']['/cpu/classes/gc/total:cpu-seconds'],
                           rss_MiB=t['process']['process_peak_rss']/2**20,sql=q['sql']['totals']['executions'])
                        for t,q in zip(ordinary,counter)],
             top10=counter[-1]['sql']['top10'],sql_totals=counter[-1]['sql']['totals'],
             sql_scopes=counter[-1]['sql']['scopes'],sql_driver=counter[-1]['sql']['driver'],
             function_work=counter[-1]['function_work'],
             bridge=[{k:v for k,v in row.items() if k!='samples'} for row in bridge],
             calibration=[{k:v for k,v in row.items() if k not in ('on','off')} for row in calibration],
             coarse=[dict(bound=row['bound'],execute_wall_s=row['execute_resources']['wall_ns']/1e9,boundaries=row['boundaries']) for row in coarse],
             normalized=normalizations,prefixes_checked=sum(bounds),load_min=min(load_values),load_max=max(load_values),
             marginal_examples=[row for row in marginals if row['fixed_B'] in (32,128,256) and row['k'] in (1,8,32,128,256)])
# All input decoding/arithmetic finished. If a racing creator claims either
# destination, fail without deleting anyone else's files. An empty directory
# created by this invocation is removed if creating the second one fails.
destination.mkdir(parents=True, exist_ok=False)
try:
    auxiliary.mkdir(parents=True, exist_ok=False)
except OSError:
    destination.rmdir()
    raise
save(destination/'baseline-data.json',data)
with (destination/'baseline-marginals.csv').open('x',newline='') as out:
    writer=csv.DictWriter(out,fieldnames=list(marginals[0]),lineterminator='\n');writer.writeheader();writer.writerows(marginals)
save(auxiliary/'preview.json',preview)
output_hashes = {p.name:sha(p) for p in (destination/'baseline-data.json',destination/'baseline-marginals.csv')}
save(auxiliary/'output-hashes.json',output_hashes)
print(json.dumps({'rows':len(summaries),'prefixes_checked':sum(bounds),'outputs':output_hashes,
                  'out':str(destination),'aux_out':str(auxiliary)}))
