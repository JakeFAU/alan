import json,gzip,pathlib,hashlib,collections
ROOT=pathlib.Path('/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan')
BASE=ROOT/'docs/foundational/phase1i/evidence/runtime'
def read(p): return json.load(gzip.open(p,'rt'))
def path(profile,test): return BASE/profile/hashlib.sha256(json.dumps(test).encode()).hexdigest()[:12]
def inspection(x): return x.get('inspection',x)
def rows(s,t,key='descriptor'): return [json.loads(r[key]) for r in s['Tables'][t]]
def run(s,name): return next(r for r in s['Tables']['runs'] if r['subject']==name)
report={}
for profile in ['normal-v1','race-v1']:
 log=[json.loads(x) for x in gzip.open(BASE/(profile+'.jsonl.gz'),'rt')]
 states={r['Test']:r['Action'] for r in log if 'Test' in r and r['Action'] in ['pass','fail','skip']}
 leaves={t:a for t,a in states.items() if not any(x.startswith(t+'/') for x in states)}
 assert len(leaves)==236 and set(leaves.values())=={'pass'}
 stats=collections.Counter(); defects=[]
 for test in leaves:
  d=path(profile,test); assert d.is_dir(),test
  data={f.name.removesuffix('.json.gz'):read(f) for f in d.glob('*.json.gz')}
  starts={x['pid']:x for k,x in data.items() if k.startswith('start-')}; exits={x['pid']:x for k,x in data.items() if k.startswith('exit-')}
  for pid,s in starts.items():
   assert pid in exits and s['race_instrumented']==(profile=='race-v1')
   assert not any(x.startswith('OPENAI_API_KEY=') for x in s['environment'])
   if 'ALAN_I_REPLAY=1' in s['environment']:
    assert s['argv'][:3]==['/usr/bin/sandbox-exec','-p','(version 1)(allow default)(deny network*)']
    assert 'ALAN_I_ENDPOINT=' in s['environment'] and exits[pid]['exit_code']==0
    stats['network_denied_replay_children']+=1
  for k,a in data.items():
   if not k.startswith('ack-'): continue
   s=starts[a['PID']]; assert a['Store']==s['filter']['Store'] and a['Token']==s['filter']['Token'] and a['Scenario']==test and a['Boundary']==s['filter']['Boundary']; stats['validated_ACKs']+=1
   before=inspection(data.get('before',data.get('closed',{}).get('before',{})));
   if a['Intent']:
    assert any(i['ID']==a['Intent'] for i in rows(before,'intents'))
   if a['Boundary']=='execution.committed':
    rr=run(before,a['Subject']); assert rr['generation']==a['Generation'] and rr['epoch']==a['Epoch'] and rr['sequence']==a['Sequence']
   if test.startswith(('TestSubstitutionRecovery/','TestSubstitutionClosure/')) or test.endswith('unknown-applied-reopen'):
    assert exits[a['PID']]['signal']=='killed'
  if 'before' in data and 'after' in data:
   b=inspection(data['before']); a=inspection(data['after'])
   for tab in ['programs','intents','composition_charges','composition_admission_decisions','composition_publications','composition_replies','composition_reservations','adapter_attempts']:
    assert all(row in a['Tables'][tab] for row in b['Tables'][tab]),(test,tab)
   for old in b['Tables']['runs']:
    new=run(a,old['subject']); assert (old['generation'],old['program_id'])==(new['generation'],new['program_id']); assert new['epoch']>=old['epoch']; assert new['sequence']>=old['sequence']
   assert b['Namespace']==a['Namespace']; assert b['Roots'][0]['RootID']==a['Roots'][0]['RootID']
   for key in ['Invocations','CreatedRuns','Effects']: assert b['Roots'][0]['Charges'][key]<=a['Roots'][0]['Charges'][key]
   stats['preserved_prefixes']+=1
  if test.startswith('TestSubstitutionRecovery/'):
   b=inspection(data['before']); a=inspection(data['after']); assert run(a,'root')['epoch']>run(b,'root')['epoch']; assert len(starts)==3; assert len(set(x['binary_sha256'] for x in starts.values()))==1
   if test.endswith('outcome-retained'):
    obs=rows(b,'observations'); rootObs=[o for o in obs if o['Run']['Subject']=='root' and o['Kind']=='invocation']; assert len(rootObs)==1
    assert next(r for r in b['Tables']['observations'] if r['id']==rootObs[0]['ID'])['consumed_sequence'] is None
  if test.startswith('TestSubstitutionClosure/'):
   a=inspection(data['after']); assert a['Roots'][0]['LocalStatus']=='cancelled'; assert a['Roots'][0]['Outcome']['Status']=='cancelled'; assert any(r['Accepted'] and r['Reason']=='closed' for r in rows(a,'lifecycle_results','result'))
   if not data['after']['normalized']['Context']['Closed']: defects.append(test+': cancelled raw root is Closed=false')
  if test.startswith('TestSubstitutionRuntime/'):
   x=data['result']; s=x['inspection']; n=x['normalized']; ad=rows(s,'composition_admission_decisions'); pubs=rows(s,'composition_publications'); charges=s['Roots'][0]['Charges']
   assert charges['Invocations']==sum(a['Accepted'] for a in ad); assert charges['CreatedRuns']==sum(len(a['Members'] or []) for a in ad if a['Accepted'])
   byInv={a['Invocation']['ID']:a['Invocation']['OperationID'] for a in ad}
   for a in ad:
    i=a['Invocation']; o=n['Operations'][i['OperationID']]; assert o['Invocation']==i['ID'] and o['Template']==i['TemplateID']; assert o['Parent']==byInv.get(i['ParentInvocationID'],'')
   for o in n['Trace']:
    if o['Kind'] in ['publish','outcome_consume']:
     p=next(p for p in pubs if p['OperationID']==o['Operation']); assert (p['Status'],p['Reason'],p['ResponsePresent'])==(o['Status'],o['Reason'],o['Present'])
     if p['ResponsePresent']: assert (p['Response']['id']['Number'],p['Response']['value']['Number'])==(o['ID'],o['Value'])
   stats['matrix_identity_charge_envelopes']+=1
  if test.endswith(('wrong-id','extra-helper')):
   x=data['negative']; assert any(v['Code']==x['required_rejection'] for v in x['normalized']['Violations']); assert x['inspection']['Roots'][0]['LocalStatus']=='succeeded'; assert 'replay' in data
  if test.endswith('unknown-applied-reopen'):
   b=data['before']; a=data['after']; si=a['inspection']; attempts=rows(si,'adapter_attempts'); ep=a['endpoint']; assert [e['Phase'] for e in ep]==['applied','connection_closed']; assert len(attempts)==1 and attempts[0]['Allowed']; assert attempts[0]['Intent']['ID']==ep[0]['Operation']; assert attempts[0]['Prepared']['Request']['Key']==ep[0]['Key']; assert si['Roots'][0]['UnknownExternal']==[ep[0]['Operation']]; assert si['Roots'][0]['Charges']['Effects']==1 and not si['Tables']['adapter_results']; assert len(starts)==3
  if test.endswith(('reserved-then-closure','late-authentic-receipt')):
   a=data['after']; si=a['inspection']; ep=a['endpoint']; assert [e['Phase'] for e in ep]==['applied','response']; assert ep[0]['Operation']==ep[1]['Operation']; rs=rows(si,'adapter_results'); assert len(rs)==2
   first=next(r for r in rs if r['Disposition']=='validated'); denied=next(r for r in rs if r['Disposition']=='source_denied'); assert first['Candidate']['ID']==ep[0]['Operation'] and first['Candidate']['HTTPStatus']==200 and first['Values']['valid']['Flag']; assert denied['Candidate']['ID']!=ep[0]['Operation']
   assert any(r['IntentID']==denied['Candidate']['ID'] and r['Allowed'] for r in rows(si,'attempts','result'))
   assert any(r['OperationID']==denied['Candidate']['ID'] and not r['Allowed'] and r['Reason']=='scope_closed' for r in rows(si,'composition_reservations')); assert si['Roots'][0]['Outcome']['Status']=='cancelled'
   assert not data['closure-before-response']['inspection']['Tables']['adapter_results']; assert len(data['before']['endpoint'])==(0 if test.endswith('reserved-then-closure') else 1)
  if test.startswith('TestSubstitutionRootSuccessFence/'):
   x=data['counterexample']; assert x['initial_entitled'] and x['peer_extra_pure_sets']==32; n=x['normalized']; assert n['Context']['RootStatus']=='succeeded' and n['Context']['PeerStatus']=='failed' and n['ModelMembership']!='accepted'; assert x['boundary_census']['Counterexamples']; assert any(not a['Accepted'] and a['Reason']=='scope_closed' and a['Invocation']['Caller']['Subject']=='peer' for a in rows(x['inspection'],'composition_admission_decisions'))
 report[profile]={'leaves':len(leaves),'checks':dict(stats),'defects':defects}
print(json.dumps(report,indent=2)); pathlib.Path('/tmp/alan-phase1i-review-records-result.json').write_text(json.dumps(report,indent=2))
