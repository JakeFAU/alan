#!/usr/bin/env python3
"""Independent original-row audit; reuses only inert Phase 1L audit functions.

No reducer, Go decoder, historical top-level script, capture, or archive writer
is executed here. This script writes only this package's audit/index outputs.
"""
from pathlib import Path
import ast
import gzip
import hashlib
import json

ROOT = Path(__file__).resolve().parents[6]
DEST = Path(__file__).resolve().parent
sha = lambda b: hashlib.sha256(b).hexdigest()
old_audit = ROOT / 'docs/foundational/phase1l/evidence/runtime/audit.py'
old_bytes = old_audit.read_bytes()
assert sha(old_bytes) == 'c2a9b4b47cceb132efd5293ebb12f8e901ba1af581f9db1dfaa78baa3d2458cf'
functions = ['descriptor', 'links', 'source_authority']
syntax = ast.parse(old_bytes, filename=str(old_audit))
selected = [n for n in syntax.body if isinstance(n, ast.FunctionDef) and n.name in functions]
assert {n.name for n in selected} == set(functions)
# Compile exactly the three function definitions, omitting every top-level
# historical load, command, assertion and write. Their source identity is pinned.
namespace = {'json': json}
exec(compile(ast.Module(body=selected, type_ignores=[]), str(old_audit), 'exec'), namespace)
descriptor, links, source_authority = (namespace[n] for n in functions)


def one(rows):
    assert len(rows) == 1, ('cardinality', len(rows))
    return rows[0]


def read(label):
    return json.loads(gzip.decompress((DEST / (label + '.json.gz')).read_bytes()))


def run_identity(row):
    return {'Subject': row['subject'], 'Generation': row['generation']}


def root_projection(s):
    return json.loads(one([r for r in s['Tables']['runs'] if r['subject'] == 'root'])['projection'])


def value(kind, data):
    result = dict(Kind=kind, Number=0, Text='', Flag=False)
    result[dict(string='Text', int='Number', bool='Flag')[kind]] = data
    return result


def operand(o, data):
    assert o['Offset'] is None
    return data[o['Ref']] if o['Ref'] else o['Literal']


def raw_links(snapshot):
    """Authenticate handle/request/binding/publication/actual selection separately."""
    source_authority(snapshot)
    original = links(snapshot)
    tables = snapshot['Tables']
    root = one([r for r in tables['runs'] if r['subject'] == 'root'])
    histories = {(r['subject'], r['generation'], r['sequence']): r for r in tables['history']}
    observations = {r['id']: r for r in tables['observations']}
    admissions = {r['operation_id']: descriptor(r) for r in tables['composition_admission_decisions']}
    bindings = {descriptor(r)['OperationID']: descriptor(r) for r in tables['composition_bindings']}
    programs = {r['id']: json.loads(r['bytes']) for r in tables['programs']}
    templates = {r['id']: descriptor(r) for r in tables['composition_templates']}
    facts = sorted((descriptor(r) for r in tables['composition_facts']), key=lambda f: f['Order'])
    assert [f['Order'] for f in facts] == list(range(1, len(facts)+1))
    handles = {role: info['operation'] for role, info in original.items()}
    entry = admissions[handles['e']]
    if entry['Accepted']:
        member = one(entry['Members'])
        run = one([r for r in tables['runs'] if run_identity(r) == member])
        data = json.loads(run['projection'])['Data']
        if data.get('helper_operation'):
            handles['h'] = data['helper_operation']['Text']
    result = {}
    for role, operation in handles.items():
        admission = admissions[operation]
        invocation = admission['Invocation']
        intent = descriptor(one([r for r in tables['intents'] if r['id'] == operation]))
        caller = invocation['Caller']
        commits = [h for h in tables['history'] if any(i['ID'] == operation for i in json.loads(h['intents']))]
        commitment = one(commits)
        register = dict(e='candidate_operation', s='s_operation', h='helper_operation')[role]
        assert json.loads(commitment['successor'])['Data'][register]['Text'] == operation
        assert invocation['Inputs'] == intent['Payload'] and invocation['TemplateID'] == intent['Target']
        item = dict(Operation=operation, InvocationID=invocation['ID'], OutcomeObservation='', RequestObservation='',
            Member={'Subject':'','Generation':0}, MemberStatus='', MemberSequence=0, MemberTerminalOrder=0,
            PublicationOrder=0, ConsumptionSequence=0, ConsumptionOrder=0, RequestConsumedSequence=0)
        if admission['Accepted']:
            member = one(admission['Members'])
            binding = bindings[operation]
            template = templates[binding['TemplateID']]
            assert binding['Caller'] == caller and binding['InvocationID'] == invocation['ID']
            assert binding['Protocol'] == 'alan/composition/v1' and binding['TemplateID'] == invocation['TemplateID']
            assert binding['Members'][template['EntryMember']] == member
            run = one([r for r in tables['runs'] if run_identity(r) == member])
            item.update(Member=member, MemberStatus=json.loads(run['projection'])['Status'], MemberSequence=run['sequence'])
            request = one([r for r in tables['composition_requests'] if r['invocation_id'] == invocation['ID']])
            o = descriptor(request)
            assert observations[o['ID']]['descriptor'] == request['descriptor']
            assert o['Run'] == member and o['Source'] == caller['Subject'] and o['SourceGeneration'] == caller['Generation']
            assert o['Operation'] == operation and o['Payload'] == intent['Payload']
            assert o['Kind'] == 'message' and o['Provenance'] == 'runtime-invocation-request'
            port = programs[binding['Programs'][template['EntryMember']]]['Author']['Ports'][template['EntryPort']]
            assert o['Obligation'] == port['Obligation']
            item.update(RequestObservation=o['ID'], RequestConsumedSequence=observations[o['ID']]['consumed_sequence'] or 0)
            freezes = [f for f in facts if f['Kind'] == 'freeze' and f['Run'] == member]
            if item['MemberStatus'] != 'open':
                freeze = one(freezes)
                item['MemberTerminalOrder'] = freeze['Order']
                terminal = histories[(member['Subject'], member['Generation'], freeze['Sequence'])]
                assert json.loads(terminal['successor'])['Status'] == item['MemberStatus']
            else:
                assert not freezes
        publications = [descriptor(r) for r in tables['composition_publications'] if r['operation_id'] == operation]
        deliveries = [r for r in tables['composition_outcome_deliveries'] if r['operation_id'] == operation]
        if publications:
            publication, delivery = one(publications), one(deliveries)
            observation = observations[delivery['observation_id']]
            o = descriptor(observation)
            assert publication['Caller'] == caller and publication['InvocationID'] == invocation['ID']
            assert publication['OperationID'] == operation and publication['TemplateID'] == invocation['TemplateID']
            assert o['Operation'] == operation and o['Run'] == caller and o['Source'] == invocation['TemplateID']
            assert o['SourceGeneration'] == 1 and o['Obligation'] == intent['Obligation']
            assert o['Kind'] == 'invocation' and o['Provenance'] == 'runtime-invocation'
            expected = dict(status=value('string',publication['Status']), reason=value('string',publication['Reason']),
                            response_present=value('bool',publication['ResponsePresent']))
            if publication['ResponsePresent']:
                expected.update({'response.'+k:v for k,v in publication['Response'].items()})
            assert o['Payload'] == expected
            fact = one([f for f in facts if f['Kind'] == 'outcome' and f['OperationID'] == operation])
            assert fact['Order'] == publication['DecisionOrder']
            item.update(PublicationOrder=fact['Order'], OutcomeObservation=o['ID'])
            consumed = [h for h in tables['history'] if h['selected_observation'] == o['ID']]
            if observation['consumed_sequence'] is not None:
                selected = one(consumed)
                seq = selected['sequence']
                assert seq == observation['consumed_sequence'] and run_identity(selected) == caller
                pre = json.loads(histories[(caller['Subject'],caller['Generation'],seq-1)]['successor'])
                run = one([r for r in tables['runs'] if run_identity(r) == caller])
                instruction = programs[run['program_id']]['Code'][pre['PC']]
                match = instruction['Match']
                assert instruction['Op'] == 'receive' and match['Operation']['Ref'] == register
                assert operand(match['Operation'],pre['Data'])['Text'] == operation
                assert operand(match['Source'],pre['Data'])['Text'] == o['Source']
                assert operand(match['Generation'],pre['Data'])['Number'] == o['SourceGeneration']
                assert operand(match['Obligation'],pre['Data'])['Text'] == o['Obligation']
                assert match['Kind'] == o['Kind'] and match['Provenance'] == o['Provenance']
                post = json.loads(selected['successor'])
                for field, destination in instruction['Bind'].items():
                    if field in o['Payload']: assert post['Data'][destination] == o['Payload'][field]
                execution = one([f for f in facts if f['Kind']=='execution' and f['Run']==caller and f['Sequence']==seq])
                assert execution['Order'] > fact['Order']
                item.update(ConsumptionSequence=seq,ConsumptionOrder=execution['Order'])
            else:
                assert not consumed
        else:
            assert not deliveries
        result[role] = item
    accepted = [a for a in admissions.values() if a['Accepted']]
    charges = snapshot['Roots'][0]['Charges']
    assert len(accepted) == len(tables['composition_charges']) == charges['Invocations'] == charges['CreatedRuns']
    assert charges['Effects'] == 0
    return result


def exactly_one(records, suffix):
    return one([v for k,v in records.items() if k.endswith(suffix)])


def good(trace):
    assert trace['RawValid'] and not trace['Prefix'] and not trace['Component'] and not trace['Guarantee']


def check_positive(record):
    s = record['inspection']
    actual = raw_links(s)
    assert actual == record['causality']
    good(record['trace'])
    assert actual['s']['ConsumptionSequence'] == 3 and actual['e']['ConsumptionSequence'] == 4
    request = record['config']['Request']
    for role in ['e','s']:
        publication = descriptor(one([r for r in s['Tables']['composition_publications']
                                      if r['operation_id']==actual[role]['Operation']]))
        assert publication['Status']=='succeeded' and publication['Reason']=='member_succeeded' and publication['ResponsePresent']
        assert publication['Response']['id']['Number']==request['ID']
        assert publication['Response']['value']['Number']==(1-request['Value'] if role=='e' else 0)
    assert root_projection(s)['Status']=='succeeded'
    assert s['Roots'][0]['Outcome']['Status']=='succeeded' and s['Roots'][0]['Outcome']['Reason']=='member_succeeded'
    if record['kind']=='automatic':
        spending = 2 if record['candidate_name']=='a' else 3
        assert s['Roots'][0]['Charges']['Invocations']==spending and record['fixed_runs']==1
    return actual


def main():
    labels = ['wrapper-before','red-order','se-normal','historical-normal','se-race','historical-race',
              'discovery','vet','toolchain','format','module-verify']
    commands = {label:read(label) for label in labels}
    for label, command in commands.items():
        assert command['exit'] == (1 if label=='red-order' else 0)
        assert command['sources_unchanged']
        assert sha(json.dumps(command['sources'],sort_keys=True,separators=(',',':')).encode())==command['source_aggregate_sha256']
        if label not in ['wrapper-before','red-order']:
            assert all(sha((ROOT/p).read_bytes())==h for p,h in command['sources'].items()), label
            assert not [r for r in command['test_results'] if r['Action'] in ['fail','skip']]
    assert commands['format']['stdout']==''
    historical_source = commands['wrapper-before']['source_snapshots']['internal/substitutiontest/invocation_join_evidence_test.go']
    assert sha(historical_source.encode())=='e8882db6ba7053a88d33a8bc06355ee63ce55444ba879767a7341bbf2d3999e0'
    current_source = commands['se-normal']['source_snapshots']['internal/substitutiontest/invocation_join_evidence_test.go']
    # The raw validation body through RawValid is text-identical: only the
    # checker wrapper was extracted. This complements executed trace equality.
    start, end = '\tout.Handles = map[string]string{}', '\tout.RawValid = true'
    body = lambda s:s[s.index(start):s.index(end)+len(end)]
    assert body(historical_source)==body(current_source)
    before = exactly_one(commands['wrapper-before']['records'], '/equivalence.json.gz')
    assert before['count']==48
    red = exactly_one(commands['red-order']['records'], '/rejected-original.json.gz')
    raw_links(red['inspection'])
    assert red['trace']['RawValid'] and any(v['Code']=='consume_order' for v in red['trace']['Prefix'])
    summary = {}
    for profile in ['normal','race']:
        command = commands['se-'+profile]
        assert len(command['selected_leaves'])==19
        records = command['records']
        assert exactly_one(records,'/equivalence.json.gz')==before
        results = [v for v in records.values() if isinstance(v,dict) and 'kind' in v]
        positives = [r for r in results if r['kind']=='automatic']
        assert len(positives)==8
        for r in positives: check_positive(r)
        controls = [r for r in results if r['kind']=='control']
        assert {r['control'] for r in controls}=={'candidate-pending','s-pending','wrong-id','budget2'}
        for r in controls:
            s=r['inspection']; actual=raw_links(s); trace=r['trace']; root=root_projection(s)
            assert actual==r['causality'] and trace['RawValid'] and not trace['Prefix'] and trace['Guarantee']
            if r['control']=='candidate-pending':
                assert actual['s']['ConsumptionSequence']==3 and actual['e']['PublicationOrder']==actual['e']['ConsumptionSequence']==0
                assert root['Status']=='open' and root['PC']==3 and s['Roots'][0]['Outcome'] is None
            elif r['control']=='s-pending':
                assert actual['e']['PublicationOrder']>0 and actual['s']['PublicationOrder']==0
                assert actual['s']['ConsumptionSequence']==actual['e']['ConsumptionSequence']==0
                assert root['Status']=='open' and root['PC']==2 and s['Roots'][0]['Outcome'] is None
            else:
                assert actual['s']['ConsumptionSequence']==3 and actual['e']['ConsumptionSequence']==4
                assert root['Status']=='failed' and s['Roots'][0]['Outcome']['Status']=='failed'
            if r['control']=='wrong-id':
                assert trace['Component']
                outcome=descriptor(one([p for p in s['Tables']['composition_publications'] if p['operation_id']==actual['e']['Operation']]))
                assert outcome['ResponsePresent'] and outcome['Response']['id']['Number']==2 and outcome['Response']['value']['Number']==1
            else:
                assert not trace['Component']
            if r['control']=='budget2':
                refusal=one(r['actual_refused']); assert refusal['Reason']=='invocation_limit' and refusal['I']==refusal['R']==0
                assert s['Roots'][0]['Charges']['Invocations']==s['Roots'][0]['Charges']['CreatedRuns']==2
        orders=[r for r in results if r['kind']=='public-schedule']
        assert {r['order'] for r in orders}=={'s-first','candidate-first'}
        witnesses={}
        for r in orders:
            final=check_positive(r)
            cut=one([v for k,v in records.items() if k.endswith('/cut.json.gz') and v['order']==r['order']])
            raw=cut['inspection']; causal=raw_links(raw); root=root_projection(raw)
            assert causal==cut['causality'] and causal['s']['ConsumptionSequence']==3 and causal['e']['ConsumptionSequence']==0
            assert root['Status']=='open' and root['PC']==3 and raw['Roots'][0]['Outcome'] is None
            assert cut['ack']['sequence']==3 and cut['ack']['handles']==cut['trace']['Handles']
            assert final['e']['Operation']==causal['e']['Operation'] and final['s']['Operation']==causal['s']['Operation']
            assert (final['s']['PublicationOrder']<final['e']['PublicationOrder'])==(r['order']=='s-first')
            if r['order']=='s-first':
                assert cut['ack']['candidate_state']=='unproduced' and causal['e']['PublicationOrder']==0
                assert causal['e']['MemberSequence']==causal['e']['RequestConsumedSequence']==0 and causal['e']['MemberStatus']=='open'
                assert 'h' not in causal and raw['Roots'][0]['Charges']['Invocations']==2
            else:
                assert cut['ack']['candidate_state']=='retained-unconsumed' and 0<causal['e']['PublicationOrder']<causal['s']['PublicationOrder']
                assert causal['e']['OutcomeObservation'] and raw['Roots'][0]['Charges']['Invocations']==3
            for row in raw['Tables']['observations']:
                if row['consumed_sequence'] is not None:
                    assert row==one([x for x in r['inspection']['Tables']['observations'] if x['id']==row['id']])
            witnesses[r['order']]=dict(candidate_state=cut['ack']['candidate_state'],cut_causality=causal,final_causality=final)
        negatives=[r for r in results if r['kind']=='policy-negative']
        assert {r['context'] for r in negatives}=={'historical-es','s-first-se'}
        for r in negatives:
            raw_links(r['inspection'])
            accepted,rejected=(r['historical'],r['s_first']) if r['context']=='historical-es' else (r['s_first'],r['historical'])
            good(accepted); assert rejected['RawValid'] and not rejected['Component']
            assert any(v['Code']=='consume_order' for v in rejected['Prefix'])
        corruptions=[r for r in results if r['kind']=='decoder-negative']
        assert {r['mutation'] for r in corruptions}=={'swapped-selected-outcomes','delivery-is-not-consumption'}
        for r in corruptions:
            raw_links(r['source_inspection']); assert r['error']
        old=commands['historical-'+profile]
        assert len(old['selected_leaves'])==39
        for r in old['records'].values():
            if isinstance(r,dict) and 'inspection' in r: source_authority(r['inspection'])
        old_before=exactly_one(old['records'],'/before.json.gz')['inspection']
        old_after=exactly_one(old['records'],'/after.json.gz')['inspection']
        bl,al=links(old_before),links(old_after)
        assert bl['e']['consumptions']==1 and bl['s']['consumptions']==0 and bl['s']['publication'] is None
        assert all(bl[k]['operation']==al[k]['operation'] and al[k]['consumptions']==1 for k in ['e','s'])
        starts=[r for k,r in old['records'].items() if '/start-' in k]
        exits=[r for k,r in old['records'].items() if '/exit-' in k]
        assert len(starts)==len(exits)==2 and all(r['race_instrumented']==(profile=='race') for r in starts)
        killed=one([r for r in exits if r['signal']=='killed']); fresh=one([r for r in exits if r['exit_code']==0])
        assert killed['pid']!=fresh['pid']
        summary[profile]=dict(new_leaves=19,automatic=8,controls=4,public_schedules=2,policy_order_negatives=2,
                              detached_decoder_negatives=2,historical_exact_inspections=48,historical_rerun_leaves=39,
                              historical_recovery=dict(killed_pid=killed['pid'],fresh_pid=fresh['pid']),witnesses=witnesses)
    output=dict(scope='Independent bounded raw arithmetic/link audit; no new runtime or universal proof.',
                reused_audit=dict(path=str(old_audit.relative_to(ROOT)),sha256=sha(old_bytes),functions=functions),
                shared_raw_body_text_unchanged=True,profiles=summary)
    (DEST/'audit.json').write_text(json.dumps(output,indent=2)+'\n')
    index=dict(scope='C_se runtime qualification and affected C_es regression; no C_reason or new crash campaign.',
               commands=[{k:c[k] for k in ['label','argv','exit','elapsed_seconds','selected_leaves','source_aggregate_sha256']} for c in commands.values()],
               files={p.name:dict(sha256=sha(p.read_bytes()),bytes=p.stat().st_size) for p in sorted(DEST.iterdir()) if p.is_file() and p.name!='index.json'},
               audit=dict(path='audit.json',sha256=sha((DEST/'audit.json').read_bytes())))
    (DEST/'index.json').write_text(json.dumps(index,indent=2)+'\n')
    print(json.dumps({p:{k:v for k,v in s.items() if k!='witnesses'} for p,s in summary.items()},indent=2))

if __name__=='__main__': main()
