#!/usr/bin/env python3
"""Freeze and record the bounded Task 5 evidence routes in a new explicit root."""
import argparse
import fcntl
import json
import os
from pathlib import Path
import platform
import shutil
import sys

from custody import EVIDENCE, HERE, P3, TaskRecorder, extract_source, inventory, read, restore_inputs, separate_inputs, sha, write
from measurement import BOUNDS, TUPLES, analyze, check_schedule, command, row, schedule, validate_report
from overlays import generate, probe_keys, scopes

BASELINE = 'ef38ca47179d86e7544885471ae43e86e1d6351e'
BASELINE_TREE = '4b8f76b5fd84df99e13c4a8381dde65d1b70746b'
TASK4 = '102c62062c61985bcbe5ea8c8c9d12fcf385891a'
TASK4_TREE = 'c69553b9754a91b38709395f13f0084077d14217'
OLD_CLI = 'af6943a5805a9a6c0e8fe77f804264f4765d07ec'
OLD_OWNER = 'd59eaf2976849e450635d252c7bca62eeccbef2e'
PYTHON = '3.9.6'
PATH = '/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin'
ENV = dict(GOMAXPROCS='4', GOGC='100', GOMEMLIMIT='off', CGO_ENABLED='1',
           GOWORK='off', GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off', GOENV='off',
           GOFLAGS='', GODEBUG='', GORACE='', GOTRACEBACK='single',
           PYTHONDONTWRITEBYTECODE='1', LANG='C.UTF-8', LC_ALL='C.UTF-8')


def environment(root, gopath):
    root = root.resolve()
    return dict(ENV, PATH=PATH, HOME=str(Path.home()), GOPATH=str(gopath),
                GOMODCACHE=str(gopath / 'pkg/mod'), TMPDIR=str(root / 'tmp'),
                GOCACHE=str(root / 'go-cache'), STATICCHECK_CACHE=str(root / 'staticcheck-cache'),
                TASK5_PYTHON=str(Path(sys.executable).resolve()))


def sanitize_hardware(data):
    info, = data['SPHardwareDataType']
    if info['chip_type'] != 'Apple M4':
        raise ValueError('measurement host is not Apple M4')
    return {k: info[k] for k in ('chip_type', 'machine_model', 'number_processors', 'physical_memory')}


def freeze(root, repo, baseline, candidate, gopath, *, excluded=False):
    if baseline != BASELINE or len(candidate) != 40 or any(c not in '0123456789abcdef' for c in candidate):
        raise ValueError('explicit exact baseline and candidate commit IDs required')
    if platform.python_version() != PYTHON:
        raise ValueError('expected Python ' + PYTHON)
    root.mkdir(parents=True)
    for name in ('tmp', 'go-cache', 'staticcheck-cache', 'sources', 'git', 'metadata', 'archives', 'builds', 'samples'):
        (root / name).mkdir()
    rec = TaskRecorder(root, env=environment(root, gopath))
    identities = {}
    for name, revision in dict(baseline=baseline, candidate=candidate, old_cli=OLD_CLI, old_owner=OLD_OWNER).items():
        tree = rec.run('tree-' + name, ['git', 'rev-parse', revision + '^{tree}'], repo).read_text().strip()
        if name == 'baseline' and tree != BASELINE_TREE:
            raise ValueError('integrated baseline tree differs')
        actual = rec.run('commit-' + name, ['git', 'rev-parse', revision + '^{commit}'], repo).read_text().strip()
        if actual != revision:
            raise ValueError('not an exact commit ID: ' + revision)
        tar = root / 'archives' / (name + '.tar')
        rec.run('archive-' + name, ['git', 'archive', '--output=' + str(tar), revision], repo)
        source = root / 'sources' / name
        extract_source(tar, source)
        rec.bind(name, source)
        identities[name] = {'commit': revision, 'tree': tree, 'archive_sha256': sha(tar)}
        if name in ('baseline', 'candidate'):
            # Detached HEAD in a separate Git directory. Only provenance-aware
            # gates get GIT_DIR/GIT_WORK_TREE. Source archives never get .git.
            gitdir = root / 'git' / (name + '.git')
            rec.run('git-init-' + name, ['git', 'init', '--bare', gitdir], repo)
            common = rec.run('git-common-' + name, ['git', 'rev-parse', '--path-format=absolute',
                                                    '--git-common-dir'], repo).read_text().strip()
            (gitdir / 'objects/info/alternates').write_text(str(Path(common) / 'objects') + '\n')
            (gitdir / 'HEAD').write_text(revision + '\n')
            provenance = dict(GIT_DIR=str(gitdir), GIT_WORK_TREE=str(source))
            observed = rec.run('git-proof-' + name, ['git', '-C', source, 'rev-parse', 'HEAD', 'HEAD^{tree}'],
                               source, provenance).read_text().splitlines()
            if observed != [revision, tree]:
                raise ValueError('detached provenance disagrees with exported commit/tree')
            rec.bind('git-' + name, gitdir)
    task4_tree = rec.run('task4-tree', ['git', 'rev-parse', TASK4 + '^{tree}'], repo).read_text().strip()
    if task4_tree != TASK4_TREE:
        raise ValueError('Task 4 reference tree differs')
    rec.run('task4-ancestry', ['git', 'merge-base', '--is-ancestor', TASK4, candidate], repo)
    executable_diff = rec.run('task4-executable-bridge',
                              ['git', 'diff', '--name-only', TASK4, candidate, '--', '*.go', 'go.mod',
                               'go.sum', '*.c', '*.h', '*.s', 'go.work', 'go.work.sum'], repo).read_text()
    if executable_diff:
        raise ValueError('reviewed executable sources changed: ' + executable_diff)
    rec.run('source-whitespace', ['git', 'diff', '--check', baseline, candidate], repo)
    # These exports must stay free of nested output and VCS metadata. The
    # recorder also hashes every member before and after each subprocess.
    live = Path(__file__).resolve()
    frozen = rec.sources['candidate'] / live.relative_to(repo)
    if not frozen.is_file() and not excluded:
        raise ValueError('candidate commit does not contain the Task 5 apparatus')
    for directory in ('execution-scaling/task5', 'execution-scaling/apparatus', 'performance-p2', 'performance-p3'):
        if not (excluded and directory == 'execution-scaling/task5') and inventory(EVIDENCE / directory) != inventory(frozen.parents[2] / directory):
            raise ValueError('live controller/import differs from frozen candidate: ' + directory)
        rec.bind('controller-' + directory.replace('/', '-'), EVIDENCE / directory)
    toolchain = bind_toolchain(rec)
    for source in ('baseline', 'candidate'):
        paths = [p for p in rec.sources[source].rglob('*.go')]
        matches = {}
        for path in sorted(paths):
            lines = [(i, line) for i, line in enumerate(path.read_text().splitlines(), 1)
                     if line.startswith('func Fuzz') or '"testing/quick"' in line]
            if lines:
                matches[str(path.relative_to(rec.sources[source]))] = lines
        write(root / 'metadata' / ('fuzz-' + source + '.json'), {
            'source': identities[source], 'matches': matches,
            'status': 'N/A: no fuzz targets or testing/quick imports' if not matches else 'inventory requires review'})
    separate_inputs(rec, 'source', [])
    write(root / 'source-freeze.json', {'format': 'alan/task5-source-v1', 'sources': identities,
                                        'reviewed_executable': {'commit': TASK4, 'tree': TASK4_TREE},
                                        'environment': rec.env, 'toolchain': toolchain, 'excluded': excluded,
                                        'source_inputs_sha256': sha(root / 'source-inputs.json')})
    print('SOURCE-FREEZE SHA256 ' + sha(root / 'source-freeze.json'), flush=True)


def bind_toolchain(rec):
    source = rec.sources['candidate']
    names = ('GOVERSION', 'GOOS', 'GOARCH', 'CGO_ENABLED', 'GOROOT', 'GOTOOLDIR', 'GOMODCACHE', 'CC')
    # system_profiler returns serial and device identifiers. Persist only the
    # required processor/model fields, using the explicit sanitized command.
    hardware = rec.run('host-hardware', [sys.executable, '-B', HERE / 'controller.py', 'hardware-probe'])
    host = read(hardware)
    rec.run('host-os', ['sw_vers'])
    go = read(rec.run('go-settings', ['go', 'env', '-json', *names], source))
    if [go[k] for k in names[:4]] != ['go1.27.4', 'darwin', 'arm64', '1']:
        raise ValueError('unexpected Go toolchain')
    staticcheck = rec.run('staticcheck-version', ['staticcheck', '-version']).read_text().strip()
    if staticcheck != 'staticcheck 2026.2.2 (0.8.2)':
        raise ValueError('unexpected Staticcheck release')
    clang = Path(rec.run('clang-path', ['xcrun', '--find', 'clang']).read_text().strip()).resolve()
    sdk = Path(rec.run('sdk-path', ['xcrun', '--show-sdk-path']).read_text().strip()).resolve()
    sdk_version = rec.run('sdk-version', ['xcrun', '--show-sdk-version']).read_text().strip()
    rec.run('clang-version', [clang, '--version'])
    linker = Path(rec.run('linker-path', ['xcrun', '--find', 'ld']).read_text().strip()).resolve()
    rec.env.update(CC=str(clang), SDKROOT=str(sdk))
    for name in ('go', 'gofmt', 'staticcheck', 'git', 'clang', 'sh', 'xcrun', 'system_profiler', 'sw_vers'):
        rec.bind_tool(shutil.which(name, path=PATH))
    for path in (clang, linker, Path(sys.executable), sdk / 'SDKSettings.json'):
        rec.bind_tool(path)
    rec.bind('go-toolchain', Path(go['GOTOOLDIR']))
    rec.bind('go-stdlib-source', Path(go['GOROOT']) / 'src')
    for relative in ('lib/time/zoneinfo.zip', 'lib/wasm/wasm_exec.js'):
        path = Path(go['GOROOT']) / relative
        if path.is_file():
            rec.bind_tool(path)
    sqlite = Path(go['GOMODCACHE']) / 'github.com/mattn/go-sqlite3@v1.14.52'
    rec.bind('sqlite', sqlite)
    rec.run('sqlite-module', ['go', 'mod', 'download', '-json', 'github.com/mattn/go-sqlite3@v1.14.52'], source)
    return {'host': host, 'go': go, 'sdk': str(sdk), 'sdk_version': sdk_version, 'clang': str(clang),
            'linker': str(linker), 'staticcheck': staticcheck, 'python': sys.version}


def open_freeze(root, digest):
    lock = root / 'source-freeze.json'
    if sha(lock) != digest:
        raise ValueError('source freeze digest mismatch')
    plan = read(lock)
    expected_env = environment(root, Path(plan['environment']['GOPATH']))
    expected_env.update(CC=plan['toolchain']['clang'], SDKROOT=plan['toolchain']['sdk'])
    if plan['environment'] != expected_env:
        raise ValueError('frozen environment differs')
    rec = TaskRecorder(root, env=expected_env)
    restore_inputs(rec, root / 'source-inputs.json', plan['source_inputs_sha256'])
    for name, identity in plan['sources'].items():
        if sha(root / 'archives' / (name + '.tar')) != identity['archive_sha256']:
            raise ValueError('frozen archive changed: ' + name)
    return rec, plan


def variant(design):
    kind = 'ordinary' if design['kind'] == 'profile' else design['kind']
    return design['side'] + '-' + kind


def diagnostic_design():
    rows = [row('diagnostic', b, side, mode, kind='counters', name=f'counters-b{b}-{side}-{mode}')
            for b in BOUNDS for side, mode in TUPLES]
    rows.extend(row('diagnostic', 256, side, mode, kind='counters', reopen=32,
                    name=f'counters-b256-{side}-{mode}-reopen32') for side, mode in TUPLES)
    rows.extend(row('profile', 256, side, mode, kind='profile', name=f'profile-b256-{side}-{mode}')
                for side, mode in TUPLES)
    for bound in (1, 8):
        for kind in ('original', 'counters'):
            rows.append(row('calibration', bound, 'baseline', 'snapshot', kind=kind,
                            name=f'gate-{"compact" if kind == "counters" else kind}-counters-b{bound}'))
    return rows


def build_measurement(rec, plan, component, rows, source_digest):
    output = rec.root / 'builds' / component
    output.mkdir()
    binaries = {}
    for design in rows:
        key = variant(design)
        if key not in binaries:
            side = design['side']
            binaries[key] = str(generate(rec, component + '-' + key, rec.sources[side], output / key,
                                        side, 'ordinary' if design['kind'] == 'profile' else design['kind']))
    records = []
    for design in rows:
        revision = plan['sources'][design['side']]['commit']
        binary = binaries[variant(design)]
        records.append({'design': design, 'revision': revision, 'binary_sha256': sha(binary),
                        'argv': command(rec.root, design, revision, binary)})
    separate_inputs(rec, component, [])
    lock = rec.root / (component + '-freeze.json')
    write(lock, {'source_freeze_sha256': source_digest, 'component': component, 'binaries': binaries,
                 'rows': records, 'inputs_sha256': sha(rec.root / (component + '-inputs.json'))})
    print(component.upper() + '-FREEZE SHA256 ' + sha(lock), flush=True)


def load_measurement(rec, plan, component, source_digest, build_digest):
    lock = rec.root / (component + '-freeze.json')
    if sha(lock) != build_digest:
        raise ValueError('measurement freeze digest mismatch')
    bound = read(lock)
    if bound['source_freeze_sha256'] != source_digest or bound['component'] != component:
        raise ValueError('measurement source/component mismatch')
    restore_inputs(rec, rec.root / (component + '-inputs.json'), bound['inputs_sha256'])
    rows = [r['design'] for r in bound['rows']]
    if component == 'timing':
        check_schedule(rows)
    elif component == 'diagnostics' and rows != diagnostic_design():
        raise ValueError('diagnostic schedule differs')
    for record in bound['rows']:
        design = record['design']
        revision = plan['sources'][design['side']]['commit']
        binary = bound['binaries'][variant(design)]
        if record != {'design': design, 'revision': revision, 'binary_sha256': sha(binary),
                      'argv': command(rec.root, design, revision, binary)}:
            raise ValueError('row command/binary/source changed')
    return bound['rows']


def check_record(rec, record):
    design = record['design']
    sample = rec.root / 'samples' / design['name']
    gate = rec.gates / ('sample-' + design['name'])
    stdout = gate / 'stdout.log'
    outcome = read(gate / 'outcome.json')
    recorded = read(gate / 'command.json')
    post = read(gate / 'after.json')
    if (outcome['exit'] != 0 or outcome['timeout'] or outcome['spawn_error'] or
            recorded['argv'] != record['argv'] or recorded['cwd'] != str(sample) or
            recorded['executable_sha256'] != record['binary_sha256'] or
            post['executable_sha256'] != record['binary_sha256'] or
            recorded['environment'] != dict(rec.env, GOFLAGS='-buildvcs=false') or
            post['stdout_sha256'] != sha(stdout) or post['stderr_sha256'] != sha(gate / 'stderr.log')):
        raise ValueError('row process result, configuration or report identity differs: ' + design['name'])
    before = read(gate / 'before.json')
    after = {k: post[k] for k in ('sources', 'tools')}
    # Sample execution uses exactly the frozen build inputs, including the
    # binary and controllers. Later phases can introduce additional tools.
    if before != after or before != rec.expected:
        raise ValueError('row source/tool custody mismatch: ' + design['name'])
    report = read(stdout)
    validate_report(report, design, record['revision'], sample / 'store/store.sqlite',
                    scopes(), probe_keys(design['side']))
    return report


def execute_rows(rec, component, rows):
    results = rec.root / (component + '-run')
    results.mkdir()
    previous = None
    for index, record in enumerate(rows, 1):
        name = record['design']['name']
        sample = rec.root / 'samples' / name
        sample.mkdir()
        start = {'name': name, 'index': index, 'previous': previous}
        write(results / (name + '-start.json'), start)
        try:
            rec.run('sample-' + name, record['argv'], sample, timeout=1800)
            check_record(rec, record)
        except Exception as error:
            write(results / (name + '-failure.json'), {'error': repr(error)})
            # Preserve the failing row and keep the predeclared serial order.
            previous = name
            continue
        write(results / (name + '-valid.json'), {'report_sha256': sha(rec.gates / ('sample-' + name) / 'stdout.log')})
        previous = name
    failed = sorted(p.name for p in results.glob('*-failure.json'))
    write(results / 'complete.json', {'rows': len(rows), 'failed': failed})
    if failed:
        raise ValueError('frozen rows failed; all attempts retained: ' + ', '.join(failed))


def read_rows(rec, component, rows):
    run = rec.root / (component + '-run')
    terminal = read(run / 'complete.json')
    if terminal != {'rows': len(rows), 'failed': []}:
        raise ValueError('run is incomplete or contains failed attempts')
    expected = {'complete.json'}
    for record in rows:
        name = record['design']['name']
        expected.update((name + '-start.json', name + '-valid.json'))
    if set(p.name for p in run.iterdir()) != expected:
        raise ValueError('missing/extra recorded rows')
    reports = {}
    previous, last = None, 0
    for index, record in enumerate(rows, 1):
        name = record['design']['name']
        if read(run / (name + '-start.json')) != {'name': name, 'index': index, 'previous': previous}:
            raise ValueError('row sequence changed')
        gate = rec.gates / ('sample-' + name)
        current = read(gate / 'command.json')['started_unix']
        if current < last:
            raise ValueError('recorded sample order/overlap changed')
        last = read(gate / 'outcome.json')['ended_unix']
        if last < current:
            raise ValueError('invalid recorded time interval')
        reports[name] = check_record(rec, record)
        if read(run / (name + '-valid.json'))['report_sha256'] != sha(gate / 'stdout.log'):
            raise ValueError('validated report hash differs')
        previous = name
    return reports


def finish_diagnostics(rec, rows, reports):
    from compare_counters import compare
    output = rec.root / 'diagnostic-analysis'
    output.mkdir()
    calibration = output / 'calibration'
    calibration.mkdir()
    for record in rows:
        name = record['design']['name']
        if record['design']['kind'] == 'profile':
            sample = rec.root / 'samples' / name
            for kind, flags in [('cpu', []), ('alloc', ['-alloc_space'])]:
                profile = sample / (kind + '.pprof')
                rec.bind_tool(profile)
                rec.run(name + '-pprof-' + kind, ['go', 'tool', 'pprof', '-top', *flags, record['argv'][0], profile], sample)
        if record['design']['experiment'] == 'calibration':
            directory = calibration / name
            directory.mkdir()
            shutil.copyfile(rec.gates / ('sample-' + name) / 'stdout.log', directory / 'stdout.json')
    write(output / 'calibration.json', compare(calibration))
    diagnostic = {}
    for record in rows:
        design = record['design']
        if design['kind'] != 'counters':
            continue
        report = reports[design['name']]
        diagnostic[design['name']] = {
            'design': design,
            'stages': {f"{s['name']}/{s.get('position', 0)}": s['counter_deltas'] for s in report['stages']},
            'execute_registry': report['execute_counter_registry'],
            'prefix_source': str(rec.gates / ('sample-' + design['name']) / 'stdout.log'),
        }
    write(output / 'counts.json', diagnostic)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('phase', choices=('freeze', 'hardware-probe', 'build-timing', 'run-timing',
                        'analyze-timing', 'build-diagnostics', 'run-diagnostics', 'analyze-diagnostics',
                        'smoke', 'build-qualification', 'qualification', 'differential', 'recovery', 'reproduce',
                        'excluded-differential-smoke', 'excluded-weather-smoke', 'excluded-historical-smoke'))
    parser.add_argument('--root', type=Path)
    parser.add_argument('--repo', type=Path, default=Path.cwd())
    parser.add_argument('--baseline')
    parser.add_argument('--candidate')
    parser.add_argument('--gopath', type=Path, default=Path.home() / 'go')
    parser.add_argument('--source-digest')
    parser.add_argument('--build-digest')
    parser.add_argument('--outside', type=Path)
    parser.add_argument('--excluded', action='store_true', help='permit uncommitted apparatus in an excluded development freeze')
    args = parser.parse_args()
    if args.phase == 'hardware-probe':
        import subprocess
        print(json.dumps(sanitize_hardware(json.loads(subprocess.check_output(
            ['system_profiler', 'SPHardwareDataType', '-json'])))))
        return
    if not args.root or (args.phase != 'freeze' and not args.source_digest):
        parser.error('root and applicable source digest are required')
    root = args.root.resolve()
    if args.phase == 'freeze':
        freeze(root, args.repo.resolve(), args.baseline, args.candidate or '', args.gopath.resolve(), excluded=args.excluded)
        return
    with (root / 'session.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        rec, plan = open_freeze(root, args.source_digest)
        if plan['excluded'] and args.phase not in ('smoke', 'build-qualification', 'excluded-historical-smoke',
                                                     'excluded-differential-smoke', 'excluded-weather-smoke'):
            raise ValueError('excluded development freeze cannot run a confirmatory phase')
        if args.phase == 'smoke':
            rows = [row('excluded-smoke', 1, 'baseline', 'snapshot', name='excluded-smoke-b1-baseline'),
                    row('excluded-smoke', 8, 'candidate', 'targeted', name='excluded-smoke-b8-targeted'),
                    row('excluded-smoke', 1, 'baseline', 'snapshot', kind='original', name='excluded-smoke-b1-original'),
                    row('excluded-smoke', 1, 'baseline', 'snapshot', kind='counters', name='excluded-smoke-b1-compact'),
                    row('excluded-smoke', 8, 'candidate', 'targeted', kind='counters', name='excluded-smoke-b8-compact')]
            rows += [row('excluded-smoke', 8, 'candidate', 'targeted', kind='counters', reopen=1,
                         name='excluded-smoke-b8-compact-reopen1'),
                     row('excluded-smoke', 8, 'baseline', 'snapshot', kind='original',
                         name='excluded-smoke-b8-original'),
                     row('excluded-smoke', 8, 'baseline', 'snapshot', kind='counters',
                         name='excluded-smoke-b8-baseline-compact')]
            build_measurement(rec, plan, 'smoke', rows, args.source_digest)
            lock_path = root / 'smoke-freeze.json'
            rows = load_measurement(rec, plan, 'smoke', args.source_digest, sha(lock_path))
            execute_rows(rec, 'smoke', rows)
        elif args.phase in ('build-qualification', 'qualification', 'differential', 'recovery', 'reproduce',
                            'excluded-differential-smoke', 'excluded-weather-smoke', 'excluded-historical-smoke'):
            import qualification
            qualification.dispatch(rec, plan, args)
        else:
            action, component = args.phase.split('-', 1)
            if action == 'build':
                build_measurement(rec, plan, component, schedule() if component == 'timing' else diagnostic_design(),
                                  args.source_digest)
            else:
                rows = load_measurement(rec, plan, component, args.source_digest, args.build_digest)
                if action == 'run':
                    execute_rows(rec, component, rows)
                elif component == 'timing':
                    reports = read_rows(rec, component, rows)
                    write(root / 'timing-analysis.json', analyze([r['design'] for r in rows], reports))
                else:
                    finish_diagnostics(rec, rows, read_rows(rec, component, rows))


if __name__ == '__main__':
    main()
