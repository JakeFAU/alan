"""Frozen structural comparisons and strict adaptation of the existing P3 proof."""
from collections import defaultdict
from copy import deepcopy
import math
from statistics import median

from custody import P3, load
from compare_counters import METRICS, validated_prefixes

BOUNDS = (1, 8, 16, 32, 64, 128, 256)
TUPLES = (('baseline', 'snapshot'), ('candidate', 'snapshot'), ('candidate', 'targeted'))
EXPERIMENTS = (('structural', TUPLES[:2]), ('targeted', TUPLES[1:]))
PROOF = load('task5_public_proof', P3 / 'campaign/validation.py')


def schedule():
    rows = []
    for bound in BOUNDS:
        for side, mode in TUPLES:
            rows.append(row('warmup', bound, side, mode))
    for experiment, arms in EXPERIMENTS:
        for bound in BOUNDS:
            for pair in range(1, 7):
                for arm in (('A', 'B') if pair % 2 else ('B', 'A')):
                    side, mode = arms[arm == 'B']
                    rows.append(row(experiment, bound, side, mode, pair, arm))
    return rows


def row(experiment, bound, side, mode, pair=0, arm='', *, kind='ordinary', reopen=0, name=None):
    name = name or (f'warmup-b{bound}-{side}-{mode}' if experiment == 'warmup'
                    else f'{experiment}-b{bound}-pair{pair}-{arm}')
    return {'name': name, 'experiment': experiment, 'bound': bound, 'side': side,
            'approval': mode, 'pair': pair, 'arm': arm, 'kind': kind, 'reopen': reopen}


def check_schedule(rows):
    if rows != schedule():
        raise ValueError('missing, extra, reordered or mode-mismatched timing schedule')


def stage_order(bound, mode, reopen):
    stages = [('construct_provision', 0), ('create', 0), ('original_state_verification', 0)]
    for i in range(1, bound + 1):
        if i > 1:
            stages.extend([(f'approval_{mode}_basis', i), ('approval_establishment', i)])
        stages.append(('run', i))
        if reopen and i < bound and i % reopen == 0:
            stages.append(('interval_close_reopen_resume', i))
    window = 'execution_window_including_reopen' if reopen else 'execute_window'
    stages.append((window, 0))
    stages.extend((name, 0) for name in PROOF.SINGLETON_PHASES[4:])
    return stages


def vectors(report):
    result = [*report['stages'], report['execute_totals'], report['prefix_residual']]
    for prefix in report['prefixes']:
        result.extend((prefix['direct'], prefix['cumulative']))
    return result


def partition(registry, scalars, scopes, *, compact=True):
    for name in ('family_overflow_events', 'cell_overflow_events', 'invalid_scope_events', 'incomplete_events'):
        key = 'sql.attribution.' + name
        if registry.get(key) != 0 or (compact and scalars.get(key) != 0):
            raise ValueError('SQL attribution is incomplete: ' + name)
    scope_names = set(scopes)
    family_totals, cell_totals = defaultdict(int), defaultdict(int)
    scoped_totals = defaultdict(int)
    for key, value in registry.items():
        if key.startswith('sql.template.'):
            family, metric = key.removeprefix('sql.template.').rsplit('.', 1)
            if metric not in METRICS:
                raise ValueError('unexpected family metric')
            family_totals[family, metric] += value
        elif key.startswith('sql.scope.'):
            scope, sep, rest = key.removeprefix('sql.scope.').partition('.class.')
            klass, _, rest = rest.partition('.post_autocommit.')
            post, _, rest = rest.partition('.template.')
            if rest and '.' in rest:
                family, metric = rest.rsplit('.', 1)
            else:
                raise ValueError('malformed SQL registry cell')
            if (not sep or scope not in scope_names or (scope == 'invalid' and value) or
                    klass not in ('unknown', 'read_write_capable', 'read_only_configured') or
                    post not in ('unknown', 'off', 'on') or metric not in METRICS):
                raise ValueError('invalid SQL registry cell')
            cell_totals[family, metric] += value
            scoped_totals[scope, metric] += value
    if any(cell_totals[key] != family_totals[key] for key in set(cell_totals) | set(family_totals)):
        raise ValueError('SQL registry family/cell partition mismatch')
    for metric in METRICS:
        key = 'sql.total.' + metric
        total = sum(scoped_totals[scope, metric] for scope in scopes)
        if (registry.get('sql.scope_overflow.' + metric) != 0 or registry.get(key) != total or
                scalars.get(key) != total):
            raise ValueError('terminal SQL registry/scalar partition mismatch: ' + metric)
        if compact and any(scalars.get('sql.scope_total.' + scope + '.' + metric) != scoped_totals[scope, metric]
                           for scope in scopes):
            raise ValueError('terminal compact SQL scope partition mismatch: ' + metric)


def difference(previous, current, direct):
    if (not set(previous) <= set(current) or set(direct) != set(current) or
            any(current[k] - previous.get(k, 0) != direct[k] for k in current)):
        raise ValueError('counter arithmetic mismatch')


def validate_counters(report, row, scopes, probe_keys, original=False):
    for vector in vectors(report):
        data = vector.get('counter_deltas')
        if not isinstance(data, dict) or not data or any(type(v) is not int or v < 0 for v in data.values()):
            raise ValueError('missing/invalid diagnostic counters')
        if not original:
            sql_keys = {'sql.total.' + m for m in METRICS}
            sql_keys.update('sql.scope_total.' + s + '.' + m for s in scopes for m in METRICS)
            sql_keys.update('sql.scope_overflow.' + m for m in METRICS)
            sql_keys.update('sql.attribution.' + n for n in (
                'family_overflow_events', 'cell_overflow_events', 'invalid_scope_events', 'incomplete_events'))
            if {k for k in data if k.startswith('sql.')} != sql_keys:
                raise ValueError('compact diagnostic SQL counter vocabulary changed')
            if set(data) - sql_keys - probe_keys:
                raise ValueError('unknown Go work counter')
            if any(v != 0 for k, v in data.items() if k.startswith(('sql.scope_overflow.', 'sql.attribution.'))):
                raise ValueError('incomplete compact SQL attribution')
            for metric in METRICS:
                if data['sql.total.' + metric] != sum(data['sql.scope_total.' + s + '.' + metric] for s in scopes):
                    raise ValueError('compact SQL vector partition mismatch')
    previous = {}
    for prefix in report['prefixes']:
        current = prefix['cumulative']['counter_deltas']
        difference(previous, current, prefix['direct']['counter_deltas'])
        previous = current
    window = next(s for s in report['stages'] if s['name'].startswith(('execute_window', 'execution_window_including_reopen')))
    counter = window['counter_deltas']
    difference(previous, counter, report['prefix_residual']['counter_deltas'])
    body = [s for s in report['stages'] if s['name'] in ('run', f"approval_{row['approval']}_basis", 'approval_establishment')]
    total = {}
    for phase in body:
        for k, value in phase['counter_deltas'].items():
            total[k] = total.get(k, 0) + value
    if total != report['execute_totals']['counter_deltas']:
        raise ValueError('Execute body counter sum mismatch')
    registry = report.get('execute_counter_registry')
    if not isinstance(registry, dict) or any(type(v) is not int or v < 0 for v in registry.values()):
        raise ValueError('invalid Execute registry')
    partition(registry, counter, scopes, compact=not original)
    if not original:
        if counter.get('work.reduce.Commit') != 4 * row['bound'] - 1:
            raise ValueError('actual committed reducer call control failed')
        if counter.get('work.report.roots_emitted') != row['bound'] * (row['bound'] + (
                (row['bound'] - 1) // row['reopen'] if row['reopen'] else 0)):
            raise ValueError('actual full RunReport root emissions control failed')
        create = report['stages'][1]['counter_deltas']
        if row['side'] == 'candidate':
            if (create.get('work.static.init_armed') != 1 or
                    create.get('work.static.init_roots') != row['bound'] or
                    not counter.get('work.static.ready_candidates') or
                    not counter.get('work.static.dirty_prepared')):
                raise ValueError('actual initialized/optimized work control failed')
            if counter.get('fn.selectReadyBaselineLocked.calls') or counter.get('fn.settleRootsBaseline.calls'):
                raise ValueError('deterministic static series unexpectedly used a fallback scheduler')
            if row['approval'] == 'targeted' and row['bound'] > 1:
                if counter.get('fn.ContinuationBasis.calls') != row['bound'] - 1 or not counter.get('work.basis.original_row'):
                    raise ValueError('actual owner-basis work control failed')
        elif not counter.get('work.selectReady.runs_considered'):
            raise ValueError('actual baseline scheduler work control failed')


def validate_report(report, design, revision, store, scopes=(), probe_keys=frozenset()):
    if not isinstance(report, dict):
        raise ValueError('expected a report object')
    name, bound, mode = design['name'], design['bound'], design['approval']
    reopen = design['reopen']
    kind = design['kind']
    runtime = {'approval_mode': mode, 'reopen_every': reopen, 'go_version': 'go1.27.4',
               'goos': 'darwin', 'goarch': 'arm64', 'gomaxprocs': 4, 'mem_profile_rate': 524288,
               'schedule': 'interval-reopen' if reopen else 'ordinary',
               'profiled': kind == 'profile', 'instrumented': kind in ('counters', 'original'),
               'store': str(store), 'prefix_enabled': True}
    if any(type(report.get(k)) is not type(v) or report[k] != v for k, v in runtime.items()):
        raise ValueError('wrong report mode, provenance or runtime configuration')
    validated_prefixes(report, bound, name)
    stages = report.get('stages')
    if not isinstance(stages, list) or [(s['name'], s.get('position', 0)) for s in stages] != stage_order(bound, mode, reopen):
        raise ValueError('stage order or completeness mismatch')
    if not report.get('invariants_ok') or report.get('error'):
        raise ValueError('driver did not retain a complete success proof')
    if runtime['instrumented']:
        validate_counters(report, design, scopes, probe_keys, original=kind == 'original')
    elif report.get('execute_counter_registry', {}) != {} or any(v.get('counter_deltas', {}) != {} for v in vectors(report)):
        raise ValueError('counter contamination in uninstrumented report')

    # P3's proof names snapshot basis and the no-interval window explicitly.
    # Validate raw phase names and all diagnostic accounting above, then pass a
    # copy through P3's immutable semantic/resource verifier.
    normalized = deepcopy(report)
    for vector in vectors(normalized):
        if runtime['instrumented']:
            vector.pop('counter_deltas', None)
        if vector.get('name') == 'approval_targeted_basis':
            vector['name'] = 'approval_snapshot_basis'
        if vector.get('name') == 'execution_window_including_reopen':
            vector['name'] = 'execute_window'
    if reopen:
        for stage in normalized['stages']:
            if stage['name'] == 'interval_close_reopen_resume':
                PROOF.validate_vector(stage, PROOF.METRIC_NAMES)
                if stage.get('error') or stage['wall_ns'] <= 0:
                    raise ValueError('interval reopen stage failure')
        normalized['stages'] = [s for s in normalized['stages'] if s['name'] != 'interval_close_reopen_resume']
    PROOF.validate_report(normalized, {'name': name, 'bound': bound, 'source': revision,
                                      'expected_runtime': runtime})


def command(root, design, revision, binary):
    directory = root / 'samples' / design['name']
    args = [str(binary), '-mode=direct', '-prefix=true', '-bound=' + str(design['bound']),
            '-sample=' + design['name'], '-source-label=' + revision,
            '-approval-mode=' + design['approval'], '-reopen-every=' + str(design['reopen']),
            '-dir=' + str(directory / 'store')]
    if design['kind'] == 'profile':
        args += ['-cpu-profile=' + str(directory / 'cpu.pprof'),
                 '-alloc-profile=' + str(directory / 'alloc.pprof')]
    return args


def resources(report):
    by_stage = defaultdict(list)
    for stage in report['stages']:
        name = 'approval_basis' if stage['name'] in ('approval_snapshot_basis', 'approval_targeted_basis') else stage['name']
        by_stage[name].append(stage)
    return {name: {**{key: sum(s[key] for s in stages) for key in PROOF.RESOURCE_KEYS},
                   'cpu_ns': sum(s['user_cpu_ns'] + s['system_cpu_ns'] for s in stages),
                   'process_maxrss_cumulative': max(s['process_maxrss_cumulative'] for s in stages)}
            for name, stages in by_stage.items()}


def summary(values):
    return {'median': median(values), 'min': min(values), 'max': max(values)}


def analyze(rows, reports):
    check_schedule(rows)
    if list(reports) != [r['name'] for r in rows]:
        raise ValueError('sample results must match the frozen ordered schedule')
    by_arm = defaultdict(list)
    for design in rows:
        if design['experiment'] != 'warmup':
            by_arm[design['experiment'], design['bound'], design['arm']].append(
                resources(reports[design['name']]))
    results = []
    for experiment, _ in EXPERIMENTS:
        for bound in BOUNDS:
            left, right = (by_arm[experiment, bound, arm] for arm in ('A', 'B'))
            metrics = {}
            for stage in sorted(left[0]):
                if stage not in right[0]:
                    continue
                metrics[stage] = {}
                for metric in left[0][stage]:
                    a, b = ([row[stage][metric] for row in group] for group in (left, right))
                    metrics[stage][metric] = {'A': summary(a), 'B': summary(b),
                                             'paired_A_minus_B': [x-y for x, y in zip(a, b)]}
            wall = metrics['execute_window']['wall_ns']
            a, b = wall['A']['median'], wall['B']['median']
            results.append({'experiment': experiment, 'bound': bound, 'stages': metrics,
                            'small_bound_regression': b-a > 1_000_000 and 10*b > 11*a if bound in (1, 8, 32) else None,
                            'b256_target_met': 4*b <= 3*a and min(wall['paired_A_minus_B']) > 0 if bound == 256 else None})
    return results
