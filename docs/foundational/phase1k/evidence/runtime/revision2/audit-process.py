"""Inspect preserved process records directly, without ALAN runtime or reducer."""
import pathlib,json,gzip,sys
base=pathlib.Path(__file__).resolve().parent
read=lambda p:json.load(gzip.open(p,'rt'))
proof=[]
for label in sys.argv[1:] or ['normal-initial','race-initial']:
 for file in sorted((base/label).glob('*/before.json.gz')):
  b=read(file);a=read(file.parent/'after.json.gz');before=b['inspection'];after=a['inspection'];cut=b['cut'];pid=b['ack']['PID']
  killed=read(file.parent/('exit-'+str(pid)+'.json.gz'));assert killed['signal']=='killed' and killed['exit_code']==-1
  starts=[read(f) for f in file.parent.glob('start-*.json.gz')]
  initial=next(x for x in starts if x['pid']==pid);assert initial['filter']['Token']==b['ack']['Token']
  profile='automatic' if cut=='report-no-authorization' else 'public-resume'
  resumed=next(x for x in starts if x['pid']!=pid and x.get('profile')==profile)
  assert resumed['pid']!=pid and a['after_epoch']>a['before_epoch']
  for table in ['programs','permissions','composition_permissions','composition_charges','composition_admission_decisions']:
   assert before['Tables'][table]==after['Tables'][table],(cut,table)
  for table in ['attempts','admissions','message_acknowledgments','history','intents']:
   assert all(row in after['Tables'][table] for row in before['Tables'][table]),(cut,table)
  for table,rows in before['Tables'].items():
   if table in ['runs','observations']:continue
   assert all(row in after['Tables'][table] for row in rows),(cut,table,'full immutable prefix')
  for row in before['Tables']['observations']:
   assert any(x['id']==row['id'] and x['descriptor']==row['descriptor'] for x in after['Tables']['observations'])
  for row in before['Tables']['runs']:
   assert any(all(x[k]==row[k] for k in ['subject','generation','program_id','initial']) for x in after['Tables']['runs'])
  if cut=='report-refused-peer-terminal':
   for table in ['attempts','admissions','message_acknowledgments','observations','history','intents','composition_facts','composition_root_publications']:
    assert before['Tables'][table]==after['Tables'][table],table
   assert a['trace']['Aggregate']=='' and after['Roots'][0]['LocalStatus']=='open'
  else:assert a['trace']['Aggregate']=='succeeded' and a['trace']['Messages'][0]['Consumption'] is not None
  m=a['trace']['Messages'][0];auth=m['FirstAuthorization'];seq=auth['Sequence']
  h=next(r for r in after['Tables']['history'] if r['subject']=='peer' and r['sequence']==seq)
  proof.append({'label':label,'cut':cut,'killed_pid':pid,'new_pid':resumed['pid'],'signal':killed['signal'],'root_epoch_before':a['before_epoch'],'root_epoch_after':a['after_epoch'],'authorization_source_epoch':auth['Epoch'],'source_sequence_history_epoch':h['epoch'],'authorization_sequence':seq,'allowed':auth['Allowed'],'admitted':m['Admission']['Accepted'],'aggregate':a['trace']['Aggregate'],'all_original_records_preserved':True,'refused_exact_row_equality_checked':cut=='report-refused-peer-terminal'})
print(json.dumps({'process_cuts':proof,'result':'all inspected real-kill/process/epoch/original-row checks passed'},indent=2))
