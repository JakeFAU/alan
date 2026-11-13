"""Derive the test-only reporting oracle from the immutable production baseline."""
import argparse
from pathlib import Path
import re
import subprocess

BASE = '5f5d382c8d1e9c7369e7624e3a962746c9bf2229'
HERE = Path(__file__).resolve().parent

FUNCTIONS = {
    'runner.go': [('(s *Store) ', 'finishRun'), ('', 'pendingDeliveries')],
    'composition_inspect.go': [('', 'readRootCompletions'), ('', 'readRootReportCharges')],
    'composition_root.go': [('', 'prepareRoot')],
    'invocation_outcome.go': [('', 'prepareInvocation'), ('', 'prepareCompositionScope')],
}


def oracle(repo):
    fragments = ['''//go:build integration

// Frozen report traversal from 5f5d382. Install as report_frozen_test.go for
// differential tests. Scheduling and authoritative leaf decoders are shared.
package executor

import (
	"context"
	"database/sql"
	"errors"
	"sort"

	"github.com/mattn/go-sqlite3"
)
''']
    for path, functions in FUNCTIONS.items():
        source = subprocess.check_output(
            ['git', 'show', f'{BASE}:internal/executor/{path}'], cwd=repo, text=True)
        for receiver, name in functions:
            match, = re.finditer(r'^func ' + re.escape(receiver + name) + r'\(', source, re.M)
            fragments.append(source[match.start():source.index('\n}', match.end()) + 2] + '\n')
    names = [name for functions in FUNCTIONS.values() for _, name in functions]
    return re.sub(r'\b(' + '|'.join(names) + r')(?=\()', r'\1FrozenReport',
                  '\n'.join(fragments))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    data = oracle(HERE.parents[2])
    fixture = HERE / 'report_frozen_test.go.txt'
    if args.check:
        if fixture.read_text() != data:
            raise SystemExit('report oracle differs from the frozen source')
    else:
        print(data, end='')
