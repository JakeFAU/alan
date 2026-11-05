#!/usr/bin/env python3
"""Check compact against original counters on the same source at B1/B8."""
import argparse
import hashlib
import json
import pathlib

METRICS = ['completed_prepares','invocation_attempts','step_calls','executions','sqlite_row_returns',
           'fullscan_steps','vm_steps','step_errors','bind_errors']


def comparable(original, compact):
    checks = 0
    for key, value in compact.items():
        if key.startswith('sql.scope_total.'):
            scope, metric = key[len('sql.scope_total.'):].rsplit('.', 1)
            expected = sum(v for k,v in original.items() if k.startswith('sql.scope.'+scope+'.class.') and k.endswith('.'+metric))
        elif key in original or key.startswith('sql.total.'):
            expected = original.get(key, 0)
        else:
            continue  # Explicitly new loop counters have no original counterpart.
        if expected != value:
            raise ValueError('counter mismatch '+key+': original='+str(expected)+' compact='+str(value))
        checks += 1
    return checks


def compare(root):
    results = []
    for bound in [1,8]:
        paths = [root / ('gate-'+kind+'-counters-b'+str(bound)) / 'stdout.json' for kind in ['original','compact']]
        original, compact = [json.loads(p.read_text()) for p in paths]
        if not original['invariants_ok'] or not compact['invariants_ok'] or original.get('error') or compact.get('error'):
            raise ValueError('independent public proof failed')
        checks = 0
        if len(original['stages']) != len(compact['stages']):
            raise ValueError('stages changed')
        for a,b in zip(original['stages'],compact['stages']):
            if (a['name'],a.get('position',0)) != (b['name'],b.get('position',0)):
                raise ValueError('stage order changed')
            checks += comparable(a.get('counter_deltas',{}), b.get('counter_deltas',{}))
        for a,b in zip(original['prefixes'],compact['prefixes']):
            for field in ['direct','cumulative']:
                checks += comparable(a[field].get('counter_deltas',{}),b[field].get('counter_deltas',{}))
        execute = next(s for s in compact['stages'] if s['name']=='execute_window')['counter_deltas']
        registry = compact['execute_counter_registry']
        for metric in METRICS:
            total = registry['sql.total.'+metric]
            scoped = sum(v for k,v in registry.items() if k.startswith('sql.scope.') and k.endswith('.'+metric))
            scoped += registry.get('sql.scope_overflow.'+metric,0)
            if total != scoped or total != execute['sql.total.'+metric]:
                raise ValueError('terminal Execute registry/scalar partition mismatch '+metric)
        if registry.get('sql.attribution.incomplete_events',0):
            raise ValueError('incomplete attribution')
        results.append({'bound':bound,'counter_equalities':checks,'independent_public_proofs':True,
                        'execute_registry_exact_partition':True,
                        'input_sha256':{str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in paths}})
    return {'ok':True,'results':results,
            'limitation':'Fresh Store namespaces differ; no same-seed raw-ID or durable-state equality claim.'}


if __name__ == '__main__':
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('root',type=pathlib.Path)
    a = p.parse_args()
    result = compare(a.root)
    (a.root / 'counter-equality.json').write_text(json.dumps(result,indent=2)+'\n')
    print(json.dumps(result,indent=2))
