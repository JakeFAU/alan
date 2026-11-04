import json,pathlib
BASE=pathlib.Path('/tmp/alan-phase3b3-runtime-development')
def read(p):return json.loads(p.read_text())
def canon(x):return json.dumps(x,sort_keys=True)
def effect(s,id):return next((e for e in s['Effects'] or [] if e['OperationID']==id),None)
def runs(s):return {x['ID']['Subject']:x for x in s['Runs']}
def facts(raw):return [json.loads(x['descriptor']) for x in raw['Tables']['composition_facts']]
def snapshot(d,phase):
 s=read(d/phase/'typed-inspection.json');raw=read(d/phase/'inspect.json');out=read(d/phase/'outcomes.json');assert all(x=='<nil>' for x in out.values());return s,raw
IMMUTABLE=['programs','subjects','permissions','composition_permissions','composition_profile','composition_static_members','composition_roots','run_origins']
APPEND=['attempts','external_attempts','adapter_attempts','lifecycle_results','composition_charges','composition_reservations','composition_facts']
def preserve(a,b,tables):
 for tab in tables:assert {canon(x) for x in a['Tables'][tab]} <= {canon(x) for x in b['Tables'][tab]},tab

def account(s,raw):
 for e in s['Effects'] or []:
  at=e['ExternalAttempt'] or e['AdapterAttempt']; cs=[json.loads(x['descriptor']) for x in raw['Tables']['composition_charges'] if x['operation_id']==e['OperationID']]
  assert len(cs)==int(at is not None and at['Allowed'])
  for c in cs:assert c['Effects']==1 and c['RootID']==runs(s)[e['Source']['Subject']]['RootID'] and c['Invocations']==c['CreatedRuns']==0

def journal(d,ep,snap,attempt=None):
 events=[json.loads(x) for x in (d/f'endpoint-{ep}.jsonl').read_text().splitlines()]
 ingress=[e for e in events if e['Event']=='ingress'];commits=[e for e in events if e['Event']=='committed'];assert len(ingress)==len(commits)==snap['Contacts']
 if attempt:
  assert all(json.loads(e['Body'])==attempt['Request'] for e in ingress)
  assert all(q==attempt['Request'] for q in snap['Requests'] or [])
 if snap['Applications']:assert all(e['Receipt']==snap['Applications'][0] for e in commits)
 return len(ingress)

results=[]
for corpus in ['qualified-ordinary','qualified-race']:
 ix=read(BASE/corpus/'case-index.json');assert len(ix)==48
 cuts=[c for c in ix if c['Name'].startswith('cut-')];assert len(cuts)==11
 for c in cuts:
  d=pathlib.Path(c['Directory']);m=c['Manifest'];aid=m['BAService'];n=c['Name'];st,sr=snapshot(d,'stopped');fo,fr=snapshot(d,'fresh-owner/post-open');fi,rr=snapshot(d,'final');ini,ir=snapshot(d,'initial');end=read(d/'endpoints-final.json');stopend=read(d/'endpoints-stopped.json');ack=read(d/'acknowledgments.json');kill=read(d/'kill.json');op1=read(d/'first-owner/open.json');op2=read(d/'fresh-owner/open.json')
  assert kill['Wait']=='signal: killed' and kill['PID']==ack['Runtime']['PID']==op1['PID'] and op2['PID']!=op1['PID']
  assert op1['Namespace']==op2['Namespace']==ini['Namespace']==fi['Namespace'];assert read(d/'owner-exit.json')['Wait']=='exit status 0'
  preserve(sr,fr,APPEND+IMMUTABLE);preserve(sr,rr,APPEND+IMMUTABLE)
  for tab in IMMUTABLE:assert ir['Tables'][tab]==rr['Tables'][tab],tab
  for a,b in zip(st['Runs'],fo['Runs']):assert a['ID']==b['ID'] and a['ProgramID']==b['ProgramID'] and b['Epoch']==a['Epoch']+int(a['Projection']['Status']=='open') and a['Sequence']==b['Sequence'] and a['Projection']==b['Projection']
  account(st,sr);account(fo,fr);account(fi,rr)
  e=effect(st,aid);f=effect(fi,aid); assert len(rr['Tables']['lifecycle_results'])==1 and len(rr['Tables']['composition_charges'])==2
  assert all(r['Projection']['Status']=='succeeded' and r['Projection']['Data']['paid']['Flag'] for r in fi['Runs'])
  assert all(len(x['Applications'])==1 for x in end.values())
  logs=[]
  for owner in ['first-owner','fresh-owner']:logs.extend(json.loads(x) for x in (d/owner/'boundaries.jsonl').read_text().splitlines())
  for g in f['AuthorityAdmissions']:
   assert g['Allowed'] and any(x['Boundary']=='authority.admission' and x['Phase']=='allowed' and x['Intent']==aid and x['Epoch']==g['Source']['Epoch'] and x['AuthorityAdmissionOrdinal']==g['Ordinal'] for x in logs)
  if e and e['ExternalAttempt']:assert e['ExternalAttempt']==f['ExternalAttempt']
  if e and e['ExternalReceipt']:assert len(e.get('AuthorityAdmissions',[]))==len(f['AuthorityAdmissions']) and stopend['b']['Contacts']==end['b']['Contacts']
  for ep,oid in [('b',aid),('a',m['ABService'])]:
   ef=effect(fi,oid);journal(d,ep,end[ep],ef['ExternalAttempt']);assert ef['ExternalReceipt']==end[ep]['Applications'][0]
  gates=len(f['AuthorityAdmissions']);contacts=end['b']['Contacts'];expected=2 if n in ['cut-06a-admitted-no-contact','cut-06b-applied-no-receipt','cut-09-mutual-wait'] else 1
  assert gates==expected and contacts==(2 if n in ['cut-06b-applied-no-receipt','cut-09-mutual-wait'] else 1)
  if n=='cut-01-before-grant':assert runs(st)['b']['Sequence']==1 and len(sr['Tables']['lifecycle_results'])==0
  if n=='cut-02-retained-grant':assert runs(st)['b']['Sequence']==2 and len(sr['Tables']['lifecycle_results'])==1
  if n=='cut-03-before-authorization':assert e['Authorization'] is None and e['ExternalAttempt'] is None
  if n in ['cut-04-complete-decision','cut-05-before-admission']:assert e['ExternalAttempt']['Allowed'] and not e.get('AuthorityAdmissions',[]) and len(sr['Tables']['composition_charges'])==1
  if n=='cut-06a-admitted-no-contact':assert e.get('AuthorityAdmissions',[])[0]['Allowed'] and stopend['b']['Contacts']==0 and e['ExternalReceipt'] is None
  if n in ['cut-06b-applied-no-receipt','cut-09-mutual-wait']:
   assert ack['Endpoint']['Boundary']=='resource.applied' and ack['Endpoint']['Phase']=='post-commit' and len(stopend['b']['Applications'])==stopend['b']['Contacts']==1
   assert e['ExternalReceipt'] is None and e['Settlement'] is None and [(r['Sequence'],r['Projection']['PC'],r['Wait']['Kind']) for r in st['Runs']]==[(4,4,'receipt'),(5,5,'message')]
  if n=='cut-07-retained-result':assert e['ExternalReceipt'] and e['Settlement'] is None
  if n=='cut-08-before-selection':assert e['Settlement'] and next(o for o in st['Observations'] if o['ID']==e['Settlement']['ObservationID'])['SelectedAt'] is None
  if n=='cut-10-one-complete':assert [(r['Projection']['Status'],r['Epoch']) for r in fo['Runs']]==[('succeeded',2),('open',3)]
  results.append({'corpus':corpus,'case':n,'gates':gates,'contacts':contacts,'applications':len(end['b']['Applications']),'request_epoch':f['ExternalAttempt']['Request']['Epoch'],'gate_epochs':[g['Source']['Epoch'] for g in f['AuthorityAdmissions']]})
 for c in [c for c in ix if c['Name'].startswith('revoke-') or c['Name'] in ['applied-unknown-revoked-before-fresh-gate','generic-unknown','first-refused-then-grant']]:
  d=pathlib.Path(c['Directory']);n=c['Name'];m=c['Manifest'];aid=m['BAService'];phase='after-grant' if n=='first-refused-then-grant' else 'final';s,raw=snapshot(d,phase);e=effect(s,aid);end=read(d/'endpoints.json');fs=facts(raw);account(s,raw)
  for ep in ['a','b']:journal(d,ep,end[ep])
  if n.startswith('revoke-') or n=='applied-unknown-revoked-before-fresh-gate':
   revoke=next(x for x in s['LifecycleDecisions'] if x['Request']['Operation']=='RevokeAuthority');assert revoke['Accepted'] and revoke['Authority']['Root']==m['BAServiceRoot'];rid=revoke['Request']['Intent']['ID'];ro=next(x['Order'] for x in fs if x['Kind']=='lifecycle' and x['OperationID']==rid);gs=[x for x in fs if x['Kind']=='authority_admission' and x['OperationID']==aid];assert e['ExternalAttempt']['Allowed']
   if n=='revoke-before-gate':assert ro<gs[0]['Order'] and not e.get('AuthorityAdmissions',[])[0]['Allowed'] and sum(x['Contacts'] for x in end.values())==0
   elif n=='applied-unknown-revoked-before-fresh-gate':
    assert gs[0]['Order']<ro<gs[1]['Order'] and [g['Allowed'] for g in e.get('AuthorityAdmissions',[])]==[True,False] and [g['Source']['Epoch'] for g in e.get('AuthorityAdmissions',[])]==[2,3]
    before,br=snapshot(d,'stopped');assert effect(before,aid)['ExternalAttempt']==e['ExternalAttempt'];assert e['ExternalReceipt'] is None and e['Settlement'] is None and e['Consequence']=='unresolved' and end['b']['Contacts']==len(end['b']['Applications'])==1 and end['a']['Contacts']==0
    assert read(d/'run.json')['Report']['UnknownExternal']==[aid]
   else:
    eo=next(x['Order'] for x in fs if x['Kind']=='external_result' and x['OperationID']==aid);assert gs[0]['Order']<ro and (ro<eo if n=='revoke-after-gate' else eo<ro)
    unrelated=effect(s,m['ABService']);assert unrelated['ExternalAttempt']['Allowed'] and unrelated['AuthorityAdmissions'][0]['Allowed'] and unrelated['ExternalReceipt'] and unrelated['ExternalAttempt']['Authority']['Use']['Root']==m['ABServiceRoot']
    assert all(x['Contacts']==len(x['Applications'])==1 for x in end.values())
  elif n=='generic-unknown':
   before,br=snapshot(d,'stopped');assert effect(before,aid)['AdapterAttempt']==e['AdapterAttempt'] and e['AdapterResult'] is None and e['Settlement'] is None and e['Consequence']=='unresolved' and len(e.get('AuthorityAdmissions',[]))==1
   report=read(d/'fresh-owner/run.json');assert report['Report']['UnknownAdapters']==[aid] and report['Report']['Dispatches']==0 and end['a']['Contacts']==len(end['a']['Applications'])==1
  else:
   before,br=snapshot(d,'before-grant');assert not before['LifecycleDecisions'] and len(s['LifecycleDecisions'])==1 and s['LifecycleDecisions'][0]['Accepted'];assert effect(before,aid)['ExternalAttempt']==e['ExternalAttempt'] and not e['ExternalAttempt']['Allowed'] and not e.get('AuthorityAdmissions',[]) and sum(x['Contacts'] for x in end.values())==0
  results.append({'corpus':corpus,'case':n,'gates':[(g['Ordinal'],g['Source']['Epoch'],g['Allowed']) for g in e.get('AuthorityAdmissions',[]) or []],'contacts':{k:v['Contacts'] for k,v in end.items()}})
print(json.dumps(results,indent=2))
print('PASS:',len(results),'scoped cases; JSON-only independent witness checks, no SQLite opens and no suite reruns')
