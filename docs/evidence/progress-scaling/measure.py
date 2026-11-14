#!/usr/bin/env python3
"""Measure deterministic bounded execution from exported ALAN revisions."""
import argparse
from collections import defaultdict
import fcntl
from itertools import permutations
import json
import os
from pathlib import Path
import platform
import re
import shutil
import sys

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
EVIDENCE = HERE.parent
REPO = EVIDENCE.parents[1]
WORKSPACE = REPO / '.superpowers/sdd/plan-progress-scaling'
P2 = EVIDENCE / 'performance-p2'
P4 = EVIDENCE / 'execution-scaling'
P5 = EVIDENCE / 'report-scaling'
DRIVER = P2 / 'driver'
sys.path.insert(0, str(P4 / 'task5'))
from custody import TaskRecorder, extract_source, inventory, read, sha, write
from controller import environment
from overlays import build_module, scopes
from measurement import partition
from apparatus import compact_counters, pinned_inputs, replace

BASE = 'e43e4f6be4500d177b50caf29181608d09b99c62'
BASE_TREE = '6fbde0ba908a80fc45733c4daf69e473e3cff47a'
SQLITE = 'github.com/mattn/go-sqlite3'
SQLITE_VERSION = 'v1.14.52'
BOUNDS = (1, 8, 16, 32, 64, 128, 256)
ARMS = ('base-full', 'candidate-full', 'candidate-progress')
ROOT_PROBE = 'work.report.roots_emitted'
METRICS = ('executions', 'sqlite_row_returns', 'vm_steps', 'fullscan_steps')
DRIVER_FILES = ('main.go', 'transport.go', 'resources.go', 'prefix_proof.go', 'go.mod', 'go.sum')
HELPER_DIRS = ('performance-p2/driver', 'performance-p2/instrument', 'execution-scaling/apparatus',
               'execution-scaling/task5', 'performance-p3/final', 'performance-p3/qualification',
               'performance-p3/campaign')


def reserve_output(path, workspace=WORKSPACE):
    path = path.resolve()
    if workspace.resolve() not in path.parents:
        raise ValueError('output must be a new directory inside the P6 workspace')
    path.mkdir()
    return path


def design(kind, bound, arm, block=0):
    name = f'{kind}-b{bound}-{arm}'
    if block:
        name += f'-block{block}'
    return {'name': name, 'kind': kind, 'bound': bound, 'arm': arm, 'block': block}


def timing_design(bounds=BOUNDS):
    rows = []
    for bound in bounds:
        rows.extend(design('warmup', bound, arm) for arm in ARMS)
        for i, order in enumerate(permutations(ARMS), 1):
            rows.extend(design('timing', bound, arm, i) for arm in order)
    return rows


def splice(text, first, next_anchor, new=''):
    # Refuse drift in either delimiter before replacing a contiguous region.
    for anchor in (first, next_anchor):
        if text.count(anchor) != 1:
            raise ValueError('anchor must occur exactly once: ' + anchor)
    start = text.index(first)
    end = text.find(next_anchor, start + len(first))
    if end < 0:
        raise ValueError('anchors out of order: ' + first)
    return text[:start] + new + text[end:]


def transform_driver(text):
    text = replace(text, 'type report struct {', '''type report struct {
        ExecutionMode string `json:"execution_mode"`
        Operations []operation `json:"operations"`
        ExecuteCounterRegistry map[string]uint64 `json:"execute_counter_registry,omitempty"`
''')
    text = replace(text, '\tPrefixConfiguration prefixConfiguration `json:"prefix_configuration"`\n', '')
    text = splice(text, '\tPrefixEnabled ', '\tServerContacts ')
    for field in ('\tReopenEvery         int                 `json:"reopen_every"`\n',
                  '\tExecuteTotals       stage               `json:"execute_totals"`\n'):
        text = replace(text, field, '')
    text = splice(text, '\tif name == "run" ||', '\tfmt.Fprintf(os.Stderr, "sample=%s bound=%d stage=%s position=%d done')
    text = replace(text, ', reopenEvery int) (err error)', ') (err error)')
    text = replace(text, '\tvar cuts *cutVerifier\n', '')
    text = splice(text, '\t\tcuts, e = newCutVerifier', '\t\treturn nil\n\t}); err != nil {\n\t\treturn err\n\t}\n\tif r.Transport',
                  '''        r.RootMembership, e = initialMembership(initial.Roots, r.Bound)
        if e != nil { return e }
''')
    text = splice(text, 'windowName := "execute_window"', '\tvar cut *aruntime.ExecutionSnapshot',
                  (HERE / 'execute.go.txt').read_text().lstrip('\t'))
    text = splice(text, '\tfor i, prefix := range r.Prefixes {', '\thistory, historyErr := cut.History',
                  '''    if err = verifyCompletedReport(finalReport, cut.Roots, m.contactCount(), r.Bound); err != nil {
        return err
    }
    for i, original := range r.RootMembership {
        if original.RootID != r.Evidence[i].RootID || original.Run != r.Evidence[i].Run {
            return errors.New("final cut changed initial root membership")
        }
    }
''')
    text = replace(text, 'if err = r.measure("close_reopen_resume"', 'if err = r.measure("close_before_reopen"')
    text = replace(text, '''		store = nil
		var e error
		store, e = aruntime.Open(ctx, path, aruntime.DefaultLimits())
		if e != nil {
			return e
		}
		_, e = store.Run(ctx, opts)
		return e''', '''        store = nil
        return nil
    }); err != nil { return err }
    if err = r.measure("open_completed", 0, func() error {
        var e error
        store, e = aruntime.Open(ctx, path, aruntime.DefaultLimits())
        return e
    }); err != nil { return err }
    var resumed aruntime.RunReport
    if err = r.measure("run_after_reopen", 0, func() error {
        var e error
        resumed, e = store.Run(ctx, opts)
        return e''')
    text = replace(text, '\tafter, e := verify(cut, view, m, r.Bound)', '''    if err = verifyCompletedReport(resumed, cut.Roots, m.contactCount(), r.Bound); err != nil { return err }
    if !reflect.DeepEqual(finalReport.Roots, cut.Roots) { return errors.New("completed reopen changed root evidence") }
    after, e := verify(cut, view, m, r.Bound)''')
    text = replace(text, '\t\tfor _, run := range replayed.Runs {', '''        if !reflect.DeepEqual(replayed.Roots, cut.Roots) {
            return errors.New("replay changed original root evidence")
        }
        for _, run := range replayed.Runs {''')
    text = replace(text, '\treopen := flag.Int("reopen-every", 0, "separate recovery schedule; reopen after every k selected positions (0 disables)")\n', '')
    text = splice(text, '\tmode := flag.String("mode",', '\ttimeout := flag.Duration')
    text = replace(text, '(*mode != "http" && *mode != "direct") || !allowed[*n] || *dir == "" || *reopen < 0',
                   '!allowed[*n] || *dir == ""')
    text = replace(text, '; reopen-every >= 0', '')
    text = replace(text, 'Version: "alan/performance-p2/v1"', 'Version: "alan/progress-scaling/v1", ExecutionMode: executionMode')
    text = replace(text, ', ReopenEvery: *reopen', '')
    text = splice(text, '\tr.Transport = frozenTransport(*mode, *endpoint)', '\tr.MetricNames',
                  '''    r.Transport = frozenTransport("direct", "")
    r.Operations = make([]operation, 0, 3*(*n)-2)
''')
    text = replace(text, 'process-wide including same-process HTTP server; GC/CPU asynchronous; RSS cumulative high-water gauge',
                   'process-wide; GC/CPU asynchronous; RSS cumulative high-water gauge')
    text = splice(text, '\tif *reopen > 0 {', '\tvar stops []func()')
    return replace(text, 'err = run(ctx, &r, *dir, *reopen)',
                   'err = r.measure("lifecycle", 0, func() error { return run(ctx, &r, *dir) })')


def make_driver(driver, mode):
    main = driver / 'main.go'
    main.write_text(transform_driver(main.read_text()))
    # Keep the P2 initial membership/root-state proof, without the mutable
    # prefix verifier or any per-position evidence materialization.
    proof = driver / 'prefix_proof.go'
    text = splice(proof.read_text(), 'type rootCut struct {', 'func newCutVerifier')
    text = replace(text, '\t"encoding/json"\n', '')
    text = replace(text, '\t"github.com/JakeFAU/alan/effects"\n', '')
    text = replace(text, '\t"reflect"\n', '')
    text = replace(text, 'func newCutVerifier(roots []aruntime.RootCompletion, bound int) (*cutVerifier, error)',
                   'func initialMembership(roots []aruntime.RootCompletion, bound int) ([]rootIdentity, error)')
    text = replace(text, 'v := &cutVerifier{membership: make([]rootIdentity, 0, bound), completed: make(map[aruntime.RunID]aruntime.RootCompletion, bound)}',
                   'membership := make([]rootIdentity, 0, bound)')
    text = replace(text, 'v.membership = append(v.membership,', 'membership = append(membership,')
    text = replace(text, 'v.membershipSHA = membershipDigest(v.membership)\n\treturn v, nil', 'return membership, nil')
    anchor = '// Check the already returned report only:'
    replace(text, anchor, anchor)  # Require a unique truncation boundary.
    proof.write_text(text.split(anchor, 1)[0])
    resource = driver / 'resources.go'
    text = splice(resource.read_text(), 'type prefixConfiguration struct {', 'func takeSnapshot')
    text = replace(text, '\t"reflect"\n', '')
    anchor = '// Treat stages as vectors of cumulative quantities.'
    replace(text, anchor, anchor)
    resource.write_text(text.split(anchor, 1)[0])
    method = {'full': 'Run', 'progress': 'RunProgress'}[mode]
    (driver / 'execution_mode.go').write_text(f'''package main
import ("context"; aruntime "github.com/JakeFAU/alan/runtime")
const executionMode = "{mode}"
type progressTuple struct {{ Steps, Dispatches int; Quiescent bool }}
func runObservation(ctx context.Context, s *aruntime.Store, opts aruntime.RunOptions) (progressTuple, error) {{
    r, err := s.{method}(ctx, opts)
    return progressTuple{{Steps: r.Steps, Dispatches: r.Dispatches, Quiescent: r.Quiescent}}, err
}}
''')
    shutil.copyfile(HERE / 'completed.go.txt', driver / 'completed.go')


def counter_generator():
    # Preserve the P2 SQLite generator; use P5's two finishRun anchors.
    original = P2 / 'instrument/generate.py'
    text = replace(original.read_text(), 'HERE = pathlib.Path(__file__).resolve().parent',
                   f'HERE = pathlib.Path({str(original.parent)!r})')
    return splice(text, 'functions={', "(a.out/'alan/measurecounter').mkdir()", f'''SCOPES = {{'runner.go': [('finishRun', 8)]}}
functions = {{'runner.go': []}}
def instrument_finish_run(s):
    s = imported(s)
    anchor = 'func (s *Store) finishRun(ctx context.Context, report RunReport, quiescent bool, runErr error) (RunReport, error) {{'
    s = replace(s, anchor, anchor + '\\n ctx = scalingprobe.Scope(ctx, 8)')
    anchor = 'report.Roots = roots'
    return replace(s, anchor, anchor + '\\n scalingprobe.Add("{ROOT_PROBE}", uint64(len(roots)))')
edit('alan/internal/executor/runner.go', instrument_finish_run)
''')


def report_identity(row, revision, mode, store):
    return {'version': 'alan/progress-scaling/v1', 'bound': row['bound'], 'sample': row['name'],
            'source_label': revision, 'execution_mode': mode, 'store': str(store)}


def check_identity(report, row, revision, mode, store):
    for key, expected in report_identity(row, revision, mode, store).items():
        if report.get(key) != expected:
            raise ValueError('driver output binding mismatch: ' + key)


def attribution(report, execute, mode):
    compact = execute['counter_deltas']
    registry = report['execute_counter_registry']
    for counters in (compact, registry):
        if not counters or any(type(v) is not int or v < 0 for v in counters.values()):
            raise ValueError('missing or invalid counter values')
    partition(registry, compact, scopes())
    bound = report['bound']
    roots = compact.get(ROOT_PROBE, 0)
    totals = {m: compact['sql.total.' + m] for m in METRICS}
    report_totals = {m: compact['sql.scope_total.finishRun.' + m] for m in METRICS}
    if roots != (bound**2 if mode == 'full' else 0) or totals['executions'] <= 0:
        raise ValueError('unexpected report emission or missing Execute SQL')
    if (mode == 'full' and not 0 < report_totals['executions'] < totals['executions'] or
            mode == 'progress' and any(report_totals.values())):
        raise ValueError('unexpected finishRun SQL partition')
    return {'roots_emitted': roots, 'execute_sql': totals, 'finishRun_sql': report_totals}


def usage_from_log(path, exit_code):
    marker = 'ALAN_P5_CHILD_USAGE '
    entries = [json.loads(line[len(marker):]) for line in path.read_text(errors='replace').splitlines()
               if line.startswith(marker)]
    if not entries:
        return None
    usage, = entries
    if usage['child_exit'] != exit_code or type(usage['process_maxrss_bytes']) is not int:
        raise ValueError('child usage does not bind to the recorded exit')
    # The reused P5 wrapper labels Darwin bytes directly; Linux ru_maxrss is KiB.
    scale = {'Darwin': 1, 'Linux': 1024}[platform.system()]
    return usage['process_maxrss_bytes'] * scale


def validated_summary(report, row, revision, mode, store, post_exit_rss):
    check_identity(report, row, revision, mode, store)
    n = row['bound']
    if (not report['invariants_ok'] or report.get('error') or report['transport']['mode'] != 'direct' or
            report['schedule'] != 'ordinary' or report['profiled'] != (row['kind'] == 'profile') or
            report['instrumented'] != (row['kind'] == 'counters')):
        raise ValueError('failed driver invariants, schedule or diagnostic lane')
    expected = dict.fromkeys(('selected', 'completed', 'spent', 'replay_intents'), n)
    expected.update(protected_approvals=n-1, reductions=4*n-1, replay_history=4*n-1)
    if (any(report[k] != v for k, v in expected.items()) or
            any(len(report[k]) != n for k in ('contacts', 'evidence', 'root_membership')) or
            report['server_contacts']):
        raise ValueError('retained workload cardinality mismatch')
    stage_names = ('construct_provision', 'create', 'original_state_verification', 'execute_window',
                   'final_full_report', 'final_inspection', 'final_series_view', 'final_series_proof',
                   'close_before_reopen', 'open_completed', 'run_after_reopen', 'resume_inspection_view',
                   'close', 'replay', 'lifecycle')
    if [s['name'] for s in report['stages']] != list(stage_names):
        raise ValueError('missing or reordered stage')
    fields = ('wall_ns', 'user_cpu_ns', 'system_cpu_ns', 'total_alloc_bytes', 'mallocs')
    stages = {}
    for stage in report['stages']:
        if stage.get('error') or any(type(stage[k]) is not int or stage[k] < 0 for k in fields):
            raise ValueError('invalid stage resource measurement')
        stages[stage['name']] = {k: stage[k] for k in fields}
    expected_ops = [('run', 1)]
    for i in range(2, n+1):
        expected_ops.extend((op, i) for op in ('approval_targeted_basis', 'approval_establishment', 'run'))
    if [(o['name'], o['position']) for o in report['operations']] != expected_ops:
        raise ValueError('missing or reordered execution operation')
    totals = defaultdict(int)
    for op in report['operations']:
        if op.get('error') or type(op['wall_ns']) is not int or op['wall_ns'] < 0:
            raise ValueError('invalid per-operation elapsed measurement')
        totals[op['name']] += op['wall_ns']
    execute = next(s for s in report['stages'] if s['name'] == 'execute_window')
    rss = execute['process_maxrss_cumulative'] * {'bytes': 1, 'KiB': 1024}[report['maxrss_unit']]
    if post_exit_rss is None or not 0 < rss <= post_exit_rss:
        raise ValueError('missing or nonmonotone post-exit child RSS')
    result = {'workload': expected, 'contact_count': n, 'resources': stages, 'operation_wall_ns': totals,
              'execute_end_maxrss_bytes': rss, 'process_exit_maxrss_bytes': post_exit_rss,
              'driver_wall_ns_before_json': report['process_wall_ns']}
    if row['kind'] == 'counters':
        result['attribution'] = attribution(report, execute, mode)
    return result


def capture_row(rec, row, revision, mode, binary, store, argv):
    name = row['name']
    gate = rec.gates / name
    result = {'design': row, 'revision': revision, 'mode': mode, 'binary_sha256': sha(binary)}
    try:
        output = rec.run(name, argv, store.parent.parent, timeout=1800)
        outcome = read(gate / 'outcome.json')
        rss = usage_from_log(gate / 'stderr.log', outcome['exit'])
        result.update(validated_summary(read(output), row, revision, mode, store, rss))
        if row['kind'] == 'profile':
            profiles = [store.parent.parent / (kind + '.pprof') for kind in ('cpu', 'allocs')]
            if any(path.stat().st_size == 0 for path in profiles):
                raise ValueError('empty profile')
            result['profiles'] = {path.name: sha(path) for path in profiles}
    except Exception as error:
        result['failure'] = repr(error)
    outcome = gate / 'outcome.json'
    if outcome.exists():
        status = read(outcome)
        result.update(exit=status['exit'], timeout=status['timeout'], spawn_error=status['spawn_error'],
                      process_wrapper_seconds=status['seconds'])
    for key, filename in [('stdout_sha256', 'stdout.log'), ('stderr_sha256', 'stderr.log'),
                          ('command_sha256', 'command.json'), ('outcome_sha256', 'outcome.json')]:
        path = gate / filename
        if path.exists():
            result[key] = sha(path)
    write(gate / 'row.json', result)
    with (rec.root / 'rows.jsonl').open('a') as stream:
        stream.write(json.dumps(result, sort_keys=True) + '\n')
        stream.flush()
        os.fsync(stream.fileno())
    return result


def freeze(rec, revision, label):
    def git(name, *args):
        return rec.run(label + '-' + name, ['git', *args], REPO).read_text().strip()

    commit = git('commit', 'rev-parse', revision + '^{commit}')
    tree = git('tree', 'rev-parse', revision + '^{tree}')
    if commit != revision or (commit == BASE and tree != BASE_TREE):
        raise ValueError('frozen commit/tree mismatch')
    archive = rec.root / (label + '.tar')
    git('archive', 'archive', '--format=tar', '-o', archive, commit)
    source = rec.root / (label + '-source')
    extract_source(archive, source)
    rec.bind(label + '-source', source)
    rec.bind_tool(archive)
    metadata = {'commit': commit, 'tree': tree, 'archive_sha256': sha(archive),
                'go_mod_sha256': sha(source / 'go.mod'), 'go_sum_sha256': sha(source / 'go.sum')}
    modules = rec.run(label + '-modules', ['go', 'list', '-mod=readonly', '-m', 'all'], source)
    metadata['modules'] = modules.read_text().splitlines()
    rec.run(label + '-module-verify', ['go', 'mod', 'verify'], source)
    return source, metadata


def bind_environment(rec):
    def run(name, args):
        return rec.run(name, args, REPO).read_text().strip()

    env = rec.env
    host = {'os': run('host-os', ['uname', '-srm']), 'logical_cpus': os.cpu_count(),
            'python': sys.version, 'machine': platform.machine()}
    if platform.system() == 'Darwin':
        host['product_version'] = run('host-product', ['sw_vers'])
        env['CC'] = run('clang-path', ['xcrun', '--find', 'clang'])
        env['SDKROOT'] = run('sdk-path', ['xcrun', '--show-sdk-path'])
        rec.bind_tool(run('linker-path', ['xcrun', '--find', 'ld']))
        rec.bind_tool(Path(env['SDKROOT']) / 'SDKSettings.json')
    else:
        env['CC'] = shutil.which('cc', path=env['PATH'])
    host['cc_version'] = run('cc-version', [env['CC'], '--version'])
    go = json.loads(run('go-settings', ['go', 'env', '-json', 'GOVERSION', 'GOOS', 'GOARCH', 'CGO_ENABLED',
                                        'GOROOT', 'GOTOOLDIR', 'GOMODCACHE', 'CC']))
    for name in ('go', 'git', 'gofmt'):
        rec.bind_tool(shutil.which(name, path=env['PATH']))
    for name in ('compile', 'link', 'cgo'):
        rec.bind_tool(Path(go['GOTOOLDIR']) / name)
    rec.bind_tool(env['CC'])
    rec.bind_tool(sys.executable)
    sqlite = Path(go['GOMODCACHE']) / (SQLITE + '@' + SQLITE_VERSION)
    rec.bind('sqlite-module', sqlite)
    sqlite_module = json.loads(run('sqlite-module', ['go', 'mod', 'download', '-json',
                                                    SQLITE + '@' + SQLITE_VERSION]))
    return sqlite, {'host': host, 'go': go, 'sqlite': sqlite_module, 'environment': dict(env)}


def build(rec, name, source, sqlite, mode, *, counters=False):
    out = rec.root / name
    driver = out / 'driver'
    if counters:
        generator = rec.root / 'generate-report-counters.py'
        if not generator.exists():
            generator.write_text(counter_generator())
            rec.bind_tool(generator)
        rec.run(name + '-generate', [sys.executable, '-B', generator, '--alan', source,
                                     '--sqlite', sqlite, '--driver', DRIVER, '--out', out])
        compact_counters(out)
        rec.run(name + '-format-sqlite', ['gofmt', '-w', out / 'sqlite/alan_measurement.go'])
        rec.bind(name + '-alan', out / 'alan')
        rec.bind(name + '-sqlite', out / 'sqlite')
        for filename in ('generation.json', 'transformations.json', 'measurement.patch'):
            rec.bind_tool(out / filename)
    else:
        driver.mkdir(parents=True)
        for filename in DRIVER_FILES:
            shutil.copyfile(DRIVER / filename, driver / filename)
    make_driver(driver, mode)
    rec.run(name + '-format-driver', ['gofmt', '-w', *(driver / n for n in (
        'main.go', 'resources.go', 'prefix_proof.go', 'transport.go', 'completed.go', 'execution_mode.go'))])
    binary = out / 'driver.bin'
    build_module(rec, name, out / 'alan' if counters else source, driver, binary,
                 sqlite=out / 'sqlite' if counters else None)
    return binary


def execute(rec, row, revision, mode, binary):
    sample = rec.root / 'samples' / row['name']
    sample.mkdir()
    store = sample / 'store/store.sqlite'
    argv = [sys.executable, '-B', P5 / 'sample_usage.py', binary, '-bound', row['bound'],
            '-dir', store.parent, '-sample', row['name'], '-source-label', revision, '-timeout', '29m']
    if row['kind'] == 'profile':
        argv += ['-cpu-profile', sample / 'cpu.pprof', '-alloc-profile', sample / 'allocs.pprof']
    return capture_row(rec, row, revision, mode, binary, store, argv)


def compare_timing(rows):
    # All three effects have one adjacent contrast per block, per bound.
    comparisons = {}
    for bound in BOUNDS:
        per_bound = [r for r in rows if r['design']['kind'] == 'timing' and r['design']['bound'] == bound]
        blocks = []
        for block in range(1, 7):
            samples = {r['design']['arm']: r for r in per_bound if r['design']['block'] == block}
            if any('failure' in samples[arm] for arm in ARMS):
                blocks.append({'block': block, 'failure': 'failed timing row; no paired estimate'})
                continue
            metrics = {arm: {key: samples[arm]['resources'][key]['wall_ns']
                             for key in ('execute_window', 'lifecycle')} for arm in ARMS}
            blocks.append({'block': block, 'differences_ns': {
                f'{a}-minus-{b}': {key: metrics[a][key] - metrics[b][key] for key in metrics[a]}
                for a, b in ((ARMS[0], ARMS[1]), (ARMS[1], ARMS[2]), (ARMS[0], ARMS[2]))
            }})
        comparisons[str(bound)] = blocks
    return comparisons


def campaign(args, root):
    for name in ('tmp', 'samples'):
        (root / name).mkdir()
    env = environment(root, args.gopath.resolve())
    if args.gocache:
        env['GOCACHE'] = str(args.gocache.resolve())
    rec = TaskRecorder(root, env=env)
    for directory in HELPER_DIRS:
        rec.bind(directory, EVIDENCE / directory)
    for tool in (HERE / 'measure.py', HERE / 'execute.go.txt', HERE / 'completed.go.txt',
                 HERE / 'test_measure.py', P5 / 'sample_usage.py'):
        rec.bind_tool(tool)
    pinned_inputs()

    status = rec.run('checkout-status', ['git', 'status', '--porcelain=v1', '--untracked-files=all'], REPO)
    head = rec.run('checkout-head', ['git', 'rev-parse', 'HEAD'], REPO).read_text().strip()
    if args.lane == 'timing' and status.read_text():
        raise ValueError('primary comparison requires clean apparatus checkout')
    if args.lane == 'timing':
        # Require the executable apparatus to match its candidate freeze, even
        # if the live evidence checkout has since added documentation commits.
        for path in [HERE / n for n in ('measure.py', 'test_measure.py', 'execute.go.txt', 'completed.go.txt')]:
            log = rec.run('freeze-' + path.name, ['git', 'show', f'{args.revision}:{path.relative_to(REPO)}'], REPO)
            if sha(log) != sha(path):
                raise ValueError('apparatus differs from frozen candidate: ' + path.name)
    sqlite, toolchain = bind_environment(rec)
    sources = {}
    if args.lane == 'timing':
        base, sources['baseline'] = freeze(rec, BASE, 'baseline')
        candidate, sources['candidate'] = freeze(rec, args.revision, 'candidate')
        for directory in HELPER_DIRS:
            if inventory(candidate / 'docs/evidence' / directory) != inventory(EVIDENCE / directory):
                raise ValueError('apparatus helper differs from frozen candidate: ' + directory)
        if sha(candidate / P5.relative_to(REPO) / 'sample_usage.py') != sha(P5 / 'sample_usage.py'):
            raise ValueError('child usage wrapper differs from frozen candidate')
        rows = timing_design()
        configs = dict(zip(ARMS, ((BASE, base, 'full'), (args.revision, candidate, 'full'),
                                  (args.revision, candidate, 'progress'))))
    else:
        source, sources['single'] = freeze(rec, args.revision, 'single')
        arm = ('base-full' if args.revision == BASE else 'candidate-' + args.mode)
        configs = {arm: (args.revision, source, args.mode)}
        bounds = {'smoke': (1, 8), 'profile': (256,), 'counters': (8, 256)}
        rows = [design(args.lane, bound, arm) for bound in bounds[args.lane]]
    write(root / 'schedule.json', rows)
    binaries = {arm: build(rec, arm, source, sqlite, mode, counters=args.lane == 'counters')
                for arm, (_, source, mode) in configs.items()}
    inputs = rec.snapshot()
    write(root / 'build-inputs.json', inputs)
    manifest = {'format': 'alan/progress-measure-v1', 'lane': args.lane, 'sources': sources,
                'checkout_head': head, 'checkout_status': status.read_text().splitlines(),
                'toolchain': toolchain, 'schedule_sha256': sha(root / 'schedule.json'),
                'build_inputs_sha256': sha(root / 'build-inputs.json'), 'builds': {
                    arm: {'revision': configs[arm][0], 'mode': configs[arm][2],
                          'binary': str(binary), 'sha256': sha(binary)}
                    for arm, binary in binaries.items()}}
    write(root / 'manifest.json', manifest)

    # Build once before the serial samples. Avoid reading all exported sources
    # between timing children; check them before and after the whole schedule.
    sample_rec = TaskRecorder(root, env=rec.env, tools=[*binaries.values(), P5 / 'sample_usage.py',
                                                        Path(sys.executable).resolve()])
    results = []
    for row in rows:
        arm = row['arm']
        revision, _, mode = configs[arm]
        results.append(execute(sample_rec, row, revision, mode, binaries[arm]))
    if rec.snapshot() != inputs:
        raise ValueError('frozen build inputs changed across samples')
    result = {'rows_sha256': sha(root / 'rows.jsonl'), 'manifest_sha256': sha(root / 'manifest.json'),
              'failures': [r['design']['name'] for r in results if 'failure' in r]}
    if args.lane == 'timing':
        result['paired_timing'] = compare_timing(results)
    write(root / 'result.json', result)
    if result['failures']:
        raise SystemExit('one or more sample rows failed; all rows retained')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--out', required=True, type=Path, help='new path under the ignored P6 plan workspace')
    parser.add_argument('--revision', required=True, help='exact source commit; candidate for the timing lane')
    parser.add_argument('--mode', choices=('full', 'progress'))
    parser.add_argument('--lane', choices=('smoke', 'profile', 'counters', 'timing'), default='smoke')
    parser.add_argument('--gopath', type=Path, default=Path.home() / 'go')
    parser.add_argument('--gocache', type=Path, help='optional preexisting build cache')
    args = parser.parse_args()
    if not re.fullmatch('[0-9a-f]{40}', args.revision):
        parser.error('revision must be an exact 40-digit commit')
    if args.lane == 'timing':
        if args.mode is not None or args.revision == BASE:
            parser.error('timing requires a candidate revision and automatically measures all three arms')
    elif args.mode is None or (args.revision == BASE and args.mode != 'full'):
        parser.error('single-arm lanes require --mode; the baseline supports full only')
    # One apparatus process in this workspace at a time. The operator must
    # still keep unrelated host builds and tests off the primary timing window.
    WORKSPACE.mkdir(parents=True, exist_ok=True)
    with (WORKSPACE / 'measurement.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        campaign(args, reserve_output(args.out))


if __name__ == '__main__':
    main()
