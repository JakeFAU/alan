#!/usr/bin/env python3
"""Serial strict P3 campaign. Refuse existing output; stop at the first failure."""
import datetime
import json
import math
import os
from pathlib import Path
import sys
import time
from custody import (bound_row, command, expected_custody, fingerprint, sha, snapshot,
                     validate_plan, write)
from validation import require, validate_report, reported_configuration


def validate_meta(meta, report, plan, row, binding_hash, report_hash, progress_hash):
    require(meta.get('exit') == 0 and not meta.get('validation_error'), 'failed child or post-exit validation')
    require(meta.get('binding_sha256') == binding_hash and meta.get('row') == row, 'binding/row changed')
    require(meta.get('report_sha256') == report_hash and meta.get('progress_sha256') == progress_hash, 'raw bytes replaced')
    custody = fingerprint(expected_custody(plan))
    require(meta.get('custody_before') == custody == meta.get('custody_after'), 'source/binary/tool custody changed')
    require(meta.get('command') == command(plan, row) and meta.get('cwd') == plan['cwd'], 'argv/cwd drift')
    require(meta.get('environment') == plan['environment'], 'environment drift')
    require(meta.get('reported_configuration') == reported_configuration(report), 'configuration drift')
    require(report.get('store') == str(Path(plan['output']) / (row['name'] + '.store') / 'store.sqlite'), 'wrong store')
    require(meta.get('process_peak_rss_unit') == 'bytes' and type(meta.get('process_peak_rss')) is int and meta['process_peak_rss'] > 0, 'RSS missing')
    for key in ('seconds', 'user_cpu_seconds', 'system_cpu_seconds'):
        require(type(meta.get(key)) in (int, float) and math.isfinite(meta[key]) and meta[key] >= 0, 'process measurement missing')
    for key in ('load_before', 'load_after'):
        require(isinstance(meta.get(key), list) and len(meta[key]) == 3 and all(type(v) in (int,float) and math.isfinite(v) and v >= 0 for v in meta[key]), 'host load missing')
    require(datetime.datetime.fromisoformat(meta['ended_utc']) >= datetime.datetime.fromisoformat(meta['started_utc']), 'timestamps missing/inverted')


def validate_records(meta, start, outcome):
    start_keys = {'binding_sha256','row','command','cwd','environment','custody_before','started_utc','load_before'}
    outcome_keys = start_keys | {'exit','seconds','user_cpu_seconds','system_cpu_seconds','process_peak_rss',
                                 'process_peak_rss_unit','ended_utc'}
    require(set(start) == start_keys and set(outcome) == outcome_keys, 'start/child outcome missing or replaced')
    require(all(outcome.get(k) == v for k,v in start.items()) and
            all(meta.get(k) == v for k,v in outcome.items()), 'child outcome replaced')


def run_one(plan, row, binding_hash, custody_reader=snapshot):
    """Also used by explicitly labelled tooling fixtures; no diagnostic acceptance."""
    output = Path(plan['output'])
    name = row['name']
    report, progress, meta = [output / (name + suffix) for suffix in ('.json', '.progress', '.meta.json')]
    outcome = output / (name + '.outcome.json')
    store = output / (name + '.store')
    require(not any(p.exists() for p in (report, progress, meta, outcome, store)), 'refuse overwritten sample')
    before = custody_reader(plan)
    require(before == expected_custody(plan), 'custody differs before child')
    argv = command(plan, row)
    record = {'binding_sha256': binding_hash, 'row':row, 'command':argv,
              'cwd':plan['cwd'], 'environment':plan['environment'], 'custody_before':fingerprint(before),
              'started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(), 'load_before':list(os.getloadavg())}
    write(output / (name + '.start.json'), record)
    with report.open('xb') as out, progress.open('xb') as err:
        start = time.monotonic()
        pid = os.fork()
        if pid == 0:
            try:
                os.chdir(plan['cwd'])
                os.dup2(out.fileno(), 1)
                os.dup2(err.fileno(), 2)
                os.execve(argv[0], argv, plan['environment'])
            except BaseException as error:
                os.write(2, ('exec failure: ' + repr(error) + '\n').encode())
                os._exit(127)
        _, status, usage = os.wait4(pid, 0)
    record.update(exit=os.waitstatus_to_exitcode(status), seconds=time.monotonic()-start,
                  user_cpu_seconds=usage.ru_utime, system_cpu_seconds=usage.ru_stime,
                  process_peak_rss=usage.ru_maxrss, process_peak_rss_unit='bytes' if sys.platform == 'darwin' else 'KiB',
                  ended_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
    # This write precedes every post-exit hash, source read, JSON parse and load probe.
    write(outcome, record)
    try:
        record['load_after'] = list(os.getloadavg())
        record['report_sha256'], record['progress_sha256'] = sha(report), sha(progress)
        after = custody_reader(plan)
        record['custody_after'] = fingerprint(after)
        require(after == before, 'custody changed during child')
        result = json.loads(report.read_text())
        record['reported_configuration'] = reported_configuration(result)
        validate_report(result, row)
        validate_meta(record, result, plan, row, binding_hash, sha(report), sha(progress))
    except (OSError, ValueError, KeyError, TypeError) as error:
        record['validation_error'] = repr(error)
    write(meta, record)
    require(record['exit'] == 0 and not record.get('validation_error'), 'sample failed: ' + name + ' ' + record.get('validation_error', ''))
    return record


def main():
    binding = Path(sys.argv[1]).resolve()
    # Caller supplies the independently reviewed digest, not an implicit current-file approval.
    require(len(sys.argv) == 3 and sha(binding) == sys.argv[2], 'reviewed binding SHA256 required')
    plan = json.loads(binding.read_text())
    validate_plan(plan)
    Path(plan['output']).mkdir(parents=True, exist_ok=False)
    for design_row in plan['design']['rows']:
        require(sha(binding) == sys.argv[2], 'binding changed during campaign')
        row = bound_row(plan, design_row)
        print('START ' + row['name'], flush=True)
        run_one(plan, row, sys.argv[2])
        print('PASS ' + row['name'], flush=True)


if __name__ == '__main__':
    main()
