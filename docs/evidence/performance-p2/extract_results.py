#!/usr/bin/env python3
"""Read-only P2 result extraction; never execute workloads or open stores.

Sampling rule fixed before extraction: use pair1-candidate-B (the first measured
pair, never a warmup) for ordinary adjacent prefixes at every fixed B. Use the
separate counters-bB process for SQL. All 505 positions are emitted, without
selecting intervals by their observed latency. Six-pair summaries remain primary.
"""
import argparse
import csv
import hashlib
import json
from pathlib import Path
import sys


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--inputs', required=True, type=Path)
    parser.add_argument('--source', required=True, type=Path)
    parser.add_argument('--out', required=True, type=Path)
    args = parser.parse_args()
    inputs, source, out = (p.resolve() for p in (args.inputs, args.source, args.out))
    if not inputs.is_dir() or not source.is_dir():
        raise ValueError('inputs and source must be existing read-only directories')
    if out.exists() or any(out == p or p in out.parents or out in p.parents for p in (inputs, source)):
        raise ValueError('output must be fresh and disjoint from inputs and source')
    hashes = {}

    def raw(path):
        path = path.resolve()
        content = path.read_bytes()
        hashes[str(path)] = hashlib.sha256(content).hexdigest()
        return content

    def read(path):
        return json.loads(raw(path))

    def require(condition, message):
        if not condition:
            raise ValueError(message)

    analysis = read(inputs / 'paired-analysis.json')
    schedule_path = inputs / 'paired-schedule.json'
    schedule = read(schedule_path)
    require(hashes[str(schedule_path)] == analysis['schedule_sha256'], 'paired schedule hash')
    require(len(schedule) == len(analysis['validated_artifacts']) == 98, '98 validated rows')
    require(sum(bool(r['warmup']) for r in schedule) == analysis['warmups_excluded'] == 14, '14 warmups')
    validated = {r['name']: r for r in analysis['validated_artifacts']}
    # Hash/verify retained raw timing inputs without opening any SQLite store.
    for row in schedule:
        for suffix, field in (('.json', 'report_sha256'), ('.progress', 'progress_sha256'),
                              ('.meta.json', 'meta_sha256')):
            path = inputs / 'paired-results' / (row['name'] + suffix)
            raw(path)
            require(hashes[str(path)] == validated[row['name']][field], 'paired raw hash: ' + str(path))

    def summary(folder, sample):
        index_path = inputs / folder / 'index.json'
        index = read(index_path)
        item = next(r for r in index['samples'] if r['sample'] == sample)
        path = inputs / folder / item['summary']
        row = read(path)
        require(hashes[str(path)] == item['summary_sha256'], 'summary hash: ' + str(path))
        require(row['input_sha256'] == item['input_sha256'], 'summary lineage')
        return row

    bounds = (1, 8, 16, 32, 64, 128, 256)
    paired = []
    for row in analysis['bounds']:
        compact = {k: v for k, v in row.items() if k != 'metrics'}
        compact['metrics'] = {variant: {k: v for k, v in values.items() if k != 'prefix_vectors'}
                              for variant, values in row['metrics'].items()}
        paired.append(compact)
    counters, marginals, normalizations = [], [], []
    family_ids = {}

    def family_id(family):
        if family not in family_ids:
            family_ids[family] = len(family_ids)
        return family_ids[family]

    def membership(interval):
        families = interval['sql']['finishRun_membership_families']
        require(len(families) == 1, 'one membership family')
        return families[0]['metrics']['executions']

    def sql_compact(sql):
        return {k: sql[k] for k in ('totals', 'driver', 'scopes', 'unknown_scope',
                                    'attribution_events', 'attribution_complete',
                                    'known_semantic_scope_attribution_complete')} | {
            'families': [{'family_id': family_id(r['family']), 'metrics': r['metrics']}
                         for r in sql['families'] if any(r['metrics'].values())],
            'zero_metric_registry_families_omitted': True,
            'top10': [{'family_id': family_id(r['family']), 'metrics': r['metrics']} for r in sql['top10']],
            'finishRun_membership_families': [{'family_id': family_id(r['family']), 'metrics': r['metrics']}
                                            for r in sql['finishRun_membership_families']]}

    for b in bounds:
        baseline = summary('baseline-diagnostic-summary-initial', 'counters-b' + str(b))
        candidate = summary('candidate-diagnostic-summary', 'counters-b' + str(b))
        ordinary = read(inputs / 'paired-results' / ('pair1-candidate-' + str(b) + '.json'))
        require(len(baseline['prefixes']) == len(candidate['prefixes']) == len(ordinary['prefixes']) == b,
                'complete prefixes')
        for k, (bp, cp, tp) in enumerate(zip(baseline['prefixes'], candidate['prefixes'], ordinary['prefixes']), 1):
            require(bp['position'] == cp['position'] == tp['position'] == k, 'prefix position')
            require(bp['declaration_bound'] == cp['declaration_bound'] == tp['declaration_bound'] == b,
                    'fixed declaration bound')
            for label, old, new in (('direct', b*k, k), ('cumulative', b*k*(k+1)//2, k*(k+1)//2)):
                require(membership(bp[label]) == old and membership(cp[label]) == new, 'membership recurrence')
                require(bp[label]['function_work']['work.finishRun.root_intent_pairs'] ==
                        cp[label]['function_work']['work.finishRun.root_intent_pairs'] == old, 'unchanged pair loop')
            direct, cumulative = tp['direct'], tp['cumulative']
            marginals.append(dict(fixed_B=b, k=k, ordinary_sample=ordinary['sample'],
                counter_sample=candidate['sample'], ordinary_adjacent_wall_ns=direct['wall_ns'],
                ordinary_cumulative_wall_ns=cumulative['wall_ns'],
                ordinary_adjacent_alloc_bytes=direct['total_alloc_bytes'],
                ordinary_cumulative_alloc_bytes=cumulative['total_alloc_bytes'],
                ordinary_adjacent_user_cpu_ns=direct['user_cpu_ns'],
                ordinary_adjacent_system_cpu_ns=direct['system_cpu_ns'],
                ordinary_adjacent_mallocs=direct['mallocs'], ordinary_adjacent_gc_count=direct['gc_count'],
                counter_adjacent_executions=cp['direct']['sql']['totals']['executions'],
                counter_cumulative_executions=cp['cumulative']['sql']['totals']['executions'],
                counter_adjacent_membership_executions=k,
                counter_cumulative_membership_executions=k*(k+1)//2,
                counter_unchanged_adjacent_root_intent_pairs=b*k))
        oldsql, newsql = (r['execute']['sql'] for r in (baseline, candidate))
        removed = (b-1)*b*(b+1)//2
        require(oldsql['totals']['executions'] - newsql['totals']['executions'] == removed,
                'total SQL reduction exactly membership reduction')
        membership_family = newsql['finishRun_membership_families'][0]['family']
        old_families = {r['family']: r['metrics'] for r in oldsql['families']}
        new_families = {r['family']: r['metrics'] for r in newsql['families']}
        require(old_families.keys() == new_families.keys(), 'same SQL family registry')
        require(all(old_families[f] == new_families[f] for f in old_families if f != membership_family),
                'all nonmembership family metrics unchanged')
        require(baseline['counts'] == candidate['counts'], 'same semantic counts')
        differences = {k: {'baseline': v, 'candidate': candidate['execute']['function_work'][k]}
                       for k, v in baseline['execute']['function_work'].items()
                       if candidate['execute']['function_work'][k] != v}
        require(all(k.endswith('.text_blob_bytes') for k in differences), 'only dynamic encoded byte differences')
        counters.append(dict(bound=b, baseline=sql_compact(oldsql), candidate=sql_compact(newsql),
            counts=candidate['counts'], function_work=candidate['execute']['function_work'],
            function_work_differences=differences, all_other_function_work_exact=True,
            prefixes_checked=b, removed_membership_executions=removed,
            ordinary_sample=ordinary['sample'], ordinary_prefix_residual=ordinary['prefix_residual'],
            stages_by_name={name: {k: stage[k] for k in ('count', 'sql_totals') if k in stage}
                            for name, stage in candidate['stages_by_name'].items()},
            attribution=candidate['run_attribution']))
        paired_row = next(r for r in paired if r['bound'] == b)
        for variant, sql in (('baseline', oldsql), ('candidate', newsql)):
            t = paired_row['metrics'][variant]['execute_seconds']['median']
            a = paired_row['metrics'][variant]['allocated_bytes']['median']
            q = sql['totals']['executions']
            normalizations.append(dict(bound=b, variant=variant, seconds_over_B=t/b,
                seconds_over_B2=t/b**2, seconds_over_B3=t/b**3,
                alloc_bytes_over_B2=a/b**2, sql_over_B2=q/b**2, sql_over_B3=q/b**3))
    require(len(marginals) == 505, '505 validated adjacent intervals')
    coarse = []
    for b in (128, 256):
        for variant, folder in (('baseline', 'baseline-diagnostic-summary-initial'),
                                ('candidate', 'candidate-diagnostic-summary')):
            row = summary(folder, 'coarse-b' + str(b))
            coarse.append(dict(bound=b, variant=variant, sample=row['sample'],
                execute_resources=row['execute']['resources'], boundaries=row['execute']['coarse']))
    profile_driver_intervals = []
    for sample in ('cpu-alloc-b32', 'cpu-alloc-b128', 'trace-b16'):
        row = summary('candidate-diagnostic-summary', sample)
        profile_driver_intervals.append(dict(sample=sample, source_label=row['source_label'],
            schedule_row=row['schedule_row'], execute_resources=row['execute']['resources'],
            process=row['process'], profiled=row['profiled'], counter_mode=row['counter_mode']))

    command_records = {}
    for folder in ('paired-command-results', 'paired-analysis-results', 'candidate-post-analysis-results',
                   'candidate-profile-remaining-results', 'candidate-trace-path-correction-results',
                   'baseline-profile-analysis'):
        path = inputs / folder / 'commands.json'
        commands = read(path)
        command_records[folder] = [{k: c[k] for k in ('name', 'argv', 'cwd', 'exit',
            'controller_sha256', 'plan_sha256', 'stdout_sha256', 'stderr_sha256') if k in c} for c in commands]
        for log in sorted((inputs / folder).glob('*')):
            if log.suffix in ('.stdout', '.stderr'):
                raw(log)
        for command in commands:
            for suffix in ('stdout', 'stderr'):
                path = inputs / folder / (command['name'] + '.' + suffix)
                require(hashes[str(path)] == command[suffix + '_sha256'], 'command output hash: ' + str(path))
    # Hash source used to interpret the mechanism, plus the extractor itself.
    for name in ('contract.md', 'invariants.md', 'driver/main.go', 'driver/resources.go',
                 'analyze_pairs.py', 'campaign_validation.py', 'summarize_diagnostics.py'):
        raw(source / 'docs/evidence/performance-p2' / name)
    raw(source / 'internal/executor/composition_inspect.go')
    raw(source / 'internal/executor/runner.go')
    raw(Path(__file__))
    for name in ('profile-tool-identities.json', 'candidate-diagnostic-schedule.json',
                 'candidate-post-analysis-plan.json', 'candidate-profile-remaining-plan.json',
                 'candidate-trace-path-correction-plan.json', 'candidate-initial-qualification-summary.json',
                 'task5-direct-campaign/summary.json', 'task5-candidate-archive-campaign/summary.json',
                 'task5-external/summary.json', 'reviews/task5-recovery-review.md',
                 'reviews/task5-http-tool-review.md', 'reviews/task5-http-tool-rereview.md',
                 'reviews/task5-http-tool-final-review.md', 'postcampaign-python-identity.json'):
        raw(inputs / name)
    data = dict(version='alan/performance-p2/results-v1',
        identities=dict(base='76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315',
            implementation='2fe47cb75f57bf3f7f79779dc1841c9b9b8f28d0',
            contract='34409c56a37a5f5917913ac56a4807479f145375',
            http_differential_tools='d11e896c7b6efebd427f577613664d10b919417e'),
        extraction=dict(argv=[sys.executable, str(Path(__file__).resolve()), '--inputs', str(inputs),
            '--source', str(source), '--out', str(out)], sampling_rule=__doc__),
        paired_metadata={k: v for k, v in analysis.items() if k not in ('bounds', 'descriptive_growth')},
        paired_bounds=paired, counter_comparison=counters, coarse_diagnostics=coarse,
        candidate_profile_driver_intervals=profile_driver_intervals,
        sql_family_registry=list(family_ids),
        prefixes_checked=505, descriptive_normalizations=normalizations,
        analysis_commands=command_records, source_input_sha256=hashes,
        notes=['Ordinary timing, SQL counters, coarse timers, CPU/allocation profiles and trace are separate processes.',
               'No fit or unrestricted asymptotic, per-query latency, or exact protocol lower bound is asserted.',
               'All initial nonzero profile-analysis exits remain in analysis_commands; path corrections use the same trace.',
               'The draft PR evidence record binds the final delivered tree and its qualification command exits.'])
    out.mkdir(parents=True, exist_ok=False)
    with (out / 'results-data.json').open('x', encoding='utf-8') as stream:
        json.dump(data, stream, sort_keys=True, separators=(',', ':'), allow_nan=False)
        stream.write('\n')
    with (out / 'candidate-marginals.csv').open('x', encoding='utf-8', newline='') as stream:
        writer = csv.DictWriter(stream, fieldnames=list(marginals[0]), lineterminator='\n')
        writer.writeheader()
        writer.writerows(marginals)
    print(json.dumps(dict(out=str(out), prefixes=505, hashes={p.name: hashlib.sha256(p.read_bytes()).hexdigest()
                     for p in sorted(out.iterdir())})))


if __name__ == '__main__':
    main()
