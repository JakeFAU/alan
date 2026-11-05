#!/usr/bin/env python3
"""Fresh final-source gates; no primary timing or recovery-campaign relabeling."""
import argparse
import importlib.util
import json
from pathlib import Path
import shutil
import sys
import tarfile

from recorder import Recorder, environment, inventory, sha, write

BASE = '926a899b651fccc48be52a054612a92afcf1ddfd'
OLD = 'af6943a5805a9a6c0e8fe77f804264f4765d07ec'
TIMED = '4f5fe2cc83316900a2f387180dd9f856db5303a0'
MODULES = ('docs/evidence/bounded-series/performance/module',
           'docs/evidence/performance-p2/driver', 'docs/evidence/performance-p2/differential',
           'docs/evidence/series-scaling/driver', 'docs/evidence/series-scaling/differential')


def freeze(root, repo):
    root.mkdir(parents=True, exist_ok=False)
    env = environment(root)
    setup = Recorder(root, env=env)
    queries = 0

    def query(argv, cwd, ignored_env):
        nonlocal queries
        queries += 1
        return setup.run('setup-query-' + str(queries), argv, cwd).read_text().strip()

    if query(['git', 'status', '--porcelain=v1'], repo, env):
        raise ValueError('tracked/untracked working source is not clean')
    commit = query(['git', 'rev-parse', 'HEAD'], repo, env)
    plan = {'version': 'alan/p3-final/v1', 'repo': str(repo), 'environment': env,
            'sources': {'candidate': commit, 'baseline': BASE, 'old': OLD},
            'nested_modules': MODULES, 'default_timeout_seconds': 2400,
            'test_timeout': '30m', 'purpose': 'final verification, no benchmark campaign',
            'staticcheck_race': 'GOFLAGS=-race, not tags-only race simulation'}
    write(root / 'plan.json', plan)
    for label, revision in plan['sources'].items():
        archive = root / (label + '.tar')
        setup.run('export-' + label, ['git', 'archive', '--output=' + str(archive), revision], repo)
        dest = root / label
        dest.mkdir()
        with tarfile.open(archive) as tar:
            tar.extractall(dest, filter='data')
        write(root / (label + '-source.json'), {
            'commit': revision, 'tree': query(['git', 'rev-parse', revision + '^{tree}'], repo, env),
            'archive_sha256': sha(archive), 'files': inventory(dest)})
    # Existing reciprocal tests obtain their real source HEAD through Git.
    # Keep a standalone shallow metadata copy outside the immutable source tree.
    metadata = root / 'provenance.git'
    setup.run('clone-provenance', ['git', 'clone', '--bare', '--no-local', '--depth=1',
              '--single-branch', '--no-tags', str(repo), str(metadata)], repo)
    metadata_head = query(['git', '--git-dir=' + str(metadata), 'rev-parse', 'HEAD'], repo, env)
    if metadata_head != commit:
        raise ValueError('provenance metadata does not identify frozen source')
    plan['git_provenance'] = str(metadata)
    candidate = root / 'candidate'
    found = sorted(str(p.relative_to(candidate)) for p in candidate.rglob('go.mod'))
    if found != sorted(['go.mod'] + [p + '/go.mod' for p in MODULES]):
        raise ValueError('module inventory differs')
    changed = query(['git', 'diff', '--name-only', TIMED, commit], repo, env).splitlines()
    executable_changes = [p for p in changed if p.endswith('.go') or Path(p).name in ('go.mod', 'go.sum')]
    if executable_changes:
        raise ValueError('timed executable bridge changed: ' + repr(executable_changes))
    task4_changes = query(['git', 'diff', '--name-only', '49df59f1f767dd63fec5e238bf565d023261a75b', commit], repo, env).splitlines()
    task4_executable = [p for p in task4_changes if p.endswith('.go') or Path(p).name in ('go.mod', 'go.sum')]
    if task4_executable:
        raise ValueError('Task4 executable bridge changed')
    setup.run('source-whitespace', ['git', 'diff', '--check', BASE, commit], repo)
    go = Path(shutil.which('go', path=env['PATH'])).resolve()
    goroot = Path(query([str(go), 'env', 'GOROOT'], candidate, env))
    tool_dir = Path(query([str(go), 'env', 'GOTOOLDIR'], candidate, env))
    modcache = Path(query([str(go), 'env', 'GOMODCACHE'], candidate, env))
    dependency = modcache / 'github.com/mattn/go-sqlite3@v1.14.52'
    tools = [Path(shutil.which(p, path=env['PATH'])).resolve()
             for p in ('go', 'gofmt', 'staticcheck', 'git', 'clang', 'sh')]
    tools += [Path(sys.executable).resolve()] + [tool_dir / p for p in ('compile', 'link', 'asm', 'cgo')]
    plan['tools'] = list(map(str, tools))
    plan['dependency'] = str(dependency)
    plan['goroot'] = str(goroot)
    write(root / 'execution-plan.json', plan)
    write(root / 'freeze.json', {'candidate_commit': commit,
          'candidate_tree': query(['git', 'rev-parse', commit + '^{tree}'], repo, env),
          'git_provenance_files': inventory(metadata),
          'tool_hashes': {str(p): sha(p) for p in tools},
          'dependency_files': inventory(dependency), 'module_paths': found,
          'task4_source': '49df59f1f767dd63fec5e238bf565d023261a75b',
          'task4_to_final_go_module_changes': task4_executable,
          'timed_source': TIMED, 'timed_to_final_changed_paths': changed,
          'timed_to_final_go_module_changes': executable_changes,
          'p2_changed_paths': query(['git', 'diff', '--name-only', BASE, commit, '--',
                                      'docs/evidence/performance-p2'], repo, env).splitlines()})
    # Import no mutable workspace controller after this point.
    print('FROZEN ' + commit, flush=True)


def make_recorder(root):
    plan = json.loads((root / 'execution-plan.json').read_text())
    sources = {name: root / name for name in plan['sources']}
    sources['git-provenance'] = Path(plan['git_provenance'])
    sources['sqlite-dependency'] = Path(plan['dependency'])
    r = Recorder(root, sources=sources, tools=[Path(p) for p in plan['tools']], env=plan['environment'])
    r.plan = plan
    frozen = json.loads((root / 'freeze.json').read_text())
    actual = r.snapshot()
    for name in plan['sources']:
        if actual['sources'][name] != json.loads((root / (name + '-source.json')).read_text())['files']:
            raise ValueError('source changed before execution: ' + name)
    if actual['sources']['git-provenance'] != frozen['git_provenance_files']:
        raise ValueError('Git provenance changed before execution')
    if actual['tools'] != frozen['tool_hashes'] or actual['sources']['sqlite-dependency'] != frozen['dependency_files']:
        raise ValueError('tool/dependency changed before execution')
    return r


def check_formats(root):
    failures = []
    for name in ('full-ordinary', 'full-integration', 'full-integration-race', 'authority-protocol'):
        log = root / 'gates' / name / 'stdout.log'
        events = [json.loads(line) for line in log.read_text().splitlines() if line.startswith('{')]
        matched = [e for e in events if e.get('Action') == 'pass' and 'frozen_old_reader' in e.get('Test', '')]
        if len(matched) != 2:
            failures.append({'gate': name, 'old_reader_pass_events': len(matched)})
    if failures:
        raise ValueError('actual historical-reader coverage missing: ' + repr(failures))


def execute(root):
    r = make_recorder(root)
    src = root / 'candidate'
    env = {'ALAN_PHASE3B3_OLD_READER': str(root / 'old-alan'),
           'GIT_DIR': r.plan['git_provenance'], 'GIT_WORK_TREE': str(src)}
    r.run('go-version', ['go', 'version'], src)
    r.run('go-settings', ['go', 'env', '-json', 'GOVERSION', 'GOOS', 'GOARCH', 'CGO_ENABLED',
                         'CC', 'GOROOT', 'GOTOOLDIR', 'GOMODCACHE', 'GOCACHE', 'GOTOOLCHAIN'], src)
    r.run('staticcheck-version', ['staticcheck', '-version'], src)
    r.run('clang-version', ['clang', '--version'], src)
    if (root / 'gates/go-version/stdout.log').read_text().strip() != 'go version go1.27.3 darwin/arm64':
        raise ValueError('Go version mismatch')
    if (root / 'gates/staticcheck-version/stdout.log').read_text().strip() != 'staticcheck 2026.2.2 (0.8.2)':
        raise ValueError('Staticcheck version mismatch')
    r.run('module-list', ['go', 'list', '-m', '-json', 'all'], src)
    r.run('build-old-reader', ['go', 'build', '-buildvcs=false', '-o', root / 'old-alan', './cmd/alan'], root / 'old')
    r.run('build-candidate-cli', ['go', 'build', '-buildvcs=false', '-o', root / 'candidate-alan', './cmd/alan'], src)
    r.tools.extend([root / 'old-alan', root / 'candidate-alan'])
    final = src / 'docs/evidence/performance-p3/final'
    r.run('recorder-tests', [sys.executable, '-B', '-m', 'unittest', '-v', 'test_recorder'], final)
    p3 = src / 'docs/evidence/performance-p3'
    r.run('p3-tool-tests', [sys.executable, '-B', '-m', 'unittest', '-v', 'test_tooling', 'test_task5_diagnostics'], p3,
          {'P3_TASK5_RESULTS': str(p3 / 'results-data.json')})
    r.run('custody-tests', [sys.executable, '-B', '-m', 'unittest', '-v', 'test_custody'], p3 / 'qualification')
    for name, flags in [('focused-p3', []), ('focused-p3-race', ['-race'])]:
        r.run(name, ['go', 'test', '-json', *flags, '-count=1', '-timeout=10m',
                     '-run=^TestP3Readiness', './internal/executor'], src, env)
    for name, flags in [('full-ordinary', []), ('full-integration', ['-tags=integration']),
                        ('full-integration-race', ['-race', '-tags=integration'])]:
        r.run(name, ['go', 'test', '-json', *flags, '-count=1', '-timeout=30m', './...'], src, env)
    r.run('authority-protocol', ['go', 'test', '-json', '-race', '-tags=integration', '-count=1',
          '-timeout=10m', '-run=^TestRequestAuthority', './internal/executor', './runtime', './author'], src, env)
    check_formats(root)
    # Reuse the qualified complete-sidecar public/old-reader route unchanged.
    route_file = p3 / 'qualification/run.py'
    spec = importlib.util.spec_from_file_location('p3_qualified_routes', route_file)
    routes = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(routes)
    routes.external(r)
    for index, rel in enumerate(MODULES):
        module = root / 'modules' / str(index)
        shutil.copytree(src / rel, module)
        r.run('module-' + str(index) + '-bind', ['go', 'mod', 'edit',
              '-replace=github.com/JakeFAU/alan=' + str(src)], module)
        r.sources['module-' + str(index)] = module
        extra = {'ALAN_SOURCE': str(src)}
        flags = []
        if index == 0:
            flags = ['-tags=series']
            fixtures = root / 'nested-consumer-fixtures'
            # The full test order creates its own format11 fixture; only seed12.
            for version in (12,):
                routes.copy_db(root / 'external/ordinary-fixtures' / ('series' + str(version) + '.sqlite'),
                               fixtures / ('series' + str(version) + '.sqlite'),
                               root / ('nested-fixture' + str(version) + '-copy.json'))
            extra['SERIES_FIXTURE_DIR'] = str(fixtures)
        build_output = root / 'module-builds' / str(index)
        build_output.mkdir(parents=True)
        for label, argv in [('test', ['go', 'test', '-json', *flags, '-count=1', '-timeout=10m', './...']),
                            ('build', ['go', 'build', *flags, '-o', str(build_output) + '/', './...']),
                            ('verify', ['go', 'mod', 'verify']), ('tidy', ['go', 'mod', 'tidy', '-diff'])]:
            r.run('module-' + str(index) + '-' + label, argv, module, extra)
    for name, argv, extra in [
        ('build', ['go', 'build', './...'], {}),
        ('vet', ['go', 'vet', './...'], {}),
        ('vet-integration', ['go', 'vet', '-tags=integration', './...'], {}),
        ('module-verify', ['go', 'mod', 'verify'], {}),
        ('module-tidy', ['go', 'mod', 'tidy', '-diff'], {}),
        ('staticcheck', ['staticcheck', './...'], {}),
        ('staticcheck-integration', ['staticcheck', '-tags=integration', './...'], {}),
        ('staticcheck-race', ['staticcheck', './...'], {'GOFLAGS': '-race'}),
        ('staticcheck-integration-race', ['staticcheck', '-tags=integration', './...'], {'GOFLAGS': '-race'}),
    ]:
        r.run(name, argv, src, extra)
    files = [str(p) for p in sorted(src.rglob('*.go'))]
    output = r.run('gofmt', ['gofmt', '-l', *files], src)
    if output.read_text().strip():
        raise ValueError('gofmt output is nonempty')
    final_snapshot = r.snapshot()
    write(root / 'final-source-tool-hashes.json', final_snapshot)
    for name in ('candidate', 'baseline', 'old'):
        expected = json.loads((root / (name + '-source.json')).read_text())['files']
        if final_snapshot['sources'][name] != expected:
            raise ValueError('export changed: ' + name)
    outcomes = {g.name: json.loads((g / 'outcome.json').read_text()) for g in sorted(r.gates.iterdir())}
    terminal = {g.name: json.loads((g / 'after.json').read_text()).get('events') for g in sorted(r.gates.iterdir())}
    write(root / 'attestation.json', {'freeze_sha256': sha(root / 'freeze.json'),
          'execution_plan_sha256': sha(root / 'execution-plan.json'),
          'tested_commit': r.plan['sources']['candidate'],
          'source_tool_hashes_sha256': sha(root / 'final-source-tool-hashes.json'),
          'outcomes': outcomes, 'terminal_events': terminal,
          'all_declared_gates_completed': True,
          'scope': 'repository qualification; earlier timing/process evidence keeps original identities'})
    write(root / 'raw-index.json', inventory(root))
    print('COMPLETE ' + r.plan['sources']['candidate'], flush=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('phase', choices=('freeze', 'execute'))
    parser.add_argument('root', type=Path)
    parser.add_argument('--repo', type=Path, default=Path.cwd())
    args = parser.parse_args()
    if args.phase == 'freeze':
        freeze(args.root.resolve(), args.repo.resolve())
    else:
        execute(args.root.resolve())
