"""Bind the Phase 2C packet, executable inputs, and the two historical inventories."""
import argparse
from hashlib import sha256
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[4]
PHASE = 'docs/foundational/phase2c/'
INDEX = PHASE + 'evidence/index.sha256'
REVIEW = PHASE + 'evidence/whole-review.md'
BASE = 'a461f25fb42d1663735f68296460eceda4270d47'


def git_files(*args):
    return set(subprocess.check_output(['git', *args, '-z'], cwd=ROOT).decode().split('\0')) - {''}


def inventory():
    files = git_files('ls-files', '-co', '--exclude-standard')
    changed = git_files('diff', '--name-only', BASE)
    paths = {p for p in files if p in changed or p.startswith((PHASE, 'research/reconciliation/', 'examples/reconciliation/', 'adapters/receiptquery/'))
             or p.endswith(('.go', '.mod', '.sum', '.c', '.h')) or p in {
                 'AGENTS.md', 'paper/alan-dissertation.md', 'paper/alan-dissertation.tex',
                 'paper/alan-dissertation.pdf', 'paper/references.bib',
                 'docs/foundational/phase2a/evidence/index.sha256',
                 'docs/foundational/phase2b/evidence/index.sha256'}}
    paths.difference_update({INDEX, REVIEW})
    return ''.join(sha256((ROOT / path).read_bytes()).hexdigest() + '  ' + path + '\n' for path in sorted(paths))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--write', action='store_true')
    args = parser.parse_args()
    result = inventory()
    if args.write:
        (ROOT / INDEX).write_text(result)
    elif result != (ROOT / INDEX).read_text():
        raise SystemExit('source/evidence inventory differs')
    print('PASS', len(result.splitlines()), 'files;', sha256(result.encode()).hexdigest(), INDEX)
