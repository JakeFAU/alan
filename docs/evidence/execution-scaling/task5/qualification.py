"""Frozen semantic qualification and supported/unsupported format controls."""
import json
from pathlib import Path
import shutil
import sys

from custody import P3, events, expect_tests, inventory, load, read, require_copy, restore_inputs, separate_inputs, sha, write
from overlays import build_module, direct_overlay, generate, patch, replace

MODULES = ('docs/evidence/bounded-series/performance/module',
           'docs/evidence/performance-p2/driver', 'docs/evidence/performance-p2/differential',
           'docs/evidence/series-scaling/driver', 'docs/evidence/series-scaling/differential')
READERS = ('TestRequestAuthorityFormatsConflictPromotionAndOldReader/13/frozen_old_reader',
           'TestRequestAuthorityFormatsConflictPromotionAndOldReader/14/frozen_old_reader',
           'TestRequestBindingConflictAndOldReader/false/frozen_reader',
           'TestRequestBindingConflictAndOldReader/true/frozen_reader')
FOCUSED = ('^Test(StaticExecution|RootCompletions|P3Readiness|OwnerContinuationBasis|ContinuationBasis|'
           'ContinuationMaskedBeforeFirstApproval|ContinuationConflictPromotion|Cancellation|'
           'RequestAuthority|RequestBinding|ReceiptConflict|CompetingReadCannotSettleAdopterDebt|'
           'CompositionFactsRejectCorruption|InvocationModeCorruptionRejectsBeforeRenewal)')
ROUTES = load('task5_p3_routes', P3 / 'qualification/run.py')
# complete_campaign imports legacy run.py by name. Its only invoked functions
# here receive explicit new inputs and never use the legacy freeze/defaults.
sys.modules['run'] = ROUTES
COMPLETE = load('task5_complete_routes', P3 / 'qualification/complete_campaign.py')


def build(rec, plan, digest):
    root = rec.root
    directory = root / 'builds/qualification'
    directory.mkdir()
    for side in ('baseline', 'candidate', 'old_cli', 'old_owner'):
        source = rec.sources[side]
        binary = directory / (side + '-alan')
        if side == 'old_owner':
            args = ['go', 'test', '-c', '-mod=readonly', '-buildvcs=false', '-tags=integration',
                    '-o', binary, './internal/durabletest']
        else:
            args = ['go', 'build', '-mod=readonly', '-buildvcs=false', '-o', binary, './cmd/alan']
        rec.run('qualify-build-' + side, args, source)
        rec.bind_tool(binary)
        rec.run('qualify-build-info-' + side, ['go', 'version', '-m', binary], source)
    for kind, evidence in [('direct', 'series-scaling'), ('http', 'performance-p2')]:
        original = rec.sources['candidate'] / 'docs/evidence' / evidence / 'differential'
        tools = directory / (kind + '-tools')
        COMPLETE.adapt(root, original, tools)
        rec.bind(kind + '-tools', tools)
        for side in ('baseline', 'candidate'):
            source = rec.sources[side]
            name = side + '-' + kind
            module = directory / (name + '-module')
            shutil.copytree(original, module)
            if kind == 'direct':
                old = direct_overlay(module, original)
                rec.run(name + '-format', ['gofmt', '-w', module / 'main.go'])
                main = module / 'main.go'
                patch(main, old, main.read_text(), module / 'run-report-and-basis.patch')
            build_module(rec, name, source, module, directory / name)
    from recovery import build_weather
    for side in ('baseline', 'candidate'):
        build_weather(rec, side, directory)
    separate_inputs(rec, 'qualification', [])
    lock = root / 'qualification-freeze.json'
    write(lock, {'source_freeze_sha256': digest, 'inputs_sha256': sha(root / 'qualification-inputs.json')})
    print('QUALIFICATION-FREEZE SHA256 ' + sha(lock), flush=True)


def verify_reciprocal(root, source, identity):
    found = sorted(root.rglob('source.json'))
    if not found:
        raise ValueError('missing reciprocal evidence with source identity')
    for path in found:
        data = read(path)
        if data['HEAD'].strip() != identity or not data['FilesSHA256']:
            raise ValueError('wrong reciprocal source HEAD: ' + str(path))
        if any(sha(source / path) != digest for path, digest in data['FilesSHA256'].items()):
            raise ValueError('reciprocal source file identity mismatch')


def integration_env(rec, plan, side):
    root = rec.root
    tools = root / 'builds/qualification'
    return {'ALAN_PHASE3B3_OLD_READER': str(tools / 'old_cli-alan'),
            'ALAN_PHASE3B4_OLD_READER': str(tools / 'old_owner-alan'),
            'GIT_DIR': str(rec.sources['git-' + side]), 'GIT_WORK_TREE': str(rec.sources[side])}


def old_controls(rec, plan, *, excluded=False):
    root = rec.root
    source = rec.sources['candidate']
    tools = root / 'builds/qualification'
    out = root / 'historical-controls'
    out.mkdir()
    cli, owner = (tools / (side + '-alan') for side in ('old_cli', 'old_owner'))
    db = out / 'cli-supported.sqlite'
    rec.run('old-cli-init-supported', [cli, 'init', '--db', db, '--demo', 'nested'])
    ins = read(rec.run('old-cli-inspect-supported', [cli, 'inspect', '--db', db]))
    rep = read(rec.run('old-cli-replay-supported', [cli, 'replay', '--db', db]))
    known_version = ins['Tables']['metadata'][0]['schema_version']
    if (not 3 <= known_version < 13 or not ins.get('Tables', {}).get('runs') or not rep.get('Runs') or
            ins['Namespace'] != rep['Namespace']):
        raise ValueError('old CLI control did not decode its provisioned Store')
    parents, promoted = out / 'parents', out / 'promoted'
    extras = integration_env(rec, plan, 'candidate')
    campaign = rec.run('public-parents', ['go', 'test', '-mod=readonly', '-json', '-count=1', '-tags=integration',
                       '-run=' + ('^TestPhase3B4CampaignCompatibility$' if excluded else '^TestPhase3B4Campaign'), './internal/durabletest'], source,
                       dict(extras, ALAN_PHASE3B4_EVIDENCE=str(parents)))
    expect_tests(campaign, ['TestPhase3B4CampaignCompatibility/K01',
                            'TestPhase3B4CampaignCompatibility/K02'])
    verify_reciprocal(parents, source, plan['sources']['candidate']['commit'])
    output = rec.run('public-promotion', ['go', 'test', '-mod=readonly', '-json', '-count=1', '-tags=integration',
                     '-run=^TestPhase3B4PromotedPublicParents$', './internal/executor'], source,
                     dict(extras, ALAN_PHASE3B4_PUBLIC_PARENTS=str(parents), ALAN_PHASE3B4_PROMOTED_OUTPUT=str(promoted)))
    expect_tests(output, ['TestPhase3B4PromotedPublicParents/K01', 'TestPhase3B4PromotedPublicParents/K02'])
    parent, = parents.glob('K01-*/final/runtime.sqlite')
    clone = out / 'owner-supported.sqlite'
    require_copy(parent, clone, out / 'owner-copy.json')
    version_copy = out / 'owner-format-probe.sqlite'
    require_copy(parent, version_copy, out / 'owner-format-probe-copy.json')
    known = read(rec.run('owner-supported-format-inspect', [tools / 'candidate-alan', 'inspect', '--db', version_copy]))
    if known['Tables']['metadata'][0]['schema_version'] != 13:
        raise ValueError('pre-binding positive parent must be genuine format 13')
    owner_out = out / 'owner-positive'
    owner_out.mkdir()
    run = rec.run('old-owner-open-supported', ['go', 'tool', 'test2json', '-p', 'historical-owner', owner,
                  '-test.run=^TestPhase3B3Owner$', '-test.v=test2json', '-test.timeout=2m'], rec.sources['old_owner'],
                  {'ALAN_PHASE3B3_CHILD': '1', 'ALAN_PHASE2_DB': str(clone), 'ALAN_PHASE3B3_OUTPUT': str(owner_out)})
    expect_tests(run, ['TestPhase3B3Owner'])
    op, rr = read(owner_out / 'open.json'), read(owner_out / 'run.json')
    if (op['Error'] != '<nil>' or op['Namespace'] != known['Namespace'] or set(op['Runs']) != {'a', 'b'} or
            any(state['ID'] != {'Subject': name, 'Generation': 1} for name, state in op['Runs'].items()) or
            rr['Error'] != '<nil>' or not rr['Report']['Quiescent'] or rr['Report']['Dispatches'] != 0 or rr['Report']['OpenRuns']):
        raise ValueError('historical owner did not actually Open and complete a known format')
    if excluded:
        negatives = go_tests(rec, 'excluded-four-historical-rejections', source, packages=('./internal/executor',),
                             extras=extras, pattern='^Test(RequestAuthorityFormatsConflictPromotionAndOldReader|RequestBindingConflictAndOldReader)$')
        expect_tests(negatives, READERS)
    return parents


def go_tests(rec, name, cwd, flags=(), packages=('./...',), extras=None, pattern=None):
    args = ['go', 'test', '-buildvcs=false', '-mod=readonly', '-json', '-count=1', '-timeout=30m', *flags]
    if pattern:
        args.append('-run=' + pattern)
    log = rec.gates / name / 'stdout.log'
    try:
        return rec.run(name, [*args, *packages], cwd, extras, timeout=2100)
    finally:
        if log.exists():
            write(log.parent / 'test-identities.json', {action: sorted(events(log, action))
                                                        for action in ('pass', 'skip', 'fail')})


def qualification(rec, plan):
    from external_inputs import ExternalInputs
    root = rec.root
    cand = rec.sources['candidate']
    failures = []

    def attempt(name, fn):
        try:
            return fn()
        except (ValueError, RuntimeError) as error:
            failures.append({'gate': name, 'error': repr(error)})
            return None

    parents = attempt('historical-controls', lambda: old_controls(rec, plan))
    for side in ('candidate', 'baseline'):
        # The baseline has previously completed ordinary and both Staticcheck
        # checks; this independently refreshes the available attribution gates.
        src = rec.sources[side]
        extra = integration_env(rec, plan, side)
        configs = [('full-ordinary', ()), ('full-integration', ('-tags=integration',)),
                   ('full-integration-race', ('-tags=integration', '-race'))] if side == 'candidate' else [('full-ordinary', ())]
        for name, flags in configs:
            env = dict(extra)
            if side == 'candidate' and flags:
                if parents:
                    env.update(ALAN_PHASE3B4_PUBLIC_PARENTS=str(parents),
                               ALAN_PHASE3B4_PROMOTED_OUTPUT=str(root / ('promoted-' + name)))
                env.update(ALAN_PHASE3B4_EVIDENCE=str(root / ('phase3b4-' + name)),
                           ALAN_PHASE3B3_EVIDENCE=str(root / ('phase3b3-' + name)))
            gate = side + '-' + name
            log = attempt(gate, lambda: go_tests(rec, gate, src, flags, extras=env))
            if log and side == 'candidate':
                def check():
                    expect_tests(log, READERS)
                    if flags:
                        for key in ('ALAN_PHASE3B3_EVIDENCE', 'ALAN_PHASE3B4_EVIDENCE'):
                            verify_reciprocal(Path(env[key]), src, plan['sources'][side]['commit'])
                attempt(gate + '-required-controls', check)
        if side == 'candidate':
            for name, flags, packages in [('focused-ordinary-race', ('-race',), ('./internal/executor', './runtime')),
                    ('focused-integration-race', ('-race', '-tags=integration'), ('./internal/executor', './runtime', './internal/durabletest'))]:
                log = attempt(name, lambda: go_tests(rec, name, src, flags, packages, extra, FOCUSED))
                if log:
                    required = ['TestStaticExecutionUnchangedReadiness']
                    if '-tags=integration' in flags:
                        required += ['TestOwnerContinuationBasisAtRetainedCuts', 'TestCompetingReadCannotSettleAdopterDebt',
                                     'TestContinuationMaskedBeforeFirstApproval', 'TestContinuationConflictPromotion']
                    required += list(READERS)
                    attempt(name + '-source-backed-cases', lambda: expect_tests(log, required))
        for name, argv, env in [
            ('root-build', ['go', 'build', '-buildvcs=false', '-mod=readonly', './...'], {}),
            ('root-vet', ['go', 'vet', '-mod=readonly', './...'], {}),
            ('root-vet-integration', ['go', 'vet', '-mod=readonly', '-tags=integration', './...'], {}),
            ('root-module-verify', ['go', 'mod', 'verify'], {}),
            ('root-module-tidy', ['go', 'mod', 'tidy', '-diff'], {}),
            ('staticcheck', ['staticcheck', './...'], {}),
            ('staticcheck-integration', ['staticcheck', '-tags=integration', './...'], {}),
            ('staticcheck-race', ['staticcheck', './...'], {'GOFLAGS': '-race'}),
            ('staticcheck-integration-race', ['staticcheck', '-tags=integration', './...'], {'GOFLAGS': '-race'}),
        ]:
            attempt(side + '-' + name, lambda: rec.run(side + '-' + name, argv, src, env))
    def format_check():
        formatted = rec.run('gofmt', ['gofmt', '-l', *sorted(cand.rglob('*.go'))], cand)
        if formatted.read_text().strip():
            raise ValueError('frozen sources have gofmt differences')
    attempt('gofmt', format_check)
    external = attempt('external-public8', lambda: run_external(rec, ExternalInputs))
    attempt('nested-modules', lambda: nested_modules(rec, external, attempt))
    write(root / 'qualification-results.json', {'failures': failures, 'source_and_tool_inputs': rec.snapshot()})
    if failures:
        raise ValueError('qualification gates failed; see qualification-results.json')


class ExternalRecorder:
    """Use the existing route with task-specific source aliases and binary binding."""
    def __init__(self, rec, root):
        self.rec = rec
        self.root = root
        self.sources = rec.sources

    def run(self, name, argv, cwd=None, extra=None, expected=0):
        if name == 'external-genuine-promotion':
            return self.promote(name, argv, cwd, extra)
        result = self.rec.run(name, argv, cwd, extra, expected)
        if name.endswith('-build') or name == 'external-build-reader':
            self.rec.bind_tool(argv[argv.index('-o') + 1])
        if name in ('external-ordinary', 'external-race'):
            output = result.read_text()
            required = ('TestFreshStorageFormats', 'TestHistoricalArchivesRemainReadOnly',
                        'TestNewSeriesFixture', 'TestCanonicalLoweringEquality')
            if any(not any(line.startswith('--- PASS: ' + test + ' ') for line in output.splitlines())
                   for test in required):
                raise ValueError('external consumer did not run every required test')
        if name in ('external-baseline-read-11', 'external-baseline-read-12'):
            proof = read(result)
            if (proof['storage_version'] != int(name[-2:]) or proof['provider_contacts'] != 0 or
                    proof['open_close'] != 'valid' or proof['replay_runs'] != 8):
                raise ValueError('baseline consumer failed the genuine retained-format proof')
        if name == 'external-retained-formats' and '--- PASS: TestRetainedSeriesFormats ' not in result.read_text():
            raise ValueError('candidate retained-format check did not run')
        return result

    def promote(self, name, argv, source, extra):
        # Go test discovery omitted the inherited virtual _test.go file even
        # though the overlay command exited 0. Bind the byte-identical helper
        # in a disposable full source copy where discovery sees a real file.
        overlay, = [arg.removeprefix('-overlay=') for arg in argv if arg.startswith('-overlay=')]
        replacement, = read(overlay)['Replace'].items()
        destination, original = map(Path, replacement)
        relative = destination.relative_to(source)
        if destination.exists() or relative.as_posix() != 'internal/executor/independent_series_promotion_test.go':
            raise ValueError('unexpected promotion-only helper path')
        test_source = self.root / 'promotion-source'
        shutil.copytree(source, test_source)
        helper = test_source / relative
        shutil.copyfile(original, helper)
        if sha(helper) != sha(original):
            raise ValueError('promotion helper bytes differ')
        self.rec.bind('external-promotion-source', test_source)
        self.rec.bind_tool(overlay)
        log = go_tests(self.rec, name, test_source, packages=('./internal/executor',), extras=extra,
                       pattern='^TestIndependentPromoteSeries$')
        expect_tests(log, ['TestIndependentPromoteSeries'])
        return log


def run_external(rec, external_inputs):
    root = rec.root / 'external-route'
    root.mkdir()
    # The pinned route expects sibling baseline/candidate directories.
    for side in ('baseline', 'candidate'):
        (root / side).symlink_to(rec.sources[side], target_is_directory=True)
    adapter = ExternalRecorder(rec, root)
    # The inherited adapter directly assigns into sources when binding a
    # module. Register that exact checked inventory with TaskRecorder as well.
    class BoundExternalInputs(external_inputs):
        def run(self, name, argv, cwd=None, extra=None, expected=0):
            result = super().run(name, argv, cwd, extra, expected)
            if name.startswith('external-bind-'):
                key = 'external-' + Path(cwd).name
                directory = adapter.sources.pop(key)
                rec.bind(key, directory)
            return result
    ROUTES.external(BoundExternalInputs(adapter))
    return root / 'external'


def nested_modules(rec, external, attempt):
    src = rec.sources['candidate']
    found = {str(path.relative_to(src)) for path in src.rglob('go.mod')}
    if found != {'go.mod', *(name + '/go.mod' for name in MODULES)}:
        raise ValueError('frozen nested module inventory differs')
    root = rec.root / 'nested-modules'
    root.mkdir()
    for index, rel in enumerate(MODULES):
        name = 'nested-' + str(index)
        module = root / (name + '-source')
        shutil.copytree(src / rel, module)
        rec.run(name + '-bind', ['go', 'mod', 'edit', '-replace=github.com/JakeFAU/alan=' + str(src)], module)
        rec.bind(name, module)
        flags = ['-tags=series'] if index == 0 else []
        output = root / (name + '-build')
        output.mkdir()
        extra = {'ALAN_SOURCE': str(src)}
        if index == 0 and external:
            fixtures = root / 'fixtures'
            require_copy(external / 'ordinary-fixtures/series12.sqlite', fixtures / 'series12.sqlite', root / 'series12-copy.json')
            extra['SERIES_FIXTURE_DIR'] = str(fixtures)
        if index != 0 or external:
            attempt(name + '-test', lambda: go_tests(rec, name + '-test', module, flags, extras=extra))
        else:
            attempt(name + '-test', lambda: (_ for _ in ()).throw(ValueError('retained format12 fixture unavailable')))
        for suffix, argv in [('build', ['go', 'build', '-buildvcs=false', '-mod=readonly', *flags, '-o', str(output) + '/', './...']),
                             ('verify', ['go', 'mod', 'verify']), ('tidy', ['go', 'mod', 'tidy', '-diff'])]:
            attempt(name + '-' + suffix, lambda: rec.run(name + '-' + suffix, argv, module, extra))


def compare(rec, directory, tools, name, reports, dbs, expected=0):
    targets = [directory / 'compare-copies' / name / str(i) / 'store.sqlite' for i in range(2)]
    for db, target in zip(dbs, targets):
        target.parent.mkdir(parents=True)
        require_copy(db, target, target.parent / 'copy.json')
    log = rec.run(name, [rec.env['TASK5_PYTHON'], '-B', tools / 'compare.py', 'same-seed', *reports,
                          '--baseline-db', targets[0], '--candidate-db', targets[1]], expected=expected)
    if read(log)['equal_for_stated_coverage'] is not (expected == 0):
        raise ValueError('comparator did not produce the expected semantic verdict')


def differential(rec, plan, *, excluded=False):
    import socket
    tools = rec.root / 'builds/qualification'
    root = rec.root / ('excluded-differential-smoke' if excluded else 'differential')
    root.mkdir()
    for kind in (('direct',) if excluded else ('direct', 'http')):
        route_tools = tools / (kind + '-tools')
        endpoint = None
        if kind == 'http':
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as reservation:
                reservation.bind(('127.0.0.1', 0))
                endpoint = '127.0.0.1:' + str(reservation.getsockname()[1])
            write(root / 'http-endpoint.json', {'endpoint': endpoint})
        for pairing in (('structural', 'basis') if kind == 'direct' else ('structural',)):
            arms = [('baseline', 'snapshot'), ('candidate', 'snapshot')] if pairing == 'structural' else [
                ('candidate', 'snapshot'), ('candidate', 'targeted')]
            prefix = ('excluded-' if excluded else '') + kind + '-' + pairing
            seed = root / ('seed/store.sqlite' if kind == 'http' else prefix + '-seed/store.sqlite')
            seed.parent.mkdir()
            seed_source = arms[0][0]
            rec.run(prefix + '-seed', [tools / (seed_source + '-' + kind), '-mode', 'create-seed', '-bound', '8',
                    '-db', seed, '-source-label', plan['sources'][seed_source]['commit']])
            initial = inventory(seed.parent)
            for scenario in (('success',) if kind == 'http' or excluded else ('success', 'refusal3')):
                for reopen in ((0,) if excluded else (0, 1)):
                    case = f'{prefix}-{scenario}-reopen{reopen}'
                    reports, dbs = [], []
                    for index, (side, approval) in enumerate(arms):
                        target = root / case / str(index) / 'store.sqlite'
                        target.parent.mkdir(parents=True)
                        # Retain the pinned seed guard (owner lock, empty WAL,
                        # quick_check and untouched history) around complete copy.
                        name = case + '-' + str(index)
                        rec.run(name + '-clone', [rec.env['TASK5_PYTHON'], '-B',
                                route_tools / 'clone_seed.py', seed, target])
                        cmd = [tools / (side + '-' + kind), '-mode', 'execute-existing', '-bound', '8', '-db', target,
                               '-source-label', plan['sources'][side]['commit'], '-scenario', scenario, '-reopen-every', str(reopen)]
                        cmd.extend(['-approval-mode', approval] if kind == 'direct' else ['-endpoint', endpoint])
                        report = rec.run(name, cmd, target.parent)
                        if kind == 'direct':
                            proof = read(report)
                            verify_direct_reports(proof, reopen)
                        reports.append(report)
                        dbs.append(target)
                    compare(rec, root, route_tools, case + '-exact', reports, dbs)
                    if kind == 'http':
                        for i, report in enumerate(reports):
                            shutil.copyfile(report, root / f'{scenario}-reopen{reopen}-{"baseline" if i == 0 else "candidate"}.json')
                            target = root / f'{scenario}-reopen{reopen}' / ('baseline' if i == 0 else 'candidate') / 'store.sqlite'
                            target.parent.mkdir(parents=True)
                            require_copy(dbs[i], target, target.parent / 'copy.json')
                            Path(str(target) + '.lock').touch(exist_ok=False)
            if initial != inventory(seed.parent):
                raise ValueError('differential seed changed')
        if kind == 'http':
            negative = root / 'http-negative'
            rec.run('http-negative-comparator', [rec.env['TASK5_PYTHON'], '-B', route_tools / 'verify_tools.py',
                    '--binary', tools / 'candidate-http', root, negative])
            for result in negative.glob('reject-*.json'):
                if not result.name.endswith('.input.json') and read(result).get('equal_for_stated_coverage') is not False:
                    raise ValueError('negative comparator did not emit a semantic rejection: ' + result.name)
            Path(str(seed) + '.lock').touch(exist_ok=True)
            rec.run('http-matching-negative', [rec.env['TASK5_PYTHON'], '-B', route_tools / 'prove_comparison.py',
                    root, root / 'http-matching-negative', '--comparator', route_tools / 'compare.py'])
            for field in ('operation_id', 'capability', 'effect', 'protocol', 'endpoint'):
                candidate = root / 'success-reopen0-candidate.json'
                mutant = read(candidate)
                def change(node):
                    if isinstance(node, dict):
                        if 'prepared' in node and 'request_wire' in node:
                            node[field] = 'deliberate-foreign-' + field
                        for value in node.values():
                            change(value)
                    elif isinstance(node, list):
                        for value in node:
                            change(value)
                change(mutant)
                if mutant == read(candidate):
                    raise ValueError('contact mutant reached no journal records')
                mutated = root / ('matching-foreign-' + field + '.json')
                write(mutated, mutant)
                compare(rec, root, route_tools, 'http-reject-contact-' + field, [mutated, mutated],
                        [root / 'success-reopen0' / side / 'store.sqlite' for side in ('baseline', 'candidate')],
                        expected=1)


def verify_direct_reports(report, reopen):
    expected = []
    for position in range(1, 9):
        expected.append(('after-run', position))
        if reopen and position < 8 and position % reopen == 0:
            expected.append(('after-interval-reopen', position))
    found = []
    for cut in report['cuts']:
        if cut['label'] in ('after-run', 'after-interval-reopen'):
            rr = cut.get('run_report')
            fields = {'Steps', 'Dispatches', 'Quiescent', 'OpenRuns', 'UnresolvedIntents', 'PendingDeliveries',
                      'UnknownExternal', 'UnknownAdapters', 'Roots', 'RejectedAdmissions'}
            if (not isinstance(rr, dict) or set(rr) != fields or rr['Quiescent'] is not True or
                    rr['Roots'] != cut['snapshot']['Roots'] or len(rr['Roots']) != 8):
                raise ValueError('missing complete RunReport or typed root mismatch')
            found.append((cut['label'], cut['position']))
        elif cut.get('run_report') is not None:
            raise ValueError('unexpected RunReport cut')
    if report['ok'] is not True or report.get('error') or found != expected:
        raise ValueError('direct report cut completeness mismatch')


def reproduce(rec, plan, outside):
    from measurement import command, row, validate_report
    if outside is None or outside.parent.resolve() != Path('/private/tmp'):
        raise ValueError('use a fresh, direct child of /private/tmp for reproduction')
    outside.mkdir()
    source = outside / 'source'
    from custody import extract_source
    extract_source(rec.root / 'archives/candidate.tar', source)
    if inventory(source) != inventory(rec.sources['candidate']):
        raise ValueError('outside export differs from frozen candidate')
    rec.bind('reproduction-source', source)
    binary = generate(rec, 'outside-reproduction', source, outside / 'build', 'candidate', 'ordinary')
    design = row('outside-reproduction', 8, 'candidate', 'targeted', name='outside-b8-candidate-targeted')
    sample = outside / 'samples' / design['name']
    sample.mkdir(parents=True)
    revision = plan['sources']['candidate']['commit']
    log = rec.run('outside-b8-targeted', command(outside, design, revision, binary), sample)
    validate_report(read(log), design, revision, sample / 'store/store.sqlite')
    custody = rec.root / 'outside-reproduction'
    shutil.copytree(outside, custody)
    if inventory(custody) != inventory(outside):
        raise ValueError('outside reproduction copy differs')
    write(rec.root / 'outside-reproduction.json', {'original': str(outside), 'persistent_copy': str(custody),
          'candidate': plan['sources']['candidate'], 'binary_sha256': sha(binary), 'report_sha256': sha(log),
          'files': inventory(custody)})


def dispatch(rec, plan, args):
    if args.phase == 'build-qualification':
        build(rec, plan, args.source_digest)
    elif args.phase == 'reproduce':
        reproduce(rec, plan, args.outside)
    else:
        lock = rec.root / 'qualification-freeze.json'
        if sha(lock) != args.build_digest or read(lock)['source_freeze_sha256'] != args.source_digest:
            raise ValueError('qualification build digest mismatch')
        restore_inputs(rec, rec.root / 'qualification-inputs.json', read(lock)['inputs_sha256'])
        if args.phase == 'recovery':
            from recovery import recovery
            recovery(rec, plan)
        elif args.phase == 'excluded-weather-smoke':
            from recovery import weather
            weather(rec, plan, excluded=True)
        elif args.phase == 'excluded-historical-smoke':
            old_controls(rec, plan, excluded=True)
        elif args.phase == 'qualification':
            qualification(rec, plan)
        elif args.phase == 'excluded-differential-smoke':
            differential(rec, plan, excluded=True)
        else:
            differential(rec, plan)
