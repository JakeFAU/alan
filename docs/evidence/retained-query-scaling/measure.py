#!/usr/bin/env python3
"""P7 bounded retained-query diagnosis with stock and disposable progress arms."""
import argparse
from collections import defaultdict
import difflib
import fcntl
import importlib.util
import json
import os
from pathlib import Path
import platform
import re
import shutil
import statistics as stats
import sys

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
EVIDENCE = HERE.parent
REPO = EVIDENCE.parents[1]
WORKSPACE = REPO / '.superpowers/sdd/plan-retained-query-scaling'
spec = importlib.util.spec_from_file_location('p6_progress_measure', EVIDENCE / 'progress-scaling/measure.py')
p6 = importlib.util.module_from_spec(spec)
spec.loader.exec_module(p6)
from recorder import events

BASE = '9db473bae28e5b806c3ccdc7e2e134c1d0b38f27'
BASE_TREE = '56669e93b11a35ab8a67d09918b3a5e055944015'
BOUNDS = p6.BOUNDS
ARMS = ('baseline', 'order-overlay')
HELPERS = (*p6.HELPER_DIRS, 'progress-scaling')
TOOLS = ('measure.py', 'test_measure.py', 'step_clock.go.txt', 'step_clock_test.go.txt',
         'fk_control_test.go.txt')
TARGET = 'internal/executor/store.go'
UPDATE = '''	if selected != "" {
		if _, err = tx.ExecContext(ctx, "UPDATE observations SET consumed_sequence=? WHERE id=?", r.Sequence, selected); err != nil {
			return RunState{}, err
		}
	}
'''
INSERT = '''	if _, err = tx.ExecContext(ctx, "INSERT INTO history VALUES(?,?,?,?,?,?,?,?)", r.ID.Subject, r.ID.Generation, r.Sequence, r.Epoch, r.Revision, selectedArg, successor, intents); err != nil {
		return RunState{}, err
	}
'''
FAMILIES = {
    'history': 'INSERT INTO history VALUES(?,?,?,?,?,?,?,?)',
    'update': 'UPDATE observations SET consumed_sequence=? WHERE id=?',
    'attempt': 'SELECT result FROM attempts WHERE intent_id=?',
    'charge': 'SELECT count(*) FROM composition_charges WHERE root_id=? AND kind=?',
}
DIAGNOSTIC_METRICS = (*p6.METRICS, 'step_calls', 'invocation_attempts', 'step_errors', 'bind_errors')


def reserve(path, workspace=WORKSPACE):
    path = path.resolve()
    if workspace.resolve() not in path.parents:
        raise ValueError('output must be a new directory inside the P7 workspace')
    path.mkdir()
    return path


def schedule(bounds=BOUNDS):
    rows = []
    for n in bounds:
        rows.extend(p6.design('warmup', n, arm) for arm in ARMS)
        for block in range(1, 7):
            order = ARMS if block % 2 else ARMS[::-1]
            rows.extend(p6.design('timing', n, arm, block) for arm in order)
    return rows


def reorder(text):
    for anchor in (UPDATE, INSERT):
        p6.replace(text, anchor, anchor)
    return p6.replace(text, UPDATE + INSERT, INSERT + UPDATE)


def patch(path, before, after):
    return ''.join(difflib.unified_diff(before.splitlines(keepends=True), after.splitlines(keepends=True),
                                        fromfile='before/' + path, tofile='after/' + path))


def timer_generator(text=None):
    text = p6.counter_generator() if text is None else text
    text = p6.replace(text, r'\n probeTemplate string\n', r'\n probeTemplate string\n p7Family int\n')
    text = p6.replace(text, 'if s != nil { ss.probeTemplate', 'if s != nil { ss.p7Family = p7Family(C.GoString(C.sqlite3_sql(s))); ss.probeTemplate')
    for line in text.splitlines():
        for statement, receiver in (
                ('rv := C._sqlite3_step_row_internal(s.s, &rowid, &changes)', 's'),
                ('rv := C._sqlite3_step_internal(rc.s.s)', 'rc.s'),
                ('rv = C._sqlite3_step_columns(s.s, &ncol, &repreps)', 's'),
                ('return C._sqlite3_step_internal(rc.s.s)', 'rc.s')):
            prefix = "    s=replace(s,'" + statement + "',"
            if line.startswith(prefix):
                wrapped = statement.replace('return C.', 'rv := C.')
                replacement = f'p7Start := {receiver}.p7Start()\\n {wrapped}\\n {receiver}.p7Finish(p7Start)'
                # Replace only the generated side of the pinned replacement.
                rhs = line[len(prefix):]
                if not rhs.startswith("'" + wrapped + r'\n '):
                    raise ValueError('generated step anchor drift: ' + statement)
                text = p6.replace(text, line, prefix + "'" + replacement + rhs[len(wrapped)+1:])
    if text.count('p7Finish(p7Start)') != 4:
        raise ValueError('missing or duplicate SQLite step wrappers')
    return text


def check_rows(design, rows, configs):
    if len(rows) != len(design) or len({r['name'] for r in design}) != len(design):
        raise ValueError('incomplete or duplicate schedule')
    for planned, actual in zip(design, rows):
        config = configs[planned['arm']]
        if (actual['design'] != planned or actual.get('failure') or actual.get('exit') != 0 or
                actual['mode'] != 'progress' or actual['revision'] != config['label'] or
                actual['binary_sha256'] != config['digest']):
            raise ValueError('failed or rebound sample: ' + planned['name'])


def stats_pair(baselines, overlays):
    savings = [a-b for a, b in zip(baselines, overlays)]
    m = stats.median(savings)
    return {'savings': savings, 'baseline_median': stats.median(baselines),
            'overlay_median': stats.median(overlays), 'savings_median': m,
            'savings_mad': stats.median(abs(s-m) for s in savings),
            'median_paired_percent': 100 * stats.median((a-b)/a for a, b in zip(baselines, overlays))
            if min(baselines) > 0 else None, 'faster': sum(s > 0 for s in savings)}


def advance(baselines, overlays):
    if len(baselines) != 6 or len(overlays) != 6 or min(baselines) <= 0:
        return False
    s = stats_pair(baselines, overlays)
    return (s['median_paired_percent'] >= 1 and s['faster'] >= 5 and s['savings_median'] > s['savings_mad'])


def timing_summary(rows):
    values = defaultdict(lambda: defaultdict(dict))
    for row in rows:
        r = row['design']
        if r['kind'] != 'timing':
            continue
        metrics = {f'{stage}.{field}': val for stage, fields in row['resources'].items()
                   for field, val in fields.items()}
        metrics.update({f'operation.{k}.wall_ns': v for k, v in row['operation_wall_ns'].items()})
        metrics.update({k: row[k] for k in ('execute_end_maxrss_bytes', 'process_exit_maxrss_bytes')})
        values[r['bound']][r['block']][r['arm']] = metrics
    out = {}
    for bound, blocks in values.items():
        vectors = {arm: {k: [blocks[i][arm][k] for i in range(1, 7)]
                         for k in blocks[1][arm]} for arm in ARMS}
        summary = {k: stats_pair(vectors[ARMS[0]][k], vectors[ARMS[1]][k]) for k in vectors[ARMS[0]]}
        base, reordered = (vectors[arm]['execute_window.wall_ns'] for arm in ARMS)
        out[str(bound)] = {'metrics': summary, 'vectors': vectors,
                           'execute_order': {'AB': stats_pair(base[::2], reordered[::2]),
                                             'BA': stats_pair(base[1::2], reordered[1::2])},
                           'b256_advance_screen': advance(base, reordered) if bound == 256 else None}
    return out


def freeze(rec, head, *, preflight=False):
    baseline, identity = p6.freeze(rec, BASE, 'baseline')
    if identity['tree'] != BASE_TREE:
        raise ValueError('baseline tree does not match frozen contract')
    tracked = rec.run('checkout-measurement-index', ['git', 'ls-files', '--', HERE], REPO).read_text().splitlines()
    for f in TOOLS:
        if str((HERE / f).relative_to(REPO)) not in tracked and not preflight:
            raise ValueError('measurement input is not tracked: ' + f)
    for directory in HELPERS:
        if p6.inventory(baseline / EVIDENCE.relative_to(REPO) / directory) != p6.inventory(EVIDENCE / directory):
            raise ValueError('P6 or transitive helper changed since baseline: ' + directory)
    wrapper = p6.P5 / 'sample_usage.py'
    if p6.sha(wrapper) != p6.sha(baseline / wrapper.relative_to(REPO)):
        raise ValueError('RSS wrapper changed since baseline')
    overlay = rec.root / 'overlay-source'
    p6.extract_source(rec.root / 'baseline.tar', overlay)
    before = p6.inventory(overlay)
    if before != p6.inventory(baseline):
        raise ValueError('exports differ before overlay')
    file = overlay / TARGET
    old = file.read_text()
    new = reorder(old)
    diff = rec.root / 'statement-order.patch'
    diff.write_text(patch(TARGET, old, new))
    file.write_text(new)
    after = p6.inventory(overlay)
    if before.keys() != after.keys() or [k for k in before if before[k] != after[k]] != [TARGET]:
        raise ValueError('disposable patch changed an unexpected source file')
    rec.bind('overlay-source', overlay)
    rec.bind_tool(diff)
    rev = 'disposable-order-overlay@' + BASE + '+' + p6.sha(diff)
    overlay_id = {'label': rev, 'parent': identity, 'patch_sha256': p6.sha(diff),
                  'file': TARGET, 'file_before': before[TARGET], 'file_after': after[TARGET]}
    p6.write(rec.root / 'sources.json', {'baseline': identity, 'order-overlay': overlay_id, 'apparatus': head})
    return {ARMS[0]: dict(source=baseline, label=BASE), ARMS[1]: dict(source=overlay, label=rev)}


def timed_build(rec, source, sqlite):
    name = 'step-clock'
    out = rec.root / name
    gen = rec.root / 'generate-step-clock.py'
    gen.write_text(timer_generator())
    rec.bind_tool(gen)
    rec.run(name + '-generate', [sys.executable, '-B', gen, '--alan', source,
                                 '--sqlite', sqlite, '--driver', p6.DRIVER, '--out', out])
    p6.compact_counters(out)
    driver = out / 'driver'
    p6.make_driver(driver, 'progress')
    main = driver / 'main.go'
    old = main.read_text()
    new = p6.replace(old, 'type report struct {', '''type report struct {
        StepClock map[string]p7StepTime `json:"p7_step_clock"`
''')
    new = p6.replace(new, 'registryStart = counterRegistry()', 'registryStart = counterRegistry()\n p7StartClock()')
    new = p6.replace(new, 'r.ExecuteCounterRegistry, e = subtractCounters', 'r.StepClock = p7StopClock()\n r.ExecuteCounterRegistry, e = subtractCounters')
    main.write_text(new)
    delta = out / 'p7-driver.patch'
    delta.write_text(patch('driver/main.go', old, new))
    hook = driver / 'p7_clock.go'
    hook.write_text('''package main
import sqlite "github.com/mattn/go-sqlite3"
type p7StepTime = sqlite.P7StepTime
var p7StartClock = sqlite.P7StartClock
var p7StopClock = sqlite.P7StopClock
''')
    for src, dest in [('step_clock.go.txt', 'p7_clock.go'), ('step_clock_test.go.txt', 'p7_clock_test.go'),
                      ('fk_control_test.go.txt', 'p7_fk_test.go')]:
        shutil.copyfile(HERE / src, out / 'sqlite' / dest)
    original_proof = p6.P2 / 'instrument/proof_test.go.txt'
    shutil.copyfile(original_proof, out / 'sqlite/p7_pinned_proof_test.go')
    schema, = re.findall(r'const schema = `([^\x60]+)`', (source / TARGET).read_text())
    (out / 'sqlite/p7_schema_test.go').write_text('package sqlite3\nconst p7Schema = `' + schema + '`\n')
    rec.run(name + '-format', ['gofmt', '-w', *driver.glob('*.go'), *(
        out / 'sqlite' / f for f in ('alan_measurement.go', 'p7_clock.go', 'p7_clock_test.go',
                                    'p7_fk_test.go', 'p7_schema_test.go'))])
    for f in ('generation.json', 'transformations.json', 'measurement.patch', 'p7-driver.patch'):
        rec.bind_tool(out / f)
    rec.bind(name + '-alan', out / 'alan')
    rec.bind(name + '-sqlite', out / 'sqlite')
    rec.run(name + '-controls', ['go', 'test', '-count=1', '-json',
                                '-run=^Test(P7|Measurement|ScalingCompact)', '.'], out / 'sqlite')
    binary = out / 'driver.bin'
    p6.build_module(rec, name, out / 'alan', driver, binary, sqlite=out / 'sqlite')
    return binary


def counter_row(root, row, *, clock=None):
    log = root / 'gates' / row['design']['name'] / 'stdout.log'
    report = p6.read(log)
    registry = report['execute_counter_registry']
    if registry['sql.attribution.incomplete_events'] != 0:
        raise ValueError('incomplete pinned SQL template registry')
    suffix = '.fullscan_steps'
    scan = {k[len('sql.template.'):-len(suffix)]: n for k, n in registry.items()
            if k.startswith('sql.template.') and k.endswith(suffix) and n}
    targets = {name: {m: registry[f'sql.template.{family}.{m}'] for m in DIAGNOSTIC_METRICS}
               for name, family in FAMILIES.items()}
    if sum(scan.values()) != row['attribution']['execute_sql']['fullscan_steps']:
        raise ValueError('template scan inventory differs from total')
    all_families = {k: n for k, n in registry.items() if k.startswith('sql.template.')}
    stages = {s['name']: {m: s['counter_deltas']['sql.total.' + m] for m in p6.METRICS}
              for s in report['stages']}
    result = {'targets': targets, 'all_scan_families': scan, 'stages': stages,
              'execute_wall_ns': row['resources']['execute_window']['wall_ns'],
              'raw_report_sha256': p6.sha(log)}
    if clock is not None:
        timed = report['p7_step_clock']
        if set(timed) != {'history', 'attempt', 'charge'}:
            raise ValueError('clock omitted a timed family')
        for name, data in timed.items():
            want = targets[name]['step_calls'] if clock != 'off' else 0
            if (data['step_calls'] != want or type(data['ns']) is not int or data['ns'] < 0 or
                    (clock == 'off' and data['ns'] != 0)):
                raise ValueError('clock and SQLite step counts disagree: ' + name)
        result.update(clock=clock, step_times=timed)
    return result, all_families


def compare_counter_pair(base, overlay, reg_base, reg_overlay, n):
    expected = {'history': (2*n-1)*(n-1), 'attempt': n*(n-1), 'charge': (n-1)*(n-2)//2}
    old = base['targets']
    new = overlay['targets']
    if any(old[k]['fullscan_steps'] != v for k, v in expected.items()) or new['history']['fullscan_steps']:
        raise ValueError('history scan mechanism did not discriminate')
    history_prefix = 'sql.template.' + FAMILIES['history'] + '.'
    # Consumption update may also dispense with FK-counter work; no new VM work
    # or increase in any count except the number of compiled VDBE operations saved.
    update_prefix = 'sql.template.' + FAMILIES['update'] + '.'
    allowed = {history_prefix + x for x in ('vm_steps', 'fullscan_steps')} | {update_prefix + 'vm_steps'}
    differences = {k: reg_base.get(k, 0)-reg_overlay.get(k, 0) for k in reg_base.keys() | reg_overlay.keys()
                   if reg_base.get(k, 0) != reg_overlay.get(k, 0)}
    if not differences.keys() <= allowed or any(v <= 0 for v in differences.values()):
        raise ValueError('SQL work moved or changed in another family: ' + repr(differences))
    for name, stage in base['stages'].items():
        want = expected['history'] if name in {'execute_window', 'lifecycle'} else 0
        if stage['fullscan_steps'] - overlay['stages'][name]['fullscan_steps'] != want:
            raise ValueError('scan steps moved to lifecycle stage: ' + name)
    return differences


def campaign(args, root):
    for name in ('tmp', 'samples'):
        (root / name).mkdir()
    env = p6.environment(root, args.gopath.resolve())
    if args.gocache:
        env['GOCACHE'] = str(args.gocache.resolve())
    rec = p6.TaskRecorder(root, env=env)
    status = rec.run('checkout-status', ['git', 'status', '--porcelain=v1', '--untracked-files=all'], REPO)
    if status.read_text() and not args.preflight:
        raise ValueError('measurement requires clean frozen harness')
    head = rec.run('checkout-head', ['git', 'rev-parse', 'HEAD'], REPO).read_text().strip()
    for directory in HELPERS:
        rec.bind(directory, EVIDENCE / directory)
    for tool in [*(HERE / f for f in TOOLS), p6.P5 / 'sample_usage.py']:
        rec.bind_tool(tool)
    p6.pinned_inputs()
    sqlite, toolchain = p6.bind_environment(rec)
    if platform.system() == 'Darwin':
        hardware = json.loads(rec.run('hardware', ['system_profiler', 'SPHardwareDataType', '-json']).read_text())
        info, = hardware['SPHardwareDataType']
        toolchain['host']['hardware'] = {k: info[k] for k in ('chip_type', 'machine_model', 'number_processors', 'physical_memory')}
    configs = freeze(rec, head, preflight=args.preflight)
    rows = []
    if args.lane == 'timing':
        rows = schedule()
    elif args.lane == 'counters':
        rows = [p6.design('counters', b, a) for b in (8, 256) for a in ARMS]
    elif args.lane == 'clock':
        configs = {a: configs[ARMS[0]].copy() for a in ('on', 'off', 'empty')}
        rows = [p6.design('counters', b, a, block) for b in (8, 256) for block in range(1, 4)
                for a in ('on', 'off', 'empty')[(block-1):] + ('on', 'off', 'empty')[:(block-1)]]
    elif args.lane == 'profile':
        configs.pop(ARMS[1])
        rows = [p6.design('profile', 256, ARMS[0])]
    else:
        rows = [p6.design('smoke', b, a) for b in (1, 8) for a in ARMS]

    p6.write(root / 'schedule.json', rows)
    if args.lane == 'clock':
        binary = timed_build(rec, configs['on']['source'], sqlite)
        if args.preflight:
            return
        binaries = dict.fromkeys(configs, binary)
    else:
        binaries = {a: p6.build(rec, a, c['source'], sqlite, 'progress', counters=args.lane == 'counters')
                    for a, c in configs.items()}
    for a, c in configs.items():
        gate = 'step-clock' if args.lane == 'clock' else a
        build_info = rec.gates / (gate + '-build-info') / 'stdout.log'
        version = '\tdep\tgithub.com/mattn/go-sqlite3\tv1.14.52\t'
        info = build_info.read_text()
        if info.count(version) != 1 or (('/sqlite\t(devel)' in info) != (args.lane in ('counters', 'clock'))):
            raise ValueError('build SQLite does not match selected lane: ' + gate)
        c.update(binary=str(binaries[a]), digest=p6.sha(binaries[a]), build_info_sha256=p6.sha(build_info))

    inputs = rec.snapshot()
    p6.write(root / 'inputs.json', inputs)
    p6.write(root / 'manifest.json', {'format': 'alan/retained-query-scaling/v1',
                'lane': args.lane, 'apparatus_commit': head, 'toolchain': toolchain,
                'checkout_status': status.read_text().splitlines(),
                'sources_sha256': p6.sha(root / 'sources.json'),
                'schedule_sha256': p6.sha(root / 'schedule.json'), 'inputs_sha256': p6.sha(root / 'inputs.json'),
                'builds': {a: dict(c, source=str(c['source'])) for a, c in configs.items()}})
    sample_rec = p6.TaskRecorder(root, env=rec.env,
        tools=[*set(binaries.values()), p6.P5 / 'sample_usage.py', Path(sys.executable).resolve()])
    results = []
    print('SERIAL SAMPLES START ' + args.lane, flush=True)
    for row in rows:
        arm = row['arm']
        if args.lane == 'clock':
            sample_rec.env = dict(rec.env, ALAN_P7_STEP_CLOCK=arm)
        results.append(p6.execute(sample_rec, row, configs[arm]['label'], 'progress', binaries[arm]))
    print('SERIAL SAMPLES END ' + args.lane, flush=True)
    p6.write(root / 'inputs-after.json', rec.snapshot())
    if p6.read(root / 'inputs-after.json') != inputs:
        raise ValueError('bound inputs changed across samples')
    result = {'rows_sha256': p6.sha(root / 'rows.jsonl'), 'manifest_sha256': p6.sha(root / 'manifest.json'),
              'failures': [r['design']['name'] for r in results if 'failure' in r]}
    try:
        check_rows(rows, results, configs)
        if args.lane == 'timing':
            result['paired'] = timing_summary(results)
        elif args.lane in ('counters', 'clock'):
            analyses = [counter_row(root, r, clock=r['design']['arm'] if args.lane == 'clock' else None)
                        for r in results]
            result['diagnostics'] = dict(zip((r['name'] for r in rows), (a for a, _ in analyses)))
            if args.lane == 'counters':
                result['structural_differences'] = {
                    str(n): compare_counter_pair(analyses[i][0], analyses[i+1][0],
                                                 analyses[i][1], analyses[i+1][1], n)
                    for n, i in ((8, 0), (256, 2))}
    except Exception as error:
        result['validation_failure'] = repr(error)
        raise
    finally:
        p6.write(root / 'result.json', result)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--lane', choices=('smoke', 'counters', 'clock', 'profile', 'timing'), required=True)
    parser.add_argument('--gopath', type=Path, default=Path.home() / 'go')
    parser.add_argument('--gocache', type=Path)
    parser.add_argument('--preflight', action='store_true', help='build/test diagnostic apparatus without running samples')
    args = parser.parse_args()
    if args.preflight and args.lane != 'clock':
        parser.error('preflight is for diagnostic controls only')
    with (WORKSPACE / 'measurement.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        campaign(args, reserve(args.out))


if __name__ == '__main__':
    main()
