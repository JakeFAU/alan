"""Task 5 bindings around the P3 recorder and complete SQLite copy helper."""
import importlib.util
import json
from pathlib import Path, PurePosixPath
import sys
import tarfile

HERE = Path(__file__).resolve().parent
EVIDENCE = HERE.parents[1]
P3 = EVIDENCE / 'performance-p3'
P2 = EVIDENCE / 'performance-p2'
APPARATUS = HERE.parent / 'apparatus'

sys.dont_write_bytecode = True
sys.path[:0] = [str(P3 / 'final'), str(P3 / 'qualification'), str(APPARATUS)]
from common import copy_db, members, inventory, sha
from recorder import Recorder, write


def load(name, source):
    spec = importlib.util.spec_from_file_location(name, source)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def read(path):
    return json.loads(Path(path).read_text())


def extract_source(archive, destination):
    # Git archives contain only files/directories in these frozen revisions.
    # Prevalidate the entire member list before writing any member on Python 3.9.
    with tarfile.open(archive) as src:
        seen = set()
        for member in src.getmembers():
            path = PurePosixPath(member.name)
            if (path.is_absolute() or not path.parts or '..' in path.parts or
                    path in seen or not (member.isfile() or member.isdir())):
                raise ValueError('unsafe Git archive member: ' + member.name)
            seen.add(path)
        destination.mkdir()
        src.extractall(destination)


def require_copy(source, target, record, *, stopped=False):
    suffixes = {'', '-wal', '-shm'}
    if stopped and set(members(source)) != suffixes:
        raise ValueError('reaped live-owner snapshot lacks expected main/WAL/SHM sidecars')
    copy_db(source, target, record)
    if stopped and set(read(record)['copy']) != suffixes:
        raise ValueError('stopped source lost sidecars during copy')


class TaskRecorder(Recorder):
    def __init__(self, root, *, env, sources=None, tools=None, expected=None):
        if not env:
            raise ValueError('explicit Task 5 environment required')
        super().__init__(root, sources, tools, env)
        self.expected = expected or super().snapshot()
        self.snapshot()

    def snapshot(self):
        actual = super().snapshot()
        if actual != self.expected:
            raise ValueError('frozen inputs changed')
        return actual

    def bind(self, name, path):
        self.snapshot()
        if name in self.sources:
            raise ValueError('duplicate bound directory: ' + name)
        self.sources[name] = path
        self.expected['sources'][name] = inventory(path)

    def bind_tool(self, path):
        self.snapshot()
        path = Path(path).resolve()
        if path not in self.tools:
            self.tools.append(path)
            self.expected['tools'][str(path)] = sha(path)

    def run(self, name, argv, cwd=None, extra=None, expected=0, timeout=2400):
        # This also reaches Go invocations spawned by inherited fixture tests.
        flags = (extra or {}).get('GOFLAGS', self.env.get('GOFLAGS', ''))
        env = dict(extra or {}, GOFLAGS=' '.join(filter(None, ['-buildvcs=false', flags])))
        try:
            return super().run(name, argv, cwd, env, expected, timeout)
        except RuntimeError:
            gate = self.gates / name
            if (gate / 'outcome.json').exists():
                output = ''.join((gate / path).read_text(errors='replace')
                                 for path in ('stderr.log', 'stdout.log'))
                if 'operation not permitted' in output.lower() and ('listen tcp' in output or 'bind:' in output):
                    write(gate / 'host-constraint.json', {'kind': 'sandbox-loopback-denial'})
            raise


def events(log, action):
    result = set()
    for line in log.read_text(errors='replace').splitlines():
        try:
            row = json.loads(line)
        except ValueError:
            continue
        if isinstance(row, dict) and row.get('Action') == action and row.get('Test'):
            result.add((row['Package'], row['Test']))
    return result


def expect_tests(log, names):
    passed = {name for _, name in events(log, 'pass')}
    missing = set(names) - passed
    if missing:
        raise ValueError('required test pass events missing: ' + ', '.join(sorted(missing)))


def separate_inputs(rec, component, paths):
    expected = rec.snapshot()
    write(rec.root / (component + '-inputs.json'), {
        'snapshot': expected, 'sources': {name: str(p) for name, p in rec.sources.items()},
        'tools': [str(p) for p in rec.tools],
        'related_files': {str(p): sha(p) for p in paths},
    })


def restore_inputs(rec, path, digest=None):
    if digest is not None and sha(path) != digest:
        raise ValueError('input binding digest mismatch: ' + str(path))
    current = read(path)
    for name, source in current['sources'].items():
        source = Path(source)
        if name not in rec.sources:
            rec.bind(name, source)
        elif rec.sources[name] != source:
            raise ValueError('source path differs from frozen input: ' + name)
    for tool in current['tools']:
        rec.bind_tool(tool)
    if rec.snapshot() != current['snapshot'] or any(sha(p) != h for p, h in current['related_files'].items()):
        raise ValueError('frozen build or controller input differs')
    return current
