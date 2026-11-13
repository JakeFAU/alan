#!/usr/bin/env python3
"""Freeze, execute and reread the P5 full-report comparison on its original host."""
import argparse
import fcntl
import os
from pathlib import Path
import platform
import re
import sys

from measure import BASE, BASE_TREE, EVIDENCE, HERE, REPO, attribution, build_driver, pinned_inputs
from controller import OLD_CLI, OLD_OWNER, bind_toolchain, environment
from custody import TaskRecorder, extract_source, inventory, members, read, restore_inputs, separate_inputs, sha, write
from measurement import BOUNDS, command, resources, row, summary, validate_report
from overlays import scopes

SIDES = ('baseline', 'candidate')
PINS = {'baseline': (BASE, BASE_TREE),
        'old_cli': (OLD_CLI, '5d1f64029c7fd764ddd1941bc77b831beb3bdcb3'),
        'old_owner': (OLD_OWNER, 'dc082ea07e4c24d8f36e56d756771f7c1154b0ec')}
HELPERS = ('execution-scaling/task5', 'execution-scaling/apparatus', 'performance-p2/driver',
           'performance-p2/instrument', 'performance-p3/campaign', 'performance-p3/final',
           'performance-p3/qualification')
WORKSPACE = REPO / '.superpowers/sdd/plan-report-scaling'


def designs(component):
    if component == 'counters':
        return [row('diagnostic', bound, side, 'targeted', kind='counters',
                    name=f'counters-b{bound}-{side}') for bound in (8, 256) for side in SIDES]
    if component == 'smoke':
        return [row('excluded-smoke', bound, side, 'targeted', kind=kind,
                    name=f'excluded-{kind}-b{bound}-{side}')
                for kind, bounds in (('ordinary', (1, 8)), ('counters', (8,)))
                for bound in bounds for side in SIDES]
    if component != 'timing':
        raise ValueError('unknown measurement component')
    rows = [row('warmup', bound, side, 'targeted') for bound in BOUNDS for side in SIDES]
    for bound in BOUNDS:
        for pair in range(1, 7):
            for arm in ('AB' if pair % 2 else 'BA'):
                rows.append(row('full-report', bound, SIDES[arm == 'B'], 'targeted', pair, arm))
    return rows


def freeze(root, revision, gopath, excluded):
    root.mkdir()
    for name in ('tmp', 'sources', 'archives', 'git', 'builds', 'samples'):
        (root / name).mkdir()
    rec = TaskRecorder(root, env=environment(root, gopath))
    rec.bind_tool(Path(__file__))
    rec.bind_tool(HERE / 'sample_usage.py')
    head = rec.run('checkout-head', ['git', 'rev-parse', 'HEAD'], REPO).read_text().strip()
    status = rec.run('checkout-status', ['git', 'status', '--porcelain=v1', '--untracked-files=all'],
                     REPO).read_text()
    if head != revision or status:
        raise ValueError('freeze requires the exact candidate checked out clean')
    if platform.python_version() != '3.9.6':
        raise ValueError('expected the pinned Python 3.9.6')
    rec.run('candidate-ancestry', ['git', 'merge-base', '--is-ancestor', BASE, revision], REPO)
    rec.run('frozen-diff-check', ['git', 'diff', '--check', BASE, revision], REPO)
    common = Path(rec.run('git-common', ['git', 'rev-parse', '--path-format=absolute',
                                       '--git-common-dir'], REPO).read_text().strip())
    identities = {}
    for side, (commit, tree) in dict(PINS, candidate=(revision, None)).items():
        got = rec.run('identity-' + side, ['git', 'rev-parse', commit + '^{commit}', commit + '^{tree}'],
                      REPO).read_text().splitlines()
        if len(got) != 2 or got[0] != commit or (tree and got[1] != tree):
            raise ValueError('frozen commit/tree differs: ' + side)
        archive = root / 'archives' / (side + '.tar')
        rec.run('archive-' + side, ['git', 'archive', '--output=' + str(archive), commit], REPO)
        source = root / 'sources' / side
        extract_source(archive, source)
        rec.bind(side, source)
        identities[side] = {'commit': commit, 'tree': got[1], 'archive_sha256': sha(archive)}
        if side in SIDES:
            gitdir = root / 'git' / (side + '.git')
            rec.run('git-init-' + side, ['git', 'init', '--bare', gitdir], REPO)
            (gitdir / 'objects/info/alternates').write_text(str(common / 'objects') + '\n')
            (gitdir / 'HEAD').write_text(commit + '\n')
            rec.bind('git-' + side, gitdir)
            proof = rec.run('git-proof-' + side, ['git', 'rev-parse', 'HEAD', 'HEAD^{tree}'], source,
                            {'GIT_DIR': str(gitdir), 'GIT_WORK_TREE': str(source)})
            if proof.read_text().splitlines() != got:
                raise ValueError('detached Git metadata disagrees with source archive')
    frozen = rec.sources['candidate'] / 'docs/evidence'
    for name in HELPERS + ('report-scaling',):
        if inventory(EVIDENCE / name) != inventory(frozen / name):
            raise ValueError('live apparatus differs from frozen candidate: ' + name)
        rec.bind('apparatus-' + name.replace('/', '-'), EVIDENCE / name)
    pinned_inputs()
    toolchain = bind_toolchain(rec)
    # Xcode command shims and their selected executable are distinct inputs.
    for name in ('git', 'clang', 'ld'):
        log = rec.run('selected-' + name, ['xcrun', '--find', name])
        rec.bind_tool(log.read_text().strip())
    for side in SIDES:
        rec.run(side + '-module-resolution', ['go', 'list', '-mod=readonly', '-m', 'all'], rec.sources[side])
    artifacts = [root / 'archives' / (side + '.tar') for side in identities]
    separate_inputs(rec, 'source', artifacts)
    lock = root / 'source-freeze.json'
    write(lock, {'format': 'alan/p5-source-v1', 'sources': identities, 'excluded': excluded,
                 'environment': rec.env, 'toolchain': toolchain,
                 'inputs_sha256': sha(root / 'source-inputs.json')})
    print('SOURCE FREEZE ' + sha(lock), flush=True)


def open_freeze(root, digest):
    lock = root / 'source-freeze.json'
    if sha(lock) != digest:
        raise ValueError('source freeze digest differs')
    plan = read(lock)
    env = environment(root, Path(plan['environment']['GOPATH']))
    env.update(CC=plan['toolchain']['clang'], SDKROOT=plan['toolchain']['sdk'])
    if plan['format'] != 'alan/p5-source-v1' or env != plan['environment']:
        raise ValueError('source format or environment differs')
    rec = TaskRecorder(root, env=env)
    restore_inputs(rec, root / 'source-inputs.json', plan['inputs_sha256'])
    for side, identity in plan['sources'].items():
        if sha(root / 'archives' / (side + '.tar')) != identity['archive_sha256']:
            raise ValueError('source archive changed: ' + side)
    return rec, plan


def binary_key(component, design):
    return design['side'] + '-' + design['kind'] if component == 'smoke' else design['side']


def row_records(root, plan, component, binaries):
    expected = []
    for design in designs(component):
        binary = binaries[binary_key(component, design)]
        revision = plan['sources'][design['side']]['commit']
        expected.append({'design': design, 'revision': revision, 'binary_sha256': sha(binary),
                         'argv': [str(Path(sys.executable).resolve()), '-B', str(HERE / 'sample_usage.py'),
                                  *command(root, design, revision, binary)]})
    return expected


def require_rows(root, plan, component, binaries, rows):
    if rows != row_records(root, plan, component, binaries):
        raise ValueError('missing, altered, extra or reordered source/binary/command rows')
    return rows


def seal_build(rec, component, source_digest, extra=None):
    separate_inputs(rec, component, [])
    lock = rec.root / (component + '-freeze.json')
    write(lock, {'component': component, 'source_sha256': source_digest,
                 'inputs_sha256': sha(rec.root / (component + '-inputs.json')), **(extra or {})})
    print(component.upper() + ' FREEZE ' + sha(lock), flush=True)


def open_build(rec, component, source_digest, build_digest):
    lock = rec.root / (component + '-freeze.json')
    if sha(lock) != build_digest:
        raise ValueError('build freeze digest differs')
    binding = read(lock)
    if (binding['component'], binding['source_sha256']) != (component, source_digest):
        raise ValueError('build/source component differs')
    restore_inputs(rec, rec.root / (component + '-inputs.json'), binding['inputs_sha256'])
    return binding


def build_measurement(rec, plan, component, digest):
    root = rec.root / 'builds' / component
    root.mkdir()
    sqlite = rec.sources['sqlite']
    binaries = {}
    for design in designs(component):
        key = binary_key(component, design)
        if key in binaries:
            continue
        directory = root / key
        directory.mkdir()
        arm = TaskRecorder(directory, env=rec.env, sources=dict(rec.sources), tools=list(rec.tools))
        kind = design['kind']
        binary = build_driver(arm, rec.sources[design['side']], kind, sqlite if kind == 'counters' else None)
        for name in arm.sources.keys() - rec.sources.keys():
            rec.bind(key + '-' + name, arm.sources[name])
        for tool in arm.tools:
            rec.bind_tool(tool)
        binaries[key] = str(binary)
    seal_build(rec, component, digest, {'binaries': binaries,
              'rows': row_records(rec.root, plan, component, binaries)})


def checked_report(rec, record):
    design = record['design']
    sample = rec.root / 'samples' / design['name']
    gate = rec.gates / ('sample-' + design['name'])
    cmd, outcome, after = (read(gate / name) for name in ('command.json', 'outcome.json', 'after.json'))
    if (type(outcome['exit']) is not int or outcome['exit'] != 0 or outcome['timeout'] is not False
            or outcome['spawn_error'] is not None or cmd['expected_exit'] != 0
            or cmd['argv'] != record['argv'] or cmd['cwd'] != str(sample)
            or cmd['executable'] != record['argv'][0]
            or cmd['executable_sha256'] != rec.expected['tools'][record['argv'][0]]
            or after['executable_sha256'] != cmd['executable_sha256']
            or rec.expected['tools'][record['argv'][3]] != record['binary_sha256']
            or cmd['environment'] != dict(rec.env, GOFLAGS='-buildvcs=false')
            or after['stdout_sha256'] != sha(gate / 'stdout.log')
            or after['stderr_sha256'] != sha(gate / 'stderr.log')
            or read(gate / 'before.json') != rec.expected
            or {k: after[k] for k in ('sources', 'tools')} != rec.expected):
        raise ValueError('recorded process, report or input identity differs: ' + design['name'])
    store = sample / 'store/store.sqlite'
    report = read(gate / 'stdout.log')
    validation = dict(design, kind='original') if design['kind'] == 'counters' else design
    validate_report(report, validation, record['revision'], store, scopes())
    if design['kind'] == 'counters':
        attribution(report, design['bound'])
    usage = read(gate / 'stderr.log')
    maximum = usage.get('process_maxrss_bytes')
    sampled = max(stage['process_maxrss_cumulative'] for stage in report['stages'])
    if (type(usage.get('child_exit')) is not int or usage['child_exit'] != 0 or
            type(maximum) is not int or maximum < sampled or maximum <= 0):
        raise ValueError('missing or inconsistent post-exit process RSS')
    report['process_high_water_bytes'] = maximum
    store_members = members(store)
    if '' not in store_members:
        raise ValueError('sample missing Store')
    return report, {'gate': inventory(gate), 'store_members': store_members}


def execute_rows(rec, component, rows):
    run = rec.root / (component + '-run')
    run.mkdir()
    previous, failed = None, []
    for index, record in enumerate(rows, 1):
        name = record['design']['name']
        sample = rec.root / 'samples' / name
        sample.mkdir()
        write(run / (name + '-start.json'), {'index': index, 'name': name, 'previous': previous})
        try:
            rec.run('sample-' + name, record['argv'], sample, timeout=1800)
            _, proof = checked_report(rec, record)
            write(run / (name + '-valid.json'), proof)
        except Exception as error:
            failed.append(name)
            write(run / (name + '-failure.json'), {'error': repr(error)})
        previous = name
    write(run / 'complete.json', {'rows': len(rows), 'failed': failed})
    if failed:
        raise ValueError('frozen row failures retained: ' + ', '.join(failed))


def read_rows(rec, component, rows):
    run = rec.root / (component + '-run')
    if read(run / 'complete.json') != {'rows': len(rows), 'failed': []}:
        raise ValueError('run incomplete or failed')
    names = [record['design']['name'] for record in rows]
    wanted = {'complete.json', *(name + suffix for name in names for suffix in ('-start.json', '-valid.json'))}
    if {p.name for p in run.iterdir()} != wanted:
        raise ValueError('missing or extra recorded rows')
    recorded = {p.name.removeprefix('sample-') for p in rec.gates.glob('sample-*')
                if p.name.removeprefix('sample-') not in
                {design['name'] for other in ('smoke', 'timing', 'counters') if other != component
                 for design in designs(other)}}
    if recorded != set(names):
        raise ValueError('missing or extra sample gates')
    reports, previous, ended = {}, None, 0
    for index, (name, record) in enumerate(zip(names, rows), 1):
        if read(run / (name + '-start.json')) != {'index': index, 'name': name, 'previous': previous}:
            raise ValueError('recorded sequence changed')
        gate = rec.gates / ('sample-' + name)
        start = read(gate / 'command.json')['started_unix']
        end = read(gate / 'outcome.json')['ended_unix']
        if not 0 < ended <= start <= end and not (index == 1 and 0 < start <= end):
            raise ValueError('invalid or overlapping row time interval')
        report, proof = checked_report(rec, record)
        if proof != read(run / (name + '-valid.json')):
            raise ValueError('validated output identity differs')
        reports[name], previous, ended = report, name, end
    return reports


def timing_analysis(rows, reports):
    results = []
    for bound in BOUNDS:
        arms = {arm: [] for arm in 'AB'}
        for record in rows:
            design = record['design']
            if design['bound'] != bound or design['experiment'] == 'warmup':
                continue
            report = reports[design['name']]
            stage = resources(report)
            stage['whole_process'] = {'process_high_water_bytes': report['process_high_water_bytes']}
            arms[design['arm']].append(stage)
        a, b = (arms[arm] for arm in 'AB')
        metrics = {}
        for stage, fields in a[0].items():
            metrics[stage] = {}
            for metric in fields:
                left, right = ([sample[stage][metric] for sample in arm] for arm in (a, b))
                metrics[stage][metric] = {'baseline': summary(left), 'candidate': summary(right),
                                          'paired_baseline_minus_candidate': [x-y for x, y in zip(left, right)]}
        results.append({'bound': bound, 'pairs': len(a), 'stages': metrics})
    return results


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', required=True, type=Path)
    parser.add_argument('--candidate')
    parser.add_argument('--gopath', type=Path, default=Path.home() / 'go')
    parser.add_argument('--source-digest')
    parser.add_argument('--build-digest')
    parser.add_argument('--outside', type=Path)
    parser.add_argument('--excluded', action='store_true')
    parser.add_argument('phase', choices=('freeze', 'build-smoke', 'run-smoke', 'analyze-smoke',
                       'build-timing', 'run-timing', 'analyze-timing',
                       'build-counters', 'run-counters', 'analyze-counters',
                       'build-qualification', 'gates', 'differential', 'recovery', 'outside-rebuild',
                       'historical-smoke', 'differential-smoke', 'weather-smoke'))
    args = parser.parse_args()
    root = args.root.resolve()
    if root.is_relative_to(REPO) and not root.is_relative_to(WORKSPACE):
        parser.error('raw evidence belongs outside the checkout or in this plan workspace')
    if args.phase == 'freeze':
        if not re.fullmatch('[0-9a-f]{40}', args.candidate or ''):
            parser.error('freeze needs an exact --candidate SHA')
    elif not args.source_digest:
        parser.error('phase needs the recorded --source-digest')
    lock = Path('/private/tmp') / ('alan-p5-campaign-' + str(os.getuid()) + '.lock')
    with lock.open('a') as host:
        fcntl.flock(host, fcntl.LOCK_EX | fcntl.LOCK_NB)
        if args.phase == 'freeze':
            freeze(root, args.candidate, args.gopath.resolve(), args.excluded)
            return
        rec, plan = open_freeze(root, args.source_digest)
        excluded_phase = args.phase.endswith('-smoke')
        if args.phase != 'build-qualification' and plan['excluded'] != excluded_phase:
            parser.error('excluded and confirmatory phases require separate roots')
        action, _, component = args.phase.partition('-')
        if component in ('smoke', 'timing', 'counters') and action in ('build', 'run', 'analyze'):
            if action == 'build':
                build_measurement(rec, plan, component, args.source_digest)
                return
            bound = open_build(rec, component, args.source_digest, args.build_digest)
            rows = require_rows(root, plan, component, bound['binaries'], bound['rows'])
            if action == 'run':
                execute_rows(rec, component, rows)
                return
            reports = read_rows(rec, component, rows)
            if component == 'timing':
                analysis = timing_analysis(rows, reports)
            else:
                analysis = {r['design']['name']: attribution(reports[r['design']['name']], r['design']['bound'])
                            if r['design']['kind'] == 'counters' else resources(reports[r['design']['name']])
                            for r in rows}
            write(root / (component + '-analysis.json'), {'source_sha256': args.source_digest,
                  'build_sha256': args.build_digest, 'excluded': plan['excluded'], 'analysis': analysis})
            return
        import campaign_qualification
        campaign_qualification.dispatch(rec, plan, args)


if __name__ == '__main__':
    main()
