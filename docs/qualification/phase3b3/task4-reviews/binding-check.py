from pathlib import Path
import json,hashlib
ROOT=Path('/tmp/alan-phase3b3-runtime-development')
def j(p):return json.loads(p.read_text())
def runid(v):return(v['Subject'],v['Generation'])
def ck(c,m):
 if not c:raise AssertionError(m)
def records(t,table,col='descriptor'):return[json.loads(r[col]) for r in t.get(table,[])]
def compact(v):return json.dumps(v,separators=(',',':'),ensure_ascii=False)
def inspect(p):return j(p/'final/inspect.json')['Tables']
results=[];fail=[]
for corpus in ['ordinary','race']:
 for entry in j(ROOT/('qualified-'+corpus)/'case-index.json'):
  p=Path(entry['Directory'])
  if not(p/'final/inspect.json').exists():continue
  t=inspect(p);runs=t['runs']
  if len(runs)!=2 or any(json.loads(r['projection'])['Status']!='succeeded' for r in runs):continue
  try:
   intents={r['ID']:r for r in records(t,'intents')};effects=records(t,'external_attempts');ck(len(effects)==2,'expected two real external originals')
   obs={r['id']:(json.loads(r['descriptor']),r)for r in t['observations']}
   hist={(r['subject'],r['generation'],r['sequence']):r for r in t['history']}
   facts=records(t,'composition_facts');orders={(runid(f['Run']),f['Sequence']):f['Order']for f in facts if f['Kind']=='execution'}
   permissions=json.loads(t['composition_permissions'][0]['descriptor']);profile=json.loads(t['composition_profile'][0]['descriptor']);receipts={r['intent_id']:json.loads(r['descriptor'])for r in t['external_receipts']};auths={r['IntentID']:r for r in records(t,'attempts','result')};grants={r['Request']['Intent']['ID']:r for r in records(t,'lifecycle_results','result')}
   def selected(oid,expected):
    o,row=obs[oid];ck(runid(o['Run'])==expected and (row['subject'],row['generation'])==expected,'observation full logical run')
    seq=row['consumed_sequence'];ck(seq is not None,'observation not selected');h=hist[expected+(seq,)];ck(h['selected_observation']==oid,'selection does not join exact full-run history');return o,seq,orders[(expected,seq)]
   def admission(i):
    matches=[]
    for row in t['admissions']:
     d=json.loads(row['descriptor']);a=json.loads(row['result'])
     if d['Kind']=='message'and d['IntentID']==i['ID']and a['Accepted']:
      ck(d['Source']==i['Run']and d['Target']==i['Target']and d['Generation']==i['Generation']and d['Payload']==i['Payload'],'message admission changed origin/payload')
      ck(a['Bound']=={'Subject':i['Target'],'Generation':i['Generation']},'admission recipient generation')
      matches.append(a['ObservationID'])
    ck(len(matches)==1,'unique authentic message admission');return matches[0]
   reqs={i['Obligation']:i for i in intents.values()if i['Kind']=='message'and i['Obligation']in ['ab-request','ba-request']};ck(len(reqs)==2,'two originals missing')
   links=[]
   for a in effects:
    i=intents[a['Intent']['ID']];iid=i['ID'];sr=runid(i['Run']);ck(a['Intent']==i and a['Allowed'],'effect original bound/allowed');w=a['Authority'];ck(w['Allowed']and w['Use']['Root']['Permission']in permissions,'full original tuple absent')
    tup=w['Use']['Root']['Permission'];ck((tup['Kind'],tup['Target'],tup['Operation'],tup['Obligation'])==(i['Kind'],i['Target'],'apply',i['Obligation']),'action component mismatch')
    root=next(r for r in profile['Roots']if i['Run']in r['Members']);ck(w['ServingRootID']==root['ID'],'serving-root mismatch')
    request=reqs['ba-request'if sr[0]=='a'else'ab-request'];ck(request['Target']==sr[0]and request['Generation']==sr[1],'original targets different server');ro,rs,request_selected_order=selected(admission(request),sr);ck(rs==i['Sequence']-1,'request not immediate previous reduction')
    if w['Use']['Path']=='delegated':
     g=grants[w['Use']['GrantID']];claim=json.loads(g['Request']['Intent']['Payload']['claim']['Text']);ck(g['Accepted']and g['Request']['Intent']==intents[w['Use']['GrantID']],'grant origin changed');ck(claim['Root']==w['Use']['Root']and claim['OriginalRequestID']==request['ID']and claim['Recipient']==i['Run']and claim['ServiceOperationID']==iid,'grant root/request/recipient/service mismatch');ck(w['OriginalRequestID']==request['ID']and w['RequestObservationID']==ro['ID']and w['RequestConsumedSequence']==rs,'effect witness selection mismatch')
    else:ck(w['Use']['Path']=='own'and w['Use']['Root']['Run']==i['Run'],'own path borrowed')
    gates=[f for f in facts if f['Kind']=='authority_admission'and f['OperationID']==iid];ck(gates and all(g['AuthorityAdmission']['Allowed']for g in gates),'missing/refused exposure for completion')
    for n,g in enumerate(gates,1):ck(g['AuthorityAdmission']['Ordinal']==n and g['AuthorityAdmission']['Source']['Run']==i['Run'],'gate original ordinal/source')
    receipt=receipts[iid];request_wire=a['Request'];auth=auths[iid];ck(request_wire['Key']==compact([2,w['Namespace'],iid])and request_wire['Epoch']==auth['Source']['Epoch']and request_wire['Authorization']==auth['ID'],'wire original key/epoch/authorization');ck(request_wire['Delta']==i['Payload']['delta']['Number']and request_wire['Profile']==i['Payload']['profile']['Text'],'wire original action bytes')
    ros=[(o,row)for o,row in obs.values()if o['Provenance']=='resource-receipt'and o['Operation']==iid and runid(o['Run'])==sr];ck(len(ros)==1,'unique original result');result,resultseq,resultorder=selected(ros[0][0]['ID'],sr);ck(resultseq==i['Sequence']+1,'server result selection wrong reduction');ck(result['Payload']['mutation']['Number']==receipt['Mutation']and result['Payload']['total']['Number']==receipt['Total']and result['Payload']['accepted']['Flag'],'result does not carry actual retained receipt')
    replies=[x for x in intents.values()if x['Kind']=='message'and x['Payload'].get('service_operation',{}).get('Text')==iid];ck(len(replies)==1,'unique result-bearing reply');reply=replies[0];ck(reply['Run']==i['Run']and reply['Target']==request['Run']['Subject']and reply['Generation']==request['Run']['Generation'],'reply producer/recipient changed');ck(reply['Payload']['request_id']['Text']==request['ID']and reply['Payload']['mutation']==result['Payload']['mutation']and reply['Payload']['total']==result['Payload']['total']and reply['Payload']['accepted']['Flag'],'reply lost actual original result');replyobs,replyseq,replyorder=selected(admission(reply),runid(request['Run']));ck(orders[(sr,reply['Sequence'])]>resultorder and replyorder>resultorder,'reply/result custody order')
    endpoint='a'if i['Target']==j(p/'endpoint-a-ready.json')['URL']else'b';ck(i['Target']==j(p/('endpoint-'+endpoint+'-ready.json'))['URL'],'unknown physical endpoint')
    events=[json.loads(l)for l in(p/('endpoint-'+endpoint+'.jsonl')).read_text().splitlines()];ing=[json.loads(e['Body'])for e in events if e['Event']=='ingress'];comm=[e['Receipt']for e in events if e['Event']=='committed'];ck(ing and all(req==request_wire for req in ing),'endpoint contacts changed immutable original wire');ck(comm and all(r==receipt for r in comm),'retained result differs from endpoint actual receipt')
    charges=[json.loads(r['descriptor'])for r in t['composition_charges']if r['operation_id']==iid];ck(len(charges)==1 and charges[0]['RootID']==root['ID']and charges[0]['Effects']==1,'original serving-root debit')
    links.append({'server':sr,'request':request['ID'],'service':iid,'resultSelected':resultseq,'reply':reply['ID'],'requesterSelected':replyseq,'serviceOrder':orders[(sr,i['Sequence'])],'replyOrder':replyorder})
   a=next(x for x in links if x['server'][0]=='a');b=next(x for x in links if x['server'][0]=='b');ck(all(orders[(runid(r['Run']),r['Sequence'])]<a['serviceOrder']for r in reqs.values()),'two originals not outstanding before A services');ck(a['replyOrder']<b['serviceOrder'],'B not serial after actual BA result reply');ck(b['replyOrder']>a['serviceOrder'],'A original paid before servicing')
   results.append({'corpus':corpus,'path':str(p),'links':links})
  except Exception as e:fail.append({'corpus':corpus,'path':str(p),'error':repr(e)})
Path('/tmp/alan-phase3b3-task4-binding-check.json').write_text(json.dumps({'verified':results,'failures':fail},indent=2)+'\n');print('completed raw custody cases',len(results),'failures',fail)
