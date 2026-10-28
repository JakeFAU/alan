import pathlib,json,gzip,hashlib,tarfile,collections,re,struct
R=pathlib.Path('/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan');D=R/'docs/foundational/phase1h';F=D/'evidence/task3/final';X=D/'evidence/task3/extracted';sha=lambda b:hashlib.sha256(b).hexdigest();summary={}
def gz(p):return gzip.decompress(p.read_bytes())
def archive(p,files):
 with tarfile.open(p) as t:
  mm=t.getmembers();assert len(mm)==len(files)==len({m.name for m in mm});assert set(files)=={m.name for m in mm};raw={}
  for m in mm:
   assert m.isfile() and not pathlib.Path(m.name).is_absolute() and '..' not in pathlib.Path(m.name).parts
   raw[m.name]=t.extractfile(m).read();assert sha(raw[m.name])==files[m.name]
  return raw
sm=json.loads((D/'source-manifest.json').read_text());sid=sm['SourceID'];assert sha(json.dumps(sm['Files'],sort_keys=True,separators=(',',':')).encode())==sid;assert all(sha((R/p).read_bytes())==h for p,h in sm['Files'].items())
am=json.loads((D/'archive-manifest.json').read_text());ac=json.loads((D/'evidence/task3/archive-context.json').read_text());raw=(D/'source.tar.gz').read_bytes();assert sha(raw)==am['SHA256']==ac['ArchiveSHA256'];assert len(raw)==am['Bytes']==ac['ArchiveBytes'];assert am['SourceID']==sid==ac['SourceID'];members=archive(D/'source.tar.gz',{f['Path']:f['SHA256'] for f in am['Files']});assert all(len(members[f['Path']])==f['Bytes'] for f in am['Files']);assert all(sha(members[p])==h for p,h in sm['Files'].items());assert json.loads(members['docs/foundational/phase1h/source-manifest.json'])==sm
extra=set(members)-set(sm['Files']);proxy={p for p in extra if p.startswith('dependencies/')};assert len(proxy)==4;assert extra-proxy-{'docs/foundational/phase1h/source-manifest.json'}==set(ac['GeneratedProseSnapshotMembers']);assert sha(members['dependencies/github.com/mattn/go-sqlite3/@v/v1.14.52.zip'])==sm['Dependency']['zip_sha256'];summary['archive']={'sha256':am['SHA256'],'bytes':len(raw),'members':len(members),'contributing':len(sm['Files']),'prose_snapshots':len(ac['GeneratedProseSnapshotMembers']),'proxy_members':sorted(proxy)}
def capture(name):
 p=F/name;m=json.loads((p/'command.json').read_text());s=json.loads((p/'sources.json').read_text());assert s['Files']==sm['Files'] and m['SourceID']==s['SourceID']==sid and m['SourceAfter']==sid and m['SourceUnchanged'];assert sha((p/'source.tar.gz').read_bytes())==m['SourceArchiveSHA256'];archive(p/'source.tar.gz',s['Files']);assert sha(gz(p/'stdout.gz'))==m['RawSHA256'];assert sha(gz(p/'stderr.gz'))==m['StderrSHA256'];assert m['ExitCode']==0;return m
normal=capture('final-normal');outer=capture('final-offline-reproduction')
pattern=re.compile(r'^\s*(?:[^\s:]+:\d+:\s*)?(QUALIFICATION|COMPOSITION|SUPERVISION|CONFORMANCE|DURABLE|ASYNC|EVIDENCE|INVESTIGATION|WORKTREE|ADAPTIVE|ADAPTIVE_BYTES|ADAPTIVE_ACCEPTANCE_BOUNDARY) (.*)$')
def parse(raw):
 events=[json.loads(x) for x in raw.splitlines()];b={};records=[]
 for e in events:
  if e.get('Action')!='output':continue
  key=(e.get('Package',''),e.get('Test',''));value=b.get(key,'')+e.get('Output','')
  while '\n' in value:
   line,_,value=value.partition('\n');m=pattern.match(line)
   if m:records.append({'Package':key[0],'Test':key[1],'Marker':m[1],'Evidence':json.loads(m[2]) if m[2].startswith(('{','[')) else {'Kind':'text_trace','Text':m[2]}})
   else:
    line=re.sub(r'^\s*[^\s:]+:\d+:\s*','',line).strip()
    if line.startswith('{'):
     try:v=json.loads(line)
     except ValueError:continue
     if isinstance(v,dict) and v.get('type')=='ASYNC':records.append({'Package':key[0],'Test':key[1],'Marker':'ASYNC','Evidence':v})
  b[key]=value
 assert not any(x.strip() for x in b.values());return events,records
ne,nr=parse(gz(F/'final-normal/stdout.gz'));stored=[json.loads(x) for x in gz(F/'final-normal/records.jsonl.gz').splitlines()];assert len(stored)==len(nr)
for i,(a,b) in enumerate(zip(nr,stored),1):assert all(b[k]==v for k,v in a.items()) and b['SourceID']==sid and b['Mode']=='normal' and b['RecordIndex']==i
assert not any(e.get('Action')=='fail' for e in ne);passes=[e for e in ne if e.get('Action')=='pass'];summary['normal']={'seconds':normal['ElapsedSeconds'],'test_passes':sum(bool(e.get('Test')) for e in passes),'package_passes':sum(not e.get('Test') for e in passes),'markers':dict(collections.Counter(x['Marker'] for x in nr))};assert summary['normal']['test_passes']==1146 and summary['normal']['package_passes']==24
rep=json.loads((X/'reproduction.json').read_text());assert rep['SourceID']==sid and rep['ArchiveSHA256']==am['SHA256'] and rep['Result']=='PASS';ex={}
for i,c in enumerate(rep['Commands'],1):
 stem=f'{i:02}';stdout=gz(X/(stem+'.stdout.gz'));stderr=gz(X/(stem+'.stderr.gz'));assert sha(stdout)==c['RawSHA256'] and sha(stderr)==c['StderrSHA256'] and c['ExitCode']==0
 if '-json' in c['Command']:
  events,records=parse(stdout);assert not any(e.get('Action')=='fail' for e in events);ex[stem]=(events,records)
for stem,required in rep['RequiredActualPasses'].items():assert set(required)<={e.get('Test') for e in ex[stem][0] if e.get('Action')=='pass'}
assert len(rep['RequiredActualPasses']['04'])==7 and len(rep['RequiredActualPasses']['05'])==8;summary['extracted']={'outer_seconds':outer['ElapsedSeconds'],'commands':len(rep['Commands']),'required_focused_passes':rep['RequiredActualPasses']['04'],'required_consumer_passes':len(rep['RequiredActualPasses']['05']),'inner_seconds':[c['ElapsedSeconds'] for c in rep['Commands']]}
# The ordinary test programs retained actual source-correlated artifacts in both executions.
def verify_artifacts(records,folder):
 n=0;ks=collections.Counter()
 for r in records:
  if r['Marker']!='ADAPTIVE':continue
  e=r['Evidence'];ks[e.get('kind')]+=1
  if e.get('kind')=='retained_artifact':
   p=folder/'artifacts'/e['path'].split('/artifacts/',1)[1];assert sha(p.read_bytes())==e['gzip_sha256'];json.loads(gz(p));n+=1
 return {'retained_artifact_hashes':n,'kinds':dict(ks)}
summary['normal_adaptive']=verify_artifacts(nr,F/'final-normal');summary['extracted_adaptive']=verify_artifacts(ex['04'][1],F/'final-offline-reproduction')
# Recompute old/new semantic manifests and compare all non-provenance behavior.
baseline=[json.loads(x) for x in gz(R/'docs/foundational/phase1e/evidence/final-normal.records.jsonl.gz').splitlines()];bridge='research/semantics/runtime_conformance_test.go';before=members['docs/foundational/phase1h/evidence/task3/comparison-inputs/runtime_conformance-before.go.txt']
def compare(records,events):
 result={}
 for marker,n in [('QUALIFICATION',15),('COMPOSITION',20),('SUPERVISION',32),('CONFORMANCE',6)]:
  a=[x['Evidence'] for x in baseline if x['Marker']==marker];b=[x['Evidence'] for x in records if x['Marker']==marker];assert len(a)==len(b)==n
  if marker=='QUALIFICATION':assert a==b
  elif marker in ('COMPOSITION','SUPERVISION'):
   for rows,historical in [(a,True),(b,False)]:
    m=rows[0]['Manifest'];h=hashlib.sha256();assert [d['Path'] for d in m['Files']]==sorted(['go.mod']+[p for p in members if p.startswith('research/semantics/') and p.endswith('_test.go')])
    for d in m['Files']:
     p=d['Path'];raw=before if historical and p==bridge else members[p];assert len(raw)==d['Bytes'] and sha(raw)==d['SHA256'];pb=p.encode();h.update(struct.pack('>Q',len(pb))+pb+struct.pack('>Q',len(raw))+raw)
    assert h.hexdigest()==m['SHA256'];assert all(x['SourceID']==m['SHA256'] for x in rows[1:])
   assert [{k:v for k,v in x.items() if k!='SourceID'} for x in a[1:]]==[{k:v for k,v in x.items() if k!='SourceID'} for x in b[1:]]
  else:
   model=('test','fixture','id_map','limits','mapping','model_config','model_seed','model_trace','observation_map','passed');public=('actual','expected','assertions','label','model_event_count');passed={e.get('Test') for e in events if e.get('Action')=='pass'}
   for x,y in zip(a,b):
    assert x['passed'] and y['passed'] and y['test'] in passed;assert {k:x[k] for k in model}=={k:y[k] for k in model};assert len(x['checkpoints'])==len(y['checkpoints'])
    for q,z in zip(x['checkpoints'],y['checkpoints']):
     assert set(q)==set(z);assert {k:q[k] for k in public if k in q}=={k:z[k] for k in public if k in z}
     if 'actual' in z:assert z['actual']==z['expected']
  result[marker]=n
 return result
summary['normal_comparison']=compare(nr,ne);summary['extracted_comparison']=compare(ex['06'][1],ex['06'][0]);p=pathlib.Path('/tmp/alan-phase1h-final-stage2-audit.json');p.write_text(json.dumps(summary,indent=2)+'\n');print(json.dumps(summary,indent=2))
