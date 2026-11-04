#!/usr/bin/env python3
"""Serial same-seed HTTP qualification. No builds, retries or endpoint replacement."""
import argparse
import hashlib
import json
import os
import pathlib
import socket
import subprocess
import sys
import time

p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--baseline-binary', required=True)
p.add_argument('--candidate-binary', required=True)
p.add_argument('--baseline-source', required=True)
p.add_argument('--candidate-source', required=True)
p.add_argument('--output', required=True)
p.add_argument('--bound', type=int, default=8, help='finite series bound; default 8 preserves the primary schedules')
a = p.parse_args()
if not 1 <= a.bound <= 256:
    p.error('--bound must be between 1 and 256')
tools = pathlib.Path(__file__).resolve().parent
out = pathlib.Path(a.output).absolute()
out.mkdir(parents=True, exist_ok=False)
base, candidate = (pathlib.Path(v).resolve(strict=True) for v in (a.baseline_binary, a.candidate_binary))
env = os.environ.copy()
env['GOMAXPROCS'] = '4'
commands = []


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(name, obj):
    (out/name).write_text(json.dumps(obj, indent=2, sort_keys=True)+'\n')


def call(name, command):
    command = [str(v) for v in command]
    start = time.time()
    with (out/(name+'.json')).open('wb') as stdout, (out/(name+'.stderr')).open('wb') as stderr:
        result = subprocess.run(command, env=env, stdout=stdout, stderr=stderr)
    commands.append({'name': name, 'argv': command, 'exit': result.returncode,
                     'started_unix': start, 'elapsed_seconds': time.time()-start})
    save('commands.json', commands)
    print(name+': exit='+str(result.returncode), flush=True)
    if result.returncode:
        raise RuntimeError('stopped at '+name+'; failure retained; no replacement')
    return out/(name+'.json')


# Choose once, release once, pass verbatim to every serial invocation. A bind
# race fails that invocation and the campaign; there is no port retry loop.
with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as reservation:
    reservation.bind(('127.0.0.1', 0))
    endpoint = '127.0.0.1:'+str(reservation.getsockname()[1])
save('inputs.json', {'version': 'alan/performance-p2-http-differential-campaign/v1',
                    'endpoint': endpoint, 'bound': a.bound, 'baseline_source': a.baseline_source,
                    'candidate_source': a.candidate_source, 'source_labels_are_caller_supplied': True,
                    'binary_sha256': {'baseline': sha(base), 'candidate': sha(candidate)},
                    'tool_sha256': {v.name: sha(v) for v in sorted(tools.iterdir()) if v.is_file()},
                    'python': sys.version, 'gomaxprocs': 4})
try:
    seed = out/'seed'/'store.sqlite'
    seed.parent.mkdir()
    call('seed', [base, '-mode', 'create-seed', '-bound', a.bound, '-db', seed, '-source-label', a.baseline_source])
    for interval in (0, 1):
        name = 'success-reopen'+str(interval)
        left, right = (out/name/side/'store.sqlite' for side in ('baseline', 'candidate'))
        call(name+'-clone', [sys.executable, tools/'clone_seed.py', seed, left, right])
        reports = []
        for side, binary, db, source in (('baseline', base, left, a.baseline_source), ('candidate', candidate, right, a.candidate_source)):
            reports.append(call(name+'-'+side, [binary, '-mode', 'execute-existing', '-bound', a.bound,
                                '-db', db, '-source-label', source, '-endpoint', endpoint,
                                '-scenario', 'success', '-reopen-every', interval]))
        call(name+'-comparison', [sys.executable, tools/'compare.py', 'same-seed', *reports,
                                 '--baseline-db', left, '--candidate-db', right])
    save('summary.json', {'ok': True, 'same_seed_http_pairs': 2, 'commands': len(commands),
                         'endpoint': endpoint, 'bound': a.bound, 'scope': 'success'+str(a.bound)+' ordinary and reopen after every selected position including final; semantic only'})
except Exception as error:
    save('summary.json', {'ok': False, 'error': str(error), 'commands': len(commands), 'endpoint': endpoint, 'bound': a.bound})
    raise
finally:
    save('output-sha256.json', {str(v.relative_to(out)): sha(v) for v in sorted(out.rglob('*'))
                              if v.is_file() and v.name != 'output-sha256.json'})
