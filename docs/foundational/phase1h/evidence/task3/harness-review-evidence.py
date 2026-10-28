from pathlib import Path
import json,gzip,hashlib,collections
D=Path('docs/foundational/phase1h/evidence/task3/development')
def read(p):return json.loads(gzip.decompress(p.read_bytes()))
def rows(s,k):return s['Tables'][k]
def proj(s,who):return json.loads(next(r['projection'] for r in rows(s,'runs') if r['subject']==who))
def ids(q,k):return [r['work_id']['Number'] for r in q['Data'][k].get('List',[])]
def records(folder):return [json.loads(r) for r in gzip.decompress((D/folder/'records.jsonl.gz').read_bytes()).splitlines()]
def artifact(folder,test,name):return read(D/folder/'artifacts'/test.replace('/','_').replace('=','-')/(name+'.json.gz'))
rec=records('rebased-focused');artifact_count=0
for folder in D.iterdir():
 if not (folder/'records.jsonl.gz').exists():continue
 for r in records(folder.name):
  e=r['Evidence']
  if e.get('kind')!='retained_artifact':continue
  rel=e['path'].split('/artifacts/',1)[1];p=folder/'artifacts'/rel
  assert hashlib.sha256(p.read_bytes()).hexdigest()==e['gzip_sha256'];json.loads(gzip.decompress(p.read_bytes()));artifact_count+=1
print('Verified retained artifact compressed hashes',artifact_count)
for r in rec:
 e=r['Evidence'];kind=e.get('kind');test=r['Test']
 if kind=='crash_cut':
  b=artifact('rebased-focused',test,'before');a=artifact('rebased-focused',test,'after');ack=e['ack'];who=ack['Subject'];q=proj(b,who)
  h=next(h for h in rows(b,'history') if (h['subject'],h['generation'],h['sequence'],h['epoch'])==(who,ack['Generation'],ack['Sequence'],ack['Epoch']))
  assert json.loads(h['successor'])==q
  assert (json.loads(h['intents'])[0]['ID'] if json.loads(h['intents']) else '')==ack['Intent']
  for table in ('history','intents','composition_roots','composition_invocations','composition_charges'):
   later={json.dumps(v,sort_keys=True) for v in rows(a,table)}
   assert all(json.dumps(v,sort_keys=True) in later for v in rows(b,table))
  siblings=[]
  if e['cut']=='partial':
   for wid in ('10','11'):
    op=e['operations'][wid]; outcomes=[json.loads(v['descriptor']) for table in ('composition_results','composition_publications') for v in rows(b,table)]
    siblings.append((wid,any(v['OperationID']==op for v in outcomes)))
   assert any(not ready for _,ready in siblings)
  assert ids(proj(a,'root'),'results')==[1,2,3,7,10,11,31]
  print(test,'ACK',ack['PID'],ack['Sequence'],'results',ids(q,'results'),'work_count',q['Data']['work_count']['Number'],'sibling outcomes',siblings)
 if kind=='unknown_failed_sibling':
  b=artifact('rebased-focused',test,'before');a=artifact('rebased-focused',test,'after');unknown=a['Roots'][0]['UnknownExternal'];assert len(unknown)==1
  assert a['Roots'][0]['Outcome'] is None and b['Roots'][0]['Charges']==a['Roots'][0]['Charges']
  op=unknown[0]; assert any(json.loads(x['descriptor'])['Intent']['ID']==op for x in rows(b,'adapter_attempts'))
  assert not any(json.loads(x['descriptor']).get('ID')==op for x in rows(a,'adapter_results'))
  print(test,'unknown original attempt preserved',op,'effects',a['Roots'][0]['Charges']['Effects'])
 if kind=='faulty_application_oracle':
  s=artifact('rebased-focused',test,'faulty_execution');op=e['operations']['3'];d=next(x for x in rows(s,'composition_outcome_deliveries') if x['operation_id']==op);ob=next(x for x in rows(s,'observations') if x['id']==d['observation_id']);assert ob['consumed_sequence'] is not None
  for wid in ('10','11','31'):
   found=next(x for x in rows(s,'composition_invocations') if json.loads(x['descriptor'])['OperationID']==e['operations'][wid]);assert json.loads(found['admission'])['Accepted']
  print(test,'gamma consumed at',ob['consumed_sequence'],'forbidden admissions verified')
 if kind=='closure_late_transport_companion':
  s=artifact('rebased-focused',test,'after');cl=artifact('rebased-focused',test,'closure_before_release');second=e['second_operation']['Text']
  auth=next(json.loads(x['result']) for x in rows(s,'attempts') if x['intent_id']==second);res=next(json.loads(x['descriptor']) for x in rows(s,'composition_reservations') if json.loads(x['descriptor'])['OperationID']==second)
  assert auth['Allowed'] and not res['Allowed'] and res['Reason']=='scope_closed';assert rows(cl,'composition_closures')==rows(s,'composition_closures');assert len(rows(cl,'adapter_results'))==0 and len(rows(s,'adapter_results'))>0
  print(test,'allowed source / denied root scope verified',second)
for r in records('rebased-capacity-fix1'):
 e=r['Evidence'];test=r['Test']
 if e.get('kind')!='capacity_held':continue
 b=artifact('rebased-capacity-fix1',test,'contact_before_progress');h=artifact('rebased-capacity-fix1',test,'held');ledger=artifact('rebased-capacity-fix1',test,'contact_ledger_before_progress')
 assert proj(b,'progress')['Data']['marker']['Number']==0 and not any(x['subject']=='progress' for x in rows(b,'observations'))
 assert proj(h,'progress')['Data']['marker']['Number']==1 and proj(h,'progress')['Status']=='succeeded'
 assert ledger['active']>=1 and any(x['Phase']=='entered' and x['Value']==e['held_value'] for x in ledger['ledger'])
 assert not any(x['Phase']=='response' and x['Value']==e['held_value'] for x in e['ledger'])
 for x in ledger['ledger']:
  if x['Phase']=='entered': assert any(json.loads(y['descriptor'])['Intent']['ID']==x['Operation'] and json.loads(y['descriptor'])['Allowed'] for y in rows(b,'adapter_attempts'))
 print(test,'held-contact then input consumption verified','peak',e['peak'],'contacts',sum(x['Phase']=='entered' for x in e['ledger']))
