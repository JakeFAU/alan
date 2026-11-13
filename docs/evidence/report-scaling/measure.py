#!/usr/bin/env python3
"""Record excluded baseline rows and separate report-scoped SQL diagnostics."""
import argparse
from collections import Counter
import difflib
import json
from pathlib import Path
import hashlib
import re
import shutil
import sys

from oracle import BASE, oracle

HERE = Path(__file__).resolve().parent
EVIDENCE = HERE.parent
REPO = EVIDENCE.parents[1]
P4 = EVIDENCE / 'execution-scaling'
sys.path.insert(0, str(P4 / 'task5'))
from custody import P2, TaskRecorder, extract_source, read, sha, write
from controller import environment
from overlays import build_module, scopes
from measurement import BOUNDS, partition, resources, row, command, validate_report
from apparatus import compact_counters, patch_driver, pinned_inputs, replace

BASE_TREE = 'cb6dc03624010d13f8af8d045af44e490110bfaa'
DRIVER = P2 / 'driver'
ROOT_PROBE = 'work.report.roots_emitted'


def strengthen_driver(driver):
    main = driver / 'main.go'
    patch_driver(driver)
    text = main.read_text()
    text = replace(text, 'name == "approval_snapshot_basis" ||',
                   'name == "approval_snapshot_basis" || name == "approval_targeted_basis" ||')
    text = replace(text, '\t\t_, e = store.Run(ctx, opts)\n\t\treturn e',
                   '''        resumed, e := store.Run(ctx, opts)
        if e != nil { return e }
        _, e = cuts.verify(resumed, r.Bound, false)
        return e''')
    text = replace(text, 'for _, run := range replayed.Runs {', '''if !reflect.DeepEqual(replayed.Roots, cut.Roots) {
        return errors.New("replay changed original root evidence")
    }
    for _, run := range replayed.Runs {''')
    main.write_text(text)

    proof = driver / 'prefix_proof.go'
    text = proof.read_text()
    text = replace(text, 'limits := aruntime.ResourceLimits', '''if root.PendingInvocations == nil || root.UnresolvedExternal == nil || root.UnknownExternal == nil {
        return errors.New("nil root evidence list")
    }
    limits := aruntime.ResourceLimits''')
    text = replace(text, 'limits := aruntime.ResourceLimits', '''if root.Charges.OperationID != "" || root.Charges.Kind != "" || root.Charges.DecisionOrder != 0 {
        return errors.New("aggregate charge contains a single-operation identity")
    }
    limits := aruntime.ResourceLimits''')
    text = replace(text, 'outcome.ResponsePresent || len(outcome.Response) != 0',
                   'outcome.ResponsePresent || outcome.Response != nil')
    text = replace(text, 'for _, root := range rr.Roots {', '''if rr.OpenRuns == nil || rr.UnresolvedIntents == nil || rr.PendingDeliveries == nil ||
        rr.UnknownExternal == nil || rr.UnknownAdapters == nil {
        return cutProof{}, errors.New("nil report evidence list")
    }
    for i, root := range rr.Roots {
        if i > 0 && rr.Roots[i-1].RootID >= root.RootID {
            return cutProof{}, errors.New("root output order")
        }''')
    text = replace(text, 'for _, run := range rr.OpenRuns {', '''for i, run := range rr.OpenRuns {
        if i > 0 {
            prev := rr.OpenRuns[i-1]
            if prev.Subject > run.Subject || (prev.Subject == run.Subject && prev.Generation >= run.Generation) {
                return cutProof{}, errors.New("open run output order")
            }
        }''')
    proof.write_text(text)


def build_driver(rec, source, kind, sqlite=None):
    out = rec.root / kind
    if kind == 'counters':
        # Leave SQLite probes untouched. Replace the obsolete P2 executor
        # anchors with the smallest report-only scope and emission counter.
        original = P2 / 'instrument/generate.py'
        code = replace(original.read_text(), 'HERE = pathlib.Path(__file__).resolve().parent',
                       f'HERE = pathlib.Path({str(original.parent)!r})')
        start = code.index('functions={')
        end = code.index("(a.out/'alan/measurecounter').mkdir()", start)
        code = code[:start] + f'''SCOPES = {{'runner.go': [('finishRun', 8)]}}
functions = {{'runner.go': []}}
def instrument_finish_run(s):
    s = imported(s)
    anchor = 'func (s *Store) finishRun(ctx context.Context, report RunReport, quiescent bool, runErr error) (RunReport, error) {{'
    s = replace(s, anchor, anchor+'\\n ctx = scalingprobe.Scope(ctx, 8)')
    anchor = 'report.Roots = roots'
    return replace(s, anchor, anchor+'\\n scalingprobe.Add("{ROOT_PROBE}", uint64(len(roots)))')
edit('alan/internal/executor/runner.go', instrument_finish_run)
''' + code[end:]
        generator = rec.root / 'generate-report-counters.py'
        generator.write_text(code)
        rec.bind_tool(generator)
        rec.run('generate-counters', [sys.executable, '-B', generator, '--alan', source,
                                     '--sqlite', sqlite, '--driver', DRIVER, '--out', out])
        compact_counters(out)
        rec.run('format-sqlite-probes', ['gofmt', '-w', out / 'sqlite/alan_measurement.go'])
        rec.bind('counter-alan', out / 'alan')
        rec.bind('counter-sqlite', out / 'sqlite')
        for name in ('generation.json', 'transformations.json', 'measurement.patch'):
            rec.bind_tool(out / name)
    else:
        out.mkdir()
        shutil.copytree(DRIVER, out / 'driver')
    driver = out / 'driver'
    strengthen_driver(driver)
    rec.run('format-' + kind, ['gofmt', '-w', driver])
    patch = out / 'driver.patch'
    differences = []
    for path in sorted(driver.glob('*.go')):
        original = DRIVER / path.name
        differences.extend(difflib.unified_diff(
            original.read_text().splitlines(True) if original.exists() else [],
            path.read_text().splitlines(True), fromfile='p2/' + path.name, tofile=kind + '/' + path.name))
    patch.write_text(''.join(differences))
    rec.bind_tool(patch)
    binary = out / 'driver.bin'
    build_module(rec, kind, out / 'alan' if sqlite else source, driver, binary,
                 sqlite=out / 'sqlite' if sqlite else None)
    return binary


def attribution(report, bound):
    # Run only after Task 5 validates monotone deltas and the complete registry.
    window = next(s for s in report['stages'] if s['name'] == 'execute_window')
    compact = window['counter_deltas']
    registry = report['execute_counter_registry']
    if compact.get(ROOT_PROBE) != bound**2:
        raise ValueError('wrong full report root emission count')
    partition(registry, compact, scopes())
    by_family = {}
    for key, count in registry.items():
        prefix = 'sql.scope.finishRun.class.'
        if not key.startswith(prefix):
            continue
        match = re.fullmatch(r'(?:unknown|read_write_capable|read_only_configured)'
                             r'\.post_autocommit\.(?:unknown|off|on)\.template\.(.+)\.([a-z_]+)',
                             key[len(prefix):])
        if not match:
            raise ValueError('malformed finishRun SQL key')
        family, metric = match.groups()
        by_family.setdefault(family, Counter())[metric] += count
    metrics = ('executions', 'sqlite_row_returns', 'vm_steps', 'fullscan_steps')
    report_totals = {metric: compact['sql.scope_total.finishRun.' + metric] for metric in metrics}
    if not 0 < report_totals['executions'] < compact['sql.total.executions']:
        raise ValueError('finishRun scope did not partition executed work')
    return {'roots_emitted': compact[ROOT_PROBE],
            'execute_sql': {metric: compact['sql.total.' + metric] for metric in metrics},
            'finishRun_sql': report_totals,
            'finishRun_families': {
                family: {metric: data[metric] for metric in metrics}
                for family, data in sorted(by_family.items())
            }}


def measure_row(rec, design, revision, binary):
    name = design['name']
    sample = rec.root / 'samples' / name
    sample.mkdir(parents=True)
    try:
        output = rec.run(name, command(rec.root, design, revision, binary), sample, timeout=1800)
        report = read(output)
        # The original-counter route validates arithmetic and registry
        # partition without imposing the old P4-specific Go work keys.
        validation = dict(design, kind='original') if design['kind'] == 'counters' else design
        validate_report(report, validation, revision, sample / 'store/store.sqlite', scopes())
        result = {'design': design, 'report_sha256': sha(output),
                  'proof_sha256': sha_proof(report), 'workload': {
                      key: report[key] for key in ('completed', 'spent', 'selected', 'protected_approvals',
                                                  'reductions', 'replay_history', 'replay_intents')},
                  'contact_count': len(report['contacts'])}
        if design['kind'] == 'counters':
            result['attribution'] = attribution(report, design['bound'])
        elif design['kind'] == 'ordinary':
            result['resources'] = resources(report)
        write(sample / 'validated.json', result)
        return result
    except Exception as error:
        result = {'design': design, 'failure': repr(error)}
        write(sample / 'failure.json', result)
        return result


def sha_proof(report):
    facts = {key: report[key] for key in ('root_membership', 'evidence', 'contacts')}
    facts['cuts'] = [prefix['cut'] for prefix in report['prefixes']]
    return hashlib.sha256(json.dumps(facts, sort_keys=True, separators=(',', ':')).encode()).hexdigest()


def freeze(rec, revision):
    def run(name, argv, cwd=REPO):
        return rec.run(name, argv, cwd).read_text().strip()

    tree = run('source-tree', ['git', 'rev-parse', revision + '^{tree}'])
    status = run('checkout-status', ['git', 'status', '--porcelain=v1', '--untracked-files=all'])
    head = run('checkout-head', ['git', 'rev-parse', 'HEAD'])
    if revision == BASE and tree != BASE_TREE:
        raise ValueError('wrong baseline tree')
    if status and revision != BASE:
        raise ValueError('non-baseline measurements require a clean checkout')
    if status:
        # During Task 1 only the new, hashed apparatus can be untracked.
        expected = '?? docs/evidence/report-scaling/'
        if any(not line.startswith(expected) for line in status.splitlines()):
            raise ValueError('dirty checkout outside apparatus')
    archive = rec.root / 'source.tar'
    run('source-archive', ['git', 'archive', '--format=tar', '-o', archive, revision])
    source = rec.root / 'source'
    extract_source(archive, source)
    rec.bind('alan-source', source)
    go = json.loads(run('go-settings', ['go', 'env', '-json', 'GOVERSION', 'GOOS', 'GOARCH',
                                        'CGO_ENABLED', 'GOROOT', 'GOTOOLDIR', 'GOMODCACHE', 'CC'], source))
    if [go[k] for k in ('GOVERSION', 'GOOS', 'GOARCH', 'CGO_ENABLED')] != ['go1.27.4', 'darwin', 'arm64', '1']:
        raise ValueError('unexpected toolchain')
    sqlite = Path(go['GOMODCACHE']) / 'github.com/mattn/go-sqlite3@v1.14.52'
    rec.bind('sqlite-source', sqlite)
    for name in ('go', 'git', 'gofmt'):
        rec.bind_tool(shutil.which(name, path=rec.env['PATH']))
    for name in ('compile', 'link', 'cgo'):
        rec.bind_tool(Path(go['GOTOOLDIR']) / name)
    rec.bind_tool(sys.executable)
    clang = run('clang-path', ['xcrun', '--find', 'clang'])
    sdk = run('sdk-path', ['xcrun', '--show-sdk-path'])
    rec.bind_tool(clang)
    rec.bind_tool(run('linker-path', ['xcrun', '--find', 'ld']))
    rec.bind_tool(Path(sdk) / 'SDKSettings.json')
    rec.env.update(CC=clang, SDKROOT=sdk)
    meta = {'commit': revision, 'tree': tree, 'archive_sha256': sha(archive),
            'checkout_head': head, 'checkout_status': status.splitlines(), 'environment': rec.env,
            'go': go, 'python': sys.version, 'os': run('host-os', ['sw_vers']),
            'cpu': run('host-cpu', ['sysctl', '-n', 'machdep.cpu.brand_string']),
            'logical_cpu_count': run('host-cpu-count', ['sysctl', '-n', 'hw.logicalcpu']),
            'memory_bytes': run('host-memory', ['sysctl', '-n', 'hw.memsize']),
            'clang': run('clang-version', [clang, '--version']),
            'sdk_version': run('sdk-version', ['xcrun', '--show-sdk-version']),
            'go_mod_sha256': sha(source / 'go.mod'), 'go_sum_sha256': sha(source / 'go.sum')}
    meta['modules'] = run('resolved-modules', ['go', 'list', '-mod=readonly', '-m', 'all'], source).splitlines()
    meta['sqlite_module'] = json.loads(run('sqlite-module', ['go', 'mod', 'download', '-json',
                                                             'github.com/mattn/go-sqlite3@v1.14.52'], source))
    run('module-verify', ['go', 'mod', 'verify'], source)
    write(rec.root / 'source-freeze.json', meta)
    return source, sqlite


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', required=True, type=Path, help='new raw evidence root')
    parser.add_argument('--revision', default=BASE, help='exact source commit')
    parser.add_argument('--mode', choices=('baseline', 'smoke', 'timing', 'counters'), default='baseline')
    parser.add_argument('--gopath', default=Path.home() / 'go', type=Path)
    parser.add_argument('--gocache', type=Path, help='optional preexisting Go build cache')
    args = parser.parse_args()
    root = args.root.resolve()
    if not re.fullmatch('[0-9a-f]{40}', args.revision):
        parser.error('an exact commit is required')
    if args.mode == 'baseline' and args.revision != BASE:
        parser.error('the baseline route is pinned to the original source')
    if root.is_relative_to(REPO) and not root.is_relative_to(REPO / '.superpowers/sdd/plan-report-scaling'):
        parser.error('keep raw evidence outside the checkout or in this plan’s ignored workspace')
    root.mkdir()
    (root / 'tmp').mkdir()
    env = environment(root, args.gopath.resolve())
    if args.gocache:
        env['GOCACHE'] = str(args.gocache.resolve())
    rec = TaskRecorder(root, env=env)
    for name in ('execution-scaling/task5', 'execution-scaling/apparatus', 'performance-p2/driver',
                 'performance-p2/instrument', 'performance-p3/campaign', 'performance-p3/final',
                 'performance-p3/qualification'):
        rec.bind(name, EVIDENCE / name)
    for name in ('measure.py', 'oracle.py', 'report_frozen_test.go.txt'):
        rec.bind_tool(HERE / name)
    pinned_inputs()
    source, sqlite = freeze(rec, args.revision)
    failures = []
    if args.mode == 'baseline':
        try:
            rec.run('ordinary-tests', ['go', 'test', '-mod=readonly', '-count=1', '-json', './...'], source)
        except RuntimeError as error:
            failures.append(repr(error))
        if (HERE / 'report_frozen_test.go.txt').read_text() != oracle(REPO):
            raise ValueError('oracle differs from original baseline')
        oracle_overlay = root / 'report_frozen_test.go'
        shutil.copyfile(HERE / 'report_frozen_test.go.txt', oracle_overlay)
        mapping = root / 'oracle-overlay.json'
        write(mapping, {'Replace': {str(source / 'internal/executor/report_frozen_test.go'): str(oracle_overlay)}})
        rec.bind_tool(oracle_overlay)
        rec.bind_tool(mapping)
        try:
            rec.run('oracle-compile', ['go', 'test', '-mod=readonly', '-tags=integration', '-run=^$',
                                       '-overlay=' + str(mapping), './internal/executor'], source)
        except RuntimeError as error:
            failures.append(repr(error))

    results = []
    if args.mode != 'counters':
        binary = build_driver(rec, source, 'ordinary')
        bounds = (1, 8) if args.mode == 'smoke' else BOUNDS
        for bound in bounds:
            design = row('excluded', bound, 'baseline', 'targeted', name=f'excluded-b{bound}')
            results.append(measure_row(rec, design, args.revision, binary))
    if args.mode != 'timing':
        binary = build_driver(rec, source, 'counters', sqlite)
        bound = 8 if args.mode == 'smoke' else 256
        design = row('diagnostic', bound, 'baseline', 'targeted', kind='counters', name=f'counters-b{bound}')
        results.append(measure_row(rec, design, args.revision, binary))

    gates = {}
    for gate in sorted(rec.gates.iterdir()):
        gates[gate.name] = {p.name: sha(p) for p in sorted(gate.iterdir())}
        if (gate / 'outcome.json').exists():
            gates[gate.name]['exit'] = read(gate / 'outcome.json')['exit']
    write(root / 'input-inventory.json', rec.snapshot())
    write(root / 'results.json', {'source_sha256': sha(root / 'source-freeze.json'),
                                  'input_inventory_sha256': sha(root / 'input-inventory.json'),
                                  'rows': results, 'failures': failures, 'gates': gates})
    if failures or any('failure' in result for result in results):
        raise SystemExit('one or more gates failed; raw evidence retained')


if __name__ == '__main__':
    main()
