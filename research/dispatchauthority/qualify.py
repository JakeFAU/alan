"""Build the bounded reference and export its evidence into a new directory."""
import argparse
import gzip
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parents[2]
MODEL = Path(__file__).resolve().parent
PACKAGE = './research/dispatchauthority'


def digest(data):
    return hashlib.sha256(data).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=Path)
    output = parser.parse_args().output.resolve()
    output.mkdir(exist_ok=True, parents=True)
    if any(output.iterdir()):
        raise SystemExit('output directory must be empty')
    env = {k: os.environ[k] for k in ('PATH', 'HOME', 'TMPDIR') if k in os.environ}
    env.update(GOTOOLCHAIN='local', GOWORK='off', CGO_ENABLED='1', GOFLAGS='-mod=readonly',
               GOPROXY='off', GOSUMDB='off', LC_ALL='C',
               GOCACHE=os.environ.get('GOCACHE', '/private/tmp/alan-phase2a-gocache'))
    go = str(Path(shutil.which('go', path=env['PATH'])).resolve())
    gofmt = str(Path(shutil.which('gofmt', path=env['PATH'])).resolve())
    sources = sorted(MODEL.rglob('*.go'))
    bound = ROOT / 'docs/foundational/phase2b'
    inputs = [*sources, Path(__file__).resolve(), ROOT/'go.mod', ROOT/'go.sum',
              bound/'01-contract.md', bound/'01-source-and-matrix.md']
    record = {'cwd': str(ROOT), 'environment': env, 'commands': [], 'inputs': {
        str(p.relative_to(ROOT)): digest(p.read_bytes()) for p in inputs}}

    def run(*argv, sink=None):
        argv = list(map(str, argv))
        proc = subprocess.run(argv, cwd=ROOT, env=env, capture_output=True)
        cmd = {'argv': argv, 'executable_sha256': digest(Path(argv[0]).read_bytes()),
               'exit': proc.returncode, 'stderr': proc.stderr.decode(),
               'stdout_sha256': digest(proc.stdout)}
        if sink is None:
            cmd['stdout'] = proc.stdout.decode()
        elif sink == 'graphs.jsonl.gz':
            (output / sink).write_bytes(gzip.compress(proc.stdout, mtime=0))
            cmd['compressed_stdout_file'] = sink
        else:
            (output / sink).write_bytes(proc.stdout)
            cmd['stdout_file'] = sink
        record['commands'].append(cmd)
        proc.check_returncode()
        return proc.stdout

    try:
        run(go, 'version')
        run(go, 'env', 'GOOS', 'GOARCH', 'CC', 'GOROOT', 'GOMODCACHE')
        if run(gofmt, '-l', *sources):
            raise RuntimeError('unformatted source')
        run(go, 'vet', PACKAGE+'/...')
        with tempfile.TemporaryDirectory(prefix='phase2b-reference-') as tmp:
            for mode, flags in [('normal', []), ('race', ['-race'])]:
                binary = Path(tmp) / mode
                run(go, 'test', '-c', *flags, '-o', binary, PACKAGE)
                run(go, 'version', '-m', binary)
                run(binary, '-test.v', '-test.count=1', sink=mode+'.txt')
            exporter = Path(tmp) / 'export'
            run(go, 'build', '-o', exporter, PACKAGE+'/cmd/export')
            lines = run(exporter, sink='graphs.jsonl.gz').splitlines()
        counts = []
        for line in lines:
            graph = json.loads(line)
            rows = graph['Rows']
            counts.append({'name': graph['Profile']['Name'], 'states': len(rows),
                           'edges': sum(len(r['Outgoing']) for r in rows),
                           'sinks': sum(not r['Outgoing'] for r in rows)})
        summary = {'profiles': counts, 'totals': {k: sum(p[k] for p in counts)
                                            for k in ('states', 'edges', 'sinks')}}
        (output / 'summary.json').write_text(json.dumps(summary, indent=2)+'\n')
        record['artifacts'] = {p.name: digest(p.read_bytes()) for p in sorted(output.iterdir())}
        print(json.dumps(summary['totals']))
    finally:
        (output/'tool-record.json').write_text(json.dumps(record, indent=2)+'\n')


if __name__ == '__main__':
    main()
