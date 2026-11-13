"""P5 gates around the tracked historical, differential and recovery routes."""
import json
from pathlib import Path
import re
import shutil

from measure import HERE, build_driver
from measurement import command, row, validate_report

from custody import expect_tests, extract_source, inventory, read, require_copy, sha, write
from external_inputs import ExternalInputs
from overlays import build_module, direct_overlay, patch
from qualification import (COMPLETE, READERS, compare, go_tests, integration_env, nested_modules,
                           old_controls, run_external, verify_direct_reports, verify_reciprocal)
from recovery import build_weather, cross_build, verify_weather_stops, weather_expected

SIDES = ('baseline', 'candidate')
PROMOTIONS = ('TestPhase3B4PromotedPublicParents/K01', 'TestPhase3B4PromotedPublicParents/K02')
ORDINARY = ('TestAsyncRunReportRejectsMalformedRetainedReceipt',
            'TestAsyncCancelAccountsReceivedCandidateAndPreservesUnknown',
            'TestUnknownExternalIsExplicitInRunReport',
            'TestInvocationPublishesAuthenticatedAbsentResponse',
            'TestRequestAuthorityRevocationOrderingAndUnaffectedTuple')
INTEGRATION = ('TestCompetingReadCannotSettleAdopterDebt',)
WEATHER = (('series', '^TestWeatherSeriesProcess$', 10),
           ('recovery', '^TestWeatherProcessRecovery$/^two_interrupted_recoveries$', 3))


def build(rec, digest):
    from campaign import seal_build

    root = rec.root
    directory = root / 'builds/qualification'
    directory.mkdir()
    for side in (*SIDES, 'old_cli', 'old_owner'):
        source = rec.sources[side]
        binary = directory / (side + '-alan')
        if side == 'old_owner':
            args = ['go', 'test', '-c', '-tags=integration', '-mod=readonly', '-buildvcs=false',
                    '-o', binary, './internal/durabletest']
        else:
            args = ['go', 'build', '-mod=readonly', '-buildvcs=false', '-o', binary, './cmd/alan']
        rec.run('qualify-build-' + side, args, source)
        rec.bind_tool(binary)
        rec.run('qualify-build-info-' + side, ['go', 'version', '-m', binary], source)
    original = rec.sources['candidate'] / 'docs/evidence/series-scaling/differential'
    tools = directory / 'direct-tools'
    COMPLETE.adapt(root, original, tools)
    rec.bind('direct-tools', tools)
    for side in SIDES:
        name = side + '-direct'
        module = directory / (name + '-module')
        shutil.copytree(original, module)
        before = direct_overlay(module, original)
        main = module / 'main.go'
        rec.run(name + '-format', ['gofmt', '-w', main])
        patch(main, before, main.read_text(), module / 'run-report-and-basis.patch')
        build_module(rec, name, rec.sources[side], module, directory / name)
        build_weather(rec, side, directory)
        rec.run(side + '-weather-build-info', ['go', 'version', '-m', directory / (side + '-weather.test')])
    seal_build(rec, 'qualification', digest)


def historical(rec, plan):
    parents = old_controls(rec, plan, excluded=True)
    rec.bind('historical-parents', parents)
    return parents


def gates(rec, plan):
    root = rec.root
    source = rec.sources['candidate']
    oracle_tests = tuple(re.findall(r'^func (Test\w+)\(', (source / 'internal/executor/report_read_test.go').read_text(), re.M))
    if len(oracle_tests) != 13 or sha(source / 'internal/executor/report_frozen_test.go') != sha(HERE / 'report_frozen_test.go.txt'):
        raise ValueError('candidate oracle fixture or test inventory differs')
    failures, static = [], []

    def attempt(name, fn):
        try:
            return fn()
        except Exception as error:
            failures.append({'gate': name, 'error': repr(error)})
            return None

    def test_gate(name, side, flags, required, *, pattern=None, packages=('./...',), parents=None):
        cwd = rec.sources[side]
        env = integration_env(rec, plan, side)
        if parents:
            env.update(ALAN_PHASE3B4_PUBLIC_PARENTS=str(parents),
                       ALAN_PHASE3B4_PROMOTED_OUTPUT=str(root / ('promoted-' + name)),
                       ALAN_PHASE3B3_EVIDENCE=str(root / ('phase3b3-' + name)),
                       ALAN_PHASE3B4_EVIDENCE=str(root / ('phase3b4-' + name)))
        log = go_tests(rec, name, cwd, flags, packages, env, pattern)
        expect_tests(log, required)
        if parents:
            for key in ('ALAN_PHASE3B3_EVIDENCE', 'ALAN_PHASE3B4_EVIDENCE'):
                verify_reciprocal(Path(env[key]), cwd, plan['sources'][side]['commit'])

    def lint(name, cwd, flags=(), race=False):
        # Staticcheck exit 1 with lint diagnostics is evidence for attribution.
        # Keep actual failure, and reject tool/build/load failures separately.
        try:
            output = rec.run(name, ['staticcheck', '-f', 'json', *flags, './...'], cwd,
                             {'GOFLAGS': '-mod=readonly' + (' -race' if race else '')})
        except RuntimeError:
            gate = rec.gates / name
            outcome = read(gate / 'outcome.json')
            output = gate / 'stdout.log'
            if outcome['exit'] != 1 or outcome['timeout'] or outcome['spawn_error']:
                raise
        rows = [json.loads(line) for line in output.read_text().splitlines()]
        diagnostic = [row for row in rows if re.fullmatch(r'(?:SA|S|ST|QF|U)\d{4}', row.get('code', ''))]
        actual = read(output.parent / 'outcome.json')['exit']
        if rows != diagnostic or actual != bool(diagnostic) or (output.parent / 'stderr.log').read_text().strip():
            raise ValueError('Staticcheck scope has a non-diagnostic failure')
        static.append({'gate': name, 'exit': actual, 'diagnostics': len(diagnostic), 'cwd': str(cwd)})

    parents = attempt('historical-controls', lambda: historical(rec, plan))
    for side in SIDES:
        src = rec.sources[side]
        attempt(side + '-full-ordinary', lambda: test_gate(side + '-full-ordinary', side, (),
                (*READERS, *ORDINARY) if side == 'candidate' else READERS))
        if side == 'candidate':
            for name, flags in (('full-integration', ('-tags=integration',)),
                                ('full-integration-race', ('-tags=integration', '-race'))):
                required = (*READERS, *ORDINARY, *INTEGRATION, *oracle_tests, *PROMOTIONS)
                attempt(name, lambda: test_gate(name, side, flags, required, parents=parents))
            for label, extra, tests in (('focused-ordinary-race', (), ORDINARY),
                                        ('focused-integration-race', ('-tags=integration',),
                                         (*ORDINARY, *INTEGRATION, *oracle_tests))):
                names = tuple(dict.fromkeys((*READERS, *tests)))
                pattern = '^(' + '|'.join(sorted({test.split('/')[0] for test in names})) + ')$'
                attempt(label, lambda: test_gate(label, side, ('-race', *extra), names,
                        pattern=pattern, packages=('./internal/executor', './internal/durabletest')))
        for label, argv in (
            ('root-build', ['go', 'build', '-mod=readonly', './...']),
            ('root-vet', ['go', 'vet', '-mod=readonly', './...']),
            ('root-vet-integration', ['go', 'vet', '-mod=readonly', '-tags=integration', './...']),
            ('root-module-verify', ['go', 'mod', 'verify']),
            ('root-module-tidy', ['go', 'mod', 'tidy', '-diff']),
        ):
            name = side + '-' + label
            attempt(name, lambda: rec.run(name, argv, src))
        for tags in ((), ('-tags=integration',)):
            for race in (False, True):
                name = side + '-staticcheck' + ('-integration' if tags else '') + ('-race' if race else '')
                attempt(name, lambda: lint(name, src, tags, race))
    attempt('frozen-oracle', lambda: rec.run('frozen-oracle',
            [rec.env['TASK5_PYTHON'], '-B', source / 'docs/evidence/report-scaling/oracle.py', '--check'],
            source, integration_env(rec, plan, 'candidate')))

    def gofmt():
        log = rec.run('candidate-gofmt', ['gofmt', '-l', *sorted(source.rglob('*.go'))], source)
        if log.read_bytes():
            raise ValueError('frozen Go sources differ from gofmt')
    attempt('gofmt', gofmt)
    external = attempt('independent-public-eight', lambda: external_consumer(rec))
    attempt('nested-modules', lambda: nested_modules(rec, external, attempt))
    for key, module in sorted(rec.sources.items()):
        if key in {f'nested-{i}' for i in range(5)}:
            name = key + '-staticcheck'
            attempt(name, lambda: lint(name, module, ('-tags=series',) if key == 'nested-0' else ()))
    write(root / 'qualification-results.json', {'failures': failures, 'staticcheck': static,
          'inputs_after': rec.snapshot(), 'gate_outputs': inventory(rec.gates)})
    if failures:
        raise ValueError('qualification gates failed; see qualification-results.json')


def external_consumer(rec):
    root = run_external(rec, ExternalInputs)
    for name in ('ordinary-consumer.test', 'race-consumer.test', 'baseline-reader'):
        rec.run('external-build-info-' + name, ['go', 'version', '-m', root / name])
    write(rec.root / 'external-inventory.json', inventory(root))
    return root


def differential(rec, plan, excluded):
    tools = rec.root / 'builds/qualification'
    route = tools / 'direct-tools'
    root = rec.root / ('excluded-differential' if excluded else 'differential')
    root.mkdir()
    seed = root / 'seed/store.sqlite'
    seed.parent.mkdir()
    rec.run('direct-seed', [tools / 'baseline-direct', '-mode', 'create-seed', '-bound', '8',
            '-db', seed, '-source-label', plan['sources']['baseline']['commit']])
    rec.bind('direct-seed', seed.parent)
    for scenario in ('success', 'refusal3'):
        for reopen in (0, 1):
            if excluded and (scenario, reopen) != ('refusal3', 1):
                continue
            name = f'direct-{scenario}-reopen{reopen}'
            dbs = [root / name / side / 'store.sqlite' for side in SIDES]
            for db in dbs:
                db.parent.mkdir(parents=True)
            rec.run(name + '-clone', [rec.env['TASK5_PYTHON'], '-B', route / 'clone_seed.py', seed, *dbs])
            reports = []
            for side, db in zip(SIDES, dbs):
                log = rec.run(name + '-' + side, [tools / (side + '-direct'), '-mode', 'execute-existing',
                              '-bound', '8', '-db', db, '-source-label', plan['sources'][side]['commit'],
                              '-scenario', scenario, '-reopen-every', str(reopen), '-approval-mode', 'targeted'],
                              db.parent)
                report = read(log)
                identity = {'source_label': plan['sources'][side]['commit'], 'store': str(db),
                            'bound': 8, 'mode': 'execute-existing', 'scenario': scenario, 'reopen_every': reopen}
                if any(report.get(k) != v for k, v in identity.items()):
                    raise ValueError('same-seed report identity differs from its command')
                verify_direct_reports(report, reopen)
                reports.append(log)
            compare(rec, root, route, name + '-exact', reports, dbs)
            if (scenario, reopen) == ('refusal3', 1):
                mutant = read(reports[1])
                cut = next(c for c in mutant['cuts'] if c.get('run_report') is not None)
                if not isinstance(cut['run_report']['OpenRuns'], list):
                    raise ValueError('expected non-nil report slice for comparator control')
                cut['run_report']['OpenRuns'] = None
                altered = root / 'mutant-null.json'
                write(altered, mutant)
                compare(rec, root, route, name + '-reject-null', [reports[0], altered], dbs, expected=1)
    rec.snapshot()
    write(rec.root / 'differential-inventory.json', inventory(root))


def controlled(log, marker):
    result, pending = {}, {}
    for event in log.read_text().splitlines():
        item = json.loads(event)
        if item['Action'] != 'output':
            continue
        key = (item['Package'], item.get('Test'))
        pieces = (pending.pop(key, '') + item['Output']).split('\n')
        if pieces[-1]:
            pending[key] = pieces[-1]
        for line in pieces[:-1]:
            _, found, data = line.partition(marker + ' ')
            if found:
                value = json.loads(data)
                if value['case'] in result:
                    raise ValueError('duplicate weather case evidence')
                result[value['case']] = value
    if any(marker + ' ' in partial for partial in pending.values()):
        raise ValueError('truncated weather case evidence')
    return result


def weather(rec, plan, excluded):
    tools = rec.root / 'builds/qualification'
    root = rec.root / ('excluded-weather' if excluded else 'weather-recovery')
    root.mkdir()
    for side in (('candidate',) if excluded else SIDES):
        source = rec.sources[side]
        for section, pattern, count in (WEATHER[1:] if excluded else WEATHER):
            folder = root / (side + '-' + section)
            folder.mkdir()
            archives, consumers = folder / 'archives', folder / 'public-consumers'
            log = rec.run(side + '-weather-' + section, ['go', 'tool', 'test2json', '-p', 'weather-' + side,
                          tools / (side + '-weather.test'), '-test.v=test2json', '-test.count=1',
                          '-test.timeout=25m', '-test.run=' + pattern], source / 'examples/weather-agent',
                          {'ALAN_SERIES_EVIDENCE_DIR': str(archives), 'ALAN_P3_CONSUMER_EVIDENCE_DIR': str(consumers)})
            required = (weather_expected(source, section) if section == 'series'
                        else {'TestWeatherProcessRecovery/two_interrupted_recoveries'})
            expect_tests(log, required)
            verify_weather_stops(archives, required, section)
            stops = sorted(archives.glob('*-raw-stop-*.sqlite'))
            if len(stops) != count or (section == 'series' and len(required) != count):
                raise ValueError('unexpected count of immediately reaped triplets')
            if section == 'series':
                facts = controlled(log, 'SERIES_CONTROLLED')
                if set(facts) != {*required, 'TestWeatherSeriesProcess'}:
                    raise ValueError('missing exact original B8 series facts')
            else:
                facts = controlled(log, 'M4_CONTROLLED')
                repeated = facts['two_interrupted_recoveries']
                if len(repeated['contacts']) != 1 or repeated['charged'] != 1 or len(repeated['killed']) != 3:
                    raise ValueError('missing original single-contact recovery proof')
            binaries = list(consumers.rglob('weather-agent'))
            if len(binaries) != 1:
                raise ValueError('weather test did not retain its actual public consumer')
            binary, = binaries
            if not binary.is_file() or any(not (binary.parent / path).is_file() for path in ('go.mod', 'go.sum', 'main.go')):
                raise ValueError('weather consumer source/module evidence is incomplete')
            rec.bind(side + '-' + section + '-weather-public-module', binary.parent)
            rec.bind_tool(binary)
            rec.run(side + '-' + section + '-weather-public-build-info', ['go', 'version', '-m', binary])
            sealed = inventory(archives)
            for index, db in enumerate(stops):
                name = f'{side}-{section}-stop{index + 1}'
                reports, branches = [], []
                for reader in SIDES:
                    branch = folder / 'readers' / name / reader / 'store.sqlite'
                    require_copy(db, branch, branch.parent / 'copy.json', stopped=True)
                    reports.append(rec.run(name + '-' + reader, [tools / (reader + '-direct'),
                                   '-mode', 'generic-reader', '-db', branch,
                                   '-source-label', plan['sources'][reader]['commit']]))
                    branches.append(branch)
                compare(rec, folder, tools / 'direct-tools', name + '-compare', reports, branches)
                if inventory(archives) != sealed:
                    raise ValueError('sealed stopped evidence changed during independent reading')
            write(folder / 'raw-inventory.json', inventory(folder))
    if not excluded:
        cross_build(rec, plan)
    write(rec.root / 'recovery-inventory.json', {name: inventory(rec.root / name)
          for name in ((root.name,) if excluded else (root.name, 'cross-build-recovery'))})


def reproduce(rec, plan, outside):
    if outside is None or outside.parent.resolve() != Path('/private/tmp'):
        raise ValueError('outside rebuild requires a fresh direct child of /private/tmp')
    outside.mkdir()
    source = outside / 'source'
    extract_source(rec.root / 'archives/candidate.tar', source)
    if inventory(source) != inventory(rec.sources['candidate']):
        raise ValueError('outside archive export differs from frozen candidate')
    rec.bind('reproduction-source', source)
    build_dir = outside / 'build'
    build_dir.mkdir()
    # A fresh external path for both the exported sources and the actual build.
    from custody import TaskRecorder
    builder = TaskRecorder(build_dir, env=rec.env, sources=dict(rec.sources), tools=list(rec.tools))
    binary = build_driver(builder, source, 'ordinary')
    for name in builder.sources.keys() - rec.sources.keys():
        rec.bind('reproduction-' + name, builder.sources[name])
    for tool in builder.tools:
        rec.bind_tool(tool)
    design = row('outside-rebuild', 8, 'candidate', 'targeted', name='outside-b8-candidate')
    sample = outside / 'samples' / design['name']
    sample.mkdir(parents=True)
    revision = plan['sources']['candidate']['commit']
    output = rec.run('outside-b8-candidate', command(outside, design, revision, binary), sample)
    validate_report(read(output), design, revision, sample / 'store/store.sqlite')
    retained = rec.root / 'outside-reproduction'
    shutil.copytree(outside, retained)
    sealed = inventory(outside)
    if inventory(retained) != sealed:
        raise ValueError('retained outside build differs')
    write(rec.root / 'outside-reproduction.json', {'outside': str(outside), 'candidate': revision,
          'binary_sha256': sha(binary), 'report_sha256': sha(output), 'files': sealed})


def dispatch(rec, plan, args):
    from campaign import open_build

    if args.phase == 'build-qualification':
        build(rec, args.source_digest)
    elif args.phase == 'outside-rebuild':
        reproduce(rec, plan, args.outside)
    else:
        rec, _ = open_build(rec, 'qualification', args.source_digest, args.build_digest)
        if args.phase == 'historical-smoke':
            historical(rec, plan)
        elif args.phase in ('differential', 'differential-smoke'):
            differential(rec, plan, plan['excluded'])
        elif args.phase in ('recovery', 'weather-smoke'):
            weather(rec, plan, plan['excluded'])
        elif args.phase == 'gates':
            gates(rec, plan)
