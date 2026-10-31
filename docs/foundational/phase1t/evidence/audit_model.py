"""Independent Phase 1T incoming-rule, maximum, domain and count audit. Python stdlib."""
from collections import Counter, defaultdict
from dataclasses import dataclass
from functools import lru_cache
from hashlib import sha256
from itertools import product, zip_longest
from pathlib import Path
import gzip
import json

HERE = Path(__file__).resolve().parent
MODES = ('answer', 'fail', 'absent', 'pending')
SHAPES = {'fork': ('J', 'J'), 'left-nested': (('J', 'J'), 'J'),
          'right-nested': ('J', ('J', 'J')), 'relay-above': (('J', 'J'),),
          'relay-left': (('J',), 'J')}


@dataclass
class Node:
    role: str
    kind: str
    request: str
    children: tuple
    end: str
    reason: str
    payload: str


def inventory(shape, assignment, requests):
    leaves = iter(zip(assignment, requests))

    def make(role, form):
        if form == 'J':
            kind, (ident, value, mode) = next(leaves)
            full = f'{ident},{value},{mode}'
            ans = f'{ident},{1-value}'
            helper = Node(role+'/e/h', 'h', f'{ident},{value},answer', (),
                          'succeeded', 'member_succeeded', ans)
            private = () if kind == 'A' else (helper,)
            end = {'answer': 'succeeded', 'absent': 'succeeded', 'fail': 'failed', 'pending': ''}[mode]
            reason = {'answer': 'member_succeeded', 'absent': 'missing_reply',
                      'fail': 'member_failed', 'pending': ''}[mode]
            e = Node(role+'/e', kind, full, private, end, reason, ans if mode == 'answer' else '')
            succ = Node(role+'/S', 'S', f'{ident},{value}', (), 'succeeded', 'member_succeeded', f'{ident},0')
            tag = {'answer': 'answer', 'fail': 'candidate_failed', 'absent': 'candidate_missing_reply'}.get(mode)
            public = '' if tag is None else f'{role}={ident},{tag},[{e.payload}]'
            status = '' if tag is None else 'succeeded' if mode == 'answer' else 'failed'
            return Node(role, form, full, (e, succ), status, 'member_'+status if status else '', public)
        branches = ('tree',) if role == 'O' else ('child',) if len(form) == 1 else ('left', 'right')
        kids = tuple(make(role+'/'+name, child) for name, child in zip(branches, form, strict=True))
        status = '' if any(not c.end for c in kids) else (
            'failed' if any(c.end == 'failed' for c in kids) else 'succeeded')
        return Node(role, 'constructor', ';'.join(c.request for c in kids), kids, status,
                    'member_'+status if status else '', ';'.join(c.payload for c in kids) if status else '')

    return make('O', (SHAPES[shape],))


def expected(root):
    events, waits, unselected = {}, [], []
    roles = [root]
    for n in roles:
        roles.extend(n.children)

    def event(n, name, owner, word, after=(), snap=None):
        key = n.role+':'+name
        assert key not in events
        events[key] = (owner, tuple(word), snap, set(after))
        return key

    for n in roles:
        r = n.role
        is_root = n is root
        prev = []
        if not is_root:
            caller = r.rsplit('/', 1)[0]
            auth = event(n, 'allow', 'store', ['first_invoke_allow'], [r+':invoke'], caller)
            admit = event(n, 'admit', 'store', ['admit', 'create_member', 'request_deliver', 'charge_I1_R1'], [auth])
            prev = [event(n, 'receive', r, ['request_consume', n.request], [admit])]
        for child in n.children:
            prev = [event(child, 'invoke', r, ['invoke_commit', child.request], prev)]
        order = n.children[::-1] if n.kind == 'J' else n.children
        for idx, c in enumerate(order):
            if not c.end:
                waits.append(r+' -> '+c.role)
                unselected.extend(sibling.role for sibling in order[idx+1:] if sibling.end)
                break
            prev = [event(c, 'select', r, ['outcome_consume', c.reason, c.payload], prev+[c.role+':publish'])]
        if not n.end:
            if n.kind in ('A', 'B'):
                event(n, 'unmatched', r, ['pending_receive'], prev)
                waits.append(r+' -> second-entry')
            continue
        additional = []
        if is_root:
            choice = 'answer' if n.end == 'succeeded' else 'diagnosed'
            prev = [event(n, 'decide', r, [choice], prev)]
        elif n.payload:
            prev = [event(n, 'send', r, ['reply_commit', n.payload], prev)]
            auth = event(n, 'reply_allow', 'store', ['first_reply_allow'], prev, r)
            additional = [event(n, 'record', 'store', ['recorded_reply'], [auth])]
        end = event(n, 'end', r, ['end_'+n.end, 'freeze'], prev)
        word = ['root_publish', n.end] if is_root else [
            'invocation_publish', n.reason, n.payload, 'outcome_fact', 'outcome_deliver']
        event(n, 'publish', 'store', word, [end]+additional)
    boundary = {'Waits': sorted(waits) or None, 'DeliveredUnselected': sorted(unselected) or None,
                'Decision': '' if not root.end else 'answer' if root.end == 'succeeded' else 'diagnosed',
                'Product': root.payload}
    return events, boundary, len(roles)-1, max(n.role.count('/') for n in roles)


def inclusive_cones(deps):
    @lru_cache(None)
    def cone(key):
        return frozenset({key}).union(*(cone(parent) for parent in deps[key]))
    return {k: cone(k) for k in deps}


def domain():
    result = set()
    for shape in SHAPES:
        n = 3 if 'nested' in shape else 2
        for ab in product('AB', repeat=n):
            for modes in product(MODES, repeat=n):
                if shape == 'fork':
                    nums = product(tuple(product((1, 2), (0, 1))), repeat=n)
                else:
                    nums = [((1, 0), (1, 1), (2, 0))[:n]]
                for values in nums:
                    result.add((shape, ''.join(ab), tuple((*iv, m) for iv, m in zip(values, modes))))
    return result


def count_layout(parents):
    """Boolean factor elimination. Factor entries (multiplicity, sum of maximal bits)."""
    children = [set() for _ in parents]
    factors = []
    for v, pred in enumerate(parents):
        for p in pred:
            children[p].add(v)
            factors.append(((v, p), [(1, 0), (0, 0), (1, 0), (1, 0)]))
    for v, successors in enumerate(children):
        scope = (v, *sorted(successors))
        factors.append((scope, [(1, int(i == 1)) for i in range(1 << len(scope))]))
    remaining = set(range(len(parents)))
    width = 0
    while remaining:
        # Greedy low-degree variable ordering only controls computational cost.
        scopes = {v: set() for v in remaining}
        for args, _ in factors:
            for v in args:
                scopes[v].update(args)
        v = min(remaining, key=lambda j: (len(scopes[j]), j))
        remaining.remove(v)
        scope = (v, *sorted(scopes[v]-{v}))
        width = max(width, len(scope))
        selected = [(a,t) for a,t in factors if v in a]
        factors = [(a,t) for a,t in factors if v not in a]
        mapped = [(tuple(scope.index(x) for x in a), t) for a,t in selected]
        out = [[0,0] for _ in range(1 << (len(scope)-1))]
        for i in range(1 << len(scope)):
            weight, marks = 1, 0
            for positions, table in mapped:
                idx = sum(((i >> p) & 1) << j for j,p in enumerate(positions))
                w, m = table[idx]
                marks = marks*w + weight*m
                weight *= w
            out[i//2][0] += weight
            out[i//2][1] += marks
        factors.append((scope[1:], out))
    states, covers = 1, 0
    for scope, ((w, m),) in factors:
        assert not scope
        states, covers = states*w, states*m+covers*w
    return states, covers, width


def no_duplicate_pairs(items):
    answer = dict(items)
    assert len(answer) == len(items), 'duplicate JSON key'
    return answer


def sig(row):
    return (row['Shape'], row['Assignment'],
            tuple((q['ID'], q['Value'], q['Mode']) for q in row['Requests']))


def audit_controls():
    controls = json.loads((HERE/'model-controls.json').read_bytes(), object_pairs_hook=no_duplicate_pairs)
    names = set()
    left, right = 'O/tree/left', 'O/tree/right'
    budgets = {'depth_limit': (9, 9, 3), 'invocation_limit': (8, 9, 4), 'created_run_limit': (9, 8, 4)}
    for row in controls:
        name, nxt = row['Name'], row['Next']
        assert name not in names
        names.add(name)
        root = inventory(*sig(row['Scenario']))
        rules, *_ = expected(root)
        done = set()
        for step in row['History']:
            assert step not in done and rules[step][3] <= done, (name, step)
            done.add(step)
        assert nxt not in done
        enabled = nxt in rules and rules[nxt][3] <= done
        if name in budgets:
            caps = budgets[name]
            assert enabled and nxt.endswith('/e/h:admit')
            spent = sum(k.endswith(':admit') for k in done)
            depth = nxt.count('/')
            failures = [d for d,hit in zip(
                ('invocation_limit', 'created_run_limit', 'depth_limit'),
                (spent+1 > caps[0], spent+1 > caps[1], depth > caps[2])) if hit]
            assert failures == [name] and row['Rejection'] == name+'; zero debit'
        elif name == 'wrong-handle':
            assert not enabled and nxt == left+':select'
            assert right+':publish' in done and left+':publish' not in done
        elif name == 'early-failure':
            assert not enabled and nxt not in rules and left+':select' in done
            assert right+'/e:unmatched' in done and right+':select' not in rules
        elif name == 'fault-as-pending':
            assert not enabled and nxt not in rules and right+'/e:select' in done
            assert rules[right+'/e:select'][1] == ('outcome_consume','missing_reply','')
        else:
            assert enabled
            owner, word, snap, _ = rules[nxt]
            if name == 'omitted-sibling':
                assert nxt == right+':allow' and left+':allow' not in done
            elif name == 'omitted-post-end-cut':
                assert nxt == right+'/S:reply_allow' and right+'/S:end' not in done
            elif name == 'omitted-pre-end-cut':
                assert nxt == right+'/S:end' and right+'/S:reply_allow' not in done
            elif name == 'wrong-owner':
                assert nxt == left+':select' and owner == 'O/tree'
            elif name == 'wrong-target':
                assert nxt == right+':allow' and right != left
            elif name == 'snapshot-source':
                assert snap == right+'/e/h'
            elif name == 'split-publication':
                assert word[-2:] == ('outcome_fact','outcome_deliver')
            elif name == 'reordered-admission':
                assert word == ('admit','create_member','request_deliver','charge_I1_R1')
            elif name in ('dropped-diagnosis','fabricated-diagnosis'):
                assert word == ('reply_commit', left+'=1,candidate_failed,[];'+right+'=1,candidate_missing_reply,[]')
            elif name == 'authentic-wrong-answer':
                assert nxt == right+'/e/h:publish' and word[1:3] == ('member_succeeded','1,1')
            else:
                raise ValueError(name)
    assert len(names) == 17
    return sorted(names)


def audit():
    files = (HERE/'model-census.json.gz', HERE/'model-graphs.jsonl.gz')
    with gzip.open(files[0], 'rt') as f:
        census = json.load(f, object_pairs_hook=no_duplicate_pairs)
    assert census['Cutoff'] is False and census['Version'] == '1T-open-ideal-v1'
    todo, totals = domain(), defaultdict(Counter)
    cached = lru_cache(None)(count_layout)
    digest = lambda path: sha256(path.read_bytes()).hexdigest()
    maxwidth = maxrank = nrows = 0
    with gzip.open(files[1], 'rb') as f:
        for row, line in zip_longest(census['Cases'], f):
            assert row and line and line.endswith(b'\n')
            assert sha256(line[:-1]).hexdigest() == row['GraphSHA256']
            case = sig(row)
            todo.remove(case)
            nrows += 1
            root = inventory(*case)
            rules, boundary, charges, depth = expected(root)
            assert row['Serviced'] == boundary and (row['Charges'], row['Depth']) == (charges, depth), case
            actual = json.loads(line, object_pairs_hook=no_duplicate_pairs)
            assert actual.keys() == rules.keys(), (case, rules.keys() ^ actual.keys())
            for k, (owner, facts, snap, _) in rules.items():
                ev = actual[k]
                assert set(ev) == ({'ID','Facts','After','Owner'} | ({'SnapshotOf'} if snap else set()))
                assert (ev['ID'], ev['Owner'], tuple(ev['Facts']), ev.get('SnapshotOf')) == (k,owner,facts,snap), (case,k)
            pred = {k: row[3] for k,row in rules.items()}
            cones = inclusive_cones(pred)
            assert cones == inclusive_cones({k: ev['After'] or [] for k,ev in actual.items()}), case
            # At first permission either source status is reachable for each bound reply.
            for k in rules:
                if k.endswith(':reply_allow'):
                    end = k.split(':')[0]+':end'
                    assert end not in cones[k] and k not in cones[end]
            for k in rules:
                if k.endswith(':publish'):
                    role = k.split(':')[0]
                    assert all(x in cones[k] for x in rules if x.startswith(role+'/')), (case,role)
            assert row['Rank'] == len(rules) and row['Maxima'] == 1
            # Normalize with sorted event names. Every case's words and enable cones
            # were checked separately before reusing a byte-identical counting layout.
            ids = {k:i for i,k in enumerate(sorted(rules))}
            layout = tuple(tuple(sorted(ids[p] for p in pred[k])) for k in sorted(rules))
            states, covers, w = cached(layout)
            assert (row['States'], row['Edges']) == (states, covers), (case,states,covers)
            maxwidth, maxrank = max(maxwidth,w), max(maxrank,len(rules))
            totals[case[0]].update({'States':states, 'Edges':covers, 'Cases':1,
                                    boundary['Decision'] or 'pending':1})
    assert not todo
    assert census['ByShape'] == {s:{k:c[k] for k in ('States','Edges')} for s,c in totals.items()}
    return {'status':'APPROVE', 'cases':nrows, 'shapes':dict(totals), 'distinct_layouts':cached.cache_info().misses,
            'max_factor_variables':maxwidth, 'max_events':maxrank,
            'control_prefixes':audit_controls(),
            'sha256':{p.name:digest(p) for p in (*files, HERE/'model-controls.json', Path(__file__))}}


if __name__ == '__main__':
    print(json.dumps(audit(), indent=2, sort_keys=True))
