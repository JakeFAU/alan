"""Finite artifact check, not a compiler proof. No runtime execution or model calls."""
from pathlib import Path
import base64, copy, functools, hashlib, json, sys
root = Path.cwd()
x = json.loads(Path(sys.argv[1]).read_text())
out = Path(sys.argv[2])
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
    a = []
    for dst, target, obligation, payload in [('candidate_operation', candidate, 'candidate', {'id': ref('request_id'), 'value': ref('request_value'), 'mode': ref('request_mode')}), ('s_operation', successor, 'successor', {'id': ref('request_id'), 'value': ref('request_value')})]:
        a.append(ins('emit', Dst=dst, Effect=dict(Kind='invocation', Target=lit(target), Generation=lit(1), Obligation=lit(obligation), Payload=payload), Updates={}))
    for prefix, target, obligation in [('candidate', candidate, 'candidate'), ('s', successor, 'successor')]:
        a.append(ins('receive', Match=dict(Kind='invocation', Provenance='runtime-invocation', Source=lit(target), Generation=lit(1), Obligation=lit(obligation), Operation=ref(prefix+'_operation')), Bind={'reason': prefix+'_reason', 'response.id': prefix+'_id', 'response.value': prefix+'_value', 'response_present': prefix+'_present', 'status': prefix+'_status'}))
    a.extend([ins('branch', A=ref('request_value'), B=lit(0), Target=5, Otherwise=7), ins('set', Dst='expected_value', A=lit(1)), ins('jump', Target=8), ins('set', Dst='expected_value', A=lit(0))])
    guards = [('candidate_status', lit('succeeded')), ('candidate_present', lit(True)), ('candidate_id', ref('request_id')), ('candidate_value', ref('expected_value')), ('s_status', lit('succeeded')), ('s_present', lit(True)), ('s_id', ref('request_id')), ('s_value', lit(0))]
    for pc, (name, rhs) in enumerate(guards, 8):
        a.append(ins('branch', A=ref(name), B=rhs, Target=pc+1, Otherwise=32-pc))
    a.append(ins('end', Outcome='succeeded'))
    a.extend(ins('end', Outcome='failed') for _ in range(8))
    return a

s_code = [ins('receive', Match=dict(Kind='message', Provenance='ordinary-message', Source=lit('caller'), Generation=lit(1), Obligation=lit('substitution'), Operation=lit('')), Bind={'id': 'id'}), ins('emit', Effect=dict(Kind='message', Target=lit('caller'), Generation=lit(1), Obligation=lit('substitution'), Payload={'id': ref('id'), 'value': lit(0)})), ins('end', Outcome='succeeded')]

def diffs(a,b,path=''):
    if type(a) is not type(b): return [path]
    if isinstance(a,dict):
        if set(a) != set(b): return [path+':keys']
        return [p for k in sorted(a) for p in diffs(a[k],b[k],path+'/'+k)]
    if isinstance(a,list):
        if len(a)!=len(b): return [path+':length']
        return [p for i,(u,v) in enumerate(zip(a,b)) for p in diffs(u,v,path+'/'+str(i))]
    return [] if a==b else [path]

parent_diff = diffs(x['a']['Parent'], x['b']['Parent'])
assert parent_diff == ['/Author/Invocations/candidate/TemplateID', '/Code/0/Effect/Target/Literal/Text', '/Code/2/Match/Source/Literal/Text'], parent_diff
assert x['a']['Successor'] == x['b']['Successor']
assert x['a']['SuccessorID'] == 'a6bfb72f823541b21e8780faf1eaab779254f5a563e5352cd631521c7ab6c61c'
records, binding_results, source_paths = [], {}, set([str(oldpath)])
all_programs = {}
for name,r in x.items():
    assert all(r['FrozenTemplateExact'].values())
    assert r['InputCases'] == [{'ID': i,'Value': v,'Mode':'answer'} for i in [1,2] for v in [0,1]]
    assert r['Parent']['Code'] == parent_code(r['CandidateID'], r['SuccessorID'])
    p = r['Provision']
    assert p['Permissions'] is None and not p['Assets']
    assert p['Subjects'] == [{'ID':'root','Generation':1,'Owner':'owner:root','Revision':1,'Retired':False,'RetryCancelled':False}]
    assert len(p['Runs']) == 1 and p['Runs'][0]['ProgramID'] == r['ParentID']
    assert p['Runs'][0]['Projection'] == {'PC':0,'Data':{'request_id':lit(1)['Literal'],'request_value':lit(0)['Literal'],'request_mode':lit('answer')['Literal']},'Status':'open'}
    rr = p['Composition']['Roots']
    assert len(rr)==1 and rr[0]['Run']=={'Subject':'root','Generation':1} and rr[0]['Members']==[{'Subject':'root','Generation':1}]
    assert rr[0]['Limits']=={'MaxInvocations':3,'MaxCreatedRuns':3,'MaxDepth':2,'MaxEffects':0,'MaxInFlight':1} and not rr[0]['Capabilities']
    templates = p['Composition']['Templates']
    tids = sorted(t['ID'] for t in templates)
    assert sorted(rr[0]['Templates']) == tids
    for t in templates:
        assert list(t['Members'])==['entry'] and t['EntryMember']=='entry' and t['EntryPort']=='public' and t['CallerPeer']=='caller' and not t['Capabilities']
        assert t['Members']['entry']['Initial']=={'PC':0,'Data':{},'Status':'open'}
    program_records=[]
    for pr in p['Programs']:
        raw=base64.b64decode(pr['Bytes']); pid=pr['ID']; program=json.loads(raw); code=program['Code']
        assert sha(raw)==pid==r['ProgramSHA256'][pid]
        assert not program['Author']['Effects']
        all_programs[pid]=raw
        edges=[]
        for pc,c in enumerate(code):
            targets=[] if c['Op']=='end' else [c['Target']] if c['Op']=='jump' else [c['Target'],c['Otherwise']] if c['Op']=='branch' else [pc+1]
            assert c['Op'] in ['end','jump','branch','set','receive','emit']
            assert all(pc < j < len(code) for j in targets), (pid,pc,targets)
            edges.append(targets)
        @functools.lru_cache(None)
        def length(pc): return 1+max((length(j) for j in edges[pc]),default=0)
        program_records.append(dict(program_id=pid,bytes=len(raw),instruction_count=len(code),forward_edges=edges,max_commits=length(0)))
        if pid==r['ParentID']:
            assert program==r['Parent'] and not program['Author']['Ports']
            assert len(program['Author']['Positions'])==len(code)
            for decl,key in [('candidate','Candidate'),('successor','Successor')]:
                invocation=program['Author']['Invocations'][decl]
                assert invocation=={'TemplateID':r[key+'ID'],'Input':r[key]['Input'],'Output':r[key]['Output']}
        else:
            ports=program['Author']['Ports']; assert list(ports)==['public']
            port=ports['public']
            assert port['Peer']=='caller' and port['Obligation']=='substitution' and port['SendUsed'] and port['ReceiveUsed'] and not port['Local']
    source_paths.update(['docs/foundational/phase1k/evidence/context/frozen/'+name+'-'+role+'-template.json' for role in ['candidate','s']])
    binding_results[name]=dict(parent_id=r['ParentID'],candidate_template_id=r['CandidateID'],s_template_id=r['SuccessorID'],template_ids=tids,programs=program_records,max_dynamic_invocations=2 if name=='a' else 3,max_fixed_runs=1,max_dynamic_created_runs=2 if name=='a' else 3,depth=2,max_local_commits=sum(pr['max_commits'] for pr in program_records),max_dispatch_calls=6 if name=='a' else 9,input_cases=r['InputCases'])

for oldrecord in old['records']:
    pid=oldrecord['artifact_bytes_sha256']; path=oldrecord['artifact_path']; source_paths.add(path)
    assert all_programs[pid] == Path(path).read_bytes()
    code=json.loads(all_programs[pid])['Code']
    assert len(code)==len(oldrecord['instructions'])
    assert all(row['pc']==pc and row['instruction']==code[pc] and row['model'] for pc,row in enumerate(oldrecord['instructions']))
    records.append(dict(role=oldrecord['name'],program_id=pid,all_instructions_exact=True,mapping_source=str(oldpath),instruction_count=len(code)))
sid='d19cc8d312c2d1f16e10fe71d9b891400bf49b5d31f0d63226c77521efe63cf8'
assert json.loads(all_programs[sid])['Code']==s_code
records.append(dict(role='S',program_id=sid,instruction_count=3,model_by_pc=['request_consume s: original id; unused value stays in original invocation envelope','reply_commit s: original id, value 0; authorization/admission remain separate','terminal_freeze s: Success; reply may be admitted before or after terminal']))
parentmap=['invoke_commit e with original id/value/mode and new candidate_operation','invoke_commit s with original id/value and new s_operation','outcome_consume e matched by original handle, template, provenance, generation, obligation','outcome_consume s matched by original handle, template, provenance, generation, obligation','private branch request_value==0, then PC5 else PC7','private expected_value=1','private jump PC8','private expected_value=0']
parentmap += ['private guard '+name+'; true -> PC'+str(pc+1)+', false -> PC'+str(32-pc) for pc,name in enumerate(['candidate_status==succeeded','candidate_present==true','candidate_id==request_id','candidate_value==expected_value','s_status==succeeded','s_present==true','s_id==request_id','s_value==0'],8)]
parentmap += ['parent_terminal Success at PC16; decide emitted at actual resolved-End boundary']+['parent_terminal Failure at PC'+str(pc)+'; decide emitted at actual resolved-End boundary' for pc in range(17,25)]
for name in ['a','b']:
    records.append(dict(role='parent['+name+']',program_id=x[name]['ParentID'],instruction_count=25,model_by_pc=parentmap))

def sets_delta(a,b): return {'shared':sorted(set(a)&set(b)),'a_only':sorted(set(a)-set(b)),'b_only':sorted(set(b)-set(a))}
catalogue_diff={'templates':sets_delta(binding_results['a']['template_ids'],binding_results['b']['template_ids']),'programs':sets_delta([p['program_id'] for p in binding_results['a']['programs']],[p['program_id'] for p in binding_results['b']['programs']])}
root_a=copy.deepcopy(x['a']['Provision']['Composition']['Roots'][0]);root_b=copy.deepcopy(x['b']['Provision']['Composition']['Roots'][0]);root_a.pop('Templates');root_b.pop('Templates');assert root_a==root_b
run_a=copy.deepcopy(x['a']['Provision']['Runs'][0]);run_b=copy.deepcopy(x['b']['Provision']['Runs'][0]);run_a.pop('ProgramID');run_b.pop('ProgramID');assert run_a==run_b
source_paths.update(['examples/invocationjoin/deployment.go','examples/invocationjoin/consumer/main.go','internal/executor/composition_binding.go','internal/executor/invocation.go','internal/executor/invocation_outcome.go','internal/executor/messages.go','internal/executor/runner.go','internal/executor/program.go'])
result=dict(scope='Inspected finite artifact transcription and syntactic bound checks; not general compiler correctness, dynamic binding execution, or runtime qualification.',artifact_input_sha256=sha(Path(sys.argv[1]).read_bytes()),bindings=binding_results,parent_exact_difference_paths=parent_diff,catalogue_differences=catalogue_diff,common_root_and_initial_except_program_id=True,original_template_bytes_unchanged=True,instruction_records=records,instruction_coverage_total=sum(r['instruction_count'] for r in records),source_sha256={p:sha(Path(p).read_bytes()) for p in sorted(source_paths)})
out.write_text(json.dumps(result,indent=2)+'\n')
print('PASS 8 input bindings; exact parent differences=3; inherited instructions=36; S=3; parent instructions=50; forward DAG bounds A=25 B=35; dispatch protocol bounds A=6 B=9')
print('output_sha256='+sha(out.read_bytes()))
