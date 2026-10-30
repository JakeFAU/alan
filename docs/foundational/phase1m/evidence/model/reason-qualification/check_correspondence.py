"""Postfreeze finite C_reason instruction check; no reducer or runtime execution.

Expected instructions/table come from the specification and inspected source.
Original component instruction mappings are reused by exact byte identity.
"""
from pathlib import Path
import base64, copy, functools, hashlib, itertools, json, sys
root = Path.cwd()
x = json.loads(Path(sys.argv[1]).read_text())
sha = lambda b: hashlib.sha256(b).hexdigest()
oldpath = Path('docs/foundational/phase1i/evidence/model/artifact-correspondence.json')
old = json.loads(oldpath.read_text())

def ref(name): return {'Ref': name, 'Literal': None, 'Offset': None}
def lit(v):
    k = 'bool' if type(v) is bool else 'int' if type(v) is int else 'string'
    return {'Ref': '', 'Literal': {'Kind': k, 'Number': v if k == 'int' else 0, 'Text': v if k == 'string' else '', 'Flag': v if k == 'bool' else False}, 'Offset': None}
def ins(op, **kw):
    d = dict(Op=op, Dst='', A=ref(''), B=ref(''), Target=0, Otherwise=0, Match=None, Bind=None, Effect=None, Updates=None, Outcome='')
    d.update(kw)
    return d

def parent_code(candidate, successor):
    code = {}
    for pc, dst, target, obligation, payload in [(0,'candidate_operation',candidate,'candidate',{'id':ref('request_id'),'value':ref('request_value'),'mode':ref('request_mode')}),(1,'s_operation',successor,'successor',{'id':ref('request_id'),'value':ref('request_value')})]:
        code[pc] = ins('emit', Dst=dst, Effect=dict(Kind='invocation',Target=lit(target),Generation=lit(1),Obligation=lit(obligation),Payload=payload),Updates={})
    for pc,prefix,target,obligation in [(2,'s',successor,'successor'),(3,'candidate',candidate,'candidate')]:
        code[pc] = ins('receive', Match=dict(Kind='invocation',Provenance='runtime-invocation',Source=lit(target),Generation=lit(1),Obligation=lit(obligation),Operation=ref(prefix+'_operation')),Bind={'reason':prefix+'_reason','response.id':prefix+'_id','response.value':prefix+'_value','response_present':prefix+'_present','status':prefix+'_status'})
    guards = [(4,'s_status',lit('succeeded'),5,50),(5,'s_reason',lit('member_succeeded'),6,48),(6,'s_present',lit(True),7,46),(7,'s_id',ref('request_id'),8,44),(8,'s_value',lit(0),9,42),(9,'request_value',lit(0),10,12),(13,'candidate_status',lit('succeeded'),14,28),(14,'candidate_reason',lit('member_succeeded'),15,26),(15,'candidate_present',lit(True),16,24),(16,'candidate_id',ref('request_id'),17,22),(17,'candidate_value',ref('expected_value'),18,20),(28,'candidate_status',lit('failed'),29,40),(29,'candidate_present',lit(False),30,38),(30,'candidate_reason',lit('member_failed'),31,33),(33,'candidate_reason',lit('missing_reply'),34,36)]
    for pc,name,rhs,yes,no in guards: code[pc] = ins('branch',A=ref(name),B=rhs,Target=yes,Otherwise=no)
    code[10],code[11],code[12] = ins('set',Dst='expected_value',A=lit(1)),ins('jump',Target=13),ins('set',Dst='expected_value',A=lit(0))
    labels = {18:'answer',31:'candidate_failed',34:'candidate_missing_reply'}
    labels.update({pc:'invalid_outcome' for pc in [20,22,24,26,36,38,40,42,44,46,48,50]})
    for pc,tag in labels.items():
        code[pc] = ins('set',Dst='classification',A=lit(tag))
        code[pc+1] = ins('end',Outcome='succeeded' if tag=='answer' else 'failed')
    assert sorted(code)==list(range(52))
    return [code[pc] for pc in range(52)]

s_code = [ins('receive',Match=dict(Kind='message',Provenance='ordinary-message',Source=lit('caller'),Generation=lit(1),Obligation=lit('substitution'),Operation=lit('')),Bind={'id':'id'}),ins('emit',Effect=dict(Kind='message',Target=lit('caller'),Generation=lit(1),Obligation=lit('substitution'),Payload={'id':ref('id'),'value':lit(0)})),ins('end',Outcome='succeeded')]

def diffs(a,b,path=''):
    if type(a) is not type(b): return [path]
    if isinstance(a,dict):
        if set(a)!=set(b): return [path+':keys']
        return [p for k in sorted(a) for p in diffs(a[k],b[k],path+'/'+k)]
    if isinstance(a,list):
        if len(a)!=len(b): return [path+':length']
        return [p for i,(u,v) in enumerate(zip(a,b)) for p in diffs(u,v,path+'/'+str(i))]
    return [] if a==b else [path]

def value(operand,data):
    if operand['Ref']:
        # A read of an absent payload is an error, not a zero-valued answer.
        return data[operand['Ref']]
    v=operand['Literal']; return v[{'string':'Text','int':'Number','bool':'Flag'}[v['Kind']]]

def classify_spec(request,e,s):
    if s['status']!='succeeded' or s['reason']!='member_succeeded' or not s['present']: return 'invalid_outcome'
    if s['id']!=request['id'] or s['value']!=0: return 'invalid_outcome'
    if e['status']=='succeeded' and e['reason']=='member_succeeded' and e['present']:
        return 'answer' if e['id']==request['id'] and e['value']==1-request['value'] else 'invalid_outcome'
    if e['status']=='failed' and not e['present']:
        if e['reason']=='member_failed': return 'candidate_failed'
        if e['reason']=='missing_reply': return 'candidate_missing_reply'
    return 'invalid_outcome'

def parent_classify(code,request,e,s):
    data={'request_id':request['id'],'request_value':request['value']}
    for prefix,outcome in [('candidate',e),('s',s)]:
        data.update({prefix+'_'+k:v for k,v in outcome.items()})
    pc=4; path=[]; setters=[]
    while True:
        c=code[pc]; path.append(pc)
        if c['Op']=='end':
            assert len(setters)==1
            return data['classification'],c['Outcome'],path,setters[0]
        if c['Op']=='branch': nextpc=c['Target'] if value(c['A'],data)==value(c['B'],data) else c['Otherwise']
        elif c['Op']=='jump': nextpc=c['Target']
        else:
            assert c['Op']=='set'
            data[c['Dst']]=value(c['A'],data); nextpc=pc+1
            if c['Dst']=='classification': setters.append(pc)
        assert nextpc>pc
        pc=nextpc

def outcome_cases():
    out=[]
    for status,reason,present in itertools.product(['succeeded','failed','rejected'],['member_succeeded','member_failed','missing_reply','invocation_limit'],[False,True]):
        for payload in ([{'id':i,'value':v} for i in [0,1,2] for v in [0,1]] if present else [{}]):
            out.append(dict(status=status,reason=reason,present=present,**payload))
    return out

parent_diff=diffs(x['a']['Parent'],x['b']['Parent'])
assert parent_diff==['/Author/Invocations/candidate/TemplateID','/Code/0/Effect/Target/Literal/Text','/Code/3/Match/Source/Literal/Text']
assert x['a']['Successor']==x['b']['Successor']
assert x['a']['SuccessorID']=='a6bfb72f823541b21e8780faf1eaab779254f5a563e5352cd631521c7ab6c61c'
source_paths={str(oldpath),__file__,'docs/foundational/phase1m/evidence/model/reason-qualification/extract.go.txt'}
records=[];bindings={};all_programs={};byte_records={};branch_checks={}
expected_inputs=[dict(ID=i,Value=v,Mode=m) for i in [1,2] for v in [0,1] for m in ['answer','fail','absent','pending']]
for name,r in x.items():
    assert all(r['FrozenTemplateExact'].values()) and r['InputCases']==expected_inputs
    assert r['Parent']['Code']==parent_code(r['CandidateID'],r['SuccessorID'])
    p=r['Provision']; assert p['Permissions'] is None and not p['Assets']
    assert p['Subjects']==[dict(ID='root',Generation=1,Owner='owner:root',Revision=1,Retired=False,RetryCancelled=False)]
    assert len(p['Runs'])==1 and p['Runs'][0]['ProgramID']==r['ParentID']
    assert p['Runs'][0]['Projection']==dict(PC=0,Data={k:lit(v)['Literal'] for k,v in [('request_id',1),('request_value',0),('request_mode','answer')]},Status='open')
    rr=p['Composition']['Roots']; assert len(rr)==1 and rr[0]['Run']==dict(Subject='root',Generation=1) and rr[0]['Members']==[dict(Subject='root',Generation=1)]
    assert rr[0]['Limits']==dict(MaxInvocations=3,MaxCreatedRuns=3,MaxDepth=2,MaxEffects=0,MaxInFlight=1) and not rr[0]['Capabilities']
    templates=p['Composition']['Templates'];tids=sorted(t['ID'] for t in templates);assert sorted(rr[0]['Templates'])==tids
    for t in templates:
        assert list(t['Members'])==['entry'] and t['EntryMember']=='entry' and t['EntryPort']=='public' and t['CallerPeer']=='caller' and not t['Capabilities']
        assert t['Members']['entry']['Initial']==dict(PC=0,Data={},Status='open')
    program_records=[];byte_records[name]=[]
    for pr in p['Programs']:
        raw=base64.b64decode(pr['Bytes']);pid=pr['ID'];program=json.loads(raw);code=program['Code']
        assert sha(raw)==pid==r['ProgramSHA256'][pid] and not program['Author']['Effects']
        all_programs[pid]=raw;edges=[]
        registers=set((program['Author'].get('Inputs') or {}).get('Record') or {});max_payload=0;max_bindings=0
        for pc,c in enumerate(code):
            targets=[] if c['Op']=='end' else [c['Target']] if c['Op']=='jump' else [c['Target'],c['Otherwise']] if c['Op']=='branch' else [pc+1]
            assert c['Op'] in ['end','jump','branch','set','receive','emit'] and not c.get('Collection')
            assert all(pc<j<len(code) for j in targets),(pid,pc,targets)
            edges.append(targets)
            if c['Dst']:registers.add(c['Dst'])
            registers.update((c['Bind'] or {}).values());registers.update((c['Updates'] or {}).keys())
            max_bindings=max(max_bindings,len(c['Bind'] or {}))
            if c['Effect']:max_payload=max(max_payload,len(c['Effect']['Payload']))
        @functools.lru_cache(None)
        def length(pc):return 1+max((length(j) for j in edges[pc]),default=0)
        program_records.append(dict(program_id=pid,bytes=len(raw),instruction_count=len(code),forward_edges=edges,max_commits=length(0)))
        assert max(map(len,registers),default=0)<=32 and len(registers)<=17 and max_payload<=3 and max_bindings<=5
        # Same finite scalar domains as C_se plus fixed classification literals.
        scalar=dict(Kind='string',Number=-(2**63),Text='\x00'*256,Flag=False)
        scalar_bytes=len(json.dumps(scalar,separators=(',',':')).encode())
        projection_upper=128+len(registers)*(34+scalar_bytes+2)
        bound_program_upper=len(raw)+len(code)*4*256
        assert projection_upper<131072 and bound_program_upper<131072
        byte_records[name].append(dict(program_id=pid,retained_program_bytes=len(raw),registers=sorted(registers),register_count=len(registers),max_emit_payload_fields=max_payload,max_receive_bindings=max_bindings,projection_bytes_upper=projection_upper,bound_program_bytes_upper=bound_program_upper))
        if pid==r['ParentID']:
            assert program==r['Parent'] and not program['Author']['Ports'] and len(program['Author']['Positions'])==len(code)
            for decl,key in [('candidate','Candidate'),('successor','Successor')]:
                assert program['Author']['Invocations'][decl]==dict(TemplateID=r[key+'ID'],Input=r[key]['Input'],Output=r[key]['Output'])
            assert all(c['A']['Ref']!='request_mode' and c['B']['Ref']!='request_mode' for c in code[4:])
        else:
            ports=program['Author']['Ports'];assert list(ports)==['public'];port=ports['public']
            assert port['Peer']=='caller' and port['Obligation']=='substitution' and port['SendUsed'] and port['ReceiveUsed'] and not port['Local']
    source_paths.update('docs/foundational/phase1k/evidence/context/frozen/'+name+'-'+role+'-template.json' for role in ['candidate','s'])
    bindings[name]=dict(parent_id=r['ParentID'],candidate_template_id=r['CandidateID'],s_template_id=r['SuccessorID'],template_ids=tids,programs=program_records,max_dynamic_invocations=2 if name=='a' else 3,max_fixed_runs=1,max_dynamic_created_runs=2 if name=='a' else 3,depth=2,max_local_commits=sum(p['max_commits'] for p in program_records),max_dispatch_calls=6 if name=='a' else 9,input_cases=r['InputCases'])
    code=r['Parent']['Code'];checked=0;reached=set();classes={}
    outcomes=outcome_cases()
    for i,v,e,s in itertools.product([1,2],[0,1],outcomes,outcomes):
        request=dict(id=i,value=v);expected=classify_spec(request,e,s)
        tag,status,path,setter=parent_classify(code,request,e,s)
        assert tag==expected and status==('succeeded' if tag=='answer' else 'failed')
        assert code[setter+1]['Op']=='end' and path[-2:]==[setter,setter+1]
        reached.update(path);classes[tag]=classes.get(tag,0)+1;checked+=1
    assert reached==set(range(4,52)),set(range(4,52))-reached
    branch_checks[name]=dict(cases=checked,outcome_domain_size=len(outcomes),expected_class_counts=classes,all_guard_setter_end_pcs_reached=sorted(reached),absent_payload_fields_omitted=True,scope='Finite independent post-join artifact evaluator vs policy table; not model census or runtime execution.')

for row in old['records']:
    pid=row['artifact_bytes_sha256'];path=row['artifact_path'];source_paths.add(path)
    assert all_programs[pid]==Path(path).read_bytes()
    code=json.loads(all_programs[pid])['Code'];assert len(code)==len(row['instructions'])
    assert all(i['pc']==pc and i['instruction']==code[pc] and i['model'] for pc,i in enumerate(row['instructions']))
    records.append(dict(role=row['name'],program_id=pid,all_instructions_exact=True,mapping_source=str(oldpath),instruction_count=len(code),instructions=row['instructions']))
sid='d19cc8d312c2d1f16e10fe71d9b891400bf49b5d31f0d63226c77521efe63cf8'
assert json.loads(all_programs[sid])['Code']==s_code
records.append(dict(role='S',program_id=sid,instruction_count=3,model_by_pc=['request_consume s: original id; unused value stays in original request','reply_commit s: original id/zero; separate authorization/admission','terminal_freeze s: Success; publication still waits for reply admission']))

for name,r in x.items():
    mapping=[]
    for pc,c in enumerate(r['Parent']['Code']):
        if pc==0: model='invoke_commit e: original id/value/mode; retain candidate_operation'
        elif pc==1: model='invoke_commit s: original id/value; retain s_operation'
        elif pc in [2,3]:model='outcome_consume '+('s' if pc==2 else 'e')+': exact original operation/template/provenance/generation/obligation; bind status/reason/presence/payload'
        elif c['Op']=='set' and c['Dst']=='classification':
            model='ordered parent_classify('+c['A']['Literal']['Text']+') then parent_decide('+r['Parent']['Code'][pc+1]['Outcome']+'): one concrete committed Set reaches the adjacent End; not two commits or a checkpoint'
        elif c['Op']=='end':model='parent_terminal('+c['Outcome']+'): executing End is its own later commit; root_publish remains subsequent protocol settlement'
        elif c['Op']=='branch':model='pure guard stutter; true PC'+str(c['Target'])+', false PC'+str(c['Otherwise'])+'; no classified answer/decision yet'
        elif c['Op']=='set':model='pure expected_value setter stutter; no original outcome/obligation/charge change'
        elif c['Op']=='jump':model='pure forward jump stutter to PC'+str(c['Target'])
        else:raise AssertionError(c)
        mapping.append(dict(pc=pc,instruction=c,model=model,author_position=r['Parent']['Author']['Positions'][pc]))
    records.append(dict(role='parent['+name+']',program_id=r['ParentID'],instruction_count=len(mapping),instructions=mapping))

pending=[]
for name,arrival,receive_pc in [('a',9,11),('b',10,12)]:
    template=x[name]['Candidate'];member_pid=template['Descriptor']['Members']['entry']['ProgramID'];code=json.loads(all_programs[member_pid])['Code']
    assert code[arrival]['Op']=='branch' and code[arrival]['Otherwise']==receive_pc
    assert code[receive_pc]['Op']=='receive' and code[receive_pc]['Match']==code[0]['Match']
    assert code[0]['Op']=='receive' and code[receive_pc]['Match']['Kind']=='message'
    pending.append(dict(binding=name,program_id=member_pid,branch_pc=arrival,unmatched_receive_pc=receive_pc,match=code[receive_pc]['Match'],bind=code[receive_pc]['Bind'],required_original_helper_await_pc=2 if name=='b' else None,model='The authenticated branch commit arriving at the actual second Receive expands to private_step then pending_receive. No second request was selected or consumed; it is an arrival/readiness event, not a receive commit.',raw_obligations='Verify original candidate identity/program binding, consumed first request, executed guard path and actual open post-PC. B also consumed its original successful helper at PC2. Require no matching release observation, candidate publication or terminal. Never infer from requested mode/open status alone.'))

def sets_delta(a,b):return dict(shared=sorted(set(a)&set(b)),a_only=sorted(set(a)-set(b)),b_only=sorted(set(b)-set(a)))
catalogue=dict(templates=sets_delta(bindings['a']['template_ids'],bindings['b']['template_ids']),programs=sets_delta([p['program_id'] for p in bindings['a']['programs']],[p['program_id'] for p in bindings['b']['programs']]))
for part,field in [('root','Templates'),('run','ProgramID')]:
    aa=copy.deepcopy(x['a']['Provision']['Composition']['Roots'][0] if part=='root' else x['a']['Provision']['Runs'][0]);bb=copy.deepcopy(x['b']['Provision']['Composition']['Roots'][0] if part=='root' else x['b']['Provision']['Runs'][0]);aa.pop(field);bb.pop(field);assert aa==bb
handle_bytes=len(json.dumps(['f'*64,1,38,0],separators=(',',':')).encode());assert handle_bytes<256
slots=dict(Observation=12,Intent=8,AdmissionDescriptor=11,compositionInvocation=9,compositionAdmission=12,compositionOutcome=9,compositionFreeze=7,compositionReply=16)
envelope_upper=512+32*(scalar_bytes+34+2);assert max(slots.values())<=32 and envelope_upper<131072
source_paths.update(['examples/consumingpolicy/reason.go','examples/consumingpolicy/deployment.go','examples/consumingpolicy/consumer/main.go','internal/executor/composition_binding.go','internal/executor/invocation.go','internal/executor/invocation_outcome.go','internal/executor/messages.go','internal/executor/runner.go','internal/executor/program.go','internal/executor/composition_types.go','internal/executor/invocation_payload.go','internal/executor/store.go','schema/schema.go','internal/substitutiontest/invocation_join_evidence_test.go'])
result=dict(scope='Postfreeze finite instruction and guard transcription, bounded artifact evaluator, source-derived operational bounds. Not a compiler theorem, raw runtime authentication or Task6 runtime qualification.',artifact_input_sha256=sha(Path(sys.argv[1]).read_bytes()),bindings=bindings,parent_exact_difference_paths=parent_diff,catalogue_differences=catalogue,common_root_and_initial_except_program_id=True,original_template_bytes_unchanged=True,instruction_records=records,instruction_coverage_total=sum(r['instruction_count'] for r in records),classification_branch_checks=branch_checks,pending_mapping=pending,classification_commit_mapping='Classification Set writes the tag and reaches End in the SAME committed history step. Its ordered abstract expansion is parent_classify then parent_decide. The intermediate abstract state does not represent a distinct durable runtime checkpoint; the model overapproximates possible protocol interleavings between the two events. Executed End is separate.',operational_bounds=dict(configured_steps=200,configured_dispatches=200,configured_capacity=1,default_byte_caps=dict(MaxProgramBytes=1048576,MaxDataBytes=1048576,MaxObservationBytes=1048576,MaxOutputBytes=1048576),maximum_original_handle_bytes=handle_bytes,scalar_text_bytes_upper=256,worst_escaped_scalar_bytes=scalar_bytes,envelope_scalar_slot_limits=slots,protocol_envelope_bytes_upper=envelope_upper,conservative_common_bytes_upper=131072,programs=byte_records,dispatch_derivation='Per admitted singleton: initial invocation admission with atomic request insertion; at most one original reply-message delivery; then ready original outcome settlement/delivery. Admitted unready invocations are omitted by pendingDeliveries, settled ones removed. Thus A+S<=6, B+helper+S<=9. Fail/absent omit reply; pending omits outcome. Root settlement is uncharged. No retry/error/crash schedule is included.',scope='All declared finite pure stable inputs, plus finite classification constants. Not arbitrary strings/adapters/errors. Longest forward code path bounds include hypothetical paths beyond pending and therefore overestimate without assuming a release.'),source_sha256={p:sha(Path(p).read_bytes()) for p in sorted(source_paths)})
Path(sys.argv[2]).write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(dict(instructions=result['instruction_coverage_total'],bindings={k:dict(inputs=len(v['input_cases']),max_steps=v['max_local_commits'],max_dispatch=v['max_dispatch_calls']) for k,v in bindings.items()},classification_cases={k:v['cases'] for k,v in branch_checks.items()},output_sha256=sha(Path(sys.argv[2]).read_bytes())),indent=2))
