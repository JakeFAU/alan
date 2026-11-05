#!/usr/bin/env python3
"""Strict complete P3 analysis; timings alone never establish semantic acceptance."""
import json
import datetime
from pathlib import Path
import statistics
import sys
from custody import bound_row, expected_custody, sha, snapshot, validate_plan, write
from run import validate_meta, validate_records
from validation import BODY_PHASES, RESOURCE_KEYS, require, validate_report


def summary(values):
    return {'values':values, 'median':statistics.median(values), 'min':min(values), 'max':max(values)}


def guard(before, after, absolute):
    return after - before > max(.1 * before, absolute)


def improvement_above_five_percent(baseline_ns, candidate_ns):
    # Integer nanosecond medians (possibly half-integers); avoid 0.95 rounding.
    return 20 * statistics.median(candidate_ns) < 19 * statistics.median(baseline_ns)


def compare(before, after):
    b, c = summary(before), summary(after)
    differences = [a-z for a,z in zip(before, after)]
    return {'baseline':b, 'candidate':c, 'baseline_minus_candidate':differences,
            'paired_candidate_over_baseline':summary([c/b for b,c in zip(before, after)]),
            'median_reduction':1-c['median']/b['median'], 'all_six_positive':len(differences)==6 and all(d>0 for d in differences)}


def phase(report, name, key='wall_ns'):
    return sum(s[key] for s in report['stages'] if s['name'] == name)


def analyze(plan, binding_hash):
    validate_plan(plan)
    require(snapshot(plan) == expected_custody(plan), 'current source/binary/tool custody differs')
    output = Path(plan['output'])
    rows = plan['design']['rows']
    expected_files = {r['name']+s for r in rows for s in ('.json','.progress','.meta.json','.start.json','.outcome.json','.store')}
    require({p.name for p in output.iterdir()} == expected_files, 'missing/extra/replaced campaign rows')
    results, artifacts = {}, []
    previous_end = None
    for original in rows:
        row = bound_row(plan, original)
        name = row['name']
        report = json.loads((output/(name+'.json')).read_text())
        meta = json.loads((output/(name+'.meta.json')).read_text())
        outcome = json.loads((output/(name+'.outcome.json')).read_text())
        start = json.loads((output/(name+'.start.json')).read_text())
        validate_records(meta,start,outcome)
        validate_report(report, row)
        validate_meta(meta, report, plan, row, binding_hash, sha(output/(name+'.json')), sha(output/(name+'.progress')))
        started = datetime.datetime.fromisoformat(meta['started_utc'])
        require(previous_end is None or started >= previous_end, 'processes overlap or violate frozen order')
        previous_end = datetime.datetime.fromisoformat(meta['ended_utc'])
        artifacts.append({s:sha(output/(name+s)) for s in ('.json','.progress','.meta.json','.start.json','.outcome.json')})
        if not row['excluded']:
            results[row['bound'],row['pair'],row['variant']] = (report,meta)
    require(len(results)==84, 'exactly 84 measured rows required')
    bounds=[]
    for n in (1,8,16,32,64,128,256):
        group = {v:[results[n,p,v] for p in range(1,7)] for v in ('A','B')}
        execute = compare(*[[phase(r,'execute_window')/1e9 for r,m in group[v]] for v in ('A','B')])
        metrics = {}
        for key in RESOURCE_KEYS:
            metrics[key] = {v:summary([phase(r,'execute_window',key) for r,m in group[v]]) for v in ('A','B')}
        metrics['process_peak_rss_bytes'] = {v:summary([m['process_peak_rss'] for r,m in group[v]]) for v in ('A','B')}
        for key in ('seconds','user_cpu_seconds','system_cpu_seconds'):
            metrics['process_'+key] = {v:summary([m[key] for r,m in group[v]]) for v in ('A','B')}
        metrics['runtime_metrics'] = {key:{v:summary([next(s for s in r['stages'] if s['name']=='execute_window')['runtime_metric_deltas'][key] for r,m in group[v]]) for v in ('A','B')} for key in group['A'][0][0]['runtime_metric_names']}
        phases = {}
        for name in BODY_PHASES:
            b,c = [summary([phase(r,name)/1e9 for r,m in group[v]]) for v in ('A','B')]
            phases[name] = {'baseline':b,'candidate':c,'guard_exceeded':guard(b['median'],c['median'],.001)}
        rss = metrics['process_peak_rss_bytes']
        bounds.append({'bound':n,'execute_seconds':execute,'metrics':metrics,'phases_seconds':phases,
                       'small_execute_guard_exceeded':n in (1,8,16,32) and guard(execute['baseline']['median'],execute['candidate']['median'],.001),
                       'rss_requires_resolution':guard(rss['A']['median'],rss['B']['median'],8*1024*1024),
                       'prefixes':{v:[r['prefixes'] for r,m in group[v]] for v in ('A','B')},
                       'prefix_residuals':{v:[r['prefix_residual'] for r,m in group[v]] for v in ('A','B')}})
    primary=bounds[-1]
    return {'version':'alan/performance-p3/analysis-v1','binding_sha256':binding_hash,'excluded_warmups':14,'measured_rows':84,
            'bounds':bounds,'artifacts':artifacts,
            'primary_timing_gate':improvement_above_five_percent(primary['metrics']['wall_ns']['A']['values'],primary['metrics']['wall_ns']['B']['values']) and primary['execute_seconds']['all_six_positive'],
            'primary_allocation_lower':primary['metrics']['total_alloc_bytes']['B']['median']<primary['metrics']['total_alloc_bytes']['A']['median'],
            'guard_violation':any(b['small_execute_guard_exceeded'] or any(p['guard_exceeded'] for p in b['phases_seconds'].values()) for b in bounds),
            'rss_resolution_required':any(b['rss_requires_resolution'] for b in bounds),
            'fair_six_sign_tail':1/64 if primary['execute_seconds']['all_six_positive'] else None,
            'inference':'Six independent fair null signs required for 1/64; shared host limits inference. Mechanism and semantic qualification remain separate acceptance gates.',
            'median_rule':'1 - median(candidate Execute) / median(baseline Execute) > 0.05; paired ratios reported separately',
            'resource_label_caveat':'Raw inherited outer resource_scope mentions HTTP. Actual validated direct transport is process-no-server, with zero server contacts. No raw label rewritten.'}


def main():
    binding=Path(sys.argv[1]); plan=json.loads(binding.read_text())
    write(sys.argv[2],analyze(plan,sha(binding)))


if __name__=='__main__':
    main()
