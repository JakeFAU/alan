"""Audit the Phase 2C captured SQLite databases, endpoint journals and controller records.

Run from any directory with python3. --evidence can name a copy of the campaign.
No ALAN code, Go decoder, generated answers or network service is used.
"""
import argparse
from collections import Counter
from contextlib import closing
import gzip
import hashlib
import json
from pathlib import Path
import re
import sqlite3
import tempfile
from urllib.parse import urlsplit


def check(condition, reason):
    if not condition:
        raise ValueError(reason)


def enc(x):
    return json.dumps(x, separators=(',', ':'), ensure_ascii=False)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def source_hash(files):
    return sha(''.join(f'{files[key]}  {key}\n' for key in sorted(files)).encode())


def identity(domain, value):
    return sha(enc([domain, value]).encode())


def single(rows):
    rows = list(rows)
    check(len(rows) == 1, f'expected one row, observed {len(rows)}')
    return rows[0]


def unpack(fields):
    return {k: x[{'bool': 'Flag', 'int': 'Number', 'string': 'Text'}[x['Kind']]]
            for k, x in fields.items()}


class Tables:
    """Retain each original column as returned by SQLite, including BLOB bytes."""
    def __init__(self, path):
        with tempfile.TemporaryDirectory() as tmp:
            image = Path(tmp) / 'backup.sqlite'
            image.write_bytes(gzip.decompress(path.read_bytes()))
            with closing(sqlite3.connect(image.as_uri() + '?mode=ro&immutable=1', uri=True)) as db:
                check(list(db.execute('pragma quick_check')) == [('ok',)], 'SQLite integrity')
                check(not list(db.execute('pragma foreign_key_check')), 'SQLite foreign key')
                self.rows = {}
                self.ddl = list(db.execute("select type,name,sql from sqlite_schema order by type,name"))
                db.row_factory = sqlite3.Row
                for kind, name, _ in self.ddl:
                    if kind == 'table':
                        check(re.fullmatch('[a-z_]+', name), 'unexpected table identifier')
                        self.rows[name] = [dict(row) for row in db.execute('select * from ' + name + ' order by rowid')]

    def __getitem__(self, name):
        return self.rows[name]

    def where(self, name, **match):
        return [row for row in self[name] if all(row[k] == v for k, v in match.items())]

    def row(self, name, **match):
        return single(self.where(name, **match))

    def desc(self, name, **match):
        return json.loads(self.row(name, **match)['descriptor'])

    def same(self, other):
        return self.ddl == other.ddl and self.rows == other.rows


def journal(path):
    return [json.loads(line) for line in path.read_bytes().splitlines()]


class Original:
    def __init__(self, db, resolver='resolver'):
        self.ns = db.row('metadata')['namespace']
        self.grant = single(g for g in db.desc('reconciliation_profile')['Grants'] if g['Resolver']['Subject'] == resolver)
        self.ledger = self.grant['Endpoint'].split('/')[-2]
        check(re.fullmatch('[0-9a-f]{64}', self.ledger) and
              urlsplit(self.grant['Endpoint']).path == '/phase2c/' + self.ledger + '/apply', 'pinned apply route')
        if self.grant['Invocation']:
            iid = identity('alan/invocation/v1', [self.ns, self.grant['Invocation']])
            member = self.grant['Member']
            self.source = identity('alan/invocation-member/v1', [iid, member])
            check(db.row('composition_members', invocation_id=iid, member=member)['subject'] == self.source and
                  db.row('composition_invocations', id=iid)['operation_id'] == self.grant['Invocation'], 'dynamic binding')
            self.table = 'adapter_attempts'
        else:
            self.source = self.grant['Original']['Subject']
            self.table = 'external_attempts'
        self.run = {'Subject': self.source, 'Generation': 1}
        self.id = enc([self.source, 1, self.grant['Sequence'], 0])
        self.key = enc([2, self.ns, self.id])
        self.attempt = db.desc(self.table, intent_id=self.id)
        self.auth = self.attempt['AuthorizationID']
        self.resolver = self.grant['Resolver']
        self.terminal = db.row('terminal_obligations', subject=self.source, generation=1)['sequence']
        verify_attempt(db, self.id, self.attempt)
        if self.table == 'adapter_attempts':
            prep = self.attempt['Prepared']
            self.wire = prep['Wire']
            check(prep['Endpoint'] == self.grant['Endpoint'] and prep['Protocol'] == 'phase2c/counter/v1' and
                  prep['Model'] == 'success' and json.loads(self.wire) ==
                  {'Key': self.key, 'Profile': 'idempotent', 'Delta': 1}, 'original frozen counter')
            prep_schema(prep, {'delta': ('int', 0)}, {'mutation': ('int', 0), 'total': ('int', 0)})
        else:
            req = self.attempt['Request']
            self.wire = enc(req)
            auth = json.loads(db.row('attempts', id=self.auth)['result'])
            check(req['Authorization'] == self.auth and req['Epoch'] == auth['Source']['Epoch'] and
                  req['Subject'] == self.source and req['Generation'] == 1 and req['Sequence'] == self.grant['Sequence']
                  and req['Key'] == self.key and self.attempt['Intent']['Target'] == self.grant['Endpoint'], 'raw original wire')


def verify_attempt(db, op, attempt):
    intent = db.desc('intents', id=op)
    check(attempt['Intent'] == intent and op == enc([intent['Run']['Subject'], intent['Run']['Generation'],
                                                    intent['Sequence'], intent['Ordinal']]), 'operation identity')
    auth = db.row('attempts', intent_id=op, id=attempt['AuthorizationID'])
    result = json.loads(auth['result'])
    check(attempt['Allowed'] and result['Allowed'] and result['IntentID'] == op and
          result['Source']['Run'] == intent['Run'] and json.loads(auth['descriptor'])['Intent'] == intent, 'allowed original/query')
    db.row('permissions', kind=intent['Kind'], source=intent['Run']['Subject'], target=intent['Target'],
           revision=result['Source']['Revision'], operation=intent['Obligation'] if intent['Kind'] == 'adapter' else 'apply',
           obligation=intent['Obligation'] if intent['Kind'] == 'adapter' else '')
    reservation = db.desc('composition_reservations', operation_id=op)
    charge = db.desc('composition_charges', operation_id=op, kind='effect')
    check(reservation['Allowed'] and reservation['Run'] == intent['Run'] and charge['Effects'] == 1 and
          charge['RootID'] == reservation['RootID'] and charge['OperationID'] == op, 'operation debit and reservation')


def prep_schema(prep, inputs, outputs):
    check(prep['Digest'] == sha(enc(dict(prep, Digest='')).encode()), 'preparation digest')
    for slot, expected in [('Input', inputs), ('Output', outputs)]:
        schema = prep['Request'][slot]
        check(schema['ID'] == sha(enc(schema['Record']).encode()) and
              {k: (v['Kind'], v['MaxBytes']) for k, v in schema['Record'].items()} == expected, 'exact schema')


def execfact(db, name, seq):
    return next(r['decision_order'] for r in db['composition_facts'] if r['kind'] == 'execution'
                and (d := json.loads(r['descriptor']))['Run']['Subject'] == name and d['Sequence'] == seq)


def evidence_order(db, op, kind):
    return single(r['decision_order'] for r in db['composition_facts'] if r['kind'] == kind and
                  json.loads(r['descriptor'])['OperationID'] == op)


def select(db, obid):
    ob = db.row('observations', id=obid)
    h = db.row('history', subject=ob['subject'], generation=ob['generation'], sequence=ob['consumed_sequence'])
    check(h['selected_observation'] == obid, 'observation consumption cross-link')
    return json.loads(ob['descriptor']), json.loads(h['successor']), h['sequence']


def message(db, sender, receiver, obligation, consumed=True):
    intent = single(d for r in db.where('intents', subject=sender) if (d := json.loads(r['descriptor']))['Kind'] == 'message'
                    and d['Target'] == receiver and d['Obligation'] == obligation)
    ack = db.row('message_acknowledgments', intent_id=intent['ID'])
    admit = db.row('admissions', id=ack['admission_id'])
    descriptor = json.loads(admit['descriptor'])
    check(json.loads(db.row('attempts', intent_id=intent['ID'])['result'])['Allowed'] and
          descriptor['IntentID'] == intent['ID'] and descriptor['Payload'] == intent['Payload'] and
          json.loads(admit['result'])['Accepted'], 'message authority/admission')
    if consumed:
        ob, _, seq = select(db, admit['observation_id'])
        check(ob['Provenance'] == 'ordinary-message' and ob['Source'] == sender and ob['Run']['Subject'] == receiver
              and ob['SourceGeneration'] == 1 and ob['Obligation'] == obligation and ob['Payload'] == intent['Payload']
              and execfact(db, sender, intent['Sequence']) < execfact(db, receiver, seq), 'ordinary message selection')
        return unpack(intent['Payload']), seq
    check(admit['observation_id'] is None and json.loads(admit['result'])['Reason'] == 'recorded_reply', 'separate bound reply')
    return intent


def handoff(db, o, failed=False):
    target = {'target': o.id}
    if o.table == 'external_attempts':
        check(message(db, o.source, 'resolver', 'target')[0] == target, 'raw target handoff')
        return
    order = []
    for a, b, ob in [(o.source, 'parent', 'identity'), ('parent', 'resolver', 'resolver-target'),
                     ('resolver', 'parent', 'resolver-target'), ('parent', o.source, 'identity')]:
        payload, seq = message(db, a, b, ob)
        check(payload == target, 'genuine relayed target')
        order.append(execfact(db, b, seq))
    check(order == sorted(set(order)) and order[-1] < execfact(db, o.source, o.terminal), 'ACK precedes End')
    reply = message(db, o.source, 'parent', 'issue', False)
    iid = db.row('composition_invocations', operation_id=o.grant['Invocation'])['id']
    freeze = db.desc('composition_freezes', scope_id=iid)
    check(freeze['Status'] == ('failed' if failed else 'succeeded') and freeze['Run'] == o.run and
          freeze['Sequence'] == o.terminal and freeze['ReplyIntents'] == [reply['ID']] and not freeze['InvocationIntents'] and
          freeze['ExternalIntents'] == [o.id] and db.desc('composition_replies', operation_id=reply['ID'])['Intent'] == reply and
          unpack(reply['Payload']) == {'target': o.id, 'issued': True}, 'original freeze/issuance')


def read_ordinals(db, o):
    ordinals = {}
    for row in db['receipt_read_ordinals']:
        qid, n = row['intent_id'], row['ordinal']
        if json.loads(qid)[0] != o.resolver['Subject']:
            continue
        op = json.loads(row['descriptor'])
        query = op['Query']
        a = db.desc('adapter_attempts', intent_id=qid)
        verify_attempt(db, qid, a)
        prep = a['Prepared']
        fields = {'status': ('string', 32), 'target': ('string', 1024), 'adoption': ('string', 128),
                  'adopted': ('bool', 0), 'mutation': ('int', 0), 'total': ('int', 0)}
        prep_schema(prep, {'target': ('string', 1024)}, fields)
        check(prep['Request']['Payload'] == a['Intent']['Payload'] and unpack(a['Intent']['Payload']) == {'target': o.id}
              and prep['Request']['Key'] == enc([2, o.ns, qid]) and prep['Request']['ID'] == qid and
              prep['Endpoint'] == o.grant['Endpoint'][:-5] + 'lookup' and prep['Wire'] == '' and
              prep['Protocol'] == 'phase2c/receipt-read/v1', 'separate lookup preparation')
        expected = {'QueryID': qid, 'QueryKey': enc([2, o.ns, qid]), 'TargetID': o.id, 'TargetKey': o.key,
                    'Ordinal': n, 'LedgerID': o.ledger, 'Wire': o.wire, 'WireSHA256': sha(o.wire.encode())}
        check(query == expected and n in (1, 2) and qid != o.id and ((qid, 1) in ordinals or n == 1), 'exact physical-query binding')
        check(op['Adopt'] == o.grant['Adopt'] and op['OriginalRun'] == o.run and op['OriginalAuthorization'] == o.auth and
              op['OriginalAttemptDigest'] == identity('alan/original-attempt/v1', o.attempt) and
              op['OriginalPreparedDigest'] == o.attempt.get('Prepared', {}).get('Digest', '') and
              op['OriginalMode'] == o.attempt.get('Prepared', {}).get('Model', 'raw') and
              op['OriginalTerminalSequence'] == o.terminal and op['Resolver']['Run'] == o.resolver == a['Intent']['Run'], 'read grant/provenance')
        fact = db.row('composition_facts', decision_order=op['Prefix'] + 1)
        fd = json.loads(fact['descriptor'])
        check(fact['kind'] == 'receipt_read' and fd['Sequence'] == n and fd['OperationID'] == qid and
              fd['RecordDigest'] == identity('alan/receipt-read-ordinal/v1', op) and
              max(execfact(db, o.source, o.terminal), evidence_order(db, qid, 'reservation')) <= op['Prefix'], 'ordered spend')
        latest = max(d['Sequence'] for r in db['composition_facts'] if r['kind'] == 'execution' and
                     r['decision_order'] <= op['Prefix'] and (d := json.loads(r['descriptor']))['Run'] == o.resolver)
        hist = db.row('history', subject=o.resolver['Subject'], sequence=latest)
        check(latest == op['Resolver']['Sequence'] and json.loads(hist['successor'])['Status'] == 'open' and
              hist['source_revision'] == op['Resolver']['Revision'] and op['Resolver']['Epoch'] >= hist['epoch'], 'live resolver at spend')
        ordinals[qid, n] = op
    return ordinals


def world_evidence(world, events, o, ordinals):
    ledger = world.row('ledger_identity')['id']
    check(world.row('metadata')['version'] == 2, 'endpoint schema')
    ingress, applied, reads, contacts = {}, {}, {}, []
    for e in events:
        call = e['Process'], e['PID'], e['Contact']
        if e['Event'] == 'ingress':
            check(call not in ingress and e['Method'] == 'POST', 'distinct endpoint ingress')
            route = e['Path']
            check(route in ('/phase2c/' + o.ledger + '/apply', '/phase2c/' + o.ledger + '/lookup'), 'literal endpoint route')
            if route.endswith('/apply'):
                check(e['Body'] == o.wire, 'original ingress bytes')
            else:
                q = json.loads(e['Body'])
                check(q == ordinals[q['QueryID'], q['Ordinal']]['Query'], 'wire ingress versus reserved read')
            ingress[call] = e
        elif e['Event'] == 'committed':
            req = ingress[call]
            check(req['Path'].endswith('/apply') and req['Body'] == o.wire, 'application from POST')
            receipt = e['Receipt']
            n = receipt['Mutation']
            contacts.append(req['Body'].encode())
            if n not in applied:
                check(n == len(applied) + 1 and receipt['PID'] == req['PID'] and receipt['Process'] == req['Process']
                      and receipt['Key'] == o.key and receipt['Delta'] == 1 and receipt['Total'] == n and
                      receipt['Profile'] == json.loads(o.wire)['Profile'], 'business receipt lineage')
                applied[n] = receipt
            else:
                check(applied[n] == receipt and receipt['Profile'] == 'idempotent', 'same immutable keyed receipt')
        elif e['Event'] == 'snapshot':
            req = ingress[call]
            check(req['Path'].endswith('/lookup'), 'physical lookup route')
            snap = json.loads(e['Response'])
            q = json.loads(req['Body'])
            known = len(applied)
            status = ('not_found' if known == 0 else 'found' if known == 1 else 'conflict')
            n = len(reads) + 1
            expected = {'Query': q, 'Number': n, 'Count': known, 'Status': status, 'AppliedWire': o.wire if known == 1 else '',
                        'Receipt': applied.get(1) if known == 1 else None}
            check(ledger == q['LedgerID'] and expected == snap == e['Snapshot'], 'snapshot truth at ordered world commit')
            read = world.row('receipt_reads', number=n)
            check(read['request'] == req['Body'].encode() and read['response'] == e['Response'].encode(), 'endpoint raw snapshot bytes')
            reads[q['QueryID'], q['Ordinal']] = e['Response']
        else:
            raise ValueError('unexpected endpoint event ' + e['Event'])
    check(len(world['receipt_reads']) == len(reads) and [r['request'] for r in world['contacts']] == contacts, 'world read/contact conservation')
    check(len(world['applications']) == len(world['application_wires']) == len(applied) and
          len(world['keys']) == bool(applied), 'world application conservation')
    for n, receipt in applied.items():
        wire = world.row('application_wires', number=n)
        check(wire['key'] == o.key and wire['wire'] == o.wire.encode() and
              json.loads(world.row('applications', number=n)['receipt']) == receipt and
              world.row('keys', key=o.key)['descriptor'] == o.wire.encode(), 'immutable business wire/receipt')
    return reads


def receipts(db, o, ordinals, reads, final, immutable=False):
    results = {r['intent_id']: json.loads(r['descriptor']) for r in db['adapter_results']}
    selected = []
    for qid in sorted({key[0] for key in ordinals}, key=lambda q: json.loads(q)[2]):
        if qid not in results:
            check(not final, 'missing terminal query result')
            continue
        r = results[qid]
        c, values = r['Candidate'], unpack(r['Values'])
        check(c['ID'] == qid and c['Key'] == enc([2, o.ns, qid]), 'query candidate identity')
        read = r.get('Read')
        if r['Disposition'] == 'validated':
            n = read['Ordinal']
            response = reads[qid, n]
            check(c['Response'] == response and c['HTTPStatus'] == 200, 'Store retains exact authenticated lookup response')
            s = json.loads(response)
            if s['Status'] != 'found':
                decision = 'none' if s['Status'] == 'not_found' else 'conflict'
            elif not o.grant['Adopt']:
                decision = 'denied'
            elif o.table == 'external_attempts':
                decision = 'evidence_only'
            else:
                decision = 'existing' if immutable else 'adopted'
            v = s['Receipt'] or {'Mutation': 0, 'Total': 0}
            check(read['Adoption'] == decision and values == {
                'valid': True, 'disposition': 'validated', 'adopted': decision == 'adopted', 'adoption': decision,
                'status': s['Status'], 'target': o.id, 'mutation': v['Mutation'], 'total': v['Total']
            }, 'program-visible adoption justified by receipt and authority')
            if decision == 'adopted':
                orig = results[o.id]
                check(orig['Reconciled'] == {'Target': o.id, 'QueryID': qid, 'Ordinal': n, 'Receipt': s['Receipt']}
                      and not any(orig['Candidate'].values()) and unpack(orig['Values']) == {
                          'valid': True, 'disposition': 'validated', 'mutation': v['Mutation'], 'total': v['Total']
                      }, 'reconstruction/receipt edge without invented original response')
                check(evidence_order(db, qid, 'external_result') < evidence_order(db, o.id, 'external_result'), 'accounting fact order')
        else:
            check(not values['valid'] and not values['adopted'] and not values['status'] and
                  r['Disposition'] == c['Disposition'] and r['Disposition'] in ('protocol_invalid', 'transport_error', 'read_exhausted'),
                  'unsuccessful read preserves uncertainty')
        if final:
            settlement = db.row('settlements', intent_id=qid)
            ob, _, seq = select(db, settlement['observation_id'])
            check(settlement['descriptor'] == db.row('adapter_results', intent_id=qid)['descriptor'] and
                  ob['Run'] == o.resolver and ob['Provenance'] == 'runtime-adapter' and ob['Operation'] == qid and
                  ob['Payload'] == r['Values'] and evidence_order(db, qid, 'external_result') < execfact(db, 'resolver', seq),
                  'actual resolver consumption after evidence')
        selected.append((values['status'] or r['Disposition'], read['Adoption'] if read else 'none'))
    return selected


def parent_branch(db, first, failed):
    run = db.row('runs', subject='parent', generation=1)
    raw = db.row('programs', id=run['program_id'])['bytes']
    check(sha(raw) == run['program_id'], 'retained parent program hash')
    code = json.loads(raw)['Code']
    comparisons = []
    for seq in range(first, run['sequence'] + 1):
        row = db.row('history', subject='parent', sequence=seq)
        prev = json.loads(db.row('history', subject='parent', sequence=seq - 1)['successor'])
        current = json.loads(row['successor'])
        pc = prev['PC']
        step = code[pc]
        if step['Op'] == 'receive':
            ob = db.desc('observations', id=row['selected_observation'])
            for field, dest in step['Bind'].items():
                check(ob['Payload'][field] == current['Data'][dest], 'actual parent receive bindings')
        elif step['Op'] == 'branch':
            left = prev['Data'][step['A']['Ref']]
            right = prev['Data'][step['B']['Ref']] if step['B']['Ref'] else step['B']['Literal']
            next_pc = step['Target'] if left == right else step['Otherwise']
            check(current['PC'] == next_pc, 'executed parent equality branch')
            comparisons.append((step['A']['Ref'], step['B']['Ref'] or unpack({'literal': right})['literal']))
        elif step['Op'] == 'set':
            check(current['Data'][step['Dst']] == step['A']['Literal'], 'executed decision assignment')
        elif step['Op'] == 'end':
            check(step['Outcome'] == current['Status'] == 'succeeded', 'executed successful parent End')
    needed = [('receipt_target', 'original'), ('receipt_status', 'found'), ('receipt_total', 1), ('receipt_mutation', 1),
              ('issued_present', True), ('issued_target', 'original'), ('source_status', 'succeeded')]
    needed.extend([('source_status', 'failed')] if failed else [('issued', True), ('receipt_adopted', True)])
    check(comparisons == needed, 'meaningful parent guard list')


def parent(db, o, ready, failed=False):
    pubs = db['composition_publications']
    if not ready:
        check(not pubs and not db['composition_outcome_deliveries'] and not db.where('settlements', intent_id=o.id), 'original still pending')
        return
    pub = db.desc('composition_publications', operation_id=o.grant['Invocation'])
    state = 'failed' if failed else 'succeeded'
    check(pub['Status'] == state and pub['ResponsePresent'] and unpack(pub['Response']) == {'issued': True, 'target': o.id},
          'actual public issuance')
    delivery = db.row('composition_outcome_deliveries', operation_id=o.grant['Invocation'])
    ob, _, seq = select(db, delivery['observation_id'])
    check(ob['Run'] == {'Subject': 'parent', 'Generation': 1} and ob['Provenance'] == 'runtime-invocation' and
          ob['Operation'] == o.grant['Invocation'] and unpack(ob['Payload']) == {
              'status': state, 'reason': 'member_' + state, 'response.issued': True, 'response.target': o.id,
              'response_present': True} and evidence_order(db, o.grant['Invocation'], 'outcome') < execfact(db, 'parent', seq), 'actual invocation selection')
    parent_branch(db, seq, failed)
    report, rseq = message(db, 'resolver', 'parent', 'receipt')
    check(seq < rseq and report == {'target': o.id, 'status': 'found', 'adoption': 'existing' if failed else 'adopted',
                                   'adopted': not failed, 'mutation': 1, 'total': 1}, 'parent selected correlated receipt fields')
    p = json.loads(db.row('runs', subject='parent')['projection'])
    decision = 'applied_after_original_failure' if failed else 'applied'
    check(p['Status'] == 'succeeded' and unpack(p['Data'])['decision'] == decision, 'terminal parent decision')
    check(db.row('settlements', intent_id=o.id)['descriptor'] == db.row('adapter_results', intent_id=o.id)['descriptor'],
          'unique original settlement')


EXPECTED = {
    'U0': [('not_found', 'none')], 'U1': [('found', 'adopted')], 'atomic-import': [('found', 'adopted')],
    'held-retained': [('not_found', 'none'), ('found', 'adopted')], 'held-lost': [('found', 'adopted')],
    'exhausted': [('read_exhausted', 'none')], 'raw-ambiguous': [('conflict', 'conflict')],
    'raw-single': [('found', 'evidence_only')], 'raw-keyed': [('found', 'evidence_only')],
    'ledger-restart': [('found', 'adopted')], 'ledger-replacement': [('protocol_invalid', 'none')],
    'unavailable-refresh': [('transport_error', 'none'), ('found', 'adopted')],
    'positive': [('found', 'adopted')], 'immutable': [('found', 'existing')], 'read-only': [('found', 'denied')],
}


def final_state(name, db, o, ordinals, reads, world, events):
    evidence = receipts(db, o, ordinals, reads, True, name == 'immutable')
    check(evidence == EXPECTED[name], 'declared case outcomes')
    queryids = {qid for qid, n in ordinals}
    check({r['operation_id'] for r in db['composition_charges'] if r['kind'] == 'effect'} == {o.id} | queryids and
          {r['operation_id'] for r in db['composition_reservations']} == {o.id} | queryids, 'logical budget conservation')
    check(all(not unpack(json.loads(ob['descriptor'])['Payload']) for ob in db['observations'] if
              json.loads(ob['descriptor'])['Provenance'] == 'local-input'), 'no host-supplied target or endpoint truth')
    ready = evidence[-1][1] in ('adopted', 'existing')
    if o.table == 'adapter_attempts':
        parent(db, o, ready, name == 'immutable')
    else:
        check(not db['external_receipts'] and not db.where('settlements', intent_id=o.id), 'raw evidence only')
    check(bool(db.where('adapter_results', intent_id=o.id)) == ready, 'no unwarranted original result')
    apps = 0 if name in ('U0', 'ledger-replacement') else 2 if name == 'raw-ambiguous' else 1
    contacts = 2 if name in ('raw-keyed', 'raw-ambiguous') else int(apps != 0)
    check(len(world['applications']) == apps and len(world['contacts']) == contacts, 'business projection bounds')
    expected_ordinals = 2 if name in ('held-lost', 'held-retained', 'unavailable-refresh', 'exhausted') else 1
    expected_reads = 0 if name in ('exhausted', 'ledger-replacement') else 2 if name in ('held-lost', 'held-retained') else 1
    check(len(ordinals) == expected_ordinals and len(reads) == expected_reads, 'physical read conservation')
    ingress = Counter(e['Path'].split('/')[-1] for e in events if e['Event'] == 'ingress')
    check(ingress['apply'] == contacts and ingress['lookup'] == expected_reads + (name == 'ledger-replacement'),
          'no additional mutation or lookup ingress')
    return {'applications': apps, 'business_contacts': contacts, 'logical_queries': len(queryids),
            'ordinals': len(ordinals), 'snapshots': len(reads), 'selected': evidence}


def preserved(before, after, o):
    for name in ('history', 'intents', 'attempts', o.table, 'composition_reservations',
                 'composition_charges', 'composition_freezes', 'composition_replies', 'composition_publications',
                 'adapter_results', 'settlements', 'terminal_obligations', 'receipt_read_ordinals'):
        for row in before[name]:
            check(row in after[name], 'immutable retained bytes in ' + name)
    for row in before['observations']:
        if row['consumed_sequence'] is not None:
            check(row in after['observations'], 'immutable selected observation')


def held_order(events, name):
    actions = [(e['Detail']['Op'] if e['Event'] == 'command' else e['Detail'] if e['Event'] == 'snapshot' else e['Event'])
               for e in events]
    cut = 'lost-absence' if name == 'held-lost' else 'retained-absence'
    sequence = ['barrier', 'closed', 'SIGKILL', 'original-cut', 'open', 'Q', 'start', cut, 'release original packet',
                'original application', 'open' if name == 'held-lost' else 'refresh', 'final']
    last = -1
    for item in sequence:
        check(item in actions[last + 1:], 'held order: ' + item)
        last = actions.index(item, last + 1)


def process_case(folder, regime):
    name = folder.name
    events = json.loads((folder / 'transcript.json').read_bytes())
    names = [e['Detail'] for e in events if e['Event'] == 'snapshot']
    inventory = {p.name.removesuffix('-local.sqlite.gz') for p in folder.glob('*-local.sqlite.gz')}
    check(len(set(names)) == len(names) and inventory == set(names), 'cut inventory')
    before = Tables(folder / 'original-cut-local.sqlite.gz')
    origin = Original(before)
    handoff(before, origin)
    check(not before['receipt_read_ordinals'] and not before['adapter_results'] and not before['composition_publications'],
          'cut before reconciliation')
    observed, worlds = {}, {}
    replay_count = 0
    for label in names:
        db = Tables(folder / (label + '-local.sqlite.gz'))
        world = Tables(folder / (label + '-world.sqlite.gz'))
        jp = folder / (label + '-journal.jsonl')
        if label.endswith('-replayed'):
            prior = label.removesuffix('-replayed')
            check(db.same(observed[prior]) and world.same(worlds[prior]) and
                  jp.read_bytes() == (folder / (prior + '-journal.jsonl')).read_bytes(), 'offline replay changed raw rows/journal')
            replay_count += 1
            continue
        observed[label], worlds[label] = db, world
        if label == 'ingress-before-R':
            preserved(db, before, origin)
        else:
            preserved(before, db, origin)
        ords = read_ordinals(db, origin)
        reads = world_evidence(world, journal(jp), origin, ords)
        if label == 'final':
            outcome = final_state(name, db, origin, ords, reads, world, journal(jp))
        elif label == 'ingress-before-R':
            source = db.row('runs', subject=origin.source, generation=1)
            ingress = single(journal(jp))
            check(source['sequence'] == origin.terminal - 1 and json.loads(source['projection'])['Status'] == 'open' and
                  db.desc(origin.table, intent_id=origin.id) == origin.attempt and not db['terminal_obligations'] and
                  not db['composition_freezes'] and not world['applications'] and not ords and not reads and
                  ingress['Event'] == 'ingress' and ingress['Body'] == origin.wire and ingress['Path'].endswith('/apply'),
                  'accepted original ingress before durable terminal R')
        elif label in ('original-cut', 'Q'):
            apps = 0 if name in ('held-retained', 'held-lost', 'U0') or (name == 'ledger-replacement' and label == 'Q') else 1
            check(len(world['applications']) == apps and not ords and not reads and not db['adapter_results'], 'world and missing local cut')
        elif label in ('atomic-committed', 'lost-absence', 'retained-absence', 'ordinal-1', 'ordinal-2', 'retained-error'):
            values = receipts(db, origin, ords, reads, False)
            if label == 'atomic-committed':
                check(values == [('found', 'adopted')] and len(db['adapter_results']) == 2 and
                      not db['settlements'] and not db['composition_publications'], 'atomic pre-settlement side')
            elif label.startswith('ordinal-'):
                check(len(ords) == int(label[-1]) and not values and not reads and not db['adapter_results'], 'spent but unsent')
            elif label == 'lost-absence':
                check(len(ords) == 1 and len(reads) == 1 and json.loads(single(reads.values()))['Status'] == 'not_found'
                      and not values and not db['adapter_results'], 'lost provisional absence')
            else:
                status = 'transport_error' if label == 'retained-error' else 'not_found'
                check(values == [(status, 'none')] and len(reads) == (label != 'retained-error'), 'earlier selected result')
                qid = single(ords)[0]
                ob, _, _ = select(db, db.row('settlements', intent_id=qid)['observation_id'])
                check(ob['Operation'] == qid and ob['Run'] == origin.resolver and
                      unpack(ob['Payload'])['disposition' if label == 'retained-error' else 'status'] == status,
                      'refresh follows actual prior query selection')
        elif label == 'pre-read':
            check(not ords and not reads, 'raw comparator before lookup')
        else:
            raise ValueError('unexpected cut ' + label)
    after = observed['final']
    for db in observed.values():
        preserved(db, after, origin)
    if name == 'atomic-import':
        check(worlds['atomic-committed'].same(worlds['final']), 'recovered import caused world I/O')
    if name.startswith('held-'):
        held_order(events, name)
    if name == 'held-lost':
        final_ord = read_ordinals(after, origin)
        qid = single(set(k[0] for k in final_ord))
        check(final_ord[qid, 1]['Resolver']['Epoch'] < final_ord[qid, 2]['Resolver']['Epoch'] and
              after.desc('adapter_results', intent_id=qid)['Read']['Ordinal'] == 2, 'lost refresh under renewed owner')
    if name in ('ledger-restart', 'ledger-replacement'):
        ready = [e['Detail'] for e in events if e['Event'] == 'endpoint ready']
        a, b = ready
        check(a['PID'] != b['PID'] and a['Token'] != b['Token'] and urlsplit(a['URL']).netloc == urlsplit(b['URL']).netloc
              and a['LedgerID'] == worlds['original-cut'].row('ledger_identity')['id'] and
              b['LedgerID'] == worlds['final'].row('ledger_identity')['id'] and
              (a['LedgerID'] == b['LedgerID']) == (name == 'ledger-restart'), 'ledger continuity across endpoint replacement')
    replay_acks = [e['Detail']['Cut'] for e in events if e['Event'] == 'replay']
    check(sorted(replay_acks) == sorted(n.removesuffix('-replayed') for n in names if n.endswith('-replayed')), 'replay transcript')
    counts = controller(folder, events, observed, origin, regime)
    return dict(outcome, **counts, replay_windows=replay_count, cuts=len(names))


def controller(folder, events, cuts, o, regime):
    opened, endpoints, pending, death = {}, {}, None, None
    q_count, kills, alive, closed = 0, 0, None, False
    staged, released = None, False
    for e in events:
        tag, d = e['Event'], e['Detail']
        if tag == 'endpoint ready':
            proc = json.loads(single(folder.glob('*-endpoint-' + str(d['PID']) + '.json')).read_bytes())
            check(proc['PID'] == d['PID'] and d['Store'] in proc['argv'] and d['Token'] in proc['argv'], 'endpoint executable binding')
            endpoints[d['PID']] = d
        elif tag == 'command':
            pid = d['PID']
            check(d['Namespace'] == o.ns, 'owner namespace')
            if d['Op'] == 'open':
                check(pid not in opened and alive is None, 'fresh exclusive owner order')
                rec = json.loads(single(folder.glob('*-process-start-' + str(pid) + '.json')).read_bytes())
                manifest = single(folder.glob('*-sources-' + rec['source_manifest_sha256'] + '.json'))
                check(rec['PID'] == pid and len(rec['executable_sha256']) == 64 and
                      source_hash(json.loads(manifest.read_bytes())) == rec['source_manifest_sha256'], 'owner source manifest')
                settings = {v['Key']: v.get('Value', '') for v in rec['child_build_info']['Settings']}
                check((settings.get('-race') == 'true') == (regime == 'race'), 'binary instrumentation')
                alive = pid
                opened[pid] = rec, d['Resolver']['Epoch']
            check(pid == alive, 'command current owner')
            if d['Op'] in ('Q', 'closed'):
                check(d['Source'] == o.run and d['Original']['ID'] == o.run and
                      d['Original']['Projection']['Status'] == 'succeeded' and d['Original']['Sequence'] == o.terminal,
                      'ack names original terminal generation')
                if d['Op'] == 'Q':
                    check(closed and kills > 0, 'Q under successor')
                    q_count += 1
                else:
                    if 'ingress-before-R' in cuts:
                        check(released, 'End not released')
                    closed = True
            if d['Op'] in ('start', 'refresh'):
                check(q_count > 0, 'resolver service following current-source Q')
            if d['Op'] == 'exit':
                alive = None
        elif tag == 'barrier':
            if d['Boundary'] == 'execution.staged':
                rec = opened[alive][0]
                configured = json.loads(rec['environment']['ALAN_BARRIER'])
                check(pending and pending['Boundary'] == 'resource.ingress' and alive == d['PID'] and
                      d['Store'] == rec['environment']['ALAN_PHASE2_DB'] and d['Phase'] == 'terminal' and
                      all(d[k] == configured[k] for k in ('Scenario', 'Token', 'Store', 'Boundary', 'Phase')) and
                      d['Subject'] == o.source and d['Generation'] == 1 and d['Sequence'] == o.terminal and
                      d['Epoch'] == cuts['ingress-before-R'].row('runs', subject=o.source)['epoch'] and
                      d['Intent'] == d['Attempt'] == '' and o.auth in d['InFlight'], 'staged original terminal held across ingress')
                staged = d
                continue
            check(pending is None, 'barrier without resolved cut')
            if d['PID'] in endpoints:
                ep = endpoints[d['PID']]
                check(d['Token'] == ep['Token'] and d['Store'] == ep['Store'] and d['Intent'] == o.key and
                      (d['Boundary'], d['Phase']) == ('resource.ingress', 'pre-apply'), 'ack original held ingress')
            else:
                rec, epoch = opened[alive]
                check(alive == d['PID'] and d['Store'] == rec['environment']['ALAN_PHASE2_DB'] and
                      d['Token'] and d['Attempt'] == enc(['authorization', d['Intent']]) and
                      json.loads(d['Intent']) == [d['Subject'], d['Generation'], d['Sequence'], 0], 'owner barrier binding')
                configured = json.loads(rec['environment']['ALAN_BARRIER'])
                check(all(configured[k] == d[k] for k in ('Scenario', 'Token', 'Store', 'Boundary', 'Phase')) and
                      (not configured['Subject'] or configured['Subject'] == d['Subject']), 'configured executor cut')
                if d['Subject'] == o.resolver['Subject']:
                    check(d['Epoch'] == epoch, 'barrier physical epoch')
            pending = d
        elif tag == 'snapshot' and d == 'ingress-before-R':
            check(staged and not released, 'causal pre-R backup')
        elif tag == 'release staged original End':
            check(staged == d and not released, 'release exact held End')
            released = True
        elif tag == 'SIGKILL':
            check(closed and pending and alive == d['PID'] and d['Wait'] == 'signal: killed' and
                  opened[alive][0]['executable_sha256'] == d['Binary'], 'acknowledged owner death')
            death, pending, alive = pending, None, None
            kills += 1
        elif tag == 'snapshot' and death:
            b = death
            expected = {'atomic-committed': ('adapter.result-recorded', 'pre-settlement'),
                        'lost-absence': ('adapter.response', 'pre-record'),
                        'ordinal-1': ('dispatch.admitted', 'pre-call'), 'ordinal-2': ('dispatch.admitted', 'pre-call')}
            if d == 'original-cut':
                wanted = ('resource.ingress', 'pre-apply') if folder.name.startswith('held-') else (
                         ('dispatch.admitted', 'pre-call') if folder.name == 'U0' else
                         ('external.response' if o.table == 'external_attempts' else 'adapter.response', 'pre-record'))
                check(b['Intent'] == (o.key if wanted[0] == 'resource.ingress' else o.id), 'cut on real original')
            else:
                wanted = expected[d]
                cut_ord = read_ordinals(cuts[d], o)
                ordinal = max(n for qid, n in cut_ord if qid == b['Intent'])
                check(cut_ord[b['Intent'], ordinal]['Resolver']['Epoch'] == b['Epoch'], 'barrier and durable physical activation')
            check((b['Boundary'], b['Phase']) == wanted, 'cut on declared boundary')
            death = None
        elif tag == 'external comparator POST':
            check(folder.name in ('raw-keyed', 'raw-ambiguous') and d['Wire'] == o.wire and
                  json.loads(d['Response'])['Key'] == o.key, 'explicit environment action')
    check(not alive and not pending and not death and q_count and kills == len(opened) - 1, 'completed process lifetimes')
    return {'owner_processes': len(opened), 'deaths': kills}


def public_case(folder):
    name = folder.name
    before, after, world = (Tables(folder / (x + '.sqlite.gz')) for x in ('before', 'after', 'world'))
    o = Original(before)
    handoff(before, o, name == 'immutable')
    preserved(before, after, o)
    check(not before['receipt_read_ordinals'], 'public cold cut')
    if name == 'immutable':
        old = before.desc('adapter_results', intent_id=o.id)
        ob, proj, _ = select(before, before.row('settlements', intent_id=o.id)['observation_id'])
        check(old['Disposition'] == 'transport_error' and ob['Operation'] == o.id and
              unpack(proj['Data'])['original_valid'] is False and unpack(proj['Data'])['original_total'] == 0,
              'original unsuccessful selection before lookup')
        select(before, before.row('composition_outcome_deliveries', operation_id=o.grant['Invocation'])['observation_id'])
    else:
        check(not before['adapter_results'] and not before['composition_publications'], 'public missing original')
    ordinals = read_ordinals(after, o)
    reads = world_evidence(world, journal(folder / 'journal.jsonl'), o, ordinals)
    outcome = final_state(name, after, o, ordinals, reads, world, journal(folder / 'journal.jsonl'))
    procs = json.loads((folder / 'commands.json').read_bytes())
    check([r['argv'][1] for r in procs] == ['-receipt-ledger', 'init', 'issue', 'resolve', 'replay'] and
          len({r['pid'] for r in procs}) == 5 and len({r['sha256'] for r in procs[1:]}) == 1 and
          [r['exit'] for r in procs] == [0, 0, 0 if name == 'immutable' else -9, 0, 0], 'public executable identities/exits')
    endpoint, _, issue, resolver, _ = procs
    ready = endpoint['ready']
    check(ready['PID'] == endpoint['pid'] and ready['URL'] == o.grant['Endpoint'] and
          ready['LedgerID'] == world.row('ledger_identity')['id'] and json.loads(resolver['stdout'])['Q'] == o.run, 'public exact Q and ledger')
    if name != 'immutable':
        b = issue['applied_barrier']
        check(b['PID'] == endpoint['pid'] and b['Store'] == ready['Store'] and b['Intent'] == o.key and
              (b['Boundary'], b['Phase']) == ('resource.applied', 'post-commit'), 'public original packet cut')
    return outcome


def campaign_digest(directory, cases):
    paths = sorted(p for case in cases for p in case.rglob('*') if p.is_file())
    manifest = {str(p.relative_to(directory)): sha(p.read_bytes()) for p in paths}
    return {'files': len(paths), 'sha256': source_hash(manifest)}


def matched(folder):
    for cut in ('original-cut', 'Q'):
        def backup(case):
            return Tables(folder / case / (cut + '-local.sqlite.gz'))
        check(backup('U0').same(backup('U1')), 'matched branches differ before lookup: ' + cut)


def attacks(evidence):
    """Mutants edit independent in-memory copies only. Require the intended rejection reason."""
    folder = evidence / 'process-normal' / 'U1'
    failures = {}
    def reject(label, reason, test):
        try:
            test()
        except ValueError as error:
            check(str(error) == reason, 'mutant failed at wrong layer: ' + str(error))
            failures[label] = str(error)
        else:
            raise ValueError('mutant wrongly accepted: ' + label)

    def changed(table, field, value, reason, test):
        db = Tables(folder / 'final-local.sqlite.gz')
        o = Original(db)
        row = db[table][-1]
        obj = json.loads(row['descriptor'])
        dest = obj
        for key in field[:-1]:
            dest = dest[key]
        dest[field[-1]] = value
        row['descriptor'] = enc(obj).encode()
        reject(table + '.' + '.'.join(field), reason, lambda: test(db, o))
    changed('receipt_read_ordinals', ['Query', 'TargetKey'], 'other-namespace', 'exact physical-query binding', read_ordinals)
    changed('receipt_read_ordinals', ['Adopt'], False, 'read grant/provenance', read_ordinals)
    changed('receipt_read_ordinals', ['Query', 'Wire'], '{}', 'exact physical-query binding', read_ordinals)
    changed('receipt_read_ordinals', ['Prefix'], 2, 'ordered spend', read_ordinals)
    def positive(db, o):
        ords = read_ordinals(db, o)
        world = Tables(folder / 'final-world.sqlite.gz')
        reads = world_evidence(world, journal(folder / 'final-journal.jsonl'), o, ords)
        return receipts(db, o, ords, reads, True)
    changed('adapter_results', ['Candidate', 'HTTPStatus'], 200, 'reconstruction/receipt edge without invented original response', positive)
    changed('adapter_results', ['Reconciled', 'QueryID'], 'wrong', 'reconstruction/receipt edge without invented original response', positive)

    db = Tables(folder / 'final-local.sqlite.gz')
    o = Original(db)
    delivery = db.row('composition_outcome_deliveries', operation_id=o.grant['Invocation'])
    seq = db.row('observations', id=delivery['observation_id'])['consumed_sequence']
    db.row('history', subject='parent', sequence=seq)['selected_observation'] = None
    reject('parent-never-selected', 'observation consumption cross-link', lambda: parent(db, o, True))

    lost = evidence / 'process-normal' / 'held-lost'
    events = json.loads((lost / 'transcript.json').read_bytes())
    release = next(e for e in events if e['Event'] == 'release original packet')
    events.remove(release)
    events.insert(0, release)
    reject('release-before-absence', 'held order: release original packet', lambda: held_order(events, 'held-lost'))

    lost_db = Tables(lost / 'final-local.sqlite.gz')
    o = Original(lost_db)
    ords = read_ordinals(lost_db, o)
    stream = journal(lost / 'final-journal.jsonl')
    # Put the one genuine commit ahead of the authentic negative snapshot.
    app = single(e for e in stream if e['Event'] == 'committed')
    stream.remove(app)
    stream.insert(1, app)
    reject('application-before-negative-snapshot', 'snapshot truth at ordered world commit',
           lambda: world_evidence(Tables(lost / 'final-world.sqlite.gz'), stream, o, ords))

    immutable = evidence / 'public-checked' / 'normal' / 'immutable'
    pre, post = (Tables(immutable / (x + '.sqlite.gz')) for x in ('before', 'after'))
    o = Original(pre)
    row = post.row('adapter_results', intent_id=o.id)
    prior = json.loads(row['descriptor'])
    prior['Disposition'] = 'validated'
    row['descriptor'] = enc(prior).encode()
    reject('rewrite-historical-failure', 'immutable retained bytes in adapter_results', lambda: preserved(pre, post, o))
    return failures


def authority_case(folder):
    name = folder.name
    events = json.loads((folder / 'transcript.json').read_bytes())
    counts = Counter(e['Event'] for e in events)
    end = Tables(folder / 'final-local.sqlite.gz')
    orig = Tables(folder / 'original-cut-local.sqlite.gz')
    base = Original(orig)
    check(not orig['receipt_read_ordinals'] and not orig['adapter_results'], 'authority original missing')
    handoff(orig, base)
    grants = end.desc('reconciliation_profile')['Grants']
    ords = {}
    for g in grants:
        target = Original(end, g['Resolver']['Subject'])
        check(target.id == base.id and target.grant['Adopt'], 'authority exact shared original')
        ords.update(read_ordinals(end, target))
    for cut in ('original-cut', 'final'):
        for kind in ('world', 'local'):
            first, second = (Tables(folder / (cut + suffix + '-' + kind + '.sqlite.gz')) for suffix in ('', '-replayed'))
            check(first.same(second), 'authority offline replay')
        check((folder / (cut + '-journal.jsonl')).read_bytes() ==
              (folder / (cut + '-replayed-journal.jsonl')).read_bytes(), 'authority replay journal')
    world = Tables(folder / 'final-world.sqlite.gz')
    reads = world_evidence(world, journal(folder / 'final-journal.jsonl'), base, ords)
    check(len(world['applications']) == len(world['contacts']) == 1, 'authority business conservation')
    q = single(e['Detail'] for e in events if e['Event'] == 'successor original Q')
    check(q['Source'] == base.run and q['Namespace'] == base.ns, 'authority Q original')
    if len(grants) == 2:
        check(name in ('twin-first', 'resolver-first') and len(ords) == len(reads) == 2, 'two independently authorized reads')
        held = Tables(folder / 'both-responses-local.sqlite.gz')
        check(not held['adapter_results'] and len(held['receipt_read_ordinals']) == 2 and
              world.same(Tables(folder / 'both-responses-world.sqlite.gz')), 'two authentic responses before accounting')
        ids = {json.loads(qid)[0]: qid for qid, n in reads}
        first = ids[name.split('-')[0]]
        second = single(s for s in ids.values() if s != first)
        origin = end.desc('adapter_results', intent_id=base.id)
        receipt = json.loads(single(set(json.loads(res)['AppliedWire'] for res in reads.values())))
        check(receipt['Key'] == base.key and origin['Reconciled']['QueryID'] == first and
              not any(origin['Candidate'].values()) and
              end.row('settlements', intent_id=base.id)['descriptor'] == end.row('adapter_results', intent_id=base.id)['descriptor'],
              'one original import/settlement from the first return')
        for qid, disposition in [(first, 'adopted'), (second, 'duplicate')]:
            query = end.desc('adapter_results', intent_id=qid)
            ob, _, _ = select(end, end.row('settlements', intent_id=qid)['observation_id'])
            check(query['Read'] == {'Ordinal': 1, 'Adoption': disposition} and query['Candidate']['Response'] == reads[qid, 1]
                  and origin['Reconciled']['Receipt'] == json.loads(reads[qid, 1])['Receipt'] and ob['Operation'] == qid
                  and ob['Payload'] == query['Values'] and unpack(ob['Payload'])['adopted'] == (qid == first),
                  'separately selected winner and duplicate')
        check(evidence_order(end, first, 'external_result') < evidence_order(end, base.id, 'external_result') <
              evidence_order(end, second, 'external_result'), 'racing import order')
    else:
        check(json.loads(end.row('runs', subject='resolver')['projection'])['Status'] == 'cancelled', 'resolver closed')
        life = single(end['lifecycle_results'])
        check(json.loads(life['result'])['Accepted'] and json.loads(life['result'])['Request']['Intent']['Target'] == 'resolver',
              'authentic cancellation')
        if name == 'before-admission':
            check(not ords and not reads and not end['adapter_results'] and counts['resolver R before A'] == 1, 'revoked before read')
        else:
            qid, n = single(ords)
            spend = ords[qid, n]['Prefix'] + 1
            check(spend < evidence_order(end, life['id'], 'lifecycle') < evidence_order(end, qid, 'external_result') and
                  receipts(end, base, ords, reads, False) == [('found', 'adopted')], 'admitted read finishes after revocation')
            peer = single(e['Detail'] for e in events if e['Event'].startswith('peer committed and resolver R'))
            check(peer['Peer']['Projection']['Status'] == 'succeeded' and peer['Resolver']['Projection']['Status'] == 'cancelled'
                  and peer['Hold']['Boundary'] == 'resource.lookup' and json.loads(end.row('runs', subject='ordinary')['projection'])['Status'] == 'succeeded',
                  'ordinary peer progress under held lookup')
            for op in (qid, base.id):
                check(end.row('settlements', intent_id=op)['descriptor'] == end.row('adapter_results', intent_id=op)['descriptor'],
                      'late owed settlement')
            if name.startswith('sticky-'):
                rejected = Tables(folder / 'accounting-rejected-local.sqlite.gz')
                important = {r['intent_id'] for r in rejected['settlements'] if r['intent_id'] in (qid, base.id)}
                check(important == ({base.id} if name == 'sticky-query-settlement' else set()) and
                      rejected['adapter_results'] == end['adapter_results'] and
                      world.same(Tables(folder / 'accounting-rejected-world.sqlite.gz')), 'recovery from retained atomic evidence')
                errors = [e['Detail'] for e in events if e['Event'] == 'same-owner Q retained SQLite error']
                check(errors == ['directed receipt settlement abort'] * 2, 'sticky actual Q errors')
    return {'authorities': len(grants), 'ordinals': len(ords), 'snapshots': len(reads), 'replay_windows': 2}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--evidence', type=Path, default=Path(__file__).resolve().parents[1] / 'evidence')
    parser.add_argument('--process-normal', type=Path)
    parser.add_argument('--process-race', type=Path)
    parser.add_argument('--public-normal', type=Path)
    parser.add_argument('--public-race', type=Path)
    parser.add_argument('--authority-normal', type=Path)
    parser.add_argument('--authority-race', type=Path)
    parser.add_argument('--only', choices=['process', 'public', 'authority', 'ordered'])
    args = parser.parse_args()
    report = {'auditor_sha256': sha(Path(__file__).read_bytes()), 'campaigns': {}}
    for regime in ('normal', 'race'):
        for scope, audit in [('process', process_case), ('public', public_case), ('authority', authority_case), ('ordered', process_case)]:
            if args.only and args.only != scope:
                continue
            if scope == 'process':
                default = 'process-' + regime
                want = set(EXPECTED) - {'positive', 'immutable', 'read-only'}
            elif scope == 'public':
                default = 'public-checked/' + regime
                want = {'positive', 'immutable', 'read-only'}
            elif scope == 'authority':
                default = 'controls-' + regime
                want = {'before-admission', 'held-read', 'sticky-original-settlement', 'sticky-query-settlement', 'twin-first', 'resolver-first'}
            else:
                default = 'held-ordered-' + regime
                want = {'held-retained', 'held-lost'}
            folder = getattr(args, scope + '_' + regime, None) or args.evidence / default
            cases = [folder / case for case in sorted(want)]
            if scope == 'process':
                matched(folder)
            group = {'directory': folder.name, **campaign_digest(folder, cases), 'cases': {}}
            for p in cases:
                try:
                    if scope == 'ordered':
                        check((p / 'ingress-before-R-local.sqlite.gz').is_file(), 'missing ordered pre-R backup')
                    group['cases'][p.name] = audit(p, regime) if scope in ('process', 'ordered') else audit(p)
                except Exception as error:
                    raise RuntimeError(str(p) + ': ' + str(error)) from error
            report['campaigns'][scope + '-' + regime] = group
    if not args.only:
        report['rejected_mutants'] = attacks(args.evidence)
    print(json.dumps(report, indent=2))


if __name__ == '__main__':
    main()
