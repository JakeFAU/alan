"""Immutable file identities and whitelist-only process settings."""
import hashlib
import json
import os
from pathlib import Path
import sys
from validation import ENVIRONMENT, require

HERE = Path(__file__).resolve().parent
P3 = HERE.parent
REPO = P3.parents[2]
TOOL_FILES = ('validation.py', 'custody.py', 'freeze.py', 'run.py', 'analyze.py', 'test_campaign.py')


def sha(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1048576), b''):
            h.update(block)
    return h.hexdigest()


def inventory(root):
    root = Path(root)
    return {str(p.relative_to(root)): sha(p) for p in sorted(root.rglob('*')) if p.is_file() and '__pycache__' not in p.parts}


def tools():
    paths = [HERE / f for f in TOOL_FILES]
    paths += [P3 / f for f in ('schedule.json', 'contract.md', 'selector_probe.py', 'instrument.py')]
    paths += sorted((P3.parent / 'performance-p2' / 'driver').glob('*'))
    return {str(p): sha(p) for p in paths if p.is_file()}


def write(path, obj):
    with Path(path).open('x') as out:
        out.write(json.dumps(obj, indent=2) + '\n')


def fingerprint(obj):
    return hashlib.sha256(json.dumps(obj, sort_keys=True, separators=(',', ':')).encode()).hexdigest()


def environment():
    # Do not inherit provider credentials or undeclared Go/ALAN knobs.
    return dict(ENVIRONMENT, PATH='/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin',
                HOME=os.environ['HOME'], TMPDIR='/private/tmp',
                GOCACHE='/private/tmp/alan-p3-j216kxoz/go-cache',
                GOPATH=os.environ.get('GOPATH', os.environ['HOME'] + '/go'))


def snapshot(plan):
    return {'tools': tools(), 'python': sha(plan['python']),
            'sources': {v: inventory(i['source_dir']) for v, i in plan['variants'].items()},
            'build_inputs': {v: inventory(i['module_dir']) for v, i in plan['variants'].items()},
            'binaries': {v: sha(i['binary']) for v, i in plan['variants'].items()},
            'dependencies':inventory(plan['dependency_dir']), 'go':sha(plan['go_binary'])}


def expected_custody(plan):
    return {'tools': plan['tool_hashes'], 'python': plan['python_sha256'],
            'sources': {v: i['source_files'] for v, i in plan['variants'].items()},
            'build_inputs': {v: i['module_files'] for v, i in plan['variants'].items()},
            'binaries': {v: i['binary_sha256'] for v, i in plan['variants'].items()},
            'dependencies':plan['dependency_files'], 'go':plan['go_binary_sha256']}


def validate_plan(plan):
    design = json.loads((P3 / 'schedule.json').read_text())
    require(plan['design'] == design and len(design['rows']) == 98, 'frozen 98-row design differs')
    expected = []
    for bound in (1, 8, 16, 32, 64, 128, 256):
        for variant in ('A', 'B'):
            expected.append({'name': f'b{bound}-warmup-{variant}', 'bound': bound, 'variant': variant, 'excluded': 'warmup'})
        for pair in range(1, 7):
            for variant in (('A', 'B') if pair % 2 else ('B', 'A')):
                expected.append({'name': f'b{bound}-pair{pair}-{variant}', 'bound': bound, 'pair': pair, 'variant': variant, 'excluded': False})
    require(design == {'version': 'alan/performance-p3/design-v1', 'variants': {'A':'baseline', 'B':'candidate'},
                       'args': ['-mode=direct', '-prefix=true'], 'rows': expected}, 'design order/flags changed')
    require(plan['version'] == 'alan/performance-p3/binding-v1', 'binding version')
    require(set(plan['variants']) == {'A', 'B'}, 'two variants required')
    require(plan['variants']['A']['commit'] == '926a899b651fccc48be52a054612a92afcf1ddfd', 'wrong baseline')
    require(plan['variants']['B']['commit'] == '4f5fe2cc83316900a2f387180dd9f856db5303a0', 'wrong candidate source')
    require(plan['environment'] == environment(), 'whitelisted environment changed')
    require(plan['runtime'] == {'go_version':'go1.27.3', 'goos':'darwin', 'goarch':'arm64', 'gomaxprocs':4,
                               'mem_profile_rate':524288, 'profiled':False, 'instrumented':False,
                               'schedule':'ordinary', 'reopen_every':0, 'prefix_enabled':True}, 'runtime drift')
    for key in ('output', 'cwd', 'python'):
        require(Path(plan[key]).is_absolute(), 'absolute path required: ' + key)
    require(plan['python'] == str(Path(sys.executable).resolve()), 'Python interpreter changed')
    require(plan['tool_hashes'] == tools(), 'tools changed after freeze')
    require(plan['commands'] == [command(plan, bound_row(plan,r)) for r in design['rows']], 'frozen argv changed')
    require(plan['controls']['source_files'] == plan['variants']['B']['source_files'], 'control source drift')
    require(plan['controls']['selector_probe_sha256'] == sha(P3 / 'selector_probe.py'), 'control generator drift')


def bound_row(plan, design_row):
    identity = plan['variants'][design_row['variant']]
    return dict(design_row, source=identity['commit'], expected_runtime=plan['runtime'])


def command(plan, row):
    return [plan['variants'][row['variant']]['binary'], '-bound', str(row['bound']),
            '-dir', str(Path(plan['output']) / (row['name'] + '.store')),
            '-sample', row['name'], '-source-label', row['source'], *plan['design']['args']]
