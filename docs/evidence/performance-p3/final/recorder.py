"""Exclusive, whitelist-only final gates; persist child status before metadata."""
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import time

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'qualification'))
from common import inventory, sha


def write(path, value):
    with Path(path).open('x') as out:
        out.write(json.dumps(value, indent=2, sort_keys=True) + '\n')
        out.flush()
        os.fsync(out.fileno())


def environment(root):
    return dict(PATH='/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin',
                HOME='/Users/jacob', TMPDIR='/private/tmp', GOPATH='/Users/jacob/go',
                GOCACHE='/private/tmp/alan-p3-j216kxoz/go-cache',
                STATICCHECK_CACHE=str(root.parent / (root.name + '-staticcheck-cache')),
                GOMAXPROCS='4', GOGC='100', GOMEMLIMIT='off', CGO_ENABLED='1',
                GOWORK='off', GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off',
                GODEBUG='', GORACE='', GOFLAGS='', GOTRACEBACK='single',
                PYTHONDONTWRITEBYTECODE='1', LANG='C.UTF-8', LC_ALL='C.UTF-8')


def events(path):
    result = {'test_pass': 0, 'test_fail': [], 'test_skip': [],
              'package_pass': [], 'package_fail': [], 'no_test_packages': []}
    for line in path.read_text(errors='replace').splitlines():
        try:
            item = json.loads(line)
        except ValueError:
            if line.strip().startswith('--- PASS:'):
                result['test_pass'] += 1
            if line.strip().startswith('--- SKIP:'):
                result['test_skip'].append(line.strip())
            continue
        if not isinstance(item, dict):
            continue
        action, test = item.get('Action'), item.get('Test')
        identity = {'package': item.get('Package'), 'test': test}
        if test and action == 'pass':
            result['test_pass'] += 1
        if test and action in ('fail', 'skip'):
            result['test_' + action].append(identity)
        if not test and action in ('pass', 'fail'):
            result['package_' + action].append(item.get('Package'))
        if '[no test files]' in item.get('Output', ''):
            result['no_test_packages'].append(item.get('Package'))
    return result


class Recorder:
    def __init__(self, root, sources=None, tools=None, env=None):
        self.root = Path(root).resolve()
        self.sources = sources or {}
        self.tools = tools or []
        self.env = env or environment(self.root)
        self.gates = self.root / 'gates'
        self.gates.mkdir(exist_ok=True)

    def snapshot(self):
        return {'sources': {name: inventory(path) for name, path in self.sources.items()},
                'tools': {str(path): sha(path) for path in self.tools}}

    def run(self, name, argv, cwd=None, extra=None, expected=0, timeout=2400):
        gate = self.gates / name
        gate.mkdir()  # Failed/partial gates are never overwritten.
        argv = list(map(str, argv))
        env = dict(self.env, **(extra or {}))
        executable = shutil.which(argv[0], path=env['PATH'])
        before = self.snapshot()
        write(gate / 'before.json', before)
        record = {'argv': argv, 'cwd': str(cwd or self.root), 'environment': env,
                  'expected_exit': expected, 'timeout_seconds': timeout,
                  'executable': executable, 'executable_sha256': sha(executable),
                  'controller_sha256': sha(__file__), 'started_unix': time.time(),
                  'host_load': os.getloadavg()}
        write(gate / 'command.json', record)
        print('START ' + name, flush=True)
        outcome = {'exit': None, 'timeout': False, 'spawn_error': None}
        with (gate / 'stdout.log').open('xb') as out, (gate / 'stderr.log').open('xb') as err:
            try:
                child = subprocess.Popen(argv, cwd=record['cwd'], env=env, stdout=out,
                                         stderr=err, start_new_session=True)
                try:
                    outcome['exit'] = child.wait(timeout=timeout)
                except subprocess.TimeoutExpired:
                    outcome['timeout'] = True
                    os.killpg(child.pid, signal.SIGKILL)
                    outcome['exit'] = child.wait()
            except OSError as error:
                outcome['spawn_error'] = repr(error)
        outcome['ended_unix'] = time.time()
        outcome['seconds'] = outcome['ended_unix'] - record['started_unix']
        # Keep actual status even if the child deletes its executable or source.
        write(gate / 'outcome.json', outcome)
        try:
            after = self.snapshot()
            after.update(executable_sha256=sha(executable),
                         stdout_sha256=sha(gate / 'stdout.log'),
                         stderr_sha256=sha(gate / 'stderr.log'),
                         events=events(gate / 'stdout.log'), host_load=os.getloadavg())
            write(gate / 'after.json', after)
            if after['sources'] != before['sources'] or after['tools'] != before['tools']:
                raise ValueError('source/tool bytes changed')
            if after['executable_sha256'] != record['executable_sha256']:
                raise ValueError('executable bytes changed')
        except Exception as error:
            write(gate / 'postmetadata-error.json', {'error': repr(error)})
            raise
        print('EXIT ' + name + ' ' + str(outcome['exit']), flush=True)
        if outcome['exit'] != expected or outcome['timeout'] or outcome['spawn_error']:
            raise RuntimeError('gate failed; retained: ' + name)
        return gate / 'stdout.log'
