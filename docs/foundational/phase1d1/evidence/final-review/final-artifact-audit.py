import gzip,hashlib,io,json,struct,tarfile
from pathlib import Path
root=Path('/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan'); doc=root/'docs/foundational/phase1d1'
sha=lambda b:hashlib.sha256(b).hexdigest()
m=json.loads((doc/'source-manifest.json').read_text()); source=m['source_sha256']; assert source=='9fef40370632bdf7416d5bdb2e8472f31ae2adc80ae5145cd7c9c90ccfe63e63'; assert len(m['files'])==58
h=hashlib.sha256()
for f in m['files']:
 p=f['path'].encode(); b=(root/f['path']).read_bytes();assert len(b)==f['bytes'] and sha(b)==f['sha256'];h.update(struct.pack('>Q',len(p))+p+struct.pack('>Q',len(b))+b)
assert h.hexdigest()==source
for f in m['qualification_tools']+m['review_probe_sources']:
 b=(root/f['path']).read_bytes();assert sha(b)==f['sha256'] and len(b)==f['bytes']
a=json.loads((doc/'archive-manifest.json').read_text());raw=(doc/a['Archive']).read_bytes();assert a['SourceID']==source and sha(raw)==a['SHA256']=='cb0763f6a9e23a1c9a35f647b4658d61adba16a8acaddc40767e70a390b5d535';assert len(raw)==a['Bytes'] and len(a['Files'])==95
with tarfile.open(fileobj=io.BytesIO(raw)) as t:
 assert set(t.getnames())=={f['Path'] for f in a['Files']}; archived={}
 for f in a['Files']:
  b=t.extractfile(f['Path']).read();assert len(b)==f['Bytes'] and sha(b)==f['SHA256'];archived[f['Path']]=b
  p=root/f['Path']
  if p.exists():assert p.read_bytes()==b,f['Path']
 for f in m['files']:assert sha(archived[f['path']])==f['sha256']
 archived_m=json.loads(archived['docs/foundational/phase1d1/source-manifest.json']);assert archived_m==m
index=[json.loads(x) for x in (doc/'qualification.jsonl').read_text().splitlines()];assert len(index)==431; assert all(r['SourceID']==source for r in index)
modes={}
for mode in ['normal','race']:
 meta=json.loads((doc/f'evidence/final-{mode}.command.json').read_text());assert meta['SourceID']==source and meta['ExitCode']==0
 gz=(doc/f'evidence/final-{mode}.go-test.jsonl.gz').read_bytes();raw=gzip.decompress(gz);assert sha(gz)==meta['GzipSHA256'] and sha(raw)==meta['RawSHA256']
 events=[json.loads(x) for x in raw.splitlines()];assert not any(e['Action']=='fail' for e in events)
 passes=sum(e['Action']=='pass' and bool(e.get('Test')) for e in events);assert passes==357==meta['PassingEvents']
 lines=gzip.decompress((doc/f'evidence/final-{mode}.records.jsonl.gz').read_bytes()).splitlines();assert len(lines)==214
 for row in index:
  if row.get('Kind')=='evidence_record' and row['Mode']==mode:
   raw=lines[row['RecordIndex']-1];assert sha(raw)==row['RecordSHA256'];r=json.loads(raw)
   for k in ['SourceID','Mode','Package','Test','RecordIndex','Marker']:assert r[k]==row[k]
 modes[mode]={'passing_events':passes,'records':len(lines),'raw_sha256':meta['RawSHA256'],'elapsed_seconds':meta['ElapsedSeconds']}
r=json.loads((doc/'evidence/archive-reproduction.json').read_text());assert r['SourceID']==source and r['ArchiveSHA256']==a['SHA256'] and r['VerifiedMembers']==95 and r['InitialModuleCacheEmpty'] and r['Result']=='PASS';assert all(c['ExitCode']==0 for c in r['Commands'])
old=doc/'evidence/pre-final-review';oldm=json.loads((old/'source-manifest.json').read_text()); assert oldm['source_sha256']=='ee144b08e6abac10c704c990e83a80314326a1eac3956bc5ecf9489f8f0afd5a'
olda=json.loads((old/'archive-manifest.json').read_text());assert sha((old/'source.tar.gz').read_bytes())==olda['SHA256']=='4ea1db7953fdc754941ea45b35fa2c5d806a36309cadd1a149af7185af5c38fd'
for mode in ['normal','race']:
 meta=json.loads((old/f'final-{mode}.command.json').read_text());assert meta['SourceID']==oldm['source_sha256'] and meta['PassingEvents']==353
 assert sha(gzip.decompress((old/f'final-{mode}.go-test.jsonl.gz').read_bytes()))==meta['RawSHA256']
out={'source_sha256':source,'source_files':58,'archive_sha256':a['SHA256'],'archive_members':95,'qualification_index_rows':len(index),'modes':modes,'offline_reproduction':{'result':r['Result'],'initial_module_cache_empty':True,'commands':[{k:c[k] for k in ['Command','ExitCode','ElapsedSeconds']} for c in r['Commands']]},'pre_review_source_and_353_event_logs_preserved':True}
print(json.dumps(out,indent=2))
