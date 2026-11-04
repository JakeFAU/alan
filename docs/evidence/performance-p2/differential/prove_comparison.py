#!/usr/bin/env python3
"""Matching finite-output mutants; --comparator also targets preserved initial source for actual RED."""
import argparse
import copy
import hashlib
import json
import pathlib
import subprocess
import sys

from finite_validation import go_json


def walk(value, change):
    if isinstance(value, dict):
        for child in list(value.values()):
            walk(child, change)
        change(value)
    elif isinstance(value, list):
        for child in value:
            walk(child, change)


def work(r, field, value):
    for c in r['cuts']:
        if c['run_report'] is not None:
            c['run_report'][field] = copy.deepcopy(value)


def wires(r, response=False):
    def change(v):
        field = 'response' if response else 'request'
        if field+'_wire' in v:
            v[field+'_wire'] = ' '+v[field+'_wire']
            v[field+'_sha256'] = hashlib.sha256(v[field+'_wire'].encode()).hexdigest()
        if response and 'ResponseID' in v and 'Response' in v:
            v['Response'] = ' '+v['Response']
    walk(r, change)


def prepared_mutation(r, mutate, forged=False):
    def change(v):
        if 'Request' in v and 'Digest' in v and 'Endpoint' in v:
            mutate(v)
            v['Digest'] = ''
            v['Digest'] = '0'*64 if forged else hashlib.sha256(go_json(v).encode()).hexdigest()
        if 'prepared' in v and 'request_wire' in v:
            v['prepared_digest'] = v['prepared']['Digest']
            v['request_wire'] = go_json(v['prepared'])
            v['request_sha256'] = hashlib.sha256(v['request_wire'].encode()).hexdigest()
    walk(r, change)


def final(r):
    return r['cuts'][-1]


def matching_cases():
    cases = {
        'matching-steps': lambda r: work(r, 'Steps', 999),
        'matching-dispatches': lambda r: work(r, 'Dispatches', 999),
        'matching-open-runs': lambda r: work(r, 'OpenRuns', []),
        'matching-steps-bool': lambda r: work(r, 'Steps', True),
        'matching-dispatches-float': lambda r: work(r, 'Dispatches', 1.0),
        'matching-open-run-generation-bool': lambda r: next(c['run_report'] for c in r['cuts'] if c['label']=='after-run')['OpenRuns'][0].update(Generation=True),
        'matching-unknown-null': lambda r: work(r, 'UnknownExternal', None),
        'matching-reopen-work': lambda r: next(c['run_report'] for c in r['cuts'] if c['label']=='after-interval-reopen').update(Steps=3, Dispatches=1),
        'matching-history-truncated': lambda r: final(r)['history'].pop(),
        'matching-history-truncated-count-adjusted': lambda r: (final(r)['history'].pop(), final(r)['replay'].update(HistoryCount=len(final(r)['history']))),
        'matching-history-reordered': lambda r: final(r)['history'].reverse(),
        'matching-replay-history-count': lambda r: final(r)['replay'].update(HistoryCount=999),
        'matching-replay-history-count-float': lambda r: final(r)['replay'].update(HistoryCount=float(final(r)['replay']['HistoryCount'])),
        'matching-replay-namespace': lambda r: final(r)['replay'].update(Namespace='0'*64),
        'matching-replay-missing-run': lambda r: final(r)['replay']['Runs'].pop(),
        'matching-replay-run-generation': lambda r: final(r)['replay']['Runs'][0]['ID'].update(Generation=2),
        'matching-series-count': lambda r: final(r)['series_view'].update(Spent=0),
        'matching-selected-history': lambda r: next(h for h in final(r)['history'] if h['SelectedObservationID']).update(SelectedObservationID='foreign-observation'),
        'matching-approval-predecessor': lambda r: final(r)['snapshot']['Continuations'][0]['Approval'].update(PredecessorID='foreign'),
        'matching-approval-selection': lambda r: final(r)['snapshot']['Continuations'][0]['SelectedAt'].update(Sequence=999),
        'matching-noncanonical-request': lambda r: wires(r),
        'matching-noncanonical-response': lambda r: wires(r, True),
        'matching-forged-prepared-digest': lambda r: prepared_mutation(r, lambda p: None, True),
        'matching-authentic-wrong-action': lambda r: prepared_mutation(r, lambda p: p.update(Wire='{}')),
        'matching-authentic-wrong-payload': lambda r: prepared_mutation(r, lambda p: p['Request']['Payload']['station'].update(Text='south')),
        'matching-authentic-wrong-request-key': lambda r: prepared_mutation(r, lambda p: p['Request'].update(Key='foreign-key')),
        'matching-root-charge': lambda r: final(r)['snapshot']['Roots'][0]['Charges'].update(Effects=2),
        'matching-root-outcome': lambda r: final(r)['snapshot']['Roots'][0]['Outcome'].update(Caller={'Subject':'foreign','Generation':1}),
    }
    # Every frozen configuration field gets an identically malformed pair.
    # Full exact scalar typing is checked, including booleans masquerading as 0/1.
    config = {
        'http_version':'HTTP/2', 'dial_timeout_ns':0, 'tcp_keepalive_ns':0, 'idle_timeout_ns':0,
        'response_header_timeout_ns':0, 'server_read_timeout_ns':0, 'server_write_timeout_ns':0,
        'max_conns_per_host':2, 'max_header_bytes':1, 'expect_continue_timeout_ns':0,
        'proxy':True, 'request_format':'foreign', 'response_format':'foreign',
        'max_request_bytes':1, 'max_response_bytes':1, 'timeout_ns':0,
        'reuse_connections':False, 'redirects':True, 'provider_retries':1,
        'post_retries':True, 'resource_scope':'foreign', 'listen_address':'127.0.0.1:0',
    }
    for field, value in config.items():
        cases['matching-policy-'+field] = lambda r, f=field, v=value: r['transport'].update({f:v})
    cases['matching-policy-scalar-type'] = lambda r: r['transport'].update(max_conns_per_host=True)
    return cases


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('campaign', help='completed disposable-copy campaign, never original archives')
    p.add_argument('output', help='new output directory')
    p.add_argument('--comparator', default=str(pathlib.Path(__file__).with_name('compare.py')),
                   help='selected comparator file; preserved initial comparator gives real old RED')
    a = p.parse_args()
    campaign, out, comparator = pathlib.Path(a.campaign).resolve(strict=True), pathlib.Path(a.output).absolute(), pathlib.Path(a.comparator).resolve(strict=True)
    out.mkdir(parents=True, exist_ok=False)
    records = []
    def call(name, value, schedule, expected):
        inp = out/(name+'.input.json')
        inp.write_text(json.dumps(value)+'\n')
        db = campaign/('success-reopen'+str(schedule))/'candidate'/'store.sqlite'
        argv = [sys.executable, str(comparator), 'same-seed', str(inp), str(inp), '--baseline-db', str(db), '--candidate-db', str(db)]
        result = subprocess.run(argv, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        (out/(name+'.stdout')).write_bytes(result.stdout)
        (out/(name+'.stderr')).write_bytes(result.stderr)
        parsed = json.loads(result.stdout)
        # A setup/import/traceback exit is not proof of semantic rejection.
        valid_reject = (result.returncode == 1 and parsed.get('equal_for_stated_coverage') is False
                        and isinstance(parsed.get('error'), str) and bool(parsed['error']))
        passed = (result.returncode == 0 and parsed.get('equal_for_stated_coverage') is True) if expected == 0 else valid_reject
        records.append({'name':name, 'argv':argv, 'expected_exit':expected, 'exit':result.returncode,
                        'expected_outcome_observed':passed, 'rejection_reason':parsed.get('error'),
                        'input_sha256':hashlib.sha256(inp.read_bytes()).hexdigest()})
        (out/'commands.json').write_text(json.dumps(records,indent=2)+'\n')
    inputs = {}
    originals = {}
    for schedule in (0, 1):
        path = campaign/('success-reopen'+str(schedule)+'-candidate.json')
        originals[schedule] = json.loads(path.read_text())
        inputs[str(path)] = hashlib.sha256(path.read_bytes()).hexdigest()
        call('positive-reopen'+str(schedule), originals[schedule], schedule, 0)
    for name, mutate in matching_cases().items():
        schedule = int(name == 'matching-reopen-work')
        changed = copy.deepcopy(originals[schedule])
        mutate(changed)
        call(name, changed, schedule, 1)
    failed = [v['name'] for v in records if not v['expected_outcome_observed']]
    summary = {'ok':not failed, 'unexpected_outcomes':failed, 'commands':len(records),
               'comparator_sha256':hashlib.sha256(comparator.read_bytes()).hexdigest(),
               'proof_sha256':hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest(), 'inputs':inputs}
    (out/'summary.json').write_text(json.dumps(summary,indent=2)+'\n')
    (out/'output-sha256.json').write_text(json.dumps({v.name:hashlib.sha256(v.read_bytes()).hexdigest() for v in sorted(out.iterdir()) if v.is_file() and v.name!='output-sha256.json'},indent=2)+'\n')
    print(json.dumps(summary,indent=2))
    return int(bool(failed))


if __name__ == '__main__':
    sys.exit(main())
