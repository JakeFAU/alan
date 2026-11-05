"""Custody and command recording for the bounded P3 qualification (no timings)."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import time


def sha(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1048576), b''):
            h.update(block)
    return h.hexdigest()


def write(path, value):
    Path(path).write_text(json.dumps(value, indent=2, sort_keys=True) + '\n')


def inventory(root):
    return {str(p.relative_to(root)): {'size': p.stat().st_size, 'sha256': sha(p)}
            for p in sorted(Path(root).rglob('*')) if p.is_file()}


def members(db):
    return {s: sha(Path(str(db) + s)) for s in ('', '-wal', '-shm')
            if Path(str(db) + s).is_file()}


def copy_db(source, target, record):
    """Only a closed/reaped source; copy every extant member without SQLite reads."""
    before = members(source)
    if '' not in before:
        raise ValueError('missing main database')
    if any(Path(str(target) + s).exists() for s in ('', '-wal', '-shm')):
        raise FileExistsError(target)
    Path(target).parent.mkdir(parents=True, exist_ok=True)
    for suffix in before:
        with Path(str(source) + suffix).open('rb') as src, Path(str(target) + suffix).open('xb') as dst:
            shutil.copyfileobj(src, dst)
    copied, after = members(target), members(source)
    write(record, {'source': str(source), 'target': str(target), 'source_before': before,
                   'copy': copied, 'source_after': after, 'equal': before == copied == after})
    if before != copied or before != after:
        raise ValueError('source/copy changed during complete-sidecar copy')


class Recorder:
    def __init__(self, root):
        self.root = Path(root).resolve()
        self.plan = json.loads((self.root / 'route-manifest.json').read_text())
        self.env = self.plan['environment']
        self.gates = self.root / 'gates'
        self.gates.mkdir(exist_ok=True)

    def run(self, name, argv, cwd=None, extra=None, expected=0):
        gate = self.gates / name
        gate.mkdir()  # Never overwrite a failed attempt.
        argv = [str(x) for x in argv]
        env = dict(self.env, **(extra or {}))
        cwd = str(cwd or self.root)
        executable = shutil.which(argv[0], path=env['PATH'])
        command = {'argv': argv, 'cwd': cwd, 'environment': env, 'expected_exit': expected,
                   'executable': executable, 'executable_sha256': sha(executable),
                   'started_unix': time.time(), 'controller_sha256': sha(__file__)}
        write(gate / 'command.json', command)
        result = None
        try:
            with (gate / 'stdout.log').open('xb') as out, (gate / 'stderr.log').open('xb') as err:
                result = subprocess.run(argv, cwd=cwd, env=env, stdout=out, stderr=err)
            # Save the child exit before any post-exit metadata access.
            write(gate / 'exit.json', {'exit': result.returncode, 'expected_exit': expected})
        finally:
            write(gate / 'outputs.json', inventory(gate))
        write(gate / 'executable-after.json', {'sha256': sha(executable)})
        if sha(executable) != command['executable_sha256']:
            raise ValueError('executable changed: ' + name)
        print(name + ': exit=' + str(result.returncode), flush=True)
        if result.returncode != expected:
            raise RuntimeError('stopped at ' + name + '; all outputs retained')
        return gate / 'stdout.log'
