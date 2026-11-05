#!/usr/bin/env python3
"""Retained complete-copy adaptation of tools/compare_stores.py's four cuts.

Same nested local fixture, inherited ACK protocol, no authority/series cross-product.
Readers and both resume branches use independent copies of the sealed stopped DB.
"""
import argparse
import importlib.util
import json
from pathlib import Path
import secrets
import signal
import subprocess
import sys

from common import Recorder, copy_db, inventory, sha, write
from run import compare


def load_original(tool):
    spec = importlib.util.spec_from_file_location('original', tool)
    original = importlib.util.module_from_spec(spec)
    sys.dont_write_bytecode = True  # Keep the frozen source export intact.
    spec.loader.exec_module(original)
    return original


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--root', type=Path, required=True)
    p.add_argument('--output-name', default='cross-build')
    a = p.parse_args()
    r = Recorder(a.root)
    root = r.root
    tool = root / 'candidate/tools/compare_stores.py'
    original = load_original(tool)
    if Path(a.output_name).name != a.output_name:
        p.error('--output-name must be one directory name')
    out = root / a.output_name
    gate = 'cross-' if a.output_name == 'cross-build' else a.output_name + '-'
    out.mkdir()
    write(out / 'controller.json', {'original_tool_sha256': sha(tool), 'wrapper_sha256': sha(__file__)})
    for writer, reader in [('baseline', 'candidate'), ('candidate', 'baseline')]:
        for boundary, phase in [('execution.staged', 'pre-commit'), ('execution.committed', 'post-commit')]:
            name = writer + '-' + phase
            case = out / name
            case.mkdir()
            db = case / 'writer.sqlite'
            writerbin = root / (writer + '-alan')
            r.run(gate + name + '-init', [writerbin, 'init', '--db', db, '--demo', 'nested'])
            statepath = r.run(gate + name + '-initial', [writerbin, 'inspect', '--db', db])
            tables = json.loads(statepath.read_text())['Tables']
            subject = next(row for row in tables['subjects'] if row['id'] == 'inner')
            run = next(row for row in tables['runs'] if row['subject'] == 'inner' and row['generation'] == subject['generation'])
            event = {'Scenario': 'correspondence', 'Token': secrets.token_hex(16), 'Boundary': boundary,
                     'Phase': phase, 'Store': str(db), 'Subject': 'inner', 'Generation': run['generation'],
                     'Epoch': run['epoch'] + 1, 'Sequence': 7,
                     'Intent': json.dumps(['inner', run['generation'], 7, 0], separators=(',', ':'))}
            argv = ['sh', '-c', 'exec 3>&1 4<&0; exec "$@"', 'barrier', str(writerbin), 'run', '--db', str(db)]
            env = dict(r.env, ALAN_BARRIER=json.dumps(event))
            write(case / 'command.json', {'argv': argv, 'environment': env, 'writer_binary_sha256': sha(writerbin), 'expected_signal': 'SIGKILL'})
            ack = None
            with (case / 'writer-stderr.log').open('xb') as err:
                with subprocess.Popen(argv, env=env, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=err) as child:
                    try:
                        ack = original.read_acknowledgement(child.stdout)
                        write(case / 'ack.json', ack)
                        for key, value in dict(event, PID=child.pid).items():
                            if ack.get(key) != value:
                                raise ValueError('unexpected acknowledged ' + key)
                    finally:
                        child.kill()
                        status = child.wait(timeout=30)
                        write(case / 'reap.json', {'pid': child.pid, 'exit': status, 'expected': -signal.SIGKILL})
                    if status != -signal.SIGKILL:
                        raise ValueError('child not reaped after SIGKILL')
            # No reader has touched this stopped history. Archive all extant files.
            sealed = case / 'sealed/store.sqlite'
            copy_db(db, sealed, case / 'seal-copy.json')
            before = inventory(sealed.parent)
            branches = []
            for owner in (writer, reader):
                target = case / owner / 'store.sqlite'
                copy_db(sealed, target, case / (owner + '-copy.json'))
                report = r.run(gate + name + '-' + owner + '-resume', [root / (owner + '-alan'), 'run', '--db', target])
                branches.append((target, report))
            if branches[0][1].read_bytes() != branches[1][1].read_bytes():
                raise ValueError('opposite-build resume RunReport differs')
            result = json.loads(branches[0][1].read_text())
            if not result['Quiescent'] or result['OpenRuns']:
                raise ValueError('nested run did not complete')
            reports = []
            for owner, (target, _) in zip((writer, reader), branches):
                reports.append(r.run(gate + name + '-' + owner + '-read', [root / (owner + '-direct'), '-mode', 'generic-reader', '-db', target, '-source-label', r.plan['sources'][owner]]))
            compare(r, gate + name + '-exact', reports, [b[0] for b in branches])
            if before != inventory(sealed.parent):
                raise ValueError('sealed stopped original changed')
            write(case / 'result.json', {'ok': True, 'writer': writer, 'reader': reader, 'boundary': boundary,
                                         'same_namespace_exact_resume_and_read': True, 'sealed_unchanged': True})
            print(name + ': exact opposite-build resume passed', flush=True)


if __name__ == '__main__':
    main()
