#!/usr/bin/env python3
"""Extract the frozen baseline pilot without rerunning or relabeling samples."""
import argparse
import hashlib
import json
from pathlib import Path
import re


def counter_check(c):
    required = ('p3.calls', 'p3.eligible_visits', 'p3.distinct_programs', 'p3.reusable_visits', 'p3.load.success', 'p3.decode.calls', 'p3.validate.calls', 'p3.binding.calls', 'work.reduce.selectReady')
    if any(k not in c or type(c[k]) is not int or c[k] < 0 for k in required) or c['p3.calls'] == 0:
        raise ValueError('missing or invalid positive-workload counters')
    histogram = {k: v for k, v in c.items() if k.startswith('p3.distribution.')}
    calls = visits = distinct = selected = failures = 0
    for k, count in histogram.items():
        m = re.fullmatch(r'p3.distribution.(waiting|selected|error).visits_(\d+).distinct_(\d+)', k)
        if not m or type(count) is not int or count < 0:
            raise ValueError('invalid distribution bin')
        v, d = map(int, m.groups()[1:])
        if d > v or (v > 0 and d == 0):
            raise ValueError('impossible distribution bin')
        calls += count; visits += count*v; distinct += count*d
        selected += count*(m[1] == 'selected'); failures += count*(m[1] == 'error')
    expected = {'p3.calls': calls, 'p3.eligible_visits': visits,
                'p3.distinct_programs': distinct, 'p3.reusable_visits': visits-distinct}
    for k, v in expected.items():
        if c.get(k, 0) != v:
            raise ValueError('distribution disagrees with ' + k)
    program_visits = sum(v for k, v in c.items() if k.startswith('p3.program_id.') and k.endswith('.visits'))
    if program_visits != visits:
        raise ValueError('ProgramID registry visits disagree')
    # This extraction owns successful baseline workload diagnostics, not arbitrary failures.
    if failures or c.get('p3.load.failure', 0):
        raise ValueError('failed readiness/load in positive pilot')
    for k in ('p3.load.success', 'p3.decode.calls', 'p3.validate.calls', 'p3.binding.calls'):
        if c.get(k, 0) != visits:
            raise ValueError('successful baseline work disagrees with ' + k)
    if c.get('work.reduce.selectReady', 0) != selected:
        raise ValueError('successful baseline reducer/selection mismatch')
    return {'calls': calls, 'eligible_visits': visits, 'distinct_per_call_sum': distinct,
            'repeat_visits': visits-distinct, 'selected': selected, 'errors': failures,
            'histogram': histogram}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('raw', type=Path)
    p.add_argument('output', type=Path)
    a = p.parse_args()
    rows = json.loads((a.raw/'pilot-schedule.json').read_text())
    out = {'baseline_commit': '926a899b651fccc48be52a054612a92afcf1ddfd', 'samples': [], 'inputs': {}}
    for row in rows:
        path = a.raw/'pilot-results'/(row['name']+'.json')
        meta_path = path.with_name(row['name']+'.meta.json')
        r, meta = json.loads(path.read_text()), json.loads(meta_path.read_text())
        if meta['report_sha256'] != hashlib.sha256(path.read_bytes()).hexdigest() or meta['schedule_sha256'] != hashlib.sha256((a.raw/'pilot-schedule.json').read_bytes()).hexdigest():
            raise ValueError('report/schedule identity mismatch')
        for f in (path, meta_path):
            out['inputs'][str(f)] = hashlib.sha256(f.read_bytes()).hexdigest()
        if meta['exit'] != 0 or meta.get('validation_error') or not r['invariants_ok'] or r.get('error'):
            raise ValueError('failed pilot: '+row['name'])
        if meta['binary_sha256'] != row['expected_binary_sha256'] or meta.get('binary_sha256_after') != row['expected_binary_sha256']:
            raise ValueError('binary identity mismatch')
        b = row['bound']
        if r['reductions'] != 4*b-1 or any(r[k] != b for k in ('completed','selected','spent')) or r['protected_approvals'] != b-1 or len(r['contacts']) != b:
            raise ValueError('logical workload mismatch')
        window = next(s for s in r['stages'] if s['name'] == 'execute_window')
        item = {'name': row['name'], 'bound': b, 'execute': {k:v for k,v in window.items() if k != 'counter_deltas'},
                'process_peak_rss_bytes': meta['process_peak_rss'],
                'load_before': meta['load_before'], 'load_after': meta['load_after'],
                'phases': {s['name']: {k:v for k,v in s.items() if k != 'counter_deltas'} for s in r['stages'] if 'position' not in s},
                'binary_sha256': meta['binary_sha256'], 'argv': meta['command'], 'cwd': meta['cwd']}
        if row['name'].startswith('counters'):
            c = window['counter_deltas']
            item['readiness'] = counter_check(c)
            item['counters'] = {k:v for k,v in c.items() if k.startswith(('p3.', 'fn.', 'work.', 'sql.total.', 'sql.attribution.')) and not k.startswith('p3.distribution.')}
        if row['name'].startswith('coarse'):
            item['coarse'] = window['counter_deltas']
        out['samples'].append(item)
    with a.output.open('x') as f:
        json.dump(out, f, indent=2, sort_keys=True); f.write('\n')


if __name__ == '__main__':
    main()
