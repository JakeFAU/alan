"""Independent Phase 2D audit. Inputs are cold SQLite images, journals and process ACKs."""
import argparse
from collections import Counter, defaultdict
import gzip
import itertools
import json
from pathlib import Path
import runpy
import sqlite3
import subprocess
import tempfile
from types import SimpleNamespace

# Audited Phase 2C SQL reader, schema/ordinal verifier and canonical identity rules.
# No imports from the Phase 2D process, grid, grade, runtime or model implementations.
OLD = runpy.run_path(str(Path(__file__).resolve().parents[2] / 'phase2c/tools/audit_raw.py'))
Tables, check, enc, sha, identity, single, select, prep_schema, read_ordinals = (
    OLD[k] for k in ('Tables', 'check', 'enc', 'sha', 'identity', 'single', 'select', 'prep_schema', 'read_ordinals'))
ROLES = ('l', 'r')
CASES = ('forward', 'reverse', 'equal-reverse', 'stable-duplicate', 'stable-error', 'mixed', 'capacity-one',
         'read-only', 'retained-absence', 'finite-reads', 'lost-absence', 'spent', 'atomic', 'pre-retention',
         'original-selected', 'fresh-ack', 'manual-ack', 'auto-acks')


def unpack(fields):
    def value(v):
        if v['Kind'] == 'list':
            return [unpack(row) for row in (v.get('List') or [])]
        return v[{'string': 'Text', 'bool': 'Flag', 'int': 'Number'}[v['Kind']]]
    return {k: value(v) for k, v in fields.items()}


def read_json(p):
    raw = p.read_bytes()
    return json.loads(gzip.decompress(raw) if p.suffix == '.gz' else raw)


def project(d, who):
    return json.loads(d.row('runs', subject=who)['projection'])


def role_at(d, role):
    resolver = 'resolver_' + role
    g = single(g for g in d.desc('reconciliation_profile')['Grants'] if g['Resolver']['Subject'] == resolver)
    ns = d.row('metadata')['namespace']
    seq = ROLES.index(role) + 1
    inv = enc(['parent', 1, seq, 0])
    iid = identity('alan/invocation/v1', [ns, inv])
    src = identity('alan/invocation-member/v1', [iid, 'source'])
    run = {'Subject': src, 'Generation': 1}
    op = enc([src, 1, 2, 0])
    key = enc([2, ns, op])
    check(g['Invocation'] == inv and g['Member'] == 'source' and g['Sequence'] == 2 and
          g['Resolver'] == {'Subject': resolver, 'Generation': 1}, 'exact designated grant')
    actual = d.row('composition_invocations', id=iid, operation_id=inv)
    check(d.row('composition_members', subject=src)['invocation_id'] == iid and d.desc('composition_templates', id=actual['template_id'])['Handoff'] ==
          {'Caller': {'Subject': 'parent', 'Generation': 1}, 'Sequence': seq, 'CallerPort': role + '_handoff', 'EntryPort': 'handoff'}, 'bound issuer occurrence')
    it = d.desc('intents', id=op)
    attempt_rows = d.where('adapter_attempts', intent_id=op)
    o = SimpleNamespace(role=role, name=resolver, source=src, run=run, ns=ns, inv=inv, id=op, key=key,
                        iid=iid, grant=g, resolver=g['Resolver'], delta=unpack(it['Payload'])['delta'],
                        ledger=g['Endpoint'].split('/')[-2], table='adapter_attempts', wire=None, intent=it)
    if attempt_rows:
        o.attempt = json.loads(single(attempt_rows)['descriptor'])
        o.auth = o.attempt['AuthorizationID']
        if o.attempt['Allowed']:
            OLD['verify_attempt'](d, op, o.attempt)
            p = o.attempt['Prepared']
            prep_schema(p, {'delta': ('int', 0)}, {'mutation': ('int', 0), 'total': ('int', 0)})
            check(p['Request']['ID'] == op and p['Request']['Key'] == key and p['Endpoint'] == g['Endpoint'] and
                  p['Protocol'] == 'phase2c/counter/v1' and p['Model'] == 'success', 'versioned counter preparation')
            o.wire = enc({'Key': key, 'Profile': 'idempotent', 'Delta': o.delta})
            check(o.wire == p['Wire'], 'frozen original wire')
    terminal = d.where('terminal_obligations', subject=src)
    o.terminal = terminal[0]['sequence'] if terminal else None
    return o


def cost(d, roles):
    roots = {r['id']: json.loads(r['descriptor']) for r in d['composition_roots']}
    root = {k: r['Run']['Subject'] for k, r in roots.items()}
    total = {r: dict(invocations=0, runs=0, orig=0, queries=0) for r in root.values()}
    expected = {}
    for r in d['composition_admission_decisions']:
        a = json.loads(r['descriptor'])
        if a['Accepted']:
            expected[r['operation_id'], 'invocation'] = (r['root_id'], 1, len(a['Members']), 0)
    for r in d['composition_reservations']:
        if json.loads(r['descriptor'])['Allowed']:
            expected[r['operation_id'], 'effect'] = (r['root_id'], 0, 0, 1)
    seen = set()
    for r in d['composition_charges']:
        c = json.loads(r['descriptor'])
        key = (r['operation_id'], r['kind'])
        check(key == (c['OperationID'], c['Kind']) and c['RootID'] == r['root_id'] and c['DecisionOrder'] == r['decision_order'] and
              expected.get(key) == (c['RootID'], c['Invocations'], c['CreatedRuns'], c['Effects']), 'debit matches actual admission/reservation')
        seen.add(key)
        totals = total[root[c['RootID']]]
        totals['invocations'] += c['Invocations']
        totals['runs'] += c['CreatedRuns']
        totals['orig' if key[0] in [o.id for o in roles] else 'queries'] += c['Effects']
    check(seen == set(expected) and sum(x['runs'] for x in total.values()) == len(d['composition_members']), 'complete cost ledger')
    for k, desc in roots.items():
        t, lim = total[root[k]], desc['Limits']
        t['limits'] = lim
        t['remaining'] = [lim['MaxInvocations'] - t['invocations'], lim['MaxCreatedRuns'] - t['runs'],
                          lim['MaxEffects'] - t['orig'] - t['queries']]
        check(min(t['remaining']) >= 0, 'root allowances')
    return total


def world(w, stream, roles, ords):
    by_key = {o.key: o for o in roles}
    ing, applied, query_bytes = {}, {}, {}
    contacts, business, query_contacts = [], Counter(), Counter()
    for e in stream:
        call = (e['PID'], e['Process'], e['Contact'])
        if e['Event'] == 'ingress':
            check(call not in ing and e['Method'] == 'POST', 'unique actual ingress')
            ing[call] = e
            p = json.loads(e['Body'])
            if e['Path'].endswith('/apply'):
                o = by_key[p['Key']]
                check(e['Body'] == o.wire and e['Path'] == '/phase2c/' + o.ledger + '/apply', 'business ingress exact wire')
                business[o.role] += 1
            else:
                o = by_key[p['TargetKey']]
                check(ords[p['QueryID'], p['Ordinal']]['Query'] == p and e['Path'] == '/phase2c/' + o.ledger + '/lookup', 'lookup ingress exact bound query')
                query_contacts[o.role] += 1
        elif e['Event'] == 'committed':
            req, receipt = ing[call], e['Receipt']
            o = by_key[receipt['Key']]
            check(o.id not in applied and req['Body'] == o.wire and req['Path'].endswith('/apply') and
                  receipt == {'Key': o.key, 'Profile': 'idempotent', 'Delta': o.delta, 'Mutation': len(applied) + 1,
                              'Total': sum(r['Delta'] for r in applied.values()) + o.delta, 'Process': e['Process'], 'PID': e['PID']}, 'world receipt order and running total')
            applied[o.id] = receipt
            contacts.append(req['Body'].encode())
            check(w.row('application_wires', number=receipt['Mutation'])['wire'] == o.wire.encode() and
                  json.loads(w.row('applications', number=receipt['Mutation'])['receipt']) == receipt, 'independent ledger application')
        else:
            check(e['Event'] == 'snapshot', 'world event')
            req = ing[call]
            q = json.loads(req['Body'])
            receipt = applied.get(q['TargetID'])
            snap = {'Query': q, 'Number': len(query_bytes) + 1, 'Status': 'found' if receipt else 'not_found', 'Count': int(bool(receipt)),
                    'AppliedWire': q['Wire'] if receipt else '', 'Receipt': receipt}
            check(e['Snapshot'] == json.loads(e['Response']) == snap and w.row('ledger_identity')['id'] == q['LedgerID'], 'snapshot truth at real application prefix')
            row = w.row('receipt_reads', number=snap['Number'])
            check(row['request'] == req['Body'].encode() and row['response'] == e['Response'].encode(), 'exact read wire and response')
            query_bytes[q['QueryID'], q['Ordinal']] = e['Response']
    check(len(w['applications']) == len(applied) and len(w['receipt_reads']) == len(query_bytes) and
          [r['request'] for r in w['contacts']] == contacts, 'world conservation')
    return applied, query_bytes, business, query_contacts


def observed(d, who):
    obs = []
    for h in d.where('history', subject=who):
        if h['selected_observation']:
            ob, state, seq = select(d, h['selected_observation'])
            check(ob['Run']['Subject'] == who, 'observation target')
            obs.append((seq, ob))
    return obs


def handoff(d, o):
    for sender, receiver, obligation in [(o.source, 'parent', 'identity'), ('parent', o.name, 'relay'),
                                         (o.name, 'parent', 'relay'), ('parent', o.source, 'identity')]:
        if o.terminal:
            v, seq = OLD['message'](d, sender, receiver, o.role + '_' + obligation)
            check(v == {'target': o.id} and (receiver != o.source or seq < o.terminal), 'role-specific original handoff before End')
    if o.terminal:
        freeze = d.desc('composition_freezes', scope_id=o.iid)
        check(freeze['ExternalIntents'] == [o.id] and not freeze['InvocationIntents'] and len(freeze['ReplyIntents']) == 1, 'issuer captured its own original')
        reply = d.desc('composition_replies', operation_id=freeze['ReplyIntents'][0])
        check(reply['Accepted'] and unpack(reply['Intent']['Payload'])['target'] == o.id, 'frozen original issuer reply')


def query_evidence(d, o, reads, ords, applied):
    selected = {}
    for seq, ob in observed(d, o.name):
        if ob['Provenance'] == 'local-input':
            check(not ob['Payload'], 'empty scheduling input')
        if ob['Provenance'] == 'runtime-adapter':
            selected[ob['Operation']] = ob
    results = {r['intent_id']: json.loads(r['descriptor']) for r in d['adapter_results']}
    selected_fields = []
    for qid in sorted({r['intent_id'] for r in d['adapter_attempts'] if json.loads(r['intent_id'])[0] == o.name}, key=lambda key:json.loads(key)[2]):
        if qid not in results:
            continue
        r = results[qid]
        c, v = r['Candidate'], unpack(r['Values'])
        check(c['ID'] == qid and c['Key'] == enc([2, o.ns, qid]), 'query transport identity')
        if r['Disposition'] == 'validated':
            n = r['Read']['Ordinal']
            check(c['HTTPStatus'] == 200 and reads[qid, n] == c['Response'], 'retained real snapshot')
            s = json.loads(c['Response'])
            prior = results.get(o.id, {})
            proof = prior.get('Reconciled')
            if s['Receipt'] is None:
                decision = 'none'
            elif not o.grant['Adopt']:
                decision = 'denied'
            elif proof:
                decision = 'adopted' if proof['QueryID'] == qid else 'duplicate'
            else:
                check(prior, 'found with import grant cannot pass as existing without prior result')
                decision = 'existing'
            rec = s['Receipt'] or {'Mutation': 0, 'Total': 0}
            check(r['Read']['Adoption'] == decision and v == {'valid': True, 'disposition':'validated', 'adopted':decision=='adopted',
                  'adoption':decision,'mutation':rec['Mutation'],'total':rec['Total'],'status':s['Status'],'target':o.id}, 'import policy and query fields')
            if decision == 'adopted':
                check(proof == {'Target':o.id,'QueryID':qid,'Ordinal':n,'Receipt':s['Receipt']} and not any(prior['Candidate'].values())
                      and unpack(prior['Values']) == {'valid':True,'disposition':'validated','mutation':rec['Mutation'],'total':rec['Total']}, 'actual local reconstruction')
        else:
            check(not v['valid'] and not v['adopted'] and not v['status'] and not v['target'] and v['disposition'] == c['Disposition'], 'unsuccessful query defaults')
        if qid in selected:
            check(selected[qid]['Payload'] == r['Values'] and d.row('settlements',intent_id=qid)['descriptor'] ==
                  d.row('adapter_results',intent_id=qid)['descriptor'], 'selected own settled query')
            selected_fields.append((qid, v))
    orig = results.get(o.id, {})
    if orig and not orig.get('Reconciled') and orig['Disposition'] == 'validated':
        expected = applied[o.id]
        check(orig['Candidate']['Key'] == o.key and json.loads(orig['Candidate']['Response']) == expected, 'direct original application')
    return selected_fields, orig


def reports(d, o, queries):
    rows = [json.loads(r['descriptor']) for r in d.where('intents',subject=o.name)]
    sent = [r for r in rows if r['Kind']=='message' and r['Obligation']==o.role+'_report']
    if not sent:
        return None
    msg = single(sent)
    v = unpack(msg['Payload'])
    check(v['target']==o.id and v['queries']==len(queries) and msg['Target']=='parent', 'designated report source/target')
    for i in (1,2):
        prefix='query'+str(i)+'_'
        if i<=len(queries):
            qid, values=queries[i-1]
            check(v[prefix+'id']==qid and all(v[prefix+k]==val for k,val in values.items()), 'report describes actually selected query')
        else:
            check(all(not val for k,val in v.items() if k.startswith(prefix)), 'unused query slot')
    found = [(key,val) for key,val in queries if val['valid'] and val['status']=='found' and val['target']==o.id]
    expected_conflict = any(val['status']=='conflict' or val['adoption']=='conflict' for _,val in queries) or len({(v['mutation'],v['total']) for _,v in found})>1
    check(v['witness']==bool(found) and v['conflict']==expected_conflict,'stable found/conflict flags')
    if found:
        key, first=found[0]
        check(v['witness_query']==key and all(v['witness_'+k]==first[k] for k in ('target','adoption','mutation','total')), 'first compatible witness survives refresh')
    else:
        check(not any(v[k] for k in v if k.startswith('witness_')), 'no invented witness')
    return msg


def selected_operation(d, operation):
    delivery = d.where('composition_outcome_deliveries', operation_id=operation)
    if not delivery:
        return None
    ob = d.row('observations', id=single(delivery)['observation_id'])
    if ob['consumed_sequence'] is None:
        return None
    data, _, seq = select(d, ob['id'])
    check(data['Operation'] == operation and data['Provenance'] == 'runtime-invocation', 'actual invocation consumption')
    pub=d.desc('composition_publications',operation_id=operation)
    payload=unpack(data['Payload'])
    check(payload['status']==pub['Status'] and payload['reason']==pub['Reason'] and payload['response_present']==pub['ResponsePresent'] and
          all(payload['response.'+k]==v for k,v in unpack(pub['Response'] or {}).items()), 'selected envelope fields')
    return seq


def waiting(d, name, op):
    who=d.row('runs', subject=name)
    p=json.loads(who['projection'])
    ins=json.loads(d.row('programs',id=who['program_id'])['bytes'])['Code'][p['PC']]
    check(p['Status']=='open' and ins['Op']=='receive' and ins['Match']['Kind']=='invocation' and
          unpack(p['Data'])[ins['Match']['Operation']['Ref']]==op,'pending ordered Await frontier')


def tree(d, check_grid):
    top=enc(['parent',1,3,0])
    req=unpack(d.desc('intents', id=top)['Payload'])
    leaves=[]
    def visit(op):
        a=d.desc('composition_admission_decisions', operation_id=op)
        who=single(a['Members'])['Subject']
        p=project(d,who)
        children={i['Obligation']:i['ID'] for row in d.where('intents',subject=who)
                  if (i:=json.loads(row['descriptor']))['Kind']=='invocation'}
        if set(children)=={'candidate','successor'}:
            args=unpack(d.desc('intents',id=op)['Payload'])
            mode=args['mode']
            c=children['candidate']
            ca=d.desc('composition_admission_decisions',operation_id=c)
            code=d.row('runs',subject=single(ca['Members'])['Subject'])['program_id']
            leaves.append(code)
            if mode=='pending':
                waiting(d,who,c)
                check(not d.where('composition_publications',operation_id=c) and selected_operation(d,c) is None,'pending candidate')
                out=None
            else:
                x=d.desc('composition_publications',operation_id=c)
                check(x['Reason']=={'answer':'member_succeeded','fail':'member_failed','absent':'missing_reply'}[mode]
                      and selected_operation(d,c) is not None,'candidate outcome')
                if mode=='answer':
                    check(unpack(x['Response'])=={'id':args['id'],'value':1-args['value']},'candidate complement')
                out={'id':args['id'],'classification':{'answer':'answer','fail':'candidate_failed','absent':'candidate_missing_reply'}[mode],
                     'answers':[{'id':args['id'],'value':1-args['value']}] if mode=='answer' else []}
            check(selected_operation(d,children['successor']) is not None,'actual successor selection')
        else:
            children_list=list(children.items())
            check(len(children_list) in (1,2),'tree occurrence arity')
            sub=[(name, child, visit(child)) for name,child in children_list]
            if any(p is None for _,_,p in sub):
                first=next(i for i,(_,_,p) in enumerate(sub) if p is None)
                waiting(d,who,sub[first][1])
                check(all((selected_operation(d,ch) is not None)==(i<first) for i,(_,ch,_) in enumerate(sub)),
                      'buffered completed sibling does not cross earlier frontier')
                out=None
            else:
                check(all(selected_operation(d,ch) is not None for _,ch,_ in sub),'complete ordered subtree join')
                out={('' if len(sub)==1 else {'left':'l_','right':'r_'}[n])+k:v for n,_,subreport in sub for k,v in subreport.items()}
        pubs=d.where('composition_publications',operation_id=op)
        if out is None:
            check(not pubs,'pending tree has no publication')
        else:
            pub=json.loads(single(pubs)['descriptor'])
            check(pub['ResponsePresent'] and unpack(pub['Response'])==out and pub['Status']==
                  ('failed' if any(k.endswith('classification') and v!='answer' for k,v in out.items()) else 'succeeded'),'role-indexed pure product')
        return out
    if check_grid:
        return visit(top),leaves,req
    return None,[],req


def final_frame(d,roles,summaries, grid_name=None):
    p=project(d,'parent')
    data=unpack(p['Data'])
    actual=[(seq,ob) for seq,ob in observed(d,'parent') if ob['Obligation'] in ('l_issue','r_issue','tree','l_report','r_report')]
    check([ob['Obligation'] for _,ob in actual]==['l_issue','r_issue','tree','l_report','r_report'][:len(actual)],'parent selects a prefix of the ordered five-way join')
    for seq,ob in actual:
        if ob['Provenance']=='runtime-invocation':
            check(selected_operation(d,ob['Operation'])==seq,'parent selected exact original handle')
        else:
            role=ob['Obligation'][0]
            report=summaries[role]['report']
            check(ob['Source']=='resolver_'+role and ob['Provenance']=='ordinary-message' and report and ob['Payload']==report['Payload'],
                  'parent ranked report has designated evidence')
    if 'decision' in data:
        check(len(actual)==5,'private decision requires all five actual selections')
        for o in roles:
            report=unpack(summaries[o.role]['report']['Payload'])
            check(data[o.role+'_report_target']==report['target']==data[o.role+'_target']==o.id and data[o.role+'_issued_target']==o.id,
                  'genuine role binding in public decision')
            if data['decision'].startswith('applied'):
                check(report['witness'] and not report['conflict'] and report['witness_adoption'] in ('adopted','existing','duplicate'),
                      'complete compatible pair')
            pub=d.desc('composition_publications',operation_id=o.inv)
            response=unpack(pub['Response'])
            check(response['issued'] and response['target']==o.id and data[o.role+'_source_status']==pub['Status'],'immutable original class in decision')
            if response['observed']:
                orig=summaries[o.role]['original']
                check(orig and response['interaction']==orig['Disposition'] and response['valid']==unpack(orig['Values'])['valid']
                      and response['mutation']==unpack(orig['Values'])['mutation'] and response['total']==unpack(orig['Values'])['total'], 'selected direct original vs later witness')
        if data['decision'].startswith('applied'):
            pairs=sorted((data[o.role+'_report_witness_mutation'],data[o.role+'_report_witness_total'],o.delta) for o in roles)
            check([p[0] for p in pairs]==[1,2] and pairs[0][1]==pairs[0][2] and pairs[1][1]==sum(o.delta for o in roles),'legal world pair')
    # Check the executed consumer rather than inferring a join from the last label.
    run=d.row('runs',subject='parent')
    prog=d.row('programs',id=run['program_id'])['bytes']
    check(sha(prog)==run['program_id'],'retained parent program identity')
    code=json.loads(prog)['Code']
    previous=json.loads(run['initial'])
    for h in d.where('history',subject='parent'):
        step=code[previous['PC']]
        now=json.loads(h['successor'])
        if step['Op']=='branch':
            def operand(expr):
                return previous['Data'][expr['Ref']] if expr['Ref'] else expr['Literal']
            check(now['PC']==step['Target' if operand(step['A'])==operand(step['B']) else 'Otherwise'],
                  'executed role/evidence conditional')
        if step['Op']=='receive' and h['selected_observation']:
            ob=d.desc('observations',id=h['selected_observation'])
            for source,dest in (step['Bind'] or {}).items():
                check(now['Data'][dest]==ob['Payload'][source],'actual selected field assignment')
        previous=now
    acked={}
    for o in roles:
        ack=[(q,ob) for q,ob in observed(d,o.name) if ob['Obligation']==o.role+'_report']
        if ack:
            v,_=OLD['message'](d,'parent',o.name,o.role+'_report')
            check(v=={'target':o.id},'resolver final ACK targets its own report')
            i=single(json.loads(r['descriptor']) for r in d.where('intents',subject='parent') if json.loads(r['descriptor'])['Obligation']==o.role+'_report')
            a=json.loads(d.row('attempts',intent_id=i['ID'])['result'])
            check(a['Allowed'] and json.loads(d.row('history',subject='parent',sequence=a['Source']['Sequence'])['successor'])['Status']=='open',
                  'final ACK authorized before parent End')
        rp=project(d,o.name)
        if rp['Status']=='succeeded':
            check(ack and single(ack)[0]<d.row('runs',subject=o.name)['sequence'],'resolver actually consumed ACK before End')
        acked[o.role]=bool(ack)
    return data.get('decision'),len(actual),acked


def sole_ack(d, role):
    intents=[json.loads(r['descriptor']) for r in d['intents']]
    pending=[i for i in intents if i['Kind']=='message' and not d.where('message_acknowledgments',intent_id=i['ID'])]
    msg=single(pending)
    check(msg['Run']['Subject']=='parent' and msg['Target']=='resolver_'+role and msg['Obligation']==role+'_report' and
          not d.where('attempts',intent_id=msg['ID']),'sole pending final ACK before authorization')
    check(all(d.where('composition_outcome_deliveries',operation_id=i['ID']) for i in intents if i['Kind']=='invocation') and
          all(i['Kind']!='lifecycle' for i in intents),'other internal work exhausted')
    for row in d['terminal_obligations']:
        check(len(json.loads(row['destinations']))==sum(json.loads(row['delivery_status']).values()),'no pending terminal subscriptions')
    check(project(d,'parent')['Status']=='open' and unpack(project(d,'parent')['Data'])['decision']=='applied','open parent with prior private decision')


def snapshot(folder,label,events,stream,grid=False):
    d=Tables(folder/(label+'-local.sqlite.gz'))
    w=Tables(folder/(label+'-world.sqlite.gz'))
    roles=[role_at(d,r) for r in ROLES]
    check(roles[0].id!=roles[1].id and roles[0].ledger==roles[1].ledger,'distinct effects one ledger')
    ords={}
    for o in roles:
        ords.update(read_ordinals(d,o))
    apps,reads,bc,qc=world(w,stream,roles,ords)
    ledger=cost(d,roles)
    summaries={}
    for o in roles:
        handoff(d,o)
        selected,original=query_evidence(d,o,reads,ords,apps)
        report=reports(d,o,selected)
        summaries[o.role]={'id':o.id,'queries':[(qid,v['status'],v['disposition'],v['adoption']) for qid,v in selected],
                           'original':original,'report':report,'receipt':apps.get(o.id)}
    decision,join,acks=final_frame(d,roles,summaries)
    if label in ('ack-committed','first-ack-committed'):
        sole_ack(d,'r' if label=='ack-committed' else 'l')
    detail={}
    for o in roles:
        v=summaries[o.role];report=v['report']
        detail[o.role]={'id':o.id,'queries':v['queries'],'receipt':v['receipt'],'original':v['original'].get('Disposition'),
                        'reported':bool(report),'settled':bool(d.where('settlements',intent_id=o.id)),
                        'published':bool(d.where('composition_publications',operation_id=o.inv)),
                        'ordinal_count':sum(key[0].startswith('["'+o.name+'"') for key in ords),
                        'contacts':[bc[o.role],qc[o.role]],'ack':acks[o.role]}
    result={'decision':decision,'join':join,'roles':detail,'roots':ledger,'root_publications':len(d['composition_root_publications'])}
    if label=='final' and grid:
        pieces=folder.name.split('-')
        deficit=next((k for k in ('invocations','runs','depth','effects') if k in pieces),'')
        if not deficit:
            product,leaves,req=tree(d,True)
            expected=None if 'pending' in pieces[1:3] else ('applied' if pieces[1:3]==['answer','answer'] else 'applied_tree_diagnosed')
            check(decision==expected and join==(2 if expected is None else 5) and
                  all(s['settled'] and s['reported'] and s['ack']==(expected is not None) for s in detail.values()),'grid public frontier')
            limits=ledger['parent']['limits']
            relay=any(w in pieces for w in ('above','left','right'))
            check(limits['MaxInvocations']==limits['MaxCreatedRuns']==11+int(relay) and limits['MaxDepth']==4+int(relay) and
                  limits['MaxEffects']==2 and limits['MaxInFlight']==2 and all(ledger[o.name]['limits']['MaxInFlight']==1 for o in roles),
                  'derived common resource envelope')
            if not expected:
                for o in roles:
                    rep=summaries[o.role]['report']
                    admit=single(row for row in d['admissions'] if json.loads(row['descriptor'])['IntentID']==rep['ID'])
                    check(json.loads(admit['result'])['Accepted'] and d.row('observations',id=admit['observation_id'])['consumed_sequence'] is None,
                          'both real resolver reports admitted but buffered at T')
                waiting(d,'parent',enc(['parent',1,3,0]))
            check(req['l_mode']==pieces[1] and req['r_mode']==pieces[2] and req['l_id']==1 and req['r_id']==(1 if pieces[0] in ('AA','AB') else 2),
                  'grid literals and ordered modes')
            result['leaves']=leaves
        else:
            reason={'invocations':'invocation_limit','runs':'created_run_limit','depth':'depth_limit','effects':'effect_limit'}[deficit]
            refused=[a['Reason'] for t in ('composition_admission_decisions','adapter_attempts') for row in d[t]
                     if not (a:=json.loads(row['descriptor'])).get('Accepted',a.get('Allowed'))]
            check(reason in refused and ledger['parent']['limits'][{'invocations':'MaxInvocations','runs':'MaxCreatedRuns','depth':'MaxDepth','effects':'MaxEffects'}[deficit]]=={'invocations':10,'runs':10,'depth':3,'effects':1}[deficit], 'real bound refusal')
    return result


def campaign_case(folder, prefix=None, grid=False):
    process_file=next(folder.glob('processes.json*'))
    events=read_json(process_file)
    stream=[json.loads(x) for x in (folder/'journal.jsonl').read_bytes().splitlines()]
    cuts={}
    acks=defaultdict(list)
    processes={}
    deaths=[]
    inherited=[] if prefix is None else prefix
    for row in events:
        tag,x=row['event'],row['data']
        if tag=='spawn':
            check(x['pid'] not in processes and len(x['sha256'])==64,'fresh executable PID/hash')
            processes[x['pid']]=x
        elif tag=='ack':
            check(x['pid'] in processes and x['namespace'] and x['event']['Generation']==1,'owner acknowledgment')
            acks[x['pid']].append(x)
        elif tag=='wait' and x['exit']==-9:
            check(acks[x['pid']],'acknowledged kernel kill')
            deaths.append(acks[x['pid']][-1]['cut'])
        elif tag=='backup':
            lab=x['label']
            cuts[lab]=snapshot(folder,lab,events,inherited+stream[:x['journal_lines']],grid)
        elif tag=='offline-replay':
            local=sha(gzip.decompress((folder/'final-local.sqlite.gz').read_bytes()))
            check(x['before']==x['after']==[local,sha((folder/'journal.jsonl').read_bytes())],'offline replay genuine backup and journal identity')
    check(set(cuts)=={p.name.removesuffix('-local.sqlite.gz') for p in folder.glob('*-local.sqlite.gz')},'archive coverage')
    terminal_cut='final' if 'final' in cuts else 'fork'
    last=Tables(folder/(terminal_cut+'-local.sqlite.gz'))
    bypid={x['pid']:x for x in processes.values()}
    for pid,steps in acks.items():
        for a in steps:
            event=a['event'];who=event['Subject'];rr=last.row('runs',subject=who,generation=event['Generation'])
            check(a['namespace']==last.row('metadata')['namespace'] and bypid[pid]['argv'] and len(bypid[pid]['sha256'])==64,'process ack binds namespace and owner')
            if a['cut']=='issuer-before-end':
                check(event['Boundary']=='execution.staged' and event['Phase']=='terminal' and event['Sequence']==last.row('terminal_obligations',subject=who)['sequence'],
                      'real original terminal cut')
            elif event['Intent']:
                intent=last.desc('intents',id=event['Intent'])
                if event['Boundary'].startswith(('adapter.','dispatch.')):
                    check(intent['Run']['Subject']==who and event['Attempt']==enc(['authorization',intent['ID']]) and
                          any(json.loads(o['descriptor'])['Resolver']['Epoch']==event['Epoch'] for o in last.where('receipt_read_ordinals',intent_id=intent['ID'])),
                          'real admitted physical read cut')
                elif a['cut'] in ('both-final-acks','last-final-ack'):
                    check(intent['Run']['Subject']==who=='parent' and intent['Sequence']==event['Sequence'] and intent['Obligation'] in ('l_report','r_report'),
                          'committed final ACK hook')
    if 'final' not in cuts:
        return cuts
    end=cuts['final'];name=folder.name
    if grid:
        return {'final':end,'deaths':deaths}
    left,right=end['roles']['l'],end['roles']['r']
    if name=='capacity-one':
        check(right['original'] is not None or not right['settled'],'capacity refusal')
        d=Tables(folder/'final-local.sqlite.gz');denied=d.desc('adapter_attempts',intent_id=right['id'])
        check(not denied['Allowed'] and right['contacts'][0]==0 and left['contacts'][0]==1,'closed second source in shared slot')
    elif name in CASES or prefix is not None:
        unresolved=name in ('spent','finite-reads','read-only')
        check(end['decision']==(None if unresolved else 'applied') and end['join']==(1 if unresolved else 5),'expected actual parent frontier')
        if not unresolved:
            check(left['published'] and right['published'] and left['ack'] and (right['ack']==(name!='manual-ack')) and
                  end['root_publications']==(2 if name=='manual-ack' else 3),'completion versus manual ACK starvation')
        check(left['receipt']['Mutation']==(2 if name in ('reverse','equal-reverse') else 1),'original application order')
        if name=='equal-reverse':
            check(left['receipt']['Delta']==right['receipt']['Delta']==1 and left['receipt']['Key']!=right['receipt']['Key'],'equal payload distinct role')
        if name=='manual-ack':
            d=Tables(folder/'final-local.sqlite.gz');msg=single(json.loads(r['descriptor']) for r in d.where('intents',subject='parent') if json.loads(r['descriptor'])['Obligation']=='r_report')
            check(json.loads(d.row('attempts',intent_id=msg['ID'])['result'])['Reason']=='source_closed','manual End blocks final ACK authority')
        if name in ('atomic','pre-retention','lost-absence','spent'):
            first=cuts['ordinal-1']['roles']['r']
            if name=='atomic':check(first['original']=='validated' and not first['settled'] and not first['published'],'atomic evidence precedes local settlement')
            else:check(not first['original'] and not first['queries'],'pre-retention has only spend')
        if name=='spent':
            check(right['ordinal_count']==2 and right['contacts'][1]==0 and right['queries'][-1][2]=='read_exhausted','exhaustion across owners')
        if name in ('stable-error','stable-duplicate','retained-absence','finite-reads','lost-absence','read-only'):
            expected={'stable-error':[('found','adopted'),('','')],'stable-duplicate':[('found','adopted'),('found','duplicate')],
                      'retained-absence':[('not_found','none'),('found','adopted')],'finite-reads':[('not_found','none'),('not_found','none')],
                      'lost-absence':[('found','adopted')],'read-only':[('found','denied')]}[name]
            check([(v[1],v[3]) for v in right['queries']]==expected,'stable/absent/read-only selected results')
        if name=='mixed':check(left['original']=='transport_error' and left['queries'][0][3]=='existing' and right['queries'][0][3]=='adopted','mixed originals')
        if name=='original-selected':check(cuts['left-original-selected']['join']==1,'selected L before report')
        if 'left-only' in cuts:
            cut=cuts['left-only']['roles']
            check(cut['l']['settled'] and cut['l']['published'] and not cut['r']['settled'] and not cut['r']['published'],'separate missing R after L import')
    return {'cuts':cuts,'deaths':deaths}


def paired(folder):
    prefix=folder/'prefix'
    first=campaign_case(prefix)
    inherited=[json.loads(x) for x in (prefix/'journal.jsonl').read_bytes().splitlines()]
    coupled={}
    local=Tables(prefix/'fork-local.sqlite.gz')
    for name in ('continuous','lost-response'):
        p=folder/name
        e=single(x['data'] for x in read_json(p/'processes.json') if x['event']=='forked')
        check(e=={'local':sha((prefix/'fork-local.sqlite.gz').read_bytes()),'world':sha((prefix/'fork-world.sqlite.gz').read_bytes()),
                  'prefix_journal':sha((prefix/'journal.jsonl').read_bytes())},'identical frozen fork inputs')
        end=Tables(p/'final-local.sqlite.gz')
        for table in ('history','intents','attempts','composition_reservations','composition_charges','adapter_results','settlements','receipt_read_ordinals'):
            check(all(row in end[table] for row in local[table]),'retained fork prefix')
        coupled[name]=campaign_case(p,inherited)['cuts']['final']
    a,b=coupled.values()
    for role in ROLES:
        for k in ('id','receipt','published','reported','original','ack'):
            check(a['roles'][role][k]==b['roles'][role][k],'matched business projection')
    check(a['roots']==b['roots'] and a['roles']['r']['ordinal_count']==1 and b['roles']['r']['ordinal_count']==2 and
          a['roles']['r']['contacts']==[1,1] and b['roles']['r']['contacts']==[1,2],'visible unequal physical cost')
    return coupled


def grid_audit(folder):
    modes=('answer','fail','absent','pending')
    base={'-'.join([a+b,l,r]) for a,b,l,r in itertools.product('AB','AB',modes,modes)}
    extra={'AB-answer-answer-above','BA-fail-answer-left','BB-answer-pending-right','BB-answer-answer-above-slow'}
    extra.update('BB-answer-answer-'+k for k in ('invocations','runs','effects','depth'))
    cases={}
    for name in sorted(base|extra):
        result=campaign_case(folder/name,grid=True)
        q=result['final']
        if 'leaves' in q:
            programs={'A':'74152796cc6bb845eee44cc19813210a85ee26173629dd76e446fd042a85d5b7','B':'ca5ab7efdb73587e2664e02f9cea2928fde178190768aa844a2c88e14c90aa67'}
            check(q['leaves']==[programs[c] for c in name[:2]],'actual Series 1 leaf placement')
        cases[name]=q
    check(Counter(cases[n]['decision'] for n in base)=={None:28,'applied':4,'applied_tree_diagnosed':32},'base grid products')
    return cases


def replay_attacks(source, consumer):
    """Attack real Store copies. Repair self-digests and ordered evidence hashes where present."""
    data=gzip.decompress((source/'final-local.sqlite.gz').read_bytes())
    journal=[json.loads(line) for line in (source/'journal.jsonl').read_bytes().splitlines()]
    source_db=Tables(source/'final-local.sqlite.gz')
    l,r=[role_at(source_db,s) for s in ROLES]
    ql=single(row['intent_id'] for row in source_db['adapter_attempts'] if json.loads(row['intent_id'])[0]==l.name)
    qr=single(row['intent_id'] for row in source_db['adapter_attempts'] if json.loads(row['intent_id'])[0]==r.name)
    out={}
    with tempfile.TemporaryDirectory() as name:
        root=Path(name)
        def invoke(path):
            return subprocess.run([str(consumer.resolve()),'replay','-db',str(path)],capture_output=True,timeout=45)
        path=root/'baseline.sqlite';path.write_bytes(data)
        baseline=invoke(path)
        check(baseline.returncode==0 and sha(path.read_bytes())==sha(data),'real CLI baseline')
        for attack in ('swapped-genuine-receipt','rehashed-false-query-status','rehashed-false-import','rehashed-free-read'):
            path=root/(attack+'.sqlite');path.write_bytes(data)
            with sqlite3.connect(path) as db:
                def update_result(key,change):
                    desc=source_db.desc('adapter_results',intent_id=key)
                    change(desc)
                    desc['Digest']=sha(enc(dict(desc,Digest='')).encode())
                    raw=enc(desc).encode()
                    db.execute('update adapter_results set descriptor=? where intent_id=?',(raw,key))
                    db.execute('update settlements set descriptor=? where intent_id=?',(raw,key))
                    fact=single(row for row in source_db['composition_facts'] if row['kind']=='external_result' and json.loads(row['descriptor'])['OperationID']==key)
                    f=json.loads(fact['descriptor'])
                    f['RecordDigest']=identity('alan/external_result-row/v1',desc)
                    db.execute('update composition_facts set descriptor=? where decision_order=?',(enc(f).encode(),fact['decision_order']))
                if attack=='swapped-genuine-receipt':
                    def wrong_role(desc):
                        desc['Reconciled']['Receipt']=source_db.desc('adapter_results',intent_id=r.id)['Reconciled']['Receipt']
                    update_result(l.id,wrong_role)
                elif attack=='rehashed-false-query-status':
                    def wrong_status(desc):
                        snap=json.loads(desc['Candidate']['Response'])
                        snap.update(Status='not_found',Count=0,AppliedWire='',Receipt=None)
                        desc['Candidate']['Response']=enc(snap)
                        for v in (desc['Candidate']['Payload'],desc['Values']):
                            v['status']['Text']='not_found'
                            v['mutation']['Number']=v['total']['Number']=0
                        desc['Read']['Adoption']='none'
                        desc['Values']['adopted']['Flag']=False
                        desc['Values']['adoption']['Text']='none'
                    update_result(ql,wrong_status)
                elif attack=='rehashed-false-import':
                    def wrong_proof(desc):
                        desc['Reconciled']['QueryID']=qr
                    update_result(l.id,wrong_proof)
                else:
                    # Refund a real scope by altering both the reservation's own ordered
                    # evidence digest and its separately recorded charge allocation.
                    res=source_db.desc('composition_reservations',operation_id=ql)
                    res['Allowed']=False
                    res['Reason']='effect_limit'
                    db.execute('update composition_reservations set descriptor=? where operation_id=?',(enc(res).encode(),ql))
                    charge=source_db.desc('composition_charges',operation_id=ql,kind='effect')
                    charge['Effects']=0
                    db.execute('update composition_charges set descriptor=? where operation_id=?',(enc(charge).encode(),ql))
                    fact=source_db.row('composition_facts',decision_order=charge['DecisionOrder'])
                    f=json.loads(fact['descriptor'])
                    f['RecordDigest']=identity('alan/reservation-row/v1',res)
                    db.execute('update composition_facts set descriptor=? where decision_order=?',(enc(f).encode(),fact['decision_order']))
            db.close()
            blob=path.read_bytes()
            local=root/'mutant-local.sqlite.gz'
            local.write_bytes(gzip.compress(blob,mtime=0))
            worldfile=root/'mutant-world.sqlite.gz'
            if not worldfile.exists():
                worldfile.write_bytes((source/'final-world.sqlite.gz').read_bytes())
            expected={'swapped-genuine-receipt':'actual local reconstruction','rehashed-false-query-status':'retained real snapshot',
                      'rehashed-false-import':'import policy and query fields','rehashed-free-read':'operation debit and reservation'}
            try:
                snapshot(root,'mutant',[],journal)
            except ValueError as err:
                why=str(err)
                check(why==expected[attack], 'mutant rejection layer: '+why)
            else:
                raise ValueError('independent auditor accepted '+attack)
            p=invoke(path)
            check(p.returncode!=0,'real Replay accepted '+attack)
            out[attack]={'image_sha256':sha(blob),'independent_rejection':why,'replay_exit':p.returncode,'replay_stderr':p.stderr.decode().strip()}
    return {'executable_sha256':sha(consumer.read_bytes()),'baseline_image_sha256':sha(data),'baseline_exit':baseline.returncode,'mutants':out}


def group_digest(root):
    return OLD['source_hash']({str(p.relative_to(root)):sha(p.read_bytes()) for p in sorted(root.rglob('*')) if p.is_file()})


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--process',type=Path)
    parser.add_argument('--grid',type=Path)
    parser.add_argument('--consumer',type=Path,help='public Replay CLI for real disposable corruption attacks')
    parser.add_argument('--case',type=Path,help='one pilot process case instead of a complete corpus')
    args=parser.parse_args()
    output={'checker_sha256':sha(Path(__file__).read_bytes()),'historical_helpers_sha256':sha(Path(OLD['__file__']).read_bytes())}
    if args.process:
        p=args.process
        output['process_sha256']=group_digest(p)
        output['cases']={name:campaign_case(p/name) for name in CASES}
        output['paired']=paired(p/'paired')
    if args.case:
        output['case']=campaign_case(args.case)
    if args.grid:
        output['grid_sha256']=group_digest(args.grid)
        output['grid']=grid_audit(args.grid)
    if args.consumer:
        output['corruptions']=replay_attacks(args.case if args.case else args.process/'forward',args.consumer)
    print(json.dumps(output,sort_keys=True,separators=(',',':')))


if __name__=='__main__':
    main()
