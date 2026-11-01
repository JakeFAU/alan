"""Independent Phase 2B outgoing and observable-history checker, Python standard library."""
import argparse
from collections import Counter, defaultdict
import copy
import gzip
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys
import tempfile


PHASE = Path(__file__).resolve().parents[1]
ROOT = PHASE.parents[2]
GRAPH = PHASE / 'evidence/model/graphs.jsonl.gz'
LIVE = {'admitted', 'entered', 'known', 'empty', 'recorded'}
ELIGIBLE_SCOPES = {'open', 'success-unpublished'}


class Rejection(ValueError):
    pass


def require(test, message):
    if not test:
        raise Rejection(message)


def canonical(obj):
    return json.dumps(obj, separators=(',', ':'), sort_keys=True)


def profiles():
    """Declared experiments, independent of both the Go package and the supplied graph."""
    result = {}

    def add(name, route, member='none', closure='none', target='source',
            reply='known', restart='', duplicate=False, peer=False):
        result[name] = dict(Name=name, Route=route, Member=member, Closure=closure,
                            Target=target, Reply=reply, Restart=restart, Duplicate=duplicate, Peer=peer)

    for route in ('adapter', 'raw-keyed'):
        add(route+'-cancel', route, closure='cancel', peer=True)
        add(route+'-root', route, 'root', 'root-close', 'scope')
    add('raw-abandon', 'raw-plain', closure='abandon')
    add('raw-retire', 'raw-keyed', closure='retire')
    add('raw-retry', 'raw-keyed', closure='retry-after-failure')
    add('raw-end', 'raw-keyed', 'root', 'end')
    add('adapter-failure', 'adapter', 'nested', 'failure')
    add('alias-source-ended-scope-open', 'adapter', 'root', 'member-end', 'scope')
    for case in ('nested-fail', 'nested-retire', 'nested-publish'):
        add(case, 'adapter', 'nested', case, 'scope')
    add('two-prepared-direct', 'raw-keyed', closure='cancel', duplicate=True)
    add('owner-close-open', 'raw-keyed', restart='close')
    add('no-receipt', 'adapter', 'root', 'cancel', reply='loss')
    add('account-error', 'adapter', 'root', 'root-close', 'scope', 'account-error')
    for route in ('raw-keyed', 'raw-plain', 'adapter'):
        add(route+'-recover-open', route, restart='kill')
    add('raw-keyed-recover-revoked', 'raw-keyed', closure='cancel', restart='kill')
    return result


def initial(p):
    return dict(Local=dict(Source='open', Scope='open', Record='missing', Generation=1,
                           Retired=False, Selected=False, Peer=False, Key='N/O',
                           Request='O@epoch1/delta1/'+p['Route'], Grant='allowed@revision1/epoch1',
                           Reservations=1, Debit=int(p['Member'] != 'none')),
                Owner='open', Epoch=1, Worker=['ready', 'ready' if p['Duplicate'] else 'unused'],
                Sticky=False, Ack=False, Wire=['none', 'none'], Applications=0)


def refusal(local):
    if local['Generation'] != 1:
        return 'source_generation_changed'
    if local['Retired']:
        return 'source_retired'
    if local['Source'] != 'open':
        return 'source_closed'
    if local['Scope'] not in ELIGIBLE_SCOPES:
        return 'scope_closed'
    return ''


def wait(p, s):
    if s['Owner'] != 'open':
        return 'owner-unavailable'
    l = s['Local']
    revoked = refusal(l) if p['Target'] == 'source' else l['Scope'] not in ELIGIBLE_SCOPES
    if not revoked:
        return 'authority-open'
    if s['Sticky']:
        return 'accounting-error'
    if LIVE.intersection(s['Worker']):
        return 'pending-worker'
    if l['Record'] == 'recorded':
        return 'needs-settlement'
    return 'quiescent'


def local_transitions(p, l):
    """Changes induced by a genuine change to the original source or its scope."""
    closure = p['Closure']
    if l['Source'] == 'open' and l['Scope'] == 'open':
        simple = {
            'cancel': ('cancel', {'Source': 'cancelled'}),
            'abandon': ('abandon', {'Source': 'abandoned'}),
            'retire': ('retire', {'Source': 'cancelled', 'Retired': True}),
            'retry-after-failure': ('failure', {'Source': 'failed'}),
            'failure': ('failure', {'Source': 'failed', 'Scope': 'nested-fail'}),
            'end': ('end', {'Source': 'succeeded', 'Scope': 'root-close'}),
            'member-end': ('end', {'Source': 'succeeded'}),
            'root-close': ('root-close', {'Scope': 'root-close'}),
            'nested-fail': ('nested-fail', {'Scope': 'nested-fail'}),
            'nested-retire': ('freeze-success', {'Scope': 'success-unpublished'}),
            'nested-publish': ('freeze-success', {'Scope': 'success-unpublished'}),
        }
        if closure in simple:
            action, fields = simple[closure]
            if closure == 'cancel' and p['Member'] == 'root':
                fields['Scope'] = 'root-close'
            yield 'operator.'+action, fields
    if closure == 'retry-after-failure' and l['Source'] == 'failed' and l['Generation'] == 1:
        yield 'operator.retry', {'Generation': 2}
    if l['Scope'] == 'success-unpublished':
        if closure == 'nested-retire':
            yield 'operator.nested-retire', {'Scope': 'nested-retire'}
        if closure == 'nested-publish' and l['Selected']:
            yield 'operator.nested-publish', {'Scope': 'nested-publish', 'Source': 'succeeded'}


def successors(p, s):
    """Return every enabled action with its complete poststate; unmentioned fields persist."""
    out = {}
    l, owner, slots = s['Local'], s['Owner'], s['Worker']

    def emit(action, *, local=None, worker=None, wire=None, **changes):
        t = copy.deepcopy(s)
        if local:
            t['Local'].update(local)
        if worker:
            t['Worker'][worker[0]] = worker[1]
        if wire:
            t['Wire'][wire[0]] = wire[1]
        t.update(changes)
        require(action not in out, f'duplicate oracle rule {action}')
        out[action] = t

    if owner == 'open':
        for action, fields in local_transitions(p, l):
            emit(action, local=fields)
        if p['Peer'] and not l['Peer']:
            emit('peer.commit', local={'Peer': True})
        nonawait = {'end', 'member-end', 'failure', 'retry-after-failure'}
        if (l['Record'] == 'settled' and l['Source'] == 'open' and not l['Selected']
                and p['Closure'] not in nonawait):
            emit('source.select', local={'Selected': True})
        if l['Record'] == 'recorded' and not LIVE.intersection(slots) and not s['Sticky']:
            emit('recovery.settle', local={'Record': 'settled'})
        if not s['Ack'] and wait(p, s) == 'quiescent':
            emit('observe.Q', Ack=True)
    for i, stage in enumerate(slots):
        packet = s['Wire'][i]

        def opportunity(action, new):
            emit(f'{action}.{i}', worker=(i, new))

        if stage == 'ready':
            reason = 'owner_closing' if owner == 'closing' else refusal(l)
            if reason:
                opportunity('gate.refuse-'+reason, 'refused')
            else:
                opportunity('gate.admit', 'admitted')
        if stage == 'admitted':
            opportunity('worker.enter', 'entered')
            if owner == 'closing':
                opportunity('worker.withdraw', 'empty')
        if stage == 'entered':
            if packet == 'none':
                emit(f'worker.send.{i}', wire=(i, 'sent'))
            if p['Reply'] == 'loss':
                opportunity('worker.loss', 'empty')
            elif packet == 'served':
                opportunity('worker.return', 'known')
        if packet == 'sent':
            emit(f'endpoint.ingress.{i}', wire=(i, 'accepted'))
        if packet == 'accepted':
            apps = s['Applications'] + 1 if p['Route'] == 'raw-plain' else 1
            emit(f'endpoint.serve.{i}', wire=(i, 'served'), Applications=apps)
        if stage == 'known':
            record = 'settled' if l['Record'] == 'settled' else 'recorded'
            emit(f'account.record.{i}', worker=(i, 'recorded'), local={'Record': record})
        if stage == 'recorded':
            if p['Reply'] == 'account-error':
                emit(f'account.error.{i}', worker=(i, 'handled'), Sticky=True)
            else:
                emit(f'account.settle.{i}', worker=(i, 'handled'), local={'Record': 'settled'})
        if stage == 'empty':
            opportunity('account.unknown', 'handled')

    if s['Epoch'] == 1 and owner == 'open':
        if p['Restart'] == 'close':
            emit('owner.begin-close', Owner='closing')
        cut = slots[0] in {'admitted', 'known', 'recorded'} or (
            slots[0] == 'entered' and s['Wire'][0] == 'accepted')
        if p['Restart'] == 'kill' and cut:
            emit('owner.kill', Owner='dead', Worker=['unused', 'unused'], Sticky=False)
    if owner == 'closing' and not (LIVE | {'ready'}).intersection(slots):
        emit('owner.release', Owner='released', Worker=['unused', 'unused'], Sticky=False)
    if owner in {'released', 'dead'}:
        prepare = l['Record'] == 'missing' and p['Route'] == 'raw-keyed'
        emit('owner.open', Owner='open', Epoch=2, Worker=['unused', 'ready' if prepare else 'unused'])
    return out


def actor(action):
    group = action.split('.')[0]
    if group == 'operator':
        return 'closure driver'
    if group == 'observe':
        return 'Q caller'
    if group == 'owner':
        return 'owner/crash driver'
    if group == 'endpoint':
        return 'independent endpoint'
    if group in {'worker', 'gate', 'account', 'recovery', 'peer', 'source'}:
        return 'local execution'
    raise Rejection(f'unowned action {action}')


def check_graph(g, p):
    name = p['Name']
    require(canonical(g['Profile']) == canonical(p), f'{name}: wrong profile parameters')
    rows = g['Rows']
    require(rows and canonical(rows[0]['State']) == canonical(initial(p)), f'{name}: wrong initial state')
    state_ids = {}
    incoming = defaultdict(set)
    ownership = Counter()
    edges = 0
    for i, row in enumerate(rows):
        where = f'{name} row {i}'
        require(type(row['ID']) is int and row['ID'] == i, f'{where}: nonconsecutive ID')
        s = row['State']
        state = canonical(s)
        require(state not in state_ids, f'{where}: duplicate state formerly {state_ids.get(state)}')
        state_ids[state] = i
        expected_wait = wait(p, s)
        require(expected_wait == row['Wait'], f'{where}: expected Wait {expected_wait}, got {row["Wait"]}')
        es = row['Outgoing']
        require(isinstance(es, list), f'{where}: malformed outgoing list')
        actual = {}
        for edge in es:
            action, target = edge['Action'], edge['To']
            require(type(target) is int and 0 <= target < len(rows), f'{where}: invalid target')
            require(action not in actual, f'{where}: repeated label {action}')
            actual[action] = target
        expected = successors(p, s)
        require(expected.keys() == actual.keys(),
                f'{where}: missing {sorted(expected.keys()-actual.keys())}, extra {sorted(actual.keys()-expected.keys())}')
        for action, post in expected.items():
            j = actual[action]
            other = rows[j]['State']
            changes = {k: (v, other.get(k)) for k, v in post.items()
                       if canonical(v) != canonical(other.get(k))}
            require(canonical(other) == canonical(post), f'{where} {action} -> {j}: wrong poststate {changes}')
            ownership[actor(action)] += 1
            incoming[i].add(j)
            edges += 1
    seen = set()
    pending = [0]
    while pending:
        i = pending.pop()
        if i not in seen:
            seen.add(i)
            pending.extend(incoming[i]-seen)
    require(len(seen) == len(rows), f'{name}: unreachable extra rows')
    return len(rows), edges, sum(not r['Outgoing'] for r in rows), ownership


def follow(g, actions):
    row = g['Rows'][0]
    for action in actions:
        options = {e['Action']: e['To'] for e in row['Outgoing']}
        require(action in options, f"{g['Profile']['Name']} row {row['ID']}: trace disabled at {action}")
        row = g['Rows'][options[action]]
    return row


def reachable_sinks(g, row):
    """All suffixes terminate iff this finite reachable suffix is acyclic."""
    todo, active, visited, sinks = [(row['ID'], False)], set(), set(), []
    while todo:
        i, leaving = todo.pop()
        if leaving:
            active.remove(i)
            visited.add(i)
        elif i not in visited:
            require(i not in active, 'cycle admits infinitely postponed service')
            active.add(i)
            todo.append((i, True))
            es = g['Rows'][i]['Outgoing']
            if not es:
                sinks.append(g['Rows'][i]['State'])
            todo.extend((e['To'], False) for e in es)
    return sinks


SEND = ['worker.enter.0', 'worker.send.0', 'endpoint.ingress.0']
RESPONSE = [*SEND, 'endpoint.serve.0', 'worker.return.0']


def recovery_pairs(graphs):
    for name, g in graphs.items():
        p = g['Profile']
        if p['Restart'] != 'kill':
            continue
        closed = p['Closure'] == 'cancel'
        admit = ['gate.admit.0'] + (['operator.cancel'] if closed else [])
        local_views = []
        for apps, suffix in enumerate(([], RESPONSE)):
            row = follow(g, [*admit, *suffix, 'owner.kill', 'owner.open'])
            s = row['State']
            require(s['Applications'] == apps, 'cut lost world distinction')
            local_views.append({k: v for k, v in s.items() if k not in {'Applications', 'Wire'}})
            for end in reachable_sinks(g, row):
                useful = not closed and p['Route'] == 'raw-keyed'
                require(end['Applications'] == (1 if useful else apps), 'bad recovered application count')
                require(end['Local']['Selected'] == useful, 'useful service/withholding violated')
                require(end['Local']['Record'] == ('settled' if useful else 'missing'), 'invented/lost evidence')
                if closed:
                    require(end['Ack'] and end['Worker'][1] == 'refused', 'closed recovery failed to drain/refuse')
        require(local_views[0] == local_views[1], 'unequal U0/U1 local projections')
        print(f'{name}: equal local U0/U1, world applications 0/1; both maximal suffixes checked')
    g = graphs['raw-keyed-recover-revoked']
    row = follow(g, ['gate.admit.0', *SEND, 'operator.cancel', 'owner.kill', 'owner.open',
                     'observe.Q', 'endpoint.serve.0'])
    require(row['State']['Applications'] == 1 and row['State']['Ack'], 'delayed packet lost')
    return True


def read_graph(path):
    print(f'graph {path} sha256={hashlib.sha256(path.read_bytes()).hexdigest()}')
    graphs = {}
    with gzip.open(path, 'rt') as stream:
        for line in stream:
            g = json.loads(line)
            name = g['Profile']['Name']
            require(name not in graphs, 'duplicate profile '+name)
            graphs[name] = g
    return graphs


def check_export(path):
    graphs = read_graph(path)
    for g in graphs.values():
        source_scope_sanity(g)
    manifest = profiles()
    totals, owners = [0, 0, 0], Counter()
    # Check shared profiles even when a rejected version also altered the manifest.
    for name, p in manifest.items():
        if name in graphs:
            states, edges, sinks, counts = check_graph(graphs[name], p)
            print(f'{name}: {states} states, {edges} edges, {sinks} sinks')
            totals = [u+v for u, v in zip(totals, (states, edges, sinks))]
            owners.update(counts)
    require(graphs.keys() == manifest.keys(), f'profile mismatch {set(graphs)^set(manifest)}')
    print(f'total {len(graphs)} profiles, {totals[0]} states, {totals[1]} edges, {totals[2]} sinks; '
          f'action ownership {dict(sorted(owners.items()))}')
    recovery_pairs(graphs)
    for name in ('raw-keyed-cancel', 'adapter-cancel'):
        g = graphs[name]
        # The endpoint has not even been entered: local R and peer service remain enabled.
        prefix = ['gate.admit.0', 'operator.cancel', 'peer.commit']
        holding = follow(g, prefix)
        require(holding['Wait'] == 'pending-worker', 'lost registration before L')
        require(follow(g, [*prefix, 'worker.enter.0'])['Wait'] == 'pending-worker', 'misclassified post-R L')
    for g in graphs.values():
        for row in g['Rows']:
            if row['State']['Ack']:
                for edge in row['Outgoing']:
                    require(not edge['Action'].startswith(('gate.admit.', 'worker.enter.')),
                            f"covered A/L after Q in {g['Profile']['Name']}")
    print('checked local peer and R before held L on both routes and all post-Q states')
    return graphs


def rejected(fn, description):
    try:
        fn()
    except Rejection as exc:
        print(f'REJECT {description}: {exc}')
    else:
        raise Rejection('false acceptance of '+description)


def mutations(graphs):
    """Challenge the reader with edits to real reachable edges, not flags in its oracle."""
    def attack(label, name, prefix, action, mutate, *, extra=False):
        g = copy.deepcopy(graphs[name])
        row = follow(g, prefix)
        if extra:
            edge = {'Action': action, 'To': row['ID']}
            row['Outgoing'].append(edge)
        else:
            edge = next(e for e in row['Outgoing'] if e['Action'] == action)
        target = copy.deepcopy(g['Rows'][edge['To']])
        mutate(target['State'])
        target['ID'] = edge['To'] = len(g['Rows'])
        g['Rows'].append(target)
        rejected(lambda: check_graph(g, profiles()[name]), label)

    def slot(index, value):
        return lambda s: s['Worker'].__setitem__(index, value)

    raw = 'raw-keyed-cancel'
    revoked = 'raw-keyed-recover-revoked'
    admit = ['gate.admit.0']
    closed = [*admit, 'operator.cancel']
    attack('A merged with L', raw, [], 'gate.admit.0', slot(0, 'entered'))
    attack('saved permit reused across kill', revoked, [*closed, 'owner.kill'],
           'owner.open', slot(1, 'admitted'))
    attack('handling one key drops its second live registration', 'two-prepared-direct',
           [*admit, 'gate.admit.1', 'operator.cancel', *RESPONSE, 'account.record.0'],
           'account.settle.0', slot(1, 'handled'))
    for cut in (RESPONSE, [*RESPONSE, 'account.record.0']):
        attack('Q before completion accounting', raw, [*closed, *cut], 'observe.Q',
               lambda s: s.update(Ack=True), extra=True)
    attack('unpublished successful nested freeze closes descendants', 'nested-publish',
           ['operator.freeze-success'], 'gate.admit.0', slot(0, 'refused'))
    attack('crash erases accepted endpoint packet', revoked, [*admit, *SEND, 'operator.cancel'],
           'owner.kill', lambda s: s['Wire'].__setitem__(0, 'none'))
    u1 = [*closed, *RESPONSE, 'owner.kill', 'owner.open']
    attack('refusal implies prior nonapplication', revoked, u1, 'gate.refuse-source_closed.1',
           lambda s: s.update(Applications=0))
    attack('Q implies prior nonapplication', revoked, [*u1, 'gate.refuse-source_closed.1'], 'observe.Q',
           lambda s: s.update(Applications=0))
    # Challenge completeness and the exported wait classification independently of the action effects.
    g = copy.deepcopy(graphs[raw])
    row = follow(g, closed)
    row['Wait'] = 'quiescent'
    rejected(lambda: check_graph(g, profiles()[raw]), 'Wait hides live worker')
    g = copy.deepcopy(graphs[raw])
    row = follow(g, closed)
    row['Outgoing'] = [e for e in row['Outgoing'] if e['Action'] != 'peer.commit']
    rejected(lambda: check_graph(g, profiles()[raw]), 'dropped enabled peer service')
    g = copy.deepcopy(graphs[raw])
    g['Rows'][0]['State']['Local']['Debit'] = 1
    rejected(lambda: check_graph(g, profiles()[raw]), 'extra initial charge')


def source_scope_sanity(g):
    p = g['Profile']
    if p['Member'] != 'root':
        return
    for r in g['Rows']:
        l = r['State']['Local']
        impossible = l['Source'] in {'failed', 'cancelled', 'abandoned'} and l['Scope'] == 'open'
        require(not impossible, f"{p['Name']} row {r['ID']} has {l['Source']} static member with open scope")


def implementation_history(output):
    """Read real named-test observations, not the Go test's PASS/FAIL or graph labels."""
    traces = defaultdict(list)
    for line in output.splitlines():
        event = json.loads(line)
        test = event.get('Test', '')
        if test.startswith('TestReviewerRecoveredService') and event['Action'] == 'output':
            traces[test].append(event['Output'])
    expected = {'TestReviewerRecoveredService'+route for route in ('Q', 'Direct', 'Run')}
    require(traces.keys() == expected, f'missing controlled traces {expected-set(traces)}')
    failures = []
    for test in sorted(traces):
        output = ''.join(traces[test])
        first = re.search(r'first failure (.+) receipt 1 settlements 0', output)
        second = re.search(r'second Q (.+) receipt 1 settlements (\d+) POSTs 1', output)
        require(first and second and first.end() < second.start() and
                'phase2b recovery settlement failure' in first.group(1), f'{test}: unrecognized experiment')
        if first.group(1) != second.group(1):
            failures.append(f"{test} first error {first.group(1)!r}, second Q {second.group(1)!r}, "
                            f"settlements {second.group(2)}")
    require(not failures, 'same original source and owner after observed H/service error: '+' | '.join(failures))


def implementation_experiment(candidate):
    """Rebuild a candidate with the same preserved observer on its contract base."""
    base = '656f84b58424b3e2a62ce9a9eeb6bd72f84a5784'
    src = PHASE / 'evidence/rejected-recovery-accounting'
    names = ['internal/executor/'+x+'.go' for x in
             ('adapters', 'dispatch', 'external', 'quiescence_test', 'runner', 'store')] + ['runtime/runtime.go']
    with tempfile.TemporaryDirectory(prefix='phase2b-model-sticky-') as tmp:
        testroot = Path(tmp)
        archive = subprocess.check_output(['git', 'archive', base, 'internal', 'runtime', 'go.mod', 'go.sum',
                                           'author', 'effects', 'schema', 'compose', 'adapters'], cwd=ROOT)
        subprocess.run(['tar', '-xf', '-'], input=archive, check=True, cwd=testroot)
        print('base', base, 'archive-sha256', hashlib.sha256(archive).hexdigest())
        for name in names:
            data = gzip.decompress((src/('reviewed-'+name.replace('/', '_')+'.gz')).read_bytes())
            if candidate == "corrected" and name != "internal/executor/quiescence_test.go":
                data = (ROOT/name).read_bytes()
            print(name, hashlib.sha256(data).hexdigest())
            (testroot/name).write_bytes(data)
        helper = 'internal/executor/dispatch_gate_test.go'
        (testroot/helper).write_bytes((ROOT/helper).read_bytes())
        print(helper, hashlib.sha256((testroot/helper).read_bytes()).hexdigest())
        patch = gzip.decompress((src/'repro.patch.gz').read_bytes())
        subprocess.run(['patch', '-p1'], input=patch, cwd=testroot, check=True)
        fixture = testroot/'internal/executor/quiescence_test.go'
        digest = hashlib.sha256(fixture.read_bytes()).hexdigest()
        require(digest == 'e30dc9e6b32d2d99d46ce4ef74b9c7003183f17e420fcf28963b6cf09c4de792',
                'observer differs from reviewed fixture')
        print('executed observer', digest)
        print(subprocess.check_output(['go', 'version'], text=True).strip())
        argv = ['go', 'test', '-json', './internal/executor', '-run', '^TestReviewerRecoveredService',
                '-count=1', '-timeout', '90s']
        print('argv', argv)
        proc = subprocess.run(argv, cwd=testroot, text=True, capture_output=True)
        print(proc.stdout, proc.stderr, sep='\n')
        if candidate == 'corrected':
            require(proc.returncode == 0, f'corrected fixture exit {proc.returncode}')
            implementation_history(proc.stdout)
            print('ACCEPT actual recovered service on three paths: subsequent Q retained first failure')
            return
        if proc.returncode != 1:
            raise RuntimeError(f'historical behavior did not reproduce: go exit {proc.returncode}')
        try:
            implementation_history(proc.stdout)
        except Rejection as err:
            # A malformed trace or environment/setup failure is not a scientific negative.
            if not str(err).startswith('same original source and owner') or str(err).count("second Q '<nil>'") != 3:
                raise
            print('REJECT actual recovered service on three paths:', err)
        else:
            raise Rejection('accepted the old same-owner Q after genuine settlement failure')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--graph', type=Path, default=GRAPH)
    parser.add_argument('--mutants', action='store_true')
    parser.add_argument('--implementation-experiment', choices=('rejected', 'corrected'))
    args = parser.parse_args()
    if args.implementation_experiment:
        implementation_experiment(args.implementation_experiment)
        return
    graphs = check_export(args.graph)
    if args.mutants:
        mutations(graphs)


if __name__ == '__main__':
    try:
        main()
    except Rejection as exc:
        sys.exit('REJECT: '+str(exc))
