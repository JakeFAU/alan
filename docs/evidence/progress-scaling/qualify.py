#!/usr/bin/env python3
"""Run P6 semantic qualification against exact exported source revisions."""
import argparse
import fcntl
from pathlib import Path
import re
import shutil
import sys

sys.dont_write_bytecode = True
from measure import (BASE, EVIDENCE, HERE, REPO, WORKSPACE, TaskRecorder, build, design,
                     environment, execute, freeze, inventory, read, sha, write)
from controller import bind_toolchain
from custody import expect_tests, extract_source, members, require_copy
from external_inputs import ExternalInputs
from qualification import MODULES, READERS, go_tests, integration_env, nested_modules, old_controls, run_external, verify_reciprocal
from recovery import build_weather, verify_weather_stops, weather_expected

OLD_CLI = 'af6943a5805a9a6c0e8fe77f804264f4765d07ec'
OLD_OWNER = 'd59eaf2976849e450635d252c7bca62eeccbef2e'
HELPERS = ('execution-scaling/task5', 'execution-scaling/apparatus', 'performance-p3',
           'performance-p2/driver', 'performance-p2/instrument', 'report-scaling')


def source_git(rec, revision, source, label, tree):
    gitdir = rec.root / (label + '.git')
    rec.run(label + '-git-init', ['git', 'init', '--bare', gitdir], REPO)
    common = rec.run(label + '-git-common', ['git', 'rev-parse', '--path-format=absolute',
                                             '--git-common-dir'], REPO).read_text().strip()
    (gitdir / 'objects/info/alternates').write_text(str(Path(common) / 'objects') + '\n')
    (gitdir / 'HEAD').write_text(revision + '\n')
    extra = dict(GIT_DIR=str(gitdir), GIT_WORK_TREE=str(source))
    observed = rec.run(label + '-git-proof', ['git', 'rev-parse', 'HEAD', 'HEAD^{tree}'],
                       source, extra).read_text().splitlines()
    if observed != [revision, tree]:
        raise ValueError('provenance disagrees with exported source')
    rec.bind('git-' + label, gitdir)


def prepare(rec, revision):
    status = rec.run('checkout-status', ['git', 'status', '--porcelain=v1', '--untracked-files=all'], REPO)
    head = rec.run('checkout-head', ['git', 'rev-parse', 'HEAD'], REPO)
    if status.read_bytes() or head.read_text().strip() != revision:
        raise ValueError('qualification requires the clean freeze checkout')
    identities = {}
    for label, commit in (('candidate', revision), ('baseline', BASE), ('old_cli', OLD_CLI), ('old_owner', OLD_OWNER)):
        source, identity = freeze(rec, commit, label)
        # Legacy routes receive explicit source paths through the recorder.
        rec.sources[label] = rec.sources.pop(label + '-source')
        rec.expected['sources'][label] = rec.expected['sources'].pop(label + '-source')
        identities[label] = identity
    candidate = rec.sources['candidate']
    for rel in (*HELPERS, 'progress-scaling'):
        current = EVIDENCE / rel
        if inventory(current) != inventory(candidate / current.relative_to(REPO)):
            raise ValueError('live apparatus differs from frozen bytes: ' + rel)
        rec.bind('apparatus-' + rel, current)
    source_git(rec, revision, candidate, 'candidate', identities['candidate']['tree'])
    toolchain = bind_toolchain(rec)
    write(rec.root / 'source-freeze.json', {'sources': identities, 'toolchain': toolchain,
                                           'environment': rec.env, 'inputs_before': rec.snapshot()})
    return {'sources': identities}


def build_readers(rec, directory):
    for side in ('candidate', 'old_cli', 'old_owner'):
        src = rec.sources[side]
        binary = directory / (side + '-alan')
        if side == 'old_owner':
            argv = ['go', 'test', '-c', '-mod=readonly', '-tags=integration', '-o', binary, './internal/durabletest']
        else:
            argv = ['go', 'build', '-mod=readonly', '-o', binary, './cmd/alan']
        rec.run('qualify-build-' + side, argv, src)
        rec.bind_tool(binary)
        rec.run('qualify-build-info-' + side, ['go', 'version', '-m', binary])


def tests(rec, plan, parents, attempt):
    source = rec.sources['candidate']
    extras = integration_env(rec, plan, 'candidate')
    executor = source / 'internal/executor'
    def names(*files):
        return tuple(name for file in files for name in re.findall(r'^func (Test\w+)\(', (executor / file).read_text(), re.M))

    ordinary = names('runner_progress_test.go')
    integration = names('runner_progress_integration_test.go')
    oracle = names('report_read_test.go')
    promotions = ('TestPhase3B4PromotedPublicParents/K01', 'TestPhase3B4PromotedPublicParents/K02')
    if len(ordinary) != 5 or len(integration) != 4 or len(oracle) != 13:
        raise ValueError('unexpected frozen test inventory')

    def run_suite(label, flags, required, promote=False):
        env = dict(extras)
        if promote and parents:
            env.update(ALAN_PHASE3B4_PUBLIC_PARENTS=str(parents),
                       ALAN_PHASE3B4_PROMOTED_OUTPUT=str(rec.root / (label + '-promoted')),
                       ALAN_PHASE3B3_EVIDENCE=str(rec.root / (label + '-phase3b3')),
                       ALAN_PHASE3B4_EVIDENCE=str(rec.root / (label + '-phase3b4')))
        log = go_tests(rec, label, source, flags, extras=env)
        expect_tests(log, (*required, *promotions) if promote else required)
        if promote and parents:
            for key in ('ALAN_PHASE3B3_EVIDENCE', 'ALAN_PHASE3B4_EVIDENCE'):
                verify_reciprocal(Path(env[key]), source, plan['sources']['candidate']['commit'])

    attempt('full-ordinary', lambda: run_suite('full-ordinary', (), (*READERS, *ordinary)))
    for label, flags in (('full-integration', ('-tags=integration',)),
                         ('full-race-integration', ('-race', '-tags=integration'))):
        attempt(label, lambda: run_suite(label, flags, (*READERS, *ordinary, *integration, *oracle,
                                                          'ExampleStore_RunProgress'), promote=True))

    pattern = ('^(Test(RunProgress|RunCompactAssessment|FinishRunReport|RootCompletionsReportBatchDuplicateFallback|'
               'ReportReaderCachedCancellationAndGenerations|RequestAuthority|RequestBinding|ReceiptConflict|'
               'Cancellation|DispatchAdmissionValidatesOriginalGate|ExecutionInspectionHistoricalSnapshots|'
               'LifecycleRecoveryAcrossFormats)|ExampleStore_RunProgress)')
    for label, flags, required in (
            ('focused-race', ('-race',), ordinary),
            ('focused-race-integration', ('-race', '-tags=integration'), (*ordinary, *integration, *oracle, 'ExampleStore_RunProgress'))):
        attempt(label, lambda: expect_tests(go_tests(rec, label, source, flags,
                ('./internal/executor', './runtime'), extras, pattern), (*required, *READERS)))
    attempt('report-oracle', lambda: rec.run('report-oracle', [sys.executable, '-B',
            source / 'docs/evidence/report-scaling/oracle.py', '--check'], source, extras))


def quality(rec, attempt):
    source = rec.sources['candidate']
    for flags, suffix in (((), ''), (('-tags=integration',), '-integration')):
        for command in ('build', 'vet'):
            name = command + suffix
            attempt(name, lambda: rec.run(name, ['go', command, '-mod=readonly', *flags, './...'], source))
        for race in (False, True):
            name = 'staticcheck' + suffix + ('-race' if race else '')
            attempt(name, lambda: rec.run(name, ['staticcheck', '-f', 'json', *flags, './...'], source,
                                         {'GOFLAGS': '-mod=readonly' + (' -race' if race else '')}))
    for name, argv in (('module-verify', ['go', 'mod', 'verify']),
                       ('module-tidy', ['go', 'mod', 'tidy', '-diff'])):
        attempt(name, lambda: rec.run(name, argv, source))
    def fmt():
        log = rec.run('format', ['gofmt', '-l', *sorted(source.rglob('*.go')),
                  source / HERE.relative_to(REPO) / 'parity.go.txt'], source)
        if log.read_bytes():
            raise ValueError('frozen Go source differs from gofmt')
    attempt('format', fmt)
    attempt('apparatus-tests', lambda: rec.run('apparatus-tests', [sys.executable, '-B', '-m', 'unittest',
                  'discover', '-s', source / HERE.relative_to(REPO), '-p', 'test_*.py'], source))
    matches = {}
    for path in sorted(source.rglob('*.go')):
        found = [line for line in path.read_text().splitlines() if line.startswith('func Fuzz') or '"testing/quick"' in line]
        if found:
            matches[str(path.relative_to(source))] = found
    write(rec.root / 'fuzz-inventory.json', matches)
    if matches:
        raise ValueError('additional fuzz qualification required')


def compare_weather_readers(inspect, replay):
    detailed = ('history', 'pending', 'refused_local_inputs')
    if (type(replay.get('replayed_commits')) is not int or
            replay['replayed_commits'] != len(inspect.get('history', ())) or
            any(key in replay for key in detailed) or 'replayed_commits' in inspect):
        raise ValueError('weather Replay did not verify the inspected history')
    if not isinstance(inspect.get('reads'), list) or not isinstance(replay.get('reads'), list):
        raise ValueError('weather reader omitted original reads')
    projected = {key: value for key, value in inspect.items() if key not in detailed}
    projected['reads'] = [{key: value for key, value in row.items() if key != 'retained_response'}
                          for row in inspect['reads']]
    if projected != {key: value for key, value in replay.items() if key != 'replayed_commits'}:
        raise ValueError('independent readers disagree on the shared weather projection')


def read_weather_stops(rec, label, stopped, binary, output):
    for index, db in enumerate(stopped):
        clone = output / 'readers' / str(index) / 'store.sqlite'
        proof = clone.parent / 'copy.json'
        require_copy(db, clone, proof, stopped=True)
        before = members(clone)
        logs, reader_members = [], {}
        for action in ('inspect', 'replay'):
            logs.append(rec.run(f'weather-{label}-{index}-{action}', [binary, action, '--db', clone, '--json']))
            after = members(clone)
            reader_members[action] = after
            # SQLite may update its shared-memory index for a reader. The
            # original archive is bound separately, including its SHM.
            if any(after.get(suffix) != before[suffix] for suffix in ('', '-wal')):
                raise ValueError('read-only weather command changed main database or WAL')
        compare_weather_readers(*map(read, logs))
        write(clone.parent / 'read-only-members.json', {'before': before, 'after': reader_members})


def weather_gates(rec, directory, attempt, parity_only=False):
    src = rec.sources['candidate']
    build_weather(rec, 'candidate', directory)
    weather = src / 'examples/weather-agent'
    binary = directory / 'candidate-weather.test'
    descriptor = next((directory / 'candidate-weather').rglob('task5-overlay.json'))
    parity = descriptor.parent / 'p6_parity_test.go'
    # Build the extra test as its own overlay, with the inherited custody
    # hooks. Keep the already bound overlay intact.
    parity_overlay = directory / 'parity-overlay'
    parity_overlay.mkdir()
    parity_file = parity_overlay / parity.name
    shutil.copyfile(HERE / 'parity.go.txt', parity_file)
    replacements = read(descriptor)['Replace']
    replacements[str(weather / parity.name)] = str(parity_file)
    desc = parity_overlay / 'overlay.json'
    write(desc, {'Replace': replacements})
    rec.bind('p6-parity-overlay', parity_overlay)
    parity_binary = directory / 'parity-weather.test'
    rec.run('parity-build', ['go', 'test', '-c', '-mod=readonly', '-tags=integration', '-overlay=' + str(desc),
                              '-o', parity_binary, './examples/weather-agent'], src)
    rec.bind_tool(parity_binary)

    def run(label, test, pattern, required, expected_stops, kind, *, parity=False):
        output = rec.root / ('weather-' + label)
        output.mkdir()
        archives = output / 'archives'
        consumers = output / 'consumers'
        extra = {'ALAN_SERIES_EVIDENCE_DIR': str(archives), 'ALAN_P3_CONSUMER_EVIDENCE_DIR': str(consumers),
                 'ALAN_P6_PARITY': str(output / 'typed-cuts')}
        log = rec.run('weather-' + label, ['go', 'tool', 'test2json', '-p', 'weather-' + label,
                        test, '-test.v=test2json', '-test.count=1', '-test.timeout=25m', '-test.run=' + pattern],
                        weather, extra, timeout=1600)
        expect_tests(log, required)
        verify_weather_stops(archives, required, kind)
        stopped = sorted(archives.glob('*-raw-stop-*.sqlite'))
        if len(stopped) != expected_stops:
            raise ValueError('unexpected reaped store count')
        retained = sorted(consumers.rglob('weather-agent'))
        if not retained or any(not (p.parent / 'go.mod').is_file() or not (p.parent / 'go.sum').is_file() for p in retained):
            raise ValueError('public weather binary and module were not retained')
        rec.bind('weather-consumer-' + label, consumers)
        sealed = inventory(archives)
        if not parity:
            read_weather_stops(rec, label, stopped, retained[0], output)
        if inventory(archives) != sealed:
            raise ValueError('sealed weather archive changed')
        write(output / 'raw-inventory.json', inventory(output))

    if not parity_only:
        for label, pattern, required, stops in (
                ('series', '^TestWeatherSeriesProcess$', weather_expected(src, 'series'), 10),
                ('recovery', '^TestWeatherProcessRecovery$/^two_interrupted_recoveries$',
                 ('TestWeatherProcessRecovery/two_interrupted_recoveries',), 3)):
            attempt('weather-' + label, lambda: run(label, binary, pattern, required, stops, label))
    parity_required = tuple('TestP6SameSeedParity/' + name for name in ('success', 'refusal', 'unknown'))
    attempt('weather-parity', lambda: run('parity', parity_binary, '^TestP6(SameSeedParity|OppositeMethodRecovery)$',
            (*parity_required, 'TestP6OppositeMethodRecovery'), 1, 'opposite', parity=True))


def recheck_weather_readers(rec, previous, previous_digest, plan):
    """Resume only the failed standalone reader comparison on sealed original cuts."""
    original_result = previous / 'qualification-result.json'
    if sha(original_result) != previous_digest:
        raise ValueError('original qualification outcome digest differs')
    original = read(original_result)
    rec.bind_tool(original_result)
    original_freeze = previous / 'source-freeze.json'
    if sha(original_freeze) != original['source_freeze_sha256']:
        raise ValueError('original qualification freeze digest differs')
    old = read(original_freeze)['sources']['candidate']
    rec.bind_tool(original_freeze)
    new = plan['sources']['candidate']
    changed = rec.run('reader-correction-bridge', ['git', 'diff', '--name-only', old['commit'], new['commit']],
                      REPO).read_text().splitlines()
    allowed = {str(HERE.relative_to(REPO) / name) for name in ('qualify.py', 'test_qualify.py')}
    if set(changed) != allowed:
        raise ValueError('reader correction must preserve all measured and tested Go sources and harnesses')
    failed = {f['gate'] for f in original['failures']}
    if failed != {'weather-series', 'weather-recovery'}:
        raise ValueError('reader-only recheck cannot close other failed qualifications')
    for label, count in (('series', 10), ('recovery', 3)):
        archived = previous / ('weather-' + label)
        gate = previous / 'gates' / ('weather-' + label)
        outcome = read(gate / 'outcome.json')
        if outcome['exit'] != 0 or outcome['timeout'] or outcome['spawn_error']:
            raise ValueError('original process test did not pass')
        for artifact in ('command.json', 'outcome.json', 'stdout.log'):
            rec.bind_tool(gate / artifact)
        archives, consumers = archived / 'archives', archived / 'consumers'
        if inventory(consumers) != original['inputs_after']['sources']['weather-consumer-' + label]:
            raise ValueError('original weather consumer bytes differ')
        rec.bind(label + '-archives', archives)
        rec.bind(label + '-consumers', consumers)
        stopped = sorted(archives.glob('*-raw-stop-*.sqlite'))
        required = (weather_expected(rec.sources['candidate'], label) if label == 'series' else
                    ('TestWeatherProcessRecovery/two_interrupted_recoveries',))
        if len(stopped) != count:
            raise ValueError('expected all original reaped Stores')
        verify_weather_stops(archives, required, label)
        binary = next(iter(sorted(consumers.rglob('weather-agent'))))
        read_weather_stops(rec, label, stopped, binary, rec.root / ('weather-' + label))
    write(rec.root / 'reader-recheck.json', {'previous_result_sha256': previous_digest,
          'tested_candidate': old, 'reader_correction': new, 'rechecked': sorted(failed)})


def reproduce(rec, revision, outside):
    if outside.parent.resolve() != Path('/private/tmp'):
        raise ValueError('use a fresh direct child of /private/tmp for reproduction')
    outside.mkdir()
    for name in ('tmp', 'samples'):
        (outside / name).mkdir()
    source = outside / 'source'
    extract_source(rec.root / 'candidate.tar', source)
    if inventory(source) != inventory(rec.sources['candidate']):
        raise ValueError('outside export differs from frozen source')
    env = dict(rec.env, TMPDIR=str(outside / 'tmp'))
    standalone = TaskRecorder(outside, env=env, sources={'candidate': source})
    sqlite = Path(env['GOMODCACHE']) / 'github.com/mattn/go-sqlite3@v1.14.52'
    standalone.bind('sqlite', sqlite)
    binary = build(standalone, 'outside-progress', source, sqlite, 'progress')
    row = execute(standalone, design('smoke', 8, 'candidate-progress'), revision, 'progress', binary)
    if row.get('failure') or row['exit'] != 0:
        raise ValueError('outside reproduction failed; outputs retained')
    copy = rec.root / 'outside-reproduction'
    shutil.copytree(outside, copy, ignore=shutil.ignore_patterns('tmp'))
    sealed = inventory(copy)
    if sealed != {name: value for name, value in inventory(outside).items() if not name.startswith('tmp/')}:
        raise ValueError('outside reproduction copy differs')
    write(rec.root / 'outside-reproduction.json', {'original': str(outside), 'revision': revision,
            'files': sealed, 'row': row, 'bound_inputs': standalone.snapshot()})


def campaign(args, root):
    for name in ('tmp', 'builds/qualification'):
        (root / name).mkdir(parents=True)
    env = environment(root, args.gopath.resolve())
    if args.gocache:
        env['GOCACHE'] = str(args.gocache.resolve())
    rec = TaskRecorder(root, env=env)
    directory = root / 'builds/qualification'
    plan = prepare(rec, args.revision)
    failures = []

    def attempt(name, fn):
        try:
            return fn()
        except Exception as error:
            failures.append({'gate': name, 'error': repr(error)})
            print(f'FAILED {name}: {error!r}', flush=True)
            return None

    if args.recheck:
        attempt('reader-controls', lambda: rec.run('reader-controls', [sys.executable, '-B', '-m', 'unittest',
                  'discover', '-s', rec.sources['candidate'] / HERE.relative_to(REPO), '-p', 'test_*.py']))
        attempt('reader-recheck', lambda: recheck_weather_readers(rec, args.recheck.resolve(),
                                                                  args.previous_digest, plan))
    elif not args.parity_only:
        build_readers(rec, directory)
        parents = attempt('historical-controls', lambda: old_controls(rec, plan))
        if parents:
            rec.bind('historical-parents', parents)
        tests(rec, plan, parents, attempt)
        quality(rec, attempt)
        external = attempt('independent-consumer', lambda: run_external(rec, ExternalInputs))
        attempt('nested-modules', lambda: nested_modules(rec, external, attempt))
        for index in range(len(MODULES)):
            name = 'nested-' + str(index)
            if name in rec.sources:
                flags = ['-tags=series'] if index == 0 else []
                for tool in (['staticcheck', '-f', 'json'], ['go', 'vet', '-mod=readonly']):
                    attempt(name + '-' + tool[0], lambda: rec.run(name + '-' + '-'.join(tool[:2]), [*tool, *flags, './...'],
                                                                  rec.sources[name]))
    if not args.recheck:
        attempt('weather', lambda: weather_gates(rec, directory, attempt, args.parity_only))
    if args.outside:
        attempt('outside-reproduction', lambda: reproduce(rec, args.revision, args.outside.resolve()))
    write(root / 'qualification-result.json', {'failures': failures, 'inputs_after': rec.snapshot(),
                                                'source_freeze_sha256': sha(root / 'source-freeze.json')})
    if failures:
        raise SystemExit('one or more qualification gates failed; outputs retained')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--revision', required=True)
    parser.add_argument('--out', required=True, type=Path)
    parser.add_argument('--gopath', default=Path.home() / 'go', type=Path)
    parser.add_argument('--gocache', type=Path)
    parser.add_argument('--parity-only', action='store_true')
    parser.add_argument('--outside', type=Path, help='fresh direct child of /private/tmp for B8 reproduction')
    parser.add_argument('--recheck', type=Path, help='previous qualification root with only reader projection failures')
    parser.add_argument('--previous-digest', help='SHA-256 of the previous qualification result')
    args = parser.parse_args()
    if args.recheck and (not re.fullmatch('[0-9a-f]{64}', args.previous_digest or '') or
                         args.parity_only or args.outside):
        parser.error('reader-only recheck requires the previous digest and no other qualification options')
    if not re.fullmatch('[0-9a-f]{40}', args.revision):
        parser.error('expected an exact Git commit')
    from measure import reserve_output
    with (WORKSPACE / 'measurement.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        campaign(args, reserve_output(args.out))


if __name__ == '__main__':
    main()
