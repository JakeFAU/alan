import pathlib,json,gzip,hashlib
root=pathlib.Path('/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan');e=root/'docs/foundational/phase1h/evidence';d=e/'task3/development';out=[]
def read(p):return json.loads(gzip.decompress(p.read_bytes()))
def records(folder):return [json.loads(x) for x in gzip.decompress((d/folder/'records.jsonl.gz').read_bytes()).splitlines()]
def artifact(folder,test,name):return read(d/folder/'artifacts'/test.replace('/','_').replace('=','-')/(name+'.json.gz'))
def projection(s,who):return json.loads(next(x['projection'] for x in s['Tables']['runs'] if x['subject']==who))
def ids(q,k):return [r['work_id']['Number'] for r in q['Data'][k].get('List',[])]
def outcome(s,op):return next((json.loads(r['descriptor']) for tab in ['composition_results','composition_publications'] for r in s['Tables'][tab] if json.loads(r['descriptor'])['OperationID']==op),None)
for r in records('rebased-focused'):
 v=r['Evidence'];kind=v.get('kind');test=r['Test']
 if kind=='crash_cut':
  a=artifact('rebased-focused',test,'before');b=artifact('rebased-focused',test,'after');ack=v['ack'];q=projection(a,ack['Subject']);h=next(x for x in a['Tables']['history'] if (x['subject'],x['generation'],x['epoch'],x['sequence'])==(ack['Subject'],ack['Generation'],ack['Epoch'],ack['Sequence']));assert json.loads(h['successor'])==q
  ints=json.loads(h['intents']);assert (ints[0]['ID'] if ints else '')==ack['Intent']
  for tab in ['history','intents','composition_invocations','composition_roots','composition_charges','composition_publications','composition_results','composition_reservations']:
   assert all(x in b['Tables'][tab] for x in a['Tables'][tab]),tab
  if v['cut']=='decision':assert q['Data']['work_count']['Number']==3 and len(v['operations'])==3 and ids(q,'results')==[1,2,3]
  if v['cut']=='intents':assert q['Data']['work_count']['Number']==q['Data']['trial']['Number']==6 and set(v['operations'])=={'1','2','3','7'}
  if v['cut']=='partial':assert ids(q,'results')==[1,2,3,7] and any(outcome(a,v['operations'][k]) is None for k in ['10','11'])
  assert ids(projection(b,'root'),'results')==[1,2,3,7,10,11,31]
  out.append({'test':test,'sequence':ack['Sequence'],'epoch':ack['Epoch'],'accepted':q['Data']['work_count']['Number'],'retained_prefix':ids(q,'results'),'charges_before':a['Roots'][0]['Charges'],'charges_after':b['Roots'][0]['Charges']})
 elif kind=='unknown_failed_sibling':
  a=artifact('rebased-focused',test,'before');b=artifact('rebased-focused',test,'after');op=b['Roots'][0]['UnknownExternal'][0];assert b['Roots'][0]['Outcome'] is None and a['Roots'][0]['Charges']==b['Roots'][0]['Charges'];assert any(json.loads(x['descriptor'])['Intent']['ID']==op for x in a['Tables']['adapter_attempts']);assert outcome(b,v['original_operations']['3']) is None;beta=outcome(b,v['original_operations']['2']);assert beta['Status']=='failed' and not beta['ResponsePresent'];assert len([x for x in v['provider_contacts'] if x['Phase']=='entered'])==3
  out.append({'test':test,'unknown':op,'charges':b['Roots'][0]['Charges'],'known_beta':beta['Status']})
 elif kind=='faulty_application_oracle':
  s=artifact('rebased-focused',test,'faulty_execution');assert outcome(s,v['operations']['3'])['Response']['observation']['Number']==0
  for k in ['10','11','31']:assert any(json.loads(x['descriptor'])['OperationID']==v['operations'][k] and json.loads(x['admission'])['Accepted'] for x in s['Tables']['composition_invocations'])
  assert s['Roots'][0]['Outcome']['Status']=='succeeded';assert ids(projection(s,'root'),'results')==[1,2,3,7,10,11,31]
  out.append({'test':test,'forbidden':v['forbidden_admitted'],'oracle_rejection':v['oracle_rejection']})
 elif kind=='closure_late_transport_companion':
  a=artifact('rebased-focused',test,'closure_before_release');b=artifact('rebased-focused',test,'after');op=v['second_operation']['Text'];assert not a['Tables']['adapter_results'];assert b['Tables']['adapter_results'];assert a['Tables']['composition_closures']==b['Tables']['composition_closures'];assert any(x['intent_id']==op and json.loads(x['result'])['Allowed'] for x in b['Tables']['attempts']);assert any(json.loads(x['descriptor'])['OperationID']==op and not json.loads(x['descriptor'])['Allowed'] and json.loads(x['descriptor'])['Reason']=='scope_closed' for x in b['Tables']['composition_reservations']);assert all(b['Roots'][0]['Charges'][k]==v for k,v in {'Invocations':0,'CreatedRuns':0,'Effects':1}.items());out.append({'test':test,'source_allowed_scope_denied':True})
for r in records('rebased-capacity-fix1'):
 v=r['Evidence'];test=r['Test']
 if v.get('kind')!='capacity_held':continue
 a=artifact('rebased-capacity-fix1',test,'contact_before_progress');b=artifact('rebased-capacity-fix1',test,'held');assert projection(a,'progress')['Data']['marker']['Number']==0;assert not any(x['subject']=='progress' for x in a['Tables']['observations']);assert projection(b,'progress')['Data']['marker']['Number']==1;assert not any(x['Value']==v['held_value'] and x['Phase']=='response' for x in v['ledger']);assert v['peak']<=v['capacity'];out.append({'test':test,'contacts_while_held':len([x for x in v['ledger'] if x['Phase']=='entered']),'peak':v['peak'],'post_contact_progress':True})
# The real original counterexample is retained, not merely quoted by a review.
for name,expected in [('cli-direct-inspect',4),('cli-nested-inspect',4),('fix-r1-cli-direct-inspect',6),('fix-r1-cli-nested-inspect',6)]:
 p=e/'task2'/(name+'.log.gz');raw=gzip.decompress(p.read_bytes());s=json.loads(raw[raw.index(b'{'):]);matches=[]
 for h in s['Snapshot']['Tables']['history']:
  q=json.loads(h['successor']);ds=q['Data'];ints=json.loads(h['intents'])
  if ds.get('rounds',{}).get('Number')==1 and ds.get('work_id',{}).get('Number')==7 and ds.get('trial',{}).get('Number')==6 and len(ints)==1 and ints[0]['Kind']=='invocation' and ds.get('handles',{}).get('List',[])==[]:
   assert ds['work_count']['Number']==expected;matches.append({'sequence':h['sequence'],'count':expected,'intent':ints[0]['ID']})
 assert matches;out.append({'original_counter_witness':name,'matches':matches})
p=pathlib.Path('/tmp/alan-phase1h-final-snapshot-audit.json');p.write_text(json.dumps(out,indent=2)+'\n');print(json.dumps(out,indent=2))
