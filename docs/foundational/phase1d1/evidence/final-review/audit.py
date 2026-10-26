import collections,gzip,hashlib,io,json,struct,subprocess,tarfile
from pathlib import Path
root=Path('/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan'); doc=root/'docs/foundational/phase1d1'
sha=lambda b:hashlib.sha256(b).hexdigest()
m=json.loads((doc/'source-manifest.json').read_text()); h=hashlib.sha256()
for f in m['files']:
 b=(root/f['path']).read_bytes(); p=f['path'].encode(); assert len(b)==f['bytes'] and sha(b)==f['sha256'],f
 h.update(struct.pack('>Q',len(p))+p+struct.pack('>Q',len(b))+b)
assert h.hexdigest()==m['source_sha256']
for f in m['qualification_tools']+m['review_probe_sources']: assert sha((root/f['path']).read_bytes())==f['sha256']
archive=json.loads((doc/'archive-manifest.json').read_text()); raw=(doc/'source.tar.gz').read_bytes(); assert sha(raw)==archive['SHA256'] and len(raw)==archive['Bytes']
with tarfile.open(fileobj=io.BytesIO(raw),mode='r:gz') as t:
 entries={e.name:e for e in t.getmembers()}; assert set(entries)=={f['Path'] for f in archive['Files']}
 archive_files={}
 for f in archive['Files']:
  b=t.extractfile(entries[f['Path']]).read(); assert len(b)==f['Bytes'] and sha(b)==f['SHA256']; archive_files[f['Path']]=b
 for f in m['files']: assert sha(archive_files[f['path']])==f['sha256']
 for f in archive['Files']:
  p=root/f['Path']
  if p.exists(): assert p.read_bytes()==archive_files[f['Path']],f['Path']
index=[json.loads(x) for x in (doc/'qualification.jsonl').read_text().splitlines()]; records_by_mode={}; modes={}
for mode in ['normal','race']:
 meta=json.loads((doc/f'evidence/final-{mode}.command.json').read_text()); gz=(doc/f'evidence/final-{mode}.go-test.jsonl.gz').read_bytes(); raw=gzip.decompress(gz)
 assert sha(gz)==meta['GzipSHA256'] and sha(raw)==meta['RawSHA256'] and meta['ExitCode']==0
 events=[json.loads(x) for x in raw.splitlines()]; assert not [e for e in events if e['Action']=='fail']
 passes=sum(e['Action']=='pass' and bool(e.get('Test')) for e in events); assert passes==353==meta['PassingEvents']
 lines=gzip.decompress((doc/f'evidence/final-{mode}.records.jsonl.gz').read_bytes()).splitlines(); records_by_mode[mode]=[json.loads(x) for x in lines]
 for row in index:
  if row.get('Kind')=='evidence_record' and row['Mode']==mode:
   b=lines[row['RecordIndex']-1]; assert sha(b)==row['RecordSHA256']; r=json.loads(b)
   for k in ['SourceID','Mode','Package','Test','RecordIndex','Marker']: assert r[k]==row[k]
 traces=[r['Evidence'] for r in records_by_mode[mode] if r['Evidence'].get('type')=='ASYNC' and 'causal_events' in r['Evidence']]
 assert len(traces)==6
 for trace in traces:
  ev=trace['causal_events']; names=[e['Name'] for e in ev]; expected=['endpoint-held','peer-committed','peer-delivered','held-live']; expected+=['response-released','result-settled'] if trace['cut']=='settle' else []
  assert names==expected
  held=ev[0]['Boundary']; assert ev[3]['Boundary']['Phase']=='holding'
  for i in [1,2]: assert held['Attempt'] in ev[i]['Boundary']['InFlight']
  for i in [0,3]+([4,5] if trace['cut']=='settle' else []):
   for k in ['Intent','Attempt','Subject','Sequence','Generation','Epoch']:assert ev[i]['Boundary'][k]==held[k]
  assert trace['race_children']==(mode=='race')
 modes[mode]={'passes':passes,'records':len(lines),'causal_cli_traces':len(traces),'raw_sha256':sha(raw)}
base=[json.loads(x) for x in gzip.decompress((doc/'evidence/baseline/normal.records.jsonl.gz').read_bytes()).splitlines()]
for mode,rows in records_by_mode.items():
 for marker in ['QUALIFICATION','COMPOSITION','SUPERVISION']:
  assert [r['Evidence'] for r in rows if r['Marker']==marker]==[r['Evidence'] for r in base if r['Marker']==marker]
# Existing phase packages must retain their original tracked bytes.
old_diff=subprocess.check_output(['git','diff','--name-only','6effdc02259af5335633bfc41e9786386430fa7c','--','docs/foundational/phase1','docs/foundational/phase1b','docs/foundational/phase1c','docs/foundational/phase1d'],cwd=root).decode(); assert not old_diff
out={'source_sha256':h.hexdigest(),'source_files':len(m['files']),'archive_sha256':archive['SHA256'],'archive_members':len(archive['Files']),'modes':modes,'index_records':len(index),'historical_packages_unchanged':True,'deterministic_research_payloads_equal':True}
print(json.dumps(out,indent=2))
