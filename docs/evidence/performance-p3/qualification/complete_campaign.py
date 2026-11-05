#!/usr/bin/env python3
"""Retain legacy tools unchanged; adapt only their seed/mutant copy mechanics.

The original clone guards, exact comparator, fixed endpoint and schedules remain.
All source patches are emitted before execution. An existing output is refused.
"""
import argparse
import copy
import difflib
import json
from pathlib import Path
import shutil

from common import Recorder, sha, write
from run import BASE, CANDIDATE, compare


def adapt(root, source, destination):
    shutil.copytree(source, destination)
    shutil.copyfile(Path(__file__).with_name('common.py'), destination / 'p3_custody.py')
    source_text = (source / 'clone_seed.py').read_text()
    patched = source_text.replace('import sqlite3\n', 'import sqlite3\nfrom p3_custody import copy_db\n')
    start = '        with destination.open("xb") as target, seed.open("rb") as source:\n'
    end = '        outputs.append({"path": str(destination), "sha256": copied})\n'
    if source_text.count(start) != 1 or source_text.count(end) != 1:
        raise ValueError('legacy clone anchor changed')
    a, b = patched.index(start), patched.index(end) + len(end)
    patched = patched[:a] + '''        proof = pathlib.Path(str(destination) + ".copy.json")
        copy_db(seed, destination, proof)
        copied = hashlib.sha256(destination.read_bytes()).hexdigest()
        if copied != digest:
            raise RuntimeError("seed clone bytes differ")
        outputs.append({"path": str(destination), "sha256": copied, "complete_sidecar_proof": str(proof)})
''' + patched[b:]
    (destination / 'clone_seed.py').write_text(patched)
    (destination / 'clone-seed.patch').write_text(''.join(difflib.unified_diff(source_text.splitlines(True), patched.splitlines(True), fromfile=str(source / 'clone_seed.py'), tofile=str(destination / 'clone_seed.py'))))
    source_text = (source / 'verify_tools.py').read_text()
    patched = source_text.replace('import sys\n', '''import sys
from p3_custody import copy_db


def copy_complete(source, target):
    copy_db(source, target, pathlib.Path(str(target) + ".copy.json"))
''')
    patched = patched.replace('shutil.copyfile(', 'copy_complete(')
    if source_text.count('shutil.copyfile(') != 2:
        raise ValueError('legacy mutant copy anchors changed')
    (destination / 'verify_tools.py').write_text(patched)
    (destination / 'mutant-copy.patch').write_text(''.join(difflib.unified_diff(source_text.splitlines(True), patched.splitlines(True), fromfile=str(source / 'verify_tools.py'), tofile=str(destination / 'verify_tools.py'))))
    write(destination / 'adaptation.json', {'scope': 'only complete SQLite member copies; guards/assertions/schedule/comparator unchanged',
          'files': {p.name: {'original': sha(source / p.name) if (source / p.name).is_file() else None, 'adapted': sha(p)}
                    for p in destination.iterdir() if p.is_file() and p.name != 'adaptation.json'}})


def direct(r, tools):
    out = r.root / 'direct-complete'
    out.mkdir()
    seed = out / 'seed.sqlite'
    r.run('direct-complete-seed', [r.root / 'baseline-direct', '-mode', 'create-seed', '-bound', '8', '-db', seed, '-source-label', BASE])
    for scenario in ('success', 'refusal3'):
        for interval in (0, 1):
            name = scenario + '-reopen' + str(interval)
            dbs = [out / name / side / 'store.sqlite' for side in ('baseline', 'candidate')]
            r.run('direct-complete-' + name + '-clone', ['python3', tools / 'clone_seed.py', seed, *dbs])
            reports = []
            for label, db in zip(('baseline', 'candidate'), dbs):
                reports.append(r.run('direct-complete-' + name + '-' + label,
                               [r.root / (label + '-direct'), '-mode', 'execute-existing', '-bound', '8', '-db', db,
                                '-source-label', r.plan['sources'][label], '-scenario', scenario, '-reopen-every', str(interval)]))
            compare(r, 'direct-complete-' + name + '-compare', reports, dbs)


def http(r, tools):
    out = r.root / 'http-complete'
    r.run('http-complete-campaign', ['python3', tools / 'campaign.py', '--baseline-binary', r.root / 'baseline-http',
          '--candidate-binary', r.root / 'candidate-http', '--baseline-source', BASE, '--candidate-source', CANDIDATE, '--output', out])
    r.run('http-complete-negative-proof', ['python3', tools / 'verify_tools.py', '--binary', r.root / 'candidate-http', out, r.root / 'http-complete-tool-proof'])
    r.run('http-complete-matching-proof', ['python3', tools / 'prove_comparison.py', out, r.root / 'http-complete-matching-proof', '--comparator', tools / 'compare.py'])
    original = json.loads((out / 'success-reopen0-candidate.json').read_text())
    mutants = r.root / 'http-complete-contact-proof'
    mutants.mkdir()
    for field in ('operation_id', 'capability', 'effect', 'protocol', 'endpoint'):
        report = copy.deepcopy(original)
        count = 0
        def mutate(value):
            nonlocal count
            if isinstance(value, dict):
                for child in list(value.values()):
                    mutate(child)
                if 'prepared' in value and 'request_wire' in value:
                    if field not in value:
                        raise ValueError('contact missing ' + field)
                    value[field] = 'deliberate-foreign-' + field
                    count += 1
            elif isinstance(value, list):
                for child in value:
                    mutate(child)
        mutate(report)
        if count == 0:
            raise ValueError('mutant touched no contacts')
        path = mutants / (field + '.json')
        write(path, report)
        result = r.run('http-complete-contact-' + field, ['python3', tools / 'compare.py', 'same-seed', path, path,
                       '--baseline-db', out / 'success-reopen0/baseline/store.sqlite',
                       '--candidate-db', out / 'success-reopen0/candidate/store.sqlite'], expected=1)
        if json.loads(result.read_text()).get('equal_for_stated_coverage') is not False:
            raise ValueError('apparatus error is not semantic rejection')


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--root', type=Path, required=True)
    a = p.parse_args()
    r = Recorder(a.root)
    tools = r.root / 'complete-http-tools'
    adapt(r.root, r.root / 'candidate/docs/evidence/performance-p2/differential', tools)
    write(r.root / 'complete-copy-amendment.json', {
        'why': 'Legacy clone_seed and SQL-mutant helpers omitted extant empty WAL/SHM; original runs remain reference only for copy custody.',
        'before_new_execution': True, 'new_direct_pairs': ['success/reopen0', 'success/reopen1', 'refusal3/reopen0', 'refusal3/reopen1'],
        'new_http_pairs': ['success8/reopen0', 'success8/reopen1'], 'negatives': 'same 38 +53 original checks and five explicit matching contacts',
        'production_binaries_unchanged': True, 'comparator_sha256': sha(tools / 'compare.py'),
        'wrapper_sha256': sha(__file__), 'patches': ['complete-http-tools/clone-seed.patch', 'complete-http-tools/mutant-copy.patch'],
        'fresh_format_label_correction': 'fresh9/11; historical4/9; genuine11->12; old unsupported13/14 (initial manifest fresh4 label was a typo)'})
    direct(r, tools)
    http(r, tools)


if __name__ == '__main__':
    main()
