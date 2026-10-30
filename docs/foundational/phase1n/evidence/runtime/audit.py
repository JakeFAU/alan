#!/usr/bin/env python3
"""Independent original-row/instruction audit. No Go decoder, reducer or oracle."""
from pathlib import Path
import copy, gzip, hashlib, json, os, sys, tarfile

DEST = Path(__file__).resolve().parent
ROOT = DEST
while not (ROOT/'go.mod').exists(): ROOT = ROOT.parent
sha = lambda b: hashlib.sha256(b).hexdigest()
encode = lambda v: json.dumps(v, ensure_ascii=False, separators=(',', ':')).encode()
desc = lambda row: json.loads(row['descriptor'])
rid = lambda row: (row['subject'], row['generation'])
runid = lambda run: (run['Subject'], run['Generation'])
identity = lambda r: dict(Subject=r[0], Generation=r[1])
histkey = lambda row: (*rid(row), row['sequence'])
def one(rows):
    assert len(rows) == 1, ('cardinality', len(rows))
    return rows[0]
def value(kind, x):
    result = dict(Kind=kind, Number=0, Text='', Flag=False)
    result[dict(int='Number', string='Text', bool='Flag', list='List')[kind]] = x
    return result
def operand(o, data):
    assert o['Offset'] is None
    if o['Ref']:
        assert o['Ref'] in data, ('absent operand', o['Ref'])
        return copy.deepcopy(data[o['Ref']])
    assert o['Literal'] is not None
    return copy.deepcopy(o['Literal'])

# Actual approved original programs and separately authored Task4 control programs.
# Control identities are from the source-bound implementation-1 construction, not
# substituted positive identities. Runtime values never select an expected tag.
PROGRAMS = set('''272c68dfe7c2b422c9819a8caf80b0277f35fb91859c9ddf80915cf46fdc4725
74152796cc6bb845eee44cc19813210a85ee26173629dd76e446fd042a85d5b7
d19cc8d312c2d1f16e10fe71d9b891400bf49b5d31f0d63226c77521efe63cf8
fd9d3e82946594e8114a84a156f56594d4f9dac6448656ff86e090aadb1b32eb
5e8fee04fd8855c501c1c20b87fdc4c2a1f96d6f4152baba3a7b5a48acb45621
698256075f67b360cacb0b20e16c5aeafc561c76255aeb3fd1ca7481ef9e9a6b
ca5ab7efdb73587e2664e02f9cea2928fde178190768aa844a2c88e14c90aa67
e87d0f268da24f572e9f8fd14a194a8dd96551838708c186dadd9ac5a3b68773
66b0b33b2db837c6ada4cffd605860572bf8d49ac9a9a2288af29521f7cc376c
745546979eb7dc81a47c4b5c8bf4c651b62c323fec390ca49128cbc2554cbc09
1a49ffc3a8178ef55849899226e5962f9b4c0841e5e0703a0ba4d7b20530fab8
1e1eed664087e2f097eb3f6cea56c6efe59d2b054c23a1372323bae3afc31f76
ce195a97e5680854255746921977b79f16b77862fadfb4d6e7b86b2d0bdb11c5
473f45487754b65bd4a1cc15019027cf2fffbdb4567c5fd3d52407b0fa11f896
1bcc4e9e36713856fc53595d554a6e459a7d15aef0d49e16f681418d4520059b
276e370e023eb3106e01e82131d4c8742365af07cddfa959fc579822fdd888c0
6503907796c54a0267aebb064366bf8ee2462170975422780b089fe60ef85e3d'''.split())

def unique(tables):
    keys = dict(programs=['id'], subjects=['id'], runs=['subject','generation'], history=['subject','generation','sequence'], intents=['id'], attempts=['intent_id'], observations=['id'], composition_templates=['id'], composition_bindings=['id'], composition_invocations=['operation_id'], composition_members=['subject','generation'], composition_admission_decisions=['operation_id'], composition_charges=['operation_id','kind'], composition_requests=['invocation_id'], composition_replies=['operation_id'], composition_publications=['operation_id'], composition_outcome_deliveries=['operation_id'], composition_freezes=['scope_id'], composition_root_publications=['scope_id'], composition_facts=['id'])
    for table, columns in keys.items():
        ids = [tuple(r[k] for k in columns) for r in tables[table]]
        assert len(ids) == len(set(ids)), ('duplicate', table)
    for table in ['lifecycle_results','external_attempts','external_receipts','adapter_attempts','adapter_results','composition_reservations','composition_closures']:
        assert not tables[table], ('outside profile', table)

def walk(run, program, histories, observations, binding):
    previous = json.loads(run['initial']); result = []
    rows = sorted([r for r in histories if rid(r) == rid(run)], key=lambda r:r['sequence'])
    for seq, row in enumerate(rows, 1):
        assert row['sequence'] == seq and previous['Status'] == 'open'
        pc = previous['PC']; instruction = program['Code'][pc]; op = instruction['Op']
        after = copy.deepcopy(previous); after['PC'] = pc + 1
        data = after['Data']; cursors = after.setdefault('Cursors', {})
        ev = lambda o: operand(o, previous['Data'])
        selected = row['selected_observation']; intents = json.loads(row['intents'])
        if op != 'receive': assert not selected
        if op != 'emit': assert not intents
        if op == 'set': data[instruction['Dst']] = ev(instruction['A'])
        elif op == 'branch': after['PC'] = instruction['Target'] if ev(instruction['A']) == ev(instruction['B']) else instruction['Otherwise']
        elif op == 'jump': after['PC'] = instruction['Target']
        elif op == 'emit':
            effect = instruction['Effect']; emitted = one(intents)
            target, generation, obligation = ev(effect['Target']), ev(effect['Generation']), ev(effect['Obligation'])
            if effect['Kind'] == 'message':
                assert binding and target == value('string', 'caller')
                target, generation = value('string',binding['Caller']['Subject']), value('int',binding['Caller']['Generation'])
            assert emitted['Kind'] == effect['Kind'] and emitted['Target'] == target['Text'] and emitted['Generation'] == generation['Number'] and emitted['Obligation'] == obligation['Text']
            assert emitted['Payload'] == {k:ev(v) for k,v in effect['Payload'].items()}, 'original Send/Invoke payload'
            for k,v in (instruction['Updates'] or {}).items(): data[k] = ev(v)
            if instruction['Dst']: data[instruction['Dst']] = value('string', emitted['ID'])
        elif op == 'receive':
            assert selected in observations
            observed = desc(observations[selected]); m = instruction['Match']
            source, gen, obligation, operation = [ev(m[k]) for k in ['Source','Generation','Obligation','Operation']]
            provenance = m['Provenance']
            if m['Kind'] == 'message':
                assert binding and source == value('string','caller')
                source, gen, operation = value('string',binding['Caller']['Subject']), value('int',binding['Caller']['Generation']), value('string',binding['OperationID'])
                provenance = 'runtime-invocation-request'
            assert observed['Kind'] == m['Kind'] and observed['Provenance'] == provenance and observed['Source'] == source['Text'] and observed['SourceGeneration'] == gen['Number'] and observed['Obligation'] == obligation['Text'] and observed['Operation'] == operation['Text']
            assert observed['Run'] == identity(rid(run)) and observations[selected]['consumed_sequence'] == seq
            for field,destination in instruction['Bind'].items():
                if m['Kind'] == 'invocation' and field.startswith('response.') and not observed['Payload']['response_present']['Flag']:
                    data.pop(destination,None)
                else:
                    assert field in observed['Payload']; data[destination] = observed['Payload'][field]
        elif op == 'append':
            col = program['Author']['Collections'][instruction['Dst']]
            rowvalue = {k:ev(v) for k,v in instruction['Updates'].items()}
            data[instruction['Dst']]['List'].append(rowvalue)
            assert len(data[instruction['Dst']]['List']) <= col['MaxItems']
            assert len(encode(data[instruction['Dst']]['List'])) <= col['MaxBytes']
        elif op == 'foreach':
            snapshot = ev(instruction['A']); col = instruction['Collection']
            assert snapshot['Kind'] == 'list' and len(snapshot['List']) <= col['MaxItems'] and len(encode(snapshot['List'])) <= col['MaxBytes']
            assert str(pc) not in cursors
            cursors[str(pc)] = dict(Snapshot=snapshot,Index=0)
        elif op == 'next':
            target = str(instruction['Target']); cursor = cursors[target]; items = cursor['Snapshot']['List']; index = cursor['Index']
            assert 0 <= index <= len(items)
            if index == len(items):
                del cursors[target]; after['PC'] = program['Code'][int(target)]['Otherwise']
            else:
                for field,destination in instruction['Bind'].items(): data[destination] = items[index][field]
                cursor['Index'] += 1
        elif op == 'loop':
            assert str(instruction['Target']) in cursors
            after['PC'] = instruction['Target']+1
        elif op == 'end': after['PC'], after['Status'] = pc, instruction['Outcome']
        else: raise AssertionError(('unmapped instruction',op))
        if not cursors: after.pop('Cursors',None)
        actual = json.loads(row['successor'])
        assert after == actual, ('instruction successor',rid(run),seq,pc,op)
        result.append(dict(row=row,pre=previous,post=actual,instruction=instruction))
        previous = actual
    assert previous == json.loads(run['projection']) and run['sequence'] == len(rows)
    return result

def expected_tag(request, e, s):
    if s.get('status') != value('string','succeeded') or s.get('reason') != value('string','member_succeeded') or s.get('response_present') != value('bool',True) or s.get('response.id') != value('int',request['ID']) or s.get('response.value') != value('int',0): return 'invalid_outcome'
    if e.get('status') == value('string','succeeded') and e.get('reason') == value('string','member_succeeded') and e.get('response_present') == value('bool',True) and e.get('response.id') == value('int',request['ID']) and e.get('response.value') == value('int',1-request['Value']): return 'answer'
    if e.get('status') == value('string','failed') and e.get('response_present') == value('bool',False):
        if e.get('reason') == value('string','member_failed'): return 'candidate_failed'
        if e.get('reason') == value('string','missing_reply'): return 'candidate_missing_reply'
    return 'invalid_outcome'

def public_policy(request, payload):
    if payload.get('response_present') != value('bool',True) or payload.get('response.id') != value('int',request['ID']): return 'invalid_join_outcome'
    tag = payload['response.classification']['Text']; rows = payload['response.answers']['List']
    if tag == 'answer' and payload['status'] == value('string','succeeded') and payload['reason'] == value('string','member_succeeded') and rows == [dict(id=value('int',request['ID']),value=value('int',1-request['Value']))]: return 'answer'
    if tag in ['candidate_failed','candidate_missing_reply','invalid_outcome'] and payload['status'] == value('string','failed') and payload['reason'] == value('string','member_failed') and rows == []: return tag
    return 'invalid_join_outcome'

def audit_snapshot(snapshot, request, trace=None):
    t = snapshot['Tables']; unique(t)
    root = desc(one(t['composition_roots'])); completion = one(snapshot['Roots'])
    assert root['Run'] == dict(Subject='root',Generation=1) and root['Members'] == [root['Run']]
    assert root['Limits'] == dict(MaxInvocations=4,MaxCreatedRuns=4,MaxDepth=3,MaxEffects=0,MaxInFlight=1)
    programs = {}
    for row in t['programs']:
        assert row['id'] in PROGRAMS and sha(row['bytes'].encode()) == row['id']
        programs[row['id']] = json.loads(row['bytes'])
    runs = {rid(r):r for r in t['runs']}; histories = {histkey(r):r for r in t['history']}
    observations = {r['id']:r for r in t['observations']}; intents = {r['id']:desc(r) for r in t['intents']}
    bindings = {desc(r)['OperationID']:desc(r) for r in t['composition_bindings']}
    templates = {r['id']:desc(r) for r in t['composition_templates']}
    admissions = {r['operation_id']:desc(r) for r in t['composition_admission_decisions']}
    member_op = {}; roles = {}; histories_by_run = {}; committed = {}
    for op,b in bindings.items():
        a = admissions[op]; template = templates[b['TemplateID']]; member = b['Members'][template['EntryMember']]
        assert b['Protocol'] == 'alan/composition/v1' and b['Namespace'] == snapshot['Namespace'] and b['RootID'] == root['ID']
        assert a['Accepted'] and a['BindingID'] == b['ID'] and a['Invocation']['ID'] == b['InvocationID'] and a['Invocation']['Caller'] == b['Caller'] and a['Members'] == [member]
        assert len(template['Members']) == 1 and template['EntryPort'] == 'public'
        run = runs[runid(member)]; entry = template['Members'][template['EntryMember']]
        assert run['program_id'] == entry['ProgramID'] == b['Programs'][template['EntryMember']] and json.loads(run['initial']) == entry['Initial']
        member_op[runid(member)] = op
    original = json.loads(runs[('root',1)]['initial'])
    assert original == dict(PC=0,Status='open',Data=dict(request_id=value('int',request['ID']),request_value=value('int',request['Value']),request_mode=value('string',request['Mode'])))
    subjects = {r['id']:r for r in t['subjects']}
    for subject in subjects.values(): assert subject['generation'] == subject['revision'] == 1 and not subject['retired']
    for runkey,run in runs.items():
        walked = walk(run,programs[run['program_id']],t['history'],observations,bindings.get(member_op.get(runkey)))
        histories_by_run[runkey] = walked
        for h in walked:
            row = h['row']; assert row['source_revision'] == subjects[runkey[0]]['revision']
            for emitted in json.loads(row['intents']):
                assert emitted['ID'] not in committed and emitted == intents[emitted['ID']]
                assert emitted['Run'] == identity(runkey) and emitted['Sequence'] == row['sequence'] and emitted['Ordinal'] == 0
                committed[emitted['ID']] = h
    assert set(committed) == set(intents)
    for row in t['intents']:
        i = intents[row['id']]
        assert (row['subject'],row['generation'],row['sequence'],row['ordinal']) == (*runid(i['Run']),i['Sequence'],i['Ordinal'])
        assert row['source_revision'] == committed[i['ID']]['row']['source_revision']
    # Resolve roles only from actual authored destination registers and caller chain.
    pending = [i for i in intents.values() if i['Kind'] == 'invocation']
    while pending:
        progress = False
        for i in pending[:]:
            caller = runid(i['Run']); dst = committed[i['ID']]['instruction']['Dst']; parentop = member_op.get(caller)
            if caller == ('root',1): assert dst == 'join_operation'; role,parent='j','root'
            elif parentop not in roles: continue
            elif roles[parentop][0] == 'j':
                role = {'candidate_operation':'e','s_operation':'s'}[dst];parent='j'
            else: assert roles[parentop][0]=='e' and dst=='helper_operation';role,parent='h','e'
            assert role not in [v[0] for v in roles.values()]
            roles[i['ID']] = (role,parent); pending.remove(i);progress=True
        assert progress
    links = {role:dict(operation=op,parent=parent,intent=intents[op]) for op,(role,parent) in roles.items()}
    for op,a in admissions.items():
        i = intents[op]; inv = a['Invocation']; role = roles[op][0]
        assert inv['OperationID']==op and inv['Caller']==i['Run'] and inv['SourceSequence']==i['Sequence'] and inv['SourceRevision']==1 and inv['Inputs']==i['Payload'] and inv['TemplateID']==i['Target'] and inv['RootID']==root['ID']
        assert inv['Depth'] == {'j':1,'e':2,'s':2,'h':3}[role]
        b=bindings[op]; member=one(a['Members']); links[role]['member']=member
        template=templates[b['TemplateID']];port=programs[runs[runid(member)]['program_id']]['Author']['Ports'][template['EntryPort']]
        reqrow=one([r for r in t['composition_requests'] if r['invocation_id']==inv['ID']]);req=desc(reqrow)
        assert reqrow['observation_id']==req['ID'] and observations[req['ID']]['descriptor']==reqrow['descriptor']
        assert req['Kind']=='message' and req['Provenance']=='runtime-invocation-request' and req['Run']==member and req['Source']==i['Run']['Subject'] and req['SourceGeneration']==i['Run']['Generation'] and req['Operation']==op and req['Obligation']==port['Obligation'] and req['Payload']==i['Payload']
        links[role]['request']=req
    facts=sorted([desc(r) for r in t['composition_facts']],key=lambda f:f['Order'])
    assert [f['Order'] for f in facts] == list(range(1,len(facts)+1))
    execution={}; source_sequences={r:0 for r in runs}; source_states={r:json.loads(row['initial']) for r,row in runs.items()}
    attempts={r['intent_id']:json.loads(r['result']) for r in t['attempts']}; replies={r['operation_id']:desc(r) for r in t['composition_replies']}
    publications={r['operation_id']:desc(r) for r in t['composition_publications']}; freezes={r['scope_id']:desc(r) for r in t['composition_freezes']}
    orders={}; seen={k:set() for k in ['execution','authorization','admission','reply','freeze','outcome']}
    for f in facts:
        kind=f['Kind'];op=f['OperationID'];run=runid(f['Run']); seq=f['Sequence']
        assert f['Version']=='alan/composition-fact/v1' and f['RootIDs']==[root['ID']]
        if kind=='execution':
            row=histories[(*run,seq)];assert seq==source_sequences[run]+1
            source=dict(Run=identity(run),Sequence=seq,Epoch=row['epoch'],Revision=row['source_revision'],Selected=row['selected_observation'] or '',Successor=json.loads(row['successor']),Intents=json.loads(row['intents']))
            assert f['RecordDigest']==sha(encode(['alan/history-row/v1',source])), 'history fact digest'
            execution[(*run,seq)]=f;source_sequences[run]=seq;source_states[run]=source['Successor'];key=(*run,seq)
        else:
            table,column={'authorization':('attempts','intent_id'),'admission':('composition_admission_decisions','operation_id'),'reply':('composition_replies','operation_id'),'freeze':('composition_freezes','scope_id'),'outcome':('composition_publications','operation_id')}[kind]
            if kind=='outcome' and op.startswith('root:'):table,column='composition_root_publications','scope_id'
            row=one([r for r in t[table] if r[column]==op]);raw=json.loads(row['result'] if kind=='authorization' else row['descriptor'])
            assert f['RecordDigest']==sha(encode(['alan/'+kind+'-row/v1',raw])), ('fact digest',kind)
            key=op
            if kind=='authorization':
                i=intents[op];a=attempts[op];req=desc(row)
                assert req['Intent']==i and req['Revision']==1 and a['Source']['Run']==i['Run'] and a['Source']['Revision']==1 and a['Source']['Epoch']>0 and a['Source']['Sequence']==source_sequences[run] and seq==i['Sequence']
                assert a['Allowed'] and a['Reason']=='authorized'
                if source_states[run]['Status']!='open': assert i['Kind']=='message' and run in member_op
                role=roles[op][0] if i['Kind']=='invocation' else roles[member_op[run]][0]
                orders[(role,'invoke_authorize' if i['Kind']=='invocation' else 'reply_authorize')]=f['Order']
            if kind in ['admission','reply','outcome'] and not op.startswith('root:'):
                role=roles[member_op[run]][0] if kind=='reply' else roles[op][0];orders[(role,kind)]=f['Order']
        assert key not in seen[kind];seen[kind].add(key)
    assert seen['execution']==set(histories) and seen['authorization']==set(attempts) and seen['admission']==set(admissions) and seen['reply']==set(replies) and seen['freeze']==set(freezes)
    assert seen['outcome']==set(publications)|{r['scope_id'] for r in t['composition_root_publications']}
    # Exact original request and outcome consumption links, independently of Go trace.
    consumed=set()
    for oid,row in observations.items():
        o=desc(row);assert oid==o['ID'] and rid(row)==runid(o['Run'])
        if row['consumed_sequence'] is not None:
            h=histories[(*runid(o['Run']),row['consumed_sequence'])]
            assert h['selected_observation']==oid;consumed.add(oid)
        if o['Kind']=='invocation':
            op=o['Operation'];pub=publications[op];inv=admissions[op]['Invocation'];i=intents[op]
            delivery=one([r for r in t['composition_outcome_deliveries'] if r['operation_id']==op]);assert delivery['observation_id']==oid
            assert o['Provenance']=='runtime-invocation' and o['Source']==inv['TemplateID'] and o['SourceGeneration']==1 and o['Run']==inv['Caller'] and o['Obligation']==i['Obligation']
            expected=dict(status=value('string',pub['Status']),reason=value('string',pub['Reason']),response_present=value('bool',pub['ResponsePresent']))
            if pub['ResponsePresent']:expected.update({'response.'+k:v for k,v in pub['Response'].items()})
            assert o['Payload']==expected
            role=roles[op][0];links[role]['publication']=pub;links[role]['observation']=o
            if oid in consumed:
                h=histories[(*runid(o['Run']),row['consumed_sequence'])];links[role]['consumed']=h
                assert execution[histkey(h)]['Order']>orders[(role,'outcome')]
    assert consumed=={r['selected_observation'] for r in t['history'] if r['selected_observation']}
    if trace is not None:
        assert trace['Handles']=={role:item['operation'] for role,item in links.items()}
        selected={}
        expected_steps=[]
        def step_of(h,run):
            f=execution[histkey(h['row'])]
            return dict(Run=identity(rid(run)),Program=run['program_id'],Sequence=h['row']['sequence'],Order=f['Order'],PC=h['pre']['PC'],NextPC=h['post']['PC'])
        for runkey,walked in histories_by_run.items():
            for h in walked:
                step=step_of(h,runs[runkey])
                expected_steps.append(dict(Step=step,Instruction=h['instruction'],Pre=h['pre'],Post=h['post'],Selected=h['row']['selected_observation'] or ''))
                if h['row']['selected_observation']:
                    o=desc(observations[h['row']['selected_observation']])
                    if o['Kind']=='invocation': selected[roles[o['Operation']][0]]=dict(Step=step,Observation=o,Bind=h['instruction']['Bind'])
        expected_steps.sort(key=lambda s:s['Step']['Order'])
        assert trace['Steps']==expected_steps, 'exported actual instruction evidence'
        assert trace['Consumptions']==selected, 'exported original selected evidence'
    for op,pub in publications.items():
        inv=admissions[op]['Invocation'];freeze=freezes[inv['ID']]
        assert pub['Caller']==inv['Caller'] and pub['TemplateID']==inv['TemplateID'] and pub['InvocationID']==inv['ID']
        responses=[replies[i]['Intent']['Payload'] for i in freeze['ReplyIntents'] if replies[i]['Accepted']]
        assert len(responses)<=1 and pub['ResponsePresent']==bool(responses) and pub['Response']==(responses[0] if responses else None)
        status=freeze['Status'];reason='member_'+status
        if status=='succeeded' and not responses:status,reason='failed','missing_reply'
        assert (pub['Status'],pub['Reason'])==(status,reason)
    for scope,f in freezes.items():
        run=runid(f['Run']);h=histories[(*run,f['Sequence'])];assert json.loads(h['successor'])['Status']==f['Status'] and f['Status']!='open' and execution[histkey(h)]['ID']==f['Trigger']
        before=[i for i in intents.values() if runid(i['Run'])==run and execution[(*run,i['Sequence'])]['Order']<f['DecisionOrder']]
        assert sorted(f['InvocationIntents'])==sorted(i['ID'] for i in before if i['Kind']=='invocation') and not f['ExternalIntents']
        assert sorted(f['ReplyIntents'])==sorted(i['ID'] for i in before if i['Kind']=='message')
        if run in member_op:orders[(roles[member_op[run]][0],'End')]=execution[histkey(h)]['Order']
    for id,reply in replies.items():
        i=intents[id];b=bindings[member_op[runid(i['Run'])]];template=templates[b['TemplateID']];port=programs[b['Programs'][template['EntryMember']]]['Author']['Ports'][template['EntryPort']]
        assert reply['Intent']==i and reply['Authorization']==attempts[id] and reply['InvocationID']==b['InvocationID'] and reply['RootID']==root['ID']
        assert i['Target']==b['Caller']['Subject'] and i['Generation']==b['Caller']['Generation'] and i['Obligation']==port['Obligation'] and reply['Accepted'] and reply['Reason']=='recorded_reply'
        transport=one([r for r in t['admissions'] if desc(r)['IntentID']==id]);d=desc(transport);result=json.loads(transport['result'])
        assert d['Kind']=='message' and d['LogicalID']==id and d['Source']==i['Run'] and d['Target']==i['Target'] and d['Payload']==i['Payload'] and result['Bound']==b['Caller'] and result['Accepted'] and result['Reason']=='recorded_reply' and not result['ObservationID']
        ack=[r for r in t['message_acknowledgments'] if r['intent_id']==id]
        if ack: assert one(ack)['admission_id']==transport['id']
    assert len(t['admissions'])==len(replies)
    charges={r['operation_id']:desc(r) for r in t['composition_charges']};assert set(charges)==set(admissions)
    for op,ch in charges.items(): assert ch['Invocations']==ch['CreatedRuns']==1 and ch['Effects']==0 and ch['RootID']==root['ID'] and ch['DecisionOrder']==orders[(roles[op][0],'admission')]
    assert completion['Charges']['Invocations']==completion['Charges']['CreatedRuns']==len(charges) and completion['Charges']['Effects']==0
    rootpub=completion['Outcome']
    if rootpub:
        assert desc(one(t['composition_root_publications']))==rootpub and rootpub['Status']==json.loads(runs[('root',1)]['projection'])['Status'] and rootpub['Reason']=='member_'+rootpub['Status']
    jrun=runid(links['j']['member']) if 'member' in links['j'] else None
    jhistory=histories_by_run.get(jrun,[]);ohistory=histories_by_run[('root',1)]
    classes=[h for h in jhistory if h['instruction']['Op']=='set' and h['instruction']['Dst']=='classification']
    reports=[i for i in intents.values() if runid(i['Run'])==jrun and i['Kind']=='message']
    decisions=[h for h in ohistory if h['instruction']['Op']=='set' and h['instruction']['Dst']=='decision']
    assert len(classes)<=1 and len(reports)<=1 and len(decisions)<=1
    tag=classes[0]['post']['Data']['classification']['Text'] if classes else None
    report=reports[0] if reports else None;decision=decisions[0]['post']['Data']['decision']['Text'] if decisions else None
    expected=None
    if classes or reports:
        assert 'consumed' in links['e'] and 'consumed' in links['s']
        assert execution[histkey(links['s']['consumed'])]['Order']<execution[histkey(links['e']['consumed'])]['Order']
        expected=expected_tag(request,links['e']['observation']['Payload'],links['s']['observation']['Payload'])
    truth=not classes or tag==expected
    if report:
        payload=report['Payload'];assert set(payload)=={'id','classification','answers'} and payload['answers']['Kind']=='list' and len(payload['answers']['List'])<=1 and len(encode(payload['answers']['List']))<=192
        assert payload['id']==value('int',request['ID']) and payload['classification']==value('string',tag)
        rows=[dict(id=value('int',request['ID']),value=value('int',1-request['Value']))] if expected=='answer' else []
        truth=truth and payload['answers']['List']==rows
        orders[('j','Send')]=execution[(*runid(report['Run']),report['Sequence'])]['Order']
    if trace is not None:
        if report:
            assert trace['Report']['Intent']==report
            assert trace['Report']['Step']==step_of(committed[report['ID']],runs[jrun])
        else: assert trace['Report'] is None
        if classes:
            assert trace['Classification']['Step']==step_of(classes[0],runs[jrun])
            assert trace['Classification']['Consumed']=={role:selected[role] for role in ['e','s']}
        else:assert trace['Classification'] is None
        if decisions:
            assert trace['Decision']['Step']==step_of(decisions[0],runs[('root',1)])
            assert trace['Decision']['Consumed']=={'j':selected['j']}
        else:assert trace['Decision'] is None
    public=True
    if decision:
        assert 'consumed' in links['j'];public=decision==public_policy(request,links['j']['observation']['Payload'])
        assert execution[histkey(links['j']['consumed'])]['Order']<execution[histkey(decisions[0]['row'])]['Order']
    assert public, 'actual O public policy'
    completed=bool(rootpub)
    if completed and not report:truth=False
    pending=False
    if request['Mode']=='pending':
        erun=runs[runid(links['e']['member'])];ep=json.loads(erun['projection']);program=programs[erun['program_id']]
        pc={'74152796cc6bb845eee44cc19813210a85ee26173629dd76e446fd042a85d5b7':11,'ca5ab7efdb73587e2664e02f9cea2928fde178190768aa844a2c88e14c90aa67':12}[erun['program_id']]
        assert ep['Status']=='open' and ep['PC']==pc and program['Code'][pc]['Op']=='receive'
        assert 'publication' not in links['e'] and 'consumed' in links['s'] and not classes and not report and not decision and not rootpub
        assert json.loads(runs[jrun]['projection'])['PC']==5 and json.loads(runs[('root',1)]['projection'])['PC']==1
        if 'h' in links: assert 'consumed' in links['h'] and links['h']['publication']['Status']=='succeeded'
        assert not [row for row in observations.values() if rid(row)==rid(erun) and row['consumed_sequence'] is None and desc(row)['Operation']==links['e']['operation']]
        pending=True
    desired={'answer':'answer','fail':'candidate_failed','absent':'candidate_missing_reply','pending':None}[request['Mode']]
    component=True
    if 'e' in links and 'publication' in links['e']:
        p=links['e']['publication'];want={'answer':('succeeded','member_succeeded',True),'fail':('failed','member_failed',False),'absent':('failed','missing_reply',False)}.get(request['Mode'])
        component=(p['Status'],p['Reason'],p['ResponsePresent'])==want
    contract=truth and component and (pending or bool(report and links['j'].get('publication') and report['Payload']['classification']['Text']==desired))
    outer=(pending or bool(completed and decision==desired))
    return dict(public=public,truth=truth,component=component,contract=contract,outer=outer,completed=completed,pending=pending,classification=tag,decision=decision,orders={a+':'+b:v for (a,b),v in orders.items()},links=links,instructions=sum(len(h) for h in histories_by_run.values()),static=sum(len(p['Code']) for p in programs.values()))

def read(label): return json.loads(gzip.decompress((DEST/label/'command.json.gz').read_bytes()))
def audit_command(label, require_final=False):
    command=read(label);assert command['exit']==0 and not command['changed_during_run']
    assert not [e for e in command['test_results'] if e['Action'] in ['fail','skip']]
    snapshot=DEST/label/'sources.tar.gz';assert sha(snapshot.read_bytes())==command['source_snapshot_sha256']
    with tarfile.open(snapshot, 'r:gz') as source:
        for path,digest in command['sources'].items():
            assert sha(source.extractfile(path).read())==digest
            if require_final:assert sha((ROOT/path).read_bytes())==digest,path
    return command

def main():
    labels=sys.argv[1:] or ['normal','race']
    summary={}
    for label in labels:
        command=audit_command(label)
        counts={};results=[]
        for name,r in command['records'].items():
            if not isinstance(r,dict):continue
            kind=r.get('kind')
            if kind=='decoder-negative':
                audit_snapshot(r['source_inspection'],r['config']['Request'])
                rejected=False
                try:audit_snapshot(r['mutated_inspection'],r['config']['Request'])
                except (AssertionError,KeyError,IndexError):rejected=True
                assert rejected, r['attack'];counts[kind]=counts.get(kind,0)+1;continue
            if 'inspection' not in r or 'config' not in r:continue
            result=audit_snapshot(r['inspection'],r['config']['Request'],r.get('trace'));counts[kind]=counts.get(kind,0)+1
            trace=r.get('trace')
            if trace:
                assert trace['RawValid'] and not trace['Public']
                assert (trace['Decision']['Tag'] if trace['Decision'] else None)==result['decision']
                assert (trace['Classification']['Tag'] if trace['Classification'] else None)==result['classification']
            if kind in ['automatic','mode','late-reply']:
                assert all(result[k] for k in ['public','truth','component','contract','outer'])
                assert not any(trace[k] for k in ['Public','Truth','Component','Contract','Outer'])
                assert r['limits']==dict(steps=512,dispatch=64,capacity=1)
                assert r['bounds']['static_instructions']==result['static'] and r['bounds']['commits']==result['static']+1
                assert result['instructions']<=r['bounds']['commits']
                if r['report'] is not None:assert r['report']['Dispatches']<=r['bounds']['dispatch'] and r['report']['Steps']==result['instructions']
                else:assert kind=='late-reply'
                if 'mapping' in r:
                    programs={row['id']:json.loads(row['bytes']) for row in r['inspection']['Tables']['programs']}
                    assert {(m['Program'],m['PC']) for m in r['mapping']}=={(pid,pc) for pid,p in programs.items() for pc in range(len(p['Code']))}
                    assert len(r['mapping'])==result['static']
                    for m in r['mapping']:
                        p=programs[m['Program']];assert m['Instruction']==p['Code'][m['PC']] and m['Position']==p['Author']['Positions'][m['PC']] and m['Mapping']
                    assert r['actual_local_commits']==result['instructions']
                    tables=r['inspection']['Tables']
                    assert r['protocol_dispatch_transitions']==len(tables['composition_admission_decisions'])+len(tables['message_acknowledgments'])+len(tables['composition_outcome_deliveries'])<=r['bounds']['dispatch']
                if kind=='late-reply':
                    order=result['orders'];assert order['j:Send']<order['j:End']<order['j:reply_authorize']<order['j:reply']<order['j:outcome']
            elif kind=='control':
                control=r['control'];assert result['public'] and not result['contract'] and not result['outer']
                if control=='absent-deliberate-failure':assert result['truth'] and not result['component'] and not trace['Truth'] and trace['Component']
                else:assert not result['truth'] and result['component'] and trace['Truth'] and not trace['Component']
                if control in ['no-report-success','no-report-failure']:
                    for role,want in [('e',1-r['config']['Request']['Value']),('s',0)]:
                        p=result['links'][role]['observation']['Payload']
                        assert p['status']==value('string','succeeded') and p['reason']==value('string','member_succeeded') and p['response_present']==value('bool',True) and p['response.id']==value('int',r['config']['Request']['ID']) and p['response.value']==value('int',want)
            elif kind in ['send-prefix','completion-claim-negative']:
                assert result['truth'] and not result['completed'] and 'j:End' not in result['orders'] and 'publication' not in result['links']['j']
                if kind=='completion-claim-negative':assert all(r['claim'].values()) and trace['Contract'] and trace['Outer']
            elif kind=='end-before-first-authorization':
                assert 'j:End' in result['orders'] and 'j:reply_authorize' not in result['orders'] and r['preparation']['Pending']=='reply_admission' and not r['preparation']['Ready']
            results.append(dict(record=name,kind=kind,**{k:result[k] for k in ['public','truth','component','contract','outer','completed','pending','classification','decision','instructions','static']}))
        assert counts=={'automatic':8,'mode':6,'control':4,'late-reply':1,'decoder-negative':7,'completion-claim-negative':2,'send-prefix':1,'end-before-first-authorization':1},counts
        summary[label]=dict(counts=counts,results=results,leaves=len(command['selected_leaves']))
    output=dict(status='PASS',scope='finite saved runtime records; no new execution or universal schedule claim',profiles=summary)
    dest=os.environ.get('ALAN_I_ARTIFACTS')
    if dest:(Path(dest)/'audit.json').write_text(json.dumps(output,sort_keys=True,indent=2)+'\n')
    print(json.dumps(output,sort_keys=True))

if __name__=='__main__':main()
