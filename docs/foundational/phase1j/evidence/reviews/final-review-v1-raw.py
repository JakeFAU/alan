from pathlib import Path
import gzip,json,hashlib
base=Path('docs/foundational/phase1j/evidence/runtime/oracle-v2-normal')
def read(p):return json.loads(gzip.decompress(p.read_bytes()))
def desc(row,key='descriptor'):return json.loads(row[key])
def values(p):return {k:v[{'int':'Number','string':'Text','bool':'Flag'}[v['Kind']]] for k,v in p.items()}
def runs(s):return {r['subject']:r for r in s['Tables']['runs']}
def status(s,subject):return desc(runs(s)[subject],'projection')['Status']
def reports(s):return [desc(r) for r in s['Tables']['intents'] if desc(r)['Kind']=='message' and desc(r)['Run']['Subject']=='peer']
def selected(s,subject,kind):
 out=[]
 for row in s['Tables']['observations']:
  o=desc(row)
  if o['Run']['Subject']==subject and o['Kind']==kind and row['consumed_sequence'] is not None:out.append((o,row['consumed_sequence']))
 return out
matrix=[];historical=[]
for f in sorted(base.glob('*/result.json.gz')):
 d=read(f)
 if 'case' in d:
  s=d['inspection'];name=d['case']; rr=reports(s); rootObs=selected(s,'root','invocation'); peerObs=selected(s,'peer','invocation'); msgObs=selected(s,'root','message')
  for p in s['Tables']['programs']:assert hashlib.sha256(p['bytes'].encode()).hexdigest()==p['id']
  assert len(rr)<=1 and len(rootObs)<=1 and len(peerObs)<=1 and len(msgObs)<=1
  if rr:
   r=rr[0];p=values(r['Payload']);assert p['s_operation']=='["peer",1,33,0]'
   po,seq=peerObs[0];op=values(po['Payload'])
   if name!='premature':
    assert seq<r['Sequence'];assert p['s_operation']==po['Operation'];assert p['status']==op['status'] and p['reason']==op['reason'] and p['response_present']==op['response_present'];assert p['response_id']==op.get('response.id',0) and p['response_value']==op.get('response.value',0)
   else:assert r['Sequence']<seq
   if msgObs:
    mo,mseq=msgObs[0];assert values(mo['Payload'])==p;assert rootObs[0][1]<mseq
    assert mo['Source']=='peer' and mo['SourceGeneration']==1 and mo['Obligation']=='phase1j_s_join'
   if name=='wrong-request':assert p['request_id']==2 and status(s,'root')=='open'
  if name in ['a','b']:
   mid=d['decisive'];assert mid is not None;assert status(mid,'root')=='open';assert desc(runs(mid)['root'],'projection')['PC']==2
   assert len(selected(mid,'root','invocation'))==1 and not reports(mid)
   assert not any(desc(r)['Invocation']['Caller']['Subject']=='peer' for r in mid['Tables']['composition_admission_decisions'])
   assert status(s,'root')=='succeeded' and status(s,'peer')=='succeeded' and s['Roots'][0]['Outcome']['Status']=='succeeded'
  matrix.append({'case':name,'root':status(s,'root'),'peer':status(s,'peer'),'aggregate':(s['Roots'][0]['Outcome'] or {}).get('Status'), 'charges':s['Roots'][0]['Charges'],'report_seq':rr[0]['Sequence'] if rr else None,'s_consumption_seq':peerObs[0][1] if peerObs else None})
 elif d.get('contract')=='K1/C0':
  s=d['inspection'];assert status(s,'root')=='succeeded' and status(s,'peer')=='failed';assert s['Roots'][0]['Outcome']['Status']=='failed' and s['Roots'][0]['Outcome']['Reason']=='member_failed'
  ad=[desc(r) for r in s['Tables']['composition_admission_decisions'] if desc(r)['Invocation']['Caller']['Subject']=='peer'];assert len(ad)==1 and not ad[0]['Accepted'] and ad[0]['Reason']=='scope_closed'
  historical.append(f'{d["candidate"]}/capacity{d["capacity"]}')
print('matrix',json.dumps(sorted(matrix,key=lambda d:d['case'])))
print('historical',sorted(historical));assert len(matrix)==13 and len(historical)==4
recoveries=[]
for f in sorted(base.glob('*/before.json.gz')):
 d=read(f);a=read(f.parent/'after.json.gz');before=d['inspection'];after=a['inspection'];ack=d['ack'];rs=runs(before);rs2=runs(after)
 assert ack['Boundary']=='execution.committed' and ack['Phase']=='post-commit'
 assert status(before,'root')=='open' and desc(rs['root'],'projection')['PC']==2
 assert len(selected(before,'root','invocation'))==1 and not selected(before,'root','message')
 rr=reports(before)
 if d['cut']=='candidate-consumed':
  assert not rr;assert not any(desc(r)['Invocation']['Caller']['Subject']=='peer' for r in before['Tables']['composition_admission_decisions']);assert ack['Subject']=='root' and ack['Sequence']==2
 else:assert len(rr)==1 and ack['Subject']=='peer' and ack['Sequence']==37 and ack['Intent']==rr[0]['ID']
 for table in ['programs','composition_bindings','composition_members','attempts','admissions','message_acknowledgments','history','intents','composition_charges','composition_admission_decisions','composition_replies','composition_publications','composition_reservations']:
  assert all(r in after['Tables'][table] for r in before['Tables'][table]),table
 for r in before['Tables']['observations']:assert any(x['id']==r['id'] and x['descriptor']==r['descriptor'] for x in after['Tables']['observations'])
 for k,r in rs.items():assert all(r[z]==rs2[k][z] for z in ['subject','generation','program_id','initial'])
 exits=[read(x) for x in f.parent.glob('exit-*.json.gz')];killed=[x for x in exits if x['pid']==ack['PID']];assert len(killed)==1 and killed[0]['signal']=='killed' and killed[0]['exit_code']==-1
 resumed=[x for x in exits if x['pid']!=ack['PID']];assert len(resumed)==2 and all(x['exit_code']==0 for x in resumed)
 assert status(after,'root')=='succeeded' and status(after,'peer')=='succeeded' and after['Roots'][0]['Outcome']['Status']=='succeeded'
 assert rs['root']['epoch']==2 and rs2['root']['epoch']==3
 recoveries.append({'case':ack['Scenario'],'ack_pid':ack['PID'],'fresh_pids':[x['pid'] for x in resumed],'before_charges':before['Roots'][0]['Charges'],'after_charges':after['Roots'][0]['Charges']})
assert len(recoveries)==4
print('recovery',json.dumps(recoveries))
