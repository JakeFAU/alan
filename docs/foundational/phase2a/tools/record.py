"""Run a qualification command in an explicit environment and archive its inputs and output."""
import argparse
import gzip
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[4]
PHASE = ROOT / 'docs/foundational/phase2a'


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(__doc__)
    parser.add_argument('label')
    parser.add_argument('--cwd', default='.')
    parser.add_argument('--env', action='append', default=[])
    parser.add_argument('--out', type=Path, default=PHASE / 'evidence/commands')
    boundary = sys.argv.index('--')
    args = parser.parse_args(sys.argv[1:boundary])
    command = sys.argv[boundary + 1:]
    directory = (ROOT / args.cwd).resolve()
    env = {key: os.environ[key] for key in ('HOME', 'PATH', 'TMPDIR', 'USER') if key in os.environ}
    env.update(GOTOOLCHAIN='local', CGO_ENABLED='1', GOPROXY='off', GOSUMDB='off',
               GOFLAGS='-mod=readonly', GOCACHE=os.getenv('GOCACHE', '/private/tmp/alan-phase2a-go-cache'),
               PYTHONDONTWRITEBYTECODE='1', LC_ALL='C')
    for setting in args.env:
        key, value = setting.split('=', 1)
        env[key] = value
    executable = Path(shutil.which(command[0], path=env['PATH']) or command[0]).resolve()
    command[0] = str(executable)
    paths = subprocess.check_output(['git', 'ls-files', '-co', '--exclude-standard', '-z'],
                                    cwd=ROOT).decode().split('\0')
    inputs = {}
    for name in sorted(set(paths)):
        p = ROOT / name
        if (name.endswith(('.go', '.mod', '.sum', '.c', '.h')) or name.startswith('docs/foundational/phase2a/tools/')
                or name in {'paper/alan-dissertation.tex', 'paper/references.bib'}) and p.is_file():
            inputs[name] = digest(p)
    args.out.mkdir(parents=True, exist_ok=True)
    dest = args.out / (args.label + '.json.gz')
    if dest.exists():
        raise SystemExit(f'refusing to replace {dest}')
    result = subprocess.run(command, cwd=directory, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    output = result.stdout.decode('utf-8', errors='backslashreplace')
    record = dict(argv=command, executable_sha256=digest(executable), cwd=str(directory),
                  environment=env, inputs=inputs, output=output, exit=result.returncode)
    dest.write_bytes(gzip.compress(json.dumps(record, sort_keys=True).encode(), mtime=0))
    print(f'{args.label}: exit={result.returncode} record={dest}')
    if result.returncode:
        print(output[-4500:])
    return result.returncode


if __name__ == '__main__':
    sys.exit(main())
