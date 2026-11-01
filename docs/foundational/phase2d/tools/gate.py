"""Run one gate on an immutable committed checkout and record the before/after inputs."""
import argparse
import gzip
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[4]


def sha(path):
    h = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(1 << 20), b''):
            h.update(block)
    return h.hexdigest()


def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT)


def inputs():
    names = git('ls-files', '-co', '--exclude-standard', '-z').decode().split('\0')
    suffixes = ('.go', '.mod', '.sum', '.c', '.cc', '.h', '.s', '.S', '.syso', '.py', '.sh')
    return {name: sha(ROOT / name) for name in sorted(set(names)) if name.endswith(suffixes) or name=='examples/evidence/corpus.txt'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('record', type=Path)
    parser.add_argument('--cwd', type=Path, default=Path('.'))
    parser.add_argument('--env', action='append', default=[])
    split = sys.argv.index('--')
    a = parser.parse_args(sys.argv[1:split])
    argv = sys.argv[split+1:]
    if not argv or a.record.exists() or git('status', '--porcelain'):
        parser.error('supply an unused record path, a command and a clean frozen checkout')
    out = a.record.resolve()
    if out.is_relative_to(ROOT):
        parser.error('write evidence outside the frozen checkout')
    cwd = (ROOT / a.cwd).resolve()
    if not cwd.is_relative_to(ROOT):
        parser.error('working directory is outside the frozen checkout')
    env = {k:os.environ[k] for k in ('HOME', 'PATH','TMPDIR','USER') if k in os.environ}
    env.update(GOTOOLCHAIN='local', CGO_ENABLED='1', GOPROXY='off', GOSUMDB='off', GOFLAGS='-mod=readonly',
               GOCACHE=os.getenv('GOCACHE','/private/tmp/alan-phase2a-gocache'), PYTHONDONTWRITEBYTECODE='1', LC_ALL='C')
    for setting in a.env:
        k,v = setting.split('=',1)
        env[k] = v
    exe = Path(shutil.which(argv[0],path=env['PATH']) or argv[0]).resolve()
    argv[0] = str(exe)
    head = git('rev-parse','HEAD').decode().strip()
    before = inputs()
    out.parent.mkdir(parents=True, exist_ok=True)
    started = time.time_ns()
    child = subprocess.run(argv,cwd=cwd,env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
    finished = time.time_ns()
    after = inputs()
    post_head = git('rev-parse','HEAD').decode().strip()
    status = git('status','--porcelain').decode()
    delta = {p:[before.get(p),after.get(p)] for p in sorted(before.keys() | after.keys()) if before.get(p)!=after.get(p)}
    record = dict(head=head, end_head=post_head, end_status=status,inputs=before,end_differences=delta,
                  tree=git('rev-parse',head+'^{tree}').decode().strip(), cwd=str(cwd),argv=argv,executable_sha256=sha(exe),
                  start_ns=started,end_ns=finished,env=env,exit=child.returncode,output=child.stdout.decode(errors='backslashreplace'))
    with out.open('xb') as stream:
        stream.write(gzip.compress(json.dumps(record,sort_keys=True).encode(),mtime=0))
    print(out, 'exit',child.returncode,'inputs',len(before),'changed',len(delta),'git status',repr(status),flush=True)
    if child.returncode:
        print(record['output'][-2500:])
    return child.returncode or int(head!=post_head or bool(delta) or bool(status))


if __name__ == '__main__':
    sys.exit(main())
