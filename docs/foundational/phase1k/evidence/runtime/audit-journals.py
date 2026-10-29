"""Check retained apparatus records against archived original program bytes.
No ALAN reducer/model is imported; this checks identities and commit evidence.
"""
import pathlib,json,hashlib,collections,sys
base=pathlib.Path(__file__).resolve().parent
counts=collections.Counter()
for label in sys.argv[1:] or ['normal-initial','race-initial']:
 for file in sorted((base/label).glob('*/journal-*.jsonl')):
  counts['journals']+=1
  raw=file.read_bytes();assert raw.endswith(b'\n'),file
  for line in raw.splitlines():
   r=json.loads(line);kind=r['kind'];counts[kind]+=1
   if kind in ('commit','rejected_commit_probe'):
    pre=r['pre'];program=file.parent/('program-'+pre['ProgramID']+'.json');praw=program.read_bytes()
    assert hashlib.sha256(praw).hexdigest()==r['program_id']==r['program_sha256']==pre['ProgramID']
    p=json.loads(praw);assert p['Code'][pre['Projection']['PC']]==r['instruction']
    expected={'Run':pre['ID'],'Epoch':pre['Epoch'],'Sequence':pre['Sequence'],'Revision':pre['Revision']}
    assert r['authority']==expected
    if kind=='rejected_commit_probe':assert r['before_hash']==r['after_hash'];continue
    post=r['post'];h=r['history'];assert post['Sequence']==pre['Sequence']+1
    assert h['subject']==pre['ID']['Subject'] and h['generation']==pre['ID']['Generation']
    assert h['sequence']==post['Sequence'];assert json.loads(h['successor'])==post['Projection']
    if r['selected_id']:
     selected=r['selected'];o=json.loads(selected['descriptor'])
     assert selected['consumed_sequence'] is None and o['ID']==r['selected_id']
     assert o['Run']==pre['ID'] and h['selected_observation']==o['ID']
    else:assert h['selected_observation'] is None
   elif kind=='authorize' and not r['repeated']:
    p=r['pre'];assert r['result']['Source']=={'Run':p['ID'],'Epoch':p['Epoch'],'Sequence':p['Sequence'],'Revision':p['Revision']}
print(json.dumps({'labels':sys.argv[1:] or ['normal-initial','race-initial'],'checked':dict(counts),'result':'all journal identity/instruction/authority/selection/history checks passed'},indent=2))
