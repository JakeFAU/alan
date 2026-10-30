"""Postfreeze finite artifact correspondence; no frozen model or oracle mutation.

Expected instructions and policy tables are explicit independent transcriptions.
The evaluator reads emitted bytecode, not author combinators or executor.Reduce.
Receive schema eligibility is measured separately by the Go extraction test.
"""
import base64
import copy
import gzip
import hashlib
import itertools
import json
import os
from pathlib import Path
import sys


def sha(b):
    return hashlib.sha256(b).hexdigest()


def ref(name):
    return dict(Ref=name, Literal=None, Offset=None)


def lit(v):
    kind = 'bool' if isinstance(v, bool) else 'int' if isinstance(v, int) else 'list' if isinstance(v, list) else 'string'
    d = dict(Kind=kind, Number=v if kind == 'int' else 0,
             Text=v if kind == 'string' else '', Flag=v if kind == 'bool' else False)
    if kind == 'list':
        d['List'] = v
    return dict(Ref='', Literal=d, Offset=None)


def ins(op, **kw):
    d = dict(Op=op, Dst='', A=ref(''), B=ref(''), Target=0, Otherwise=0,
             Match=None, Bind=None, Effect=None, Updates=None, Outcome='')
    d.update(kw)
    return d


def invoke(target, name, payload):
    return ins('emit', Dst=name+'_operation', Updates={}, Effect=dict(
        Kind='invocation', Target=lit(target), Generation=lit(1),
        Obligation=lit('successor' if name == 's' else name), Payload=payload))


def receive(target, name, payload):
    bind = {'status': name+'_status', 'reason': name+'_reason', 'response_present': name+'_present'}
    bind.update(payload)
    return ins('receive', Match=dict(Kind='invocation', Provenance='runtime-invocation',
        Source=lit(target), Generation=lit(1), Obligation=lit('successor' if name == 's' else name),
        Operation=ref(name+'_operation')), Bind=bind)


integer = dict(Kind='int', MaxBytes=0, Enum=None)
rows = dict(Kind='list', MaxBytes=192, Enum=None,
            Element={'id': integer, 'value': integer}, MaxItems=1)
request = {'id': ref('request_id'), 'value': ref('request_value'), 'mode': ref('request_mode')}
tags = ['answer', 'candidate_failed', 'candidate_missing_reply', 'invalid_outcome']


def join_code(candidate, successor):
    c = {0: ins('set', Dst='answers', A=lit([]), Collection=rows),
         1: ins('receive', Match=dict(Kind='message', Provenance='ordinary-message',
            Source=lit('caller'), Generation=lit(1), Obligation=lit('work'), Operation=lit('')),
            Bind={'id': 'request_id', 'value': 'request_value', 'mode': 'request_mode'}),
         2: invoke(candidate, 'candidate', request),
         3: invoke(successor, 's', {k: request[k] for k in ['id', 'value']}),
         4: receive(successor, 's', {'response.id': 's_id', 'response.value': 's_value'}),
         5: receive(candidate, 'candidate', {'response.id': 'candidate_id', 'response.value': 'candidate_value'})}
    guards = [(6,'s_status',lit('succeeded'),7,53),(7,'s_reason',lit('member_succeeded'),8,51),
        (8,'s_present',lit(True),9,49),(9,'s_id',ref('request_id'),10,47),(10,'s_value',lit(0),11,45),
        (11,'request_value',lit(0),12,14),(15,'candidate_status',lit('succeeded'),16,31),
        (16,'candidate_reason',lit('member_succeeded'),17,29),(17,'candidate_present',lit(True),18,27),
        (18,'candidate_id',ref('request_id'),19,25),(19,'candidate_value',ref('expected_value'),20,23),
        (31,'candidate_status',lit('failed'),32,43),(32,'candidate_present',lit(False),33,41),
        (33,'candidate_reason',lit('member_failed'),34,36),(36,'candidate_reason',lit('missing_reply'),37,39),
        (55,'classification',lit('answer'),56,57)]
    for pc, name, rhs, yes, no in guards:
        c[pc] = ins('branch', A=ref(name), B=rhs, Target=yes, Otherwise=no)
    for pc, v in [(12,1),(14,0)]:
        c[pc] = ins('set', Dst='expected_value', A=lit(v))
    c[20] = ins('append', Dst='answers', Updates={'id': ref('candidate_id'), 'value': ref('candidate_value')})
    labels = {21:'answer',34:'candidate_failed',37:'candidate_missing_reply'}
    labels.update({pc:'invalid_outcome' for pc in [23,25,27,29,39,41,43,45,47,49,51,53]})
    for pc, tag in labels.items():
        c[pc] = ins('set', Dst='classification', A=lit(tag))
    for pc, target in [(13,15),(22,24),(24,26),(26,28),(28,30),(30,44),(35,40),(38,40),
                        (40,42),(42,44),(44,46),(46,48),(48,50),(50,52),(52,54)]:
        c[pc] = ins('jump', Target=target)
    c[54] = ins('emit', Effect=dict(Kind='message', Target=lit('caller'), Generation=lit(1),
        Obligation=lit('work'), Payload={'id':ref('request_id'), 'classification':ref('classification'), 'answers':ref('answers')}))
    c[56], c[57] = ins('end', Outcome='succeeded'), ins('end', Outcome='failed')
    assert sorted(c) == list(range(58))
    return [c[i] for i in range(58)]


def outer_code(join):
    c = {0: invoke(join, 'join', request),
         1: receive(join, 'join', {'response.id':'report_id', 'response.classification':'report_classification', 'response.answers':'report_answers'})}
    guards = [(2,'join_present',lit(True),3,60),(3,'report_id',ref('request_id'),4,58),
        (4,'request_value',lit(0),5,7),(12,'answer_seen',lit(False),13,14),
        (16,'answer_id',ref('request_id'),17,21),(17,'answer_value',ref('expected_value'),18,19),
        (23,'report_classification',lit('answer'),24,38),(24,'join_status',lit('succeeded'),25,36),
        (25,'join_reason',lit('member_succeeded'),26,34),(26,'answer_seen',lit(True),27,32),
        (27,'answers_valid',lit(True),28,30),(38,'join_status',lit('failed'),39,56),
        (39,'join_reason',lit('member_failed'),40,54),(40,'answer_seen',lit(False),41,52),
        (41,'report_classification',lit('candidate_failed'),42,44),
        (44,'report_classification',lit('candidate_missing_reply'),45,47),
        (47,'report_classification',lit('invalid_outcome'),48,50),(61,'decision',lit('answer'),62,63)]
    for pc, name, rhs, yes, no in guards:
        c[pc] = ins('branch', A=ref(name), B=rhs, Target=yes, Otherwise=no)
    for pc, name, v in [(5,'expected_value',1),(7,'expected_value',0),(8,'answer_seen',False),
                         (9,'answers_valid',True),(14,'answers_valid',False),(15,'answer_seen',True),
                         (19,'answers_valid',False),(21,'answers_valid',False)]:
        c[pc] = ins('set', Dst=name, A=lit(v))
    c[10] = ins('foreach', A=ref('report_answers'), Otherwise=23, Collection=rows)
    c[11] = ins('next', Target=10, Bind={'id':'answer_id','value':'answer_value'})
    c[22] = ins('loop', Target=10)
    labels = {28:'answer',42:'candidate_failed',45:'candidate_missing_reply',48:'invalid_outcome'}
    labels.update({pc:'invalid_join_outcome' for pc in [30,32,34,36,50,52,54,56,58,60]})
    for pc, tag in labels.items():
        c[pc] = ins('set', Dst='decision', A=lit(tag))
    for pc, target in [(6,8),(13,15),(18,20),(20,22),(29,31),(31,33),(33,35),(35,37),(37,57),
                        (43,51),(46,51),(49,51),(51,53),(53,55),(55,57),(57,59),(59,61)]:
        c[pc] = ins('jump', Target=target)
    c[62], c[63] = ins('end', Outcome='succeeded'), ins('end', Outcome='failed')
    assert sorted(c) == list(range(64))
    return [c[i] for i in range(64)]


def decode(v):
    if v['Kind'] == 'list':
        return [{k:decode(x) for k,x in row.items()} for row in v['List']]
    return v[{'int':'Number','string':'Text','bool':'Flag'}[v['Kind']]]


def value(v, data):
    assert v['Offset'] is None
    return data[v['Ref']] if v['Ref'] else decode(v['Literal'])


def bind(c, payload):
    # Exact omission is material: absent response operands must not become zero.
    return {dst:decode(payload[src]) for src,dst in c['Bind'].items() if src in payload}


def evaluate(code, start, data):
    pc, path, setters, cursors, trace, reports = start, [], [], {}, [], []
    while True:
        assert len(path) < 200, ('unexpected control cycle', path)
        c = code[pc]; op = c['Op']; path.append(pc); nextpc = pc+1
        if op == 'end':
            assert len(setters) == 1 and not cursors
            return dict(tag=data[setters[0][1]], status=c['Outcome'], path=path,
                        setter=setters[0][0], reports=reports, cursor_trace=trace)
        if op == 'branch':
            nextpc = c['Target'] if value(c['A'],data) == value(c['B'],data) else c['Otherwise']
        elif op == 'jump':
            nextpc = c['Target']
        elif op == 'set':
            data[c['Dst']] = copy.deepcopy(value(c['A'],data))
            if c['Dst'] in ['classification','decision']:
                setters.append((pc,c['Dst']))
        elif op == 'append':
            assert c['Dst'] == 'answers' and len(data[c['Dst']]) == 0
            data[c['Dst']].append({k:value(v,data) for k,v in c['Updates'].items()})
        elif op == 'emit':
            assert c['Effect']['Kind'] == 'message'
            reports.append({k:copy.deepcopy(value(v,data)) for k,v in c['Effect']['Payload'].items()})
        elif op == 'foreach':
            snapshot = copy.deepcopy(value(c['A'],data))
            assert len(snapshot) <= 1 and pc not in cursors
            cursors[pc] = dict(Snapshot=snapshot, Index=0)
        elif op == 'next':
            cursor = cursors[c['Target']]
            if cursor['Index'] == len(cursor['Snapshot']):
                del cursors[c['Target']]; nextpc = code[c['Target']]['Otherwise']
            else:
                row = cursor['Snapshot'][cursor['Index']]
                data.update({dst:row[src] for src,dst in c['Bind'].items()})
                cursor['Index'] += 1
        elif op == 'loop':
            nextpc = c['Target']+1
        else:
            raise AssertionError(('unsupported post-selection instruction', pc, op))
        if op in ['foreach','next','loop']:
            trace.append(dict(pc=pc,next_pc=nextpc,cursors=copy.deepcopy(cursors)))
        pc = nextpc


def outcome(payload):
    return {k:decode(v) for k,v in payload.items()}


def classify(req, e, s):
    if s['status'] != 'succeeded' or s['reason'] != 'member_succeeded' or not s['response_present']:
        return 'invalid_outcome'
    if s['response.id'] != req['id'] or s['response.value'] != 0:
        return 'invalid_outcome'
    if e['status'] == 'succeeded' and e['reason'] == 'member_succeeded' and e['response_present']:
        if e['response.id'] == req['id'] and e['response.value'] == 1-req['value']:
            return 'answer'
    if e['status'] == 'failed' and not e['response_present']:
        if e['reason'] == 'member_failed': return 'candidate_failed'
        if e['reason'] == 'missing_reply': return 'candidate_missing_reply'
    return 'invalid_outcome'


def outer_policy(req, o):
    if not o['response_present'] or o['response.id'] != req['id']:
        return 'invalid_join_outcome'
    tag, answers = o['response.classification'], o['response.answers']
    if (tag == 'answer' and o['status'] == 'succeeded' and o['reason'] == 'member_succeeded'
            and answers == [dict(id=req['id'],value=1-req['value'])]):
        return 'answer'
    if (tag in tags[1:] and o['status'] == 'failed' and o['reason'] == 'member_failed' and not answers):
        return tag
    return 'invalid_join_outcome'


def differences(a, b, path=''):
    if type(a) is not type(b): return [path]
    if isinstance(a, dict):
        if set(a) != set(b): return [path+':keys']
        return [p for k in sorted(a) for p in differences(a[k],b[k],path+'/'+k)]
    if isinstance(a, list):
        if len(a) != len(b): return [path+':length']
        return [p for i,(u,v) in enumerate(zip(a,b)) for p in differences(u,v,path+'/'+str(i))]
    return [] if a == b else [path]


def mapping(role, code, positions):
    result = []
    for pc,c in enumerate(code):
        op = c['Op']
        if role == 'J' and pc == 0:
            model = 'private empty-list initializer stutter before original J request consumption'
        elif role == 'J' and pc == 1:
            model = 'join_request_consume: original bound public request; selected/bound/consumed atomically'
        elif op == 'emit' and c['Effect']['Kind'] == 'invocation':
            model = 'invoke_commit '+c['Dst']+': original handle and intent are one atomic local commit'
        elif op == 'receive':
            model = 'outcome_consume '+c['Match']['Operation']['Ref']+': exact original correlated handle; selection/binding/consumption atomic; absent payload omitted'
        elif op == 'set' and c['Dst'] == 'classification':
            model = ('ordered join_classify('+c['A']['Literal']['Text']+') then join_decide: ghost decision is fixed by immutable tag and verified later PC55; one concrete Set, no separate durable decision checkpoint')
        elif op == 'set' and c['Dst'] == 'decision':
            model = 'outer_decide('+c['A']['Literal']['Text']+'): immutable decision tag fixes verified PC61 terminal choice'
        elif op == 'emit':
            model = 'join_reply_commit: original id and schema-valid tag/list; authorization/admission are separate protocol events and can follow J End'
        elif op == 'end':
            model = ('join_terminal' if role == 'J' else 'outer_terminal')+'('+c['Outcome']+'): End/freeze atomic; invocation or root publication later'
        elif op == 'append':
            model = 'private list stutter: append exactly the checked candidate id/value; tag not yet set; list bound one'
        elif op in ['foreach','next','loop']:
            model = 'private cursor stutter with checked snapshot/index/bind/exhaustion: foreach creates index0, next advances or erases, loop returns to next; no public protocol change'
        elif op == 'branch':
            model = 'pure guard stutter: true PC'+str(c['Target'])+', false PC'+str(c['Otherwise'])+'; exact operands checked'
        elif op == 'jump':
            model = 'pure forward jump stutter to PC'+str(c['Target'])
        else:
            assert op == 'set'
            model = 'private '+c['Dst']+' setter stutter; original outcome/ledger/obligation unchanged'
        result.append(dict(pc=pc,instruction=c,model=model,author_position=positions[pc]))
    return result


def finite_check(r, role):
    code = r['JoinProgram' if role == 'J' else 'Outer']['Code']
    domain = r['InnerEnvelopes' if role == 'J' else 'OuterEnvelopes']
    eligible = [row['Payload'] for row in domain if row['Accepted']]
    cases, counts, reached, paths, witnesses, max_steps = 0, {}, set(), set(), {}, 0
    tuples = itertools.product([1,2],[0,1],eligible,eligible) if role == 'J' else itertools.product([1,2],[0,1],eligible)
    for row in tuples:
        req = dict(id=row[0],value=row[1]); data = dict(request_id=row[0],request_value=row[1])
        if role == 'J':
            e,s = row[2:]; expected = classify(req,outcome(e),outcome(s))
            data.update(answers=[]); data.update(bind(code[4],s)); data.update(bind(code[5],e))
            run = evaluate(code,6,data)
            report = dict(id=req['id'],classification=expected,answers=[dict(id=req['id'],value=1-req['value'])] if expected == 'answer' else [])
            assert run['reports'] == [report], (req,e,s,expected,run)
        else:
            o = row[2]; expected = outer_policy(req,outcome(o)); data.update(bind(code[1],o))
            run = evaluate(code,2,data); assert not run['reports']
        assert run['tag'] == expected and run['status'] == ('succeeded' if expected == 'answer' else 'failed'), (row,expected,run)
        reached.update(run['path']); paths.add(tuple(run['path'])); max_steps = max(max_steps,len(run['path']))
        counts[expected] = counts.get(expected,0)+1; cases += 1
        # Keep a concrete complete input/trace for every distinct path, including cursor snapshots.
        key = ','.join(map(str,run['path']))
        if key not in witnesses: witnesses[key] = dict(request=req,envelopes=row[2:],execution=run)
    unreachable = sorted(set(range(6 if role == 'J' else 2,len(code)))-reached)
    # Successful absent outcomes are schema-rejected. Two rows and unknown tags are also rejected.
    # The adjacent shared jumps remain reachable: PC26->28 and PC48->50.
    assert unreachable == ([27,49] if role == 'J' else [14,50]), unreachable
    return dict(cases=cases,schema_inputs=len(domain),accepted_inputs=len(eligible),
        schema_rejected_inputs=len(domain)-len(eligible),counts=counts,visited_pcs=sorted(reached),
        schema_unreachable_pcs=unreachable,distinct_paths=len(paths),max_commits_after_selection=max_steps,
        witnesses=list(witnesses.values()),absent_payload_omitted=True)


def program_bounds(program, raw):
    code = program['Code']; registers = set((program['Author'].get('Inputs') or {}).get('Record') or {})
    payloads, bindings = 0, 0
    for pc,c in enumerate(code):
        if c['Dst']: registers.add(c['Dst'])
        registers.update((c['Bind'] or {}).values())
        if c['Effect']: payloads = max(payloads,len(c['Effect']['Payload']))
        bindings = max(bindings,len(c['Bind'] or {}))
        if c['Op'] == 'loop':
            assert c['Target'] == 10 and pc == 22 and code[10]['Collection'] == rows
        else:
            targets = [] if c['Op'] == 'end' else [c['Target']] if c['Op'] == 'jump' else [c['Target'],c['Otherwise']] if c['Op'] == 'branch' else [pc+1]
            if c['Op'] == 'next': targets.append(code[c['Target']]['Otherwise'])
            if c['Op'] == 'foreach': targets.append(c['Otherwise'])
            assert all(pc < target < len(code) for target in targets), (pc,targets)
    # Each acyclic PC executes at most once, except next executes once more for exhaustion.
    max_commits = len(code)+sum(c['Op'] == 'next' for c in code)
    assert len(registers) <= 32 and max(map(len,registers),default=0) <= 64
    # Actual finite handles <=256 bytes of text; bound includes JSON escaping all 256 bytes.
    scalar = len(json.dumps(dict(Kind='string',Number=-(2**63),Text='\x00'*256,Flag=False),separators=(',',':')).encode())
    slot = max(scalar,512)
    projection_upper = 2048+32*(66+slot+2)  # one bounded cursor/snapshot and envelope allowance
    program_upper = len(raw)+len(code)*4*6*256  # four possibly escaped replacements per PC
    assert projection_upper < 1048576 and program_upper < 1048576
    return dict(program_id=sha(raw),instruction_count=len(code),retained_bytes=len(raw),
        registers=sorted(registers),max_emit_fields=payloads,max_receive_bindings=bindings,
        max_commits=max_commits,projection_bytes_upper=projection_upper,bound_program_bytes_upper=program_upper)


def main():
    source = Path(sys.argv[1]); x = json.loads(gzip.decompress(source.read_bytes()))
    expected_inputs = [dict(ID=i,Value=v,Mode=m) for i in [1,2] for v in [0,1] for m in ['answer','fail','absent','pending']]
    diffs = {k:differences(x['a'][k],x['b'][k]) for k in ['JoinProgram','Outer']}
    assert diffs['JoinProgram'] == ['/Author/Invocations/candidate/TemplateID','/Code/2/Effect/Target/Literal/Text','/Code/5/Match/Source/Literal/Text']
    assert diffs['Outer'] == ['/Author/Invocations/join/TemplateID','/Code/0/Effect/Target/Literal/Text','/Code/1/Match/Source/Literal/Text']
    assert x['a']['Successor'] == x['b']['Successor']
    bindings, all_programs, maps, pending = {}, {}, [], []
    source_paths = {str(source),__file__,'docs/foundational/phase1i/evidence/model/artifact-correspondence.json'}
    for name,r in x.items():
        assert r['InputCases'] == expected_inputs and all(r['FrozenTemplates'].values())
        assert r['JoinProgram']['Code'] == join_code(r['CandidateID'],r['SuccessorID']), name
        assert r['Outer']['Code'] == outer_code(r['JoinID']), name
        assert not r['Outer']['Author']['Ports']
        port = r['JoinProgram']['Author']['Ports']['public']
        assert port['Peer'] == 'caller' and port['Obligation'] == 'work' and port['SendUsed'] and port['ReceiveUsed'] and not port['Local']
        assert port['Send']['Record'] == dict(id=integer,classification=dict(Kind='string',MaxBytes=32,Enum=tags),answers=rows)
        assert r['JoinProgram']['Author']['Collections'] == dict(answers=rows)
        assert r['Join']['Input'] == port['Receive'] and r['Join']['Output'] == port['Send']
        for role,key in [('candidate','Candidate'),('successor','Successor')]:
            assert r['JoinProgram']['Author']['Invocations'][role] == dict(TemplateID=r[key+'ID'],Input=r[key]['Input'],Output=r[key]['Output'])
        assert r['Outer']['Author']['Invocations']['join'] == dict(TemplateID=r['JoinID'],Input=r['Join']['Input'],Output=r['Join']['Output'])
        p = r['Provision']; root = p['Composition']['Roots'][0]
        assert len(p['Composition']['Roots']) == 1 and root['Limits'] == dict(MaxInvocations=4,MaxCreatedRuns=4,MaxDepth=3,MaxEffects=0,MaxInFlight=1)
        assert not p['Assets'] and not p['Permissions'] and not root['Capabilities']
        assert p['Subjects'] == [dict(ID='root',Generation=1,Owner='owner:root',Revision=1,Retired=False,RetryCancelled=False)]
        assert len(p['Runs']) == 1 and p['Runs'][0]['ProgramID'] == r['OuterProgramID']
        assert p['Runs'][0]['Projection'] == dict(PC=0,Data={k:lit(v)['Literal'] for k,v in [('request_id',1),('request_value',0),('request_mode','answer')]},Status='open')
        assert root['Run'] == dict(Subject='root',Generation=1) and root['Members'] == [root['Run']]
        tids = sorted(t['ID'] for t in p['Composition']['Templates']); assert sorted(root['Templates']) == tids
        for t in p['Composition']['Templates']:
            assert list(t['Members']) == ['entry'] and t['EntryMember'] == 'entry' and t['EntryPort'] == 'public' and t['CallerPeer'] == 'caller' and not t['Capabilities']
            assert t['Members']['entry']['Initial'] == dict(PC=0,Data={},Status='open')
        bounds = []
        for pr in p['Programs']:
            raw = base64.b64decode(pr['Bytes']); program = json.loads(raw)
            assert sha(raw) == pr['ID'] and not program['Author']['Effects']
            all_programs[pr['ID']] = raw; bounds.append(program_bounds(program,raw))
        for key,role in [('JoinProgram','J'),('Outer','O')]:
            program = r[key]; pid = r[key+'ID' if key == 'JoinProgram' else 'OuterProgramID']
            assert json.loads(all_programs[pid]) == program
            assert all(c['A']['Ref'] != 'request_mode' and c['B']['Ref'] != 'request_mode' for c in program['Code'][6 if role == 'J' else 2:])
            maps.append(dict(role=role+'['+name+']',program_id=pid,instructions=mapping(role,program['Code'],program['Author']['Positions'])))
        arrival, blocked = (9,11) if name == 'a' else (10,12)
        pid = r['Candidate']['Descriptor']['Members']['entry']['ProgramID']; code = json.loads(all_programs[pid])['Code']
        assert code[arrival]['Op'] == 'branch' and code[arrival]['Otherwise'] == blocked and code[blocked]['Match'] == code[0]['Match']
        pending.append(dict(binding=name,program_id=pid,branch_pc=arrival,unmatched_receive_pc=blocked,
            required_original_helper_await_pc=2 if name == 'b' else None,
            mapping='branch arrival expands private_step then pending_receive; second Receive is not consumed; original request/helper consumption and no release must be checked in retained history'))
        bindings[name] = dict(input_cases=len(expected_inputs),template_ids=tids,programs=bounds,
            join_policy=finite_check(r,'J'),outer_policy=finite_check(r,'O'),
            max_local_commits=sum(row['max_commits'] for row in bounds),
            max_dynamic_invocations=3 if name == 'a' else 4,max_dispatch_calls=9 if name == 'a' else 12)
        for role in ['candidate','s']:
            source_paths.add('docs/foundational/phase1k/evidence/context/frozen/'+name+'-'+role+'-template.json')
    old = json.loads(Path('docs/foundational/phase1i/evidence/model/artifact-correspondence.json').read_text())
    for row in old['records']:
        pid,path = row['artifact_bytes_sha256'],row['artifact_path']; source_paths.add(path)
        assert all_programs[pid] == Path(path).read_bytes()
        code = json.loads(all_programs[pid])['Code']
        assert len(code) == len(row['instructions']) and all(i['pc'] == pc and i['instruction'] == code[pc] and i['model'] for pc,i in enumerate(row['instructions']))
        maps.append(dict(role=row['name'],program_id=pid,instructions=row['instructions'],mapping_source='historical Phase1I exact bytes; shared reviewed semantics, not new independent authorship'))
    sid = x['a']['Successor']['Descriptor']['Members']['entry']['ProgramID']
    s_code = [ins('receive',Match=dict(Kind='message',Provenance='ordinary-message',Source=lit('caller'),Generation=lit(1),Obligation=lit('substitution'),Operation=lit('')),Bind={'id':'id'}),
        ins('emit',Effect=dict(Kind='message',Target=lit('caller'),Generation=lit(1),Obligation=lit('substitution'),Payload={'id':ref('id'),'value':lit(0)})),ins('end',Outcome='succeeded')]
    assert json.loads(all_programs[sid])['Code'] == s_code
    maps.append(dict(role='S',program_id=sid,instructions=[dict(pc=pc,instruction=c,model=m) for pc,c,m in zip(range(3),s_code,['request_consume s: original id','reply_commit s: original id/zero; authorization/admission separate','terminal_freeze s: Success; publication waits for admitted reply'])]))
    for part,field in [('root','Templates'),('run','ProgramID')]:
        a = copy.deepcopy(x['a']['Provision']['Composition']['Roots'][0] if part == 'root' else x['a']['Provision']['Runs'][0])
        b = copy.deepcopy(x['b']['Provision']['Composition']['Roots'][0] if part == 'root' else x['b']['Provision']['Runs'][0])
        a.pop(field); b.pop(field); assert a == b
    # Observation/Intent have at most ten scalar metadata positions after RunID
    # expansion, plus at most six payload fields here. One Emit produces one intent.
    # A 32-slot allowance, charging even scalar metadata as full schema.Value
    # objects, exceeds both actual envelopes and their JSON delimiters/keys.
    escaped_scalar = len(json.dumps(dict(Kind='string',Number=-(2**63),Text='\x00'*256,Flag=False),separators=(',',':')).encode())
    envelope_upper = 2048+32*(66+max(escaped_scalar,512)+2)
    handle_bytes = len(json.dumps(['f'*64,1,max(b['max_local_commits'] for b in bindings.values()),0],separators=(',',':')).encode())
    assert handle_bytes < 256 and envelope_upper < 1048576
    finite_payload_bytes = max(len(json.dumps(row['Payload'],separators=(',',':')).encode())
                              for r in x.values() for key in ['InnerEnvelopes','OuterEnvelopes']
                              for row in r[key] if row['Accepted'])
    assert finite_payload_bytes < 2048
    consumer_source = 'examples/callablejoin/consumer/main.go'
    assert 'MaxSteps: 512, MaxDispatch: 64, MaxInFlight: 1' in Path(consumer_source).read_text()
    assert all(b['max_local_commits'] < 512 and b['max_dispatch_calls'] < 64 for b in bindings.values())
    source_paths.add(consumer_source)
    source_paths.update(['examples/callablejoin/join.go','examples/callablejoin/outer.go','examples/callablejoin/deployment.go',
        'internal/executor/collections.go','internal/executor/invocation_payload.go','internal/executor/composition_binding.go',
        'internal/executor/invocation.go','internal/executor/invocation_outcome.go','internal/executor/messages.go',
        'internal/executor/runner.go','internal/executor/program.go','internal/executor/composition_types.go','internal/executor/store.go','schema/schema.go'])
    result = dict(scope='Postfreeze static complete instruction correspondence and schema-eligible finite policy evaluation; not runtime qualification or a compiler theorem.',
        artifact_sha256=sha(source.read_bytes()),bindings=bindings,difference_paths=diffs,instruction_records=maps,
        mapped_instructions=sum(len(r['instructions']) for r in maps),pending_mapping=pending,
        classification_commit_mapping='J classification Set expands classify then ghost decide, fixed by immutable tag; only jumps, one Send, and exact tag-to-End PC55 follow. Concrete decision checkpoint is not asserted. O decision Set similarly fixes PC61. The abstract model may interleave protocol transitions between the expansion events; the concrete atomic commit chooses no interleaving.',
        atomic_groups=['Invoke+original handle retention+intent commit','Receive+selection+binding+consumption','End+freeze','runtime outcome publish+delivery; abstract stages split'],
        raw_binding_obligation='Task4 must authenticate original source/generation/revision/caller/route/schema/binding and first disposition; static matching and normalized ReplyBound do not establish raw authentication.',
        operational_bounds=dict(configured_steps=512,configured_dispatches=64,capacity=1,byte_caps=1048576,
            original_handle_bytes_upper=handle_bytes,worst_escaped_scalar_bytes=escaped_scalar,
            observation_and_single_intent_array_bytes_upper=envelope_upper,
            largest_checked_schema_payload_bytes=finite_payload_bytes,
            dispatch_derivation='At most 3 calls per admitted singleton: admission/request insertion, optional reply delivery, ready outcome settlement/delivery; roots uncharged. No retry/error/crash/closure. A has J+e+S, B adds helper.',
            instruction_derivation='All control edges forward except one statically bounded O loop. Every PC at most once; next twice for at most one element plus exhaustion. Sum includes even mutually exclusive branches and impossible pending-release paths.',
            bytes_scope='Finite requests/answers/tags, fixed original identities/handles (at most 256 text bytes), finite reasons and no additional error/retry strings. Projection bound includes 32 slots and cursor snapshot; bound program permits four escaped 256-byte substitutions per instruction. Not arbitrary external values.'),
        schema_unreachability=dict(J='PC27 and49 require succeeded absent response, rejected by original receive schema; their shared successor jumps28/50 remain reachable. Other post-selection PCs witnessed.',
            O='PC14 requires second list row, excluded by MaxItems1; PC50 requires a tag outside the four enum members. All other post-selection PCs witnessed.'),
        sources={p:sha(Path(p).read_bytes()) for p in sorted(source_paths)})
    dest = Path(os.environ['ALAN_MODEL_EVIDENCE'])/'correspondence.json'
    dest.write_text(json.dumps(result,sort_keys=True,indent=2)+'\n')
    print(json.dumps(dict(mapped_instructions=result['mapped_instructions'],bindings={k:dict(join_cases=v['join_policy']['cases'],outer_cases=v['outer_policy']['cases'],max_steps=v['max_local_commits'],max_dispatch=v['max_dispatch_calls']) for k,v in bindings.items()},output_sha256=sha(dest.read_bytes()))))


if __name__ == '__main__':
    main()
