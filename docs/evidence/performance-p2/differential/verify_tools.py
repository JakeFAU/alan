#!/usr/bin/env python3
"""Reject report/SQL mutants using already completed closed campaign copies."""
import argparse
import copy
import fcntl
import hashlib
import json
import pathlib
import shutil
import sqlite3
import subprocess
import sys

p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--binary', required=True, help='runner for pre-open interface refusals')
p.add_argument('campaign', help='completed campaign.py output; disposable copies only')
p.add_argument('output', help='new mutation output directory')
a = p.parse_args()
campaign = pathlib.Path(a.campaign).resolve(strict=True)
out = pathlib.Path(a.output).absolute()
out.mkdir(parents=True, exist_ok=False)
tools = pathlib.Path(__file__).resolve().parent
left = campaign/'success-reopen0-baseline.json'
right = campaign/'success-reopen0-candidate.json'
ldb = campaign/'success-reopen0'/'baseline'/'store.sqlite'
rdb = campaign/'success-reopen0'/'candidate'/'store.sqlite'
original = json.loads(right.read_text())
commands = []


def call(name, argv, expected):
    argv = [str(v) for v in argv]
    result = subprocess.run(argv, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    (out/(name+'.json')).write_bytes(result.stdout)
    (out/(name+'.stderr')).write_bytes(result.stderr)
    commands.append({'name': name, 'argv': argv, 'exit': result.returncode, 'expected_exit': expected})
    (out/'commands.json').write_text(json.dumps(commands, indent=2)+'\n')
    if result.returncode != expected:
        raise RuntimeError('unexpected exit: '+name)


def compare(name, lrep, rrep, ldb=ldb, rdb=rdb, expected=1):
    call(name, [sys.executable, tools/'compare.py', 'same-seed', lrep, rrep,
                '--baseline-db', ldb, '--candidate-db', rdb], expected)


def final(r):
    return r['cuts'][-1]


def returned(r):
    return next(c['run_report'] for c in r['cuts'] if c['label'] == 'after-run')


mutants = {
 'wrong-original': lambda r: r['contacts'][0]['prepared']['Request'].update(ID='foreign-operation'),
 'wrong-key': lambda r: r['contacts'][0].update(key='foreign-key'),
 'wrong-prepared-digest': lambda r: r['contacts'][0]['prepared'].update(Digest='0'*64),
 'wrong-result': lambda r: final(r)['snapshot']['Effects'][0]['AdapterResult'].update(Digest='0'*64),
 'wrong-response-identity': lambda r: r['server_contacts'][0].update(response_id='foreign-response'),
 'handler-adapter-mismatch': lambda r: r['server_contacts'][0].update(request_wire='{}'),
 'missing-contact': lambda r: r['server_contacts'].pop(),
 'duplicate-contact': lambda r: r['server_contacts'].append(copy.deepcopy(r['server_contacts'][0])),
 'runreport-root': lambda r: returned(r)['Roots'][0].update(RootID='foreign-root'),
 'runreport-charge': lambda r: returned(r)['Roots'][0]['Charges'].update(Effects=2),
 'runreport-pending': lambda r: returned(r).update(PendingDeliveries=['fabricated-pending']),
 'runreport-unknown': lambda r: returned(r).update(UnknownExternal=['fabricated-unknown']),
 'epoch': lambda r: final(r)['snapshot']['Runs'][0].update(Epoch=999999),
 'approval-predecessor': lambda r: final(r)['snapshot']['Continuations'][0]['Approval'].update(PredecessorID='foreign'),
 'selected-observation': lambda r: next(h for h in final(r)['history'] if h.get('SelectedObservationID')).update(SelectedObservationID='foreign'),
 'missing-runreport': lambda r: next(c for c in r['cuts'] if c['label']=='after-run').update(run_report=None),
 'missing-prefix': lambda r: r['cuts'].pop(2),
 'reordered-prefix': lambda r: r['cuts'].reverse(),
 'fabricated-matching-weather-value': lambda r: r.update(cuts=[{'sample': 'present'}], contacts=[], server_contacts=[]),
}
# Single-side identity/accounting mutations exercise complete equality; matching
# malformed journals/prefixes/value-only reports must also fail shape/provenance.
compare('positive-control', left, right, expected=0)
for name, mutate in mutants.items():
    changed = copy.deepcopy(original)
    mutate(changed)
    path = out/(name+'.input.json')
    path.write_text(json.dumps(changed)+'\n')
    compare('reject-'+name, left, path)
    if name in ('missing-runreport', 'missing-prefix', 'reordered-prefix', 'fabricated-matching-weather-value', 'handler-adapter-mismatch', 'missing-contact', 'duplicate-contact'):
        compare('reject-matching-'+name, path, path)

sql_mutants = {
 'scoped-grant': "UPDATE permissions SET revision=revision+1 WHERE source='series-1'",
 'debit': "UPDATE composition_charges SET descriptor=CAST(json_set(descriptor,'$.Effects',0) AS BLOB) WHERE operation_id='[\"series-1\",1,1,0]'",
 'epoch': "UPDATE runs SET epoch=epoch+1 WHERE subject='series-1'",
}
for name, sql in sql_mutants.items():
    db = out/(name+'.sqlite')
    shutil.copyfile(rdb, db)
    connection = sqlite3.connect(db)
    try:
        result = connection.execute(sql)
        if result.rowcount < 1:
            raise RuntimeError('mutant touched no record: '+name)
        connection.commit()
    finally:
        connection.close()
    compare('reject-sql-'+name, left, right, rdb=db)

seed = campaign/'seed'/'store.sqlite'
for name, flags in (('missing-endpoint', []), ('ephemeral-port', ['-endpoint', '127.0.0.1:0']),
                    ('nonloopback', ['-endpoint', '192.0.2.1:40001']),
                    ('failure-policy', ['-endpoint', '127.0.0.1:40001', '-scenario', 'refusal3'])):
    call('interface-reject-'+name, [a.binary, '-mode', 'execute-existing', '-db', seed, *flags], 2)

call('clone-reject-existing', [sys.executable, tools/'clone_seed.py', seed, ldb], 1)
call('clone-reject-executed', [sys.executable, tools/'clone_seed.py', ldb, out/'must-not-exist-executed.sqlite'], 1)
with open(str(seed)+'.lock', 'rb+') as lock:
    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    call('clone-reject-owner', [sys.executable, tools/'clone_seed.py', seed, out/'must-not-exist-owner.sqlite'], 1)
wal_seed = out/'wal-seed.sqlite'
shutil.copyfile(seed, wal_seed)
pathlib.Path(str(wal_seed)+'.lock').touch()
pathlib.Path(str(wal_seed)+'-wal').write_bytes(b'nonempty WAL rejected before SQLite read')
call('clone-reject-wal', [sys.executable, tools/'clone_seed.py', wal_seed, out/'must-not-exist-wal.sqlite'], 1)
(out/'summary.json').write_text(json.dumps({'ok': True, 'report_mutants': len(mutants), 'sql_mutants': len(sql_mutants), 'clone_guards': 4, 'interface_refusals': 4, 'commands': len(commands)}, indent=2)+'\n')
(out/'output-sha256.json').write_text(json.dumps({v.name: hashlib.sha256(v.read_bytes()).hexdigest() for v in sorted(out.iterdir()) if v.is_file() and v.name != 'output-sha256.json'}, indent=2)+'\n')
