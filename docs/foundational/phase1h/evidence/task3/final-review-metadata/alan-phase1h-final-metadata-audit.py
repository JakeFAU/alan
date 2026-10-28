import pathlib,json,gzip,hashlib,subprocess,tarfile,collections,re
R=pathlib.Path('/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan');D=R/'docs/foundational/phase1h';sha=lambda b:hashlib.sha256(b).hexdigest();changed=subprocess.check_output(['git','diff','--name-only','680be37..564b4a4'],cwd=R,text=True).splitlines();m=json.loads((D/'source-manifest.json').read_text());assert not set(changed)&set(m['Files']);assert all(sha((R/p).read_bytes())==v for p,v in m['Files'].items());result={'range':'680be37..564b4a48bf6aceba1e84070cc68ac82db9b3416d','changed_paths':len(changed),'source_inputs_unchanged':len(m['Files'])}
# All command-index rows must reproduce exact metadata fields, including failed results.
idx=[json.loads(l) for l in (D/'qualification.jsonl').read_text().splitlines()];kind=collections.Counter(x['Kind'] for x in idx);seen=set();failed=[]
for row in idx:
 if row['Kind']!='captured_command':continue
 p=D/row['Metadata'];x=json.loads(p.read_text());assert str(p.parent.relative_to(D))==row['Capture'];assert row['Capture'] not in seen;seen.add(row['Capture'])
 for k in ['SourceID','HeadAtStart','Command','ExitCode','ElapsedSeconds','SourceUnchanged','RawSHA256','StderrSHA256','SourceArchiveSHA256']:assert row[k]==x[k],(p,k)
 assert row['GoPassEvents']==(len(x['passes']) if 'passes' in x else None);assert row['Markers']==x.get('Markers')
 if row['ExitCode']:failed.append({'capture':row['Capture'],'exit':row['ExitCode']})
actual={str(p.parent.relative_to(D)) for p in (D/'evidence/task3/final').rglob('command.json')};assert seen==actual,(actual-seen,seen-actual);result['index']={'kinds':dict(kind),'captures':len(seen),'failed':failed,'other_rows':[r for r in idx if r['Kind']!='captured_command']}
# Only inspect newly committed captures; prior staged audits already cover older results.
newcaptures=[];archives={}
for p in changed:
 if not p.endswith('/command.json') or '/final/' not in p:continue
 folder=(R/p).parent;x=json.loads((R/p).read_text());s=json.loads((folder/'sources.json').read_text());assert s['SourceID']==x['SourceID']==m['SourceID'];assert s['Files']==m['Files'];assert x['SourceAfter']==m['SourceID'] and x['SourceUnchanged'];assert sha(gzip.decompress((folder/'stdout.gz').read_bytes()))==x['RawSHA256'];assert sha(gzip.decompress((folder/'stderr.gz').read_bytes()))==x['StderrSHA256'];raw=(folder/'source.tar.gz').read_bytes();h=sha(raw);assert h==x['SourceArchiveSHA256']
 if h not in archives:
  with tarfile.open(folder/'source.tar.gz') as t:
   members=t.getmembers();assert len(members)==len(m['Files'])==len({v.name for v in members});assert {v.name for v in members}==set(m['Files']);assert all(v.isfile() and not pathlib.Path(v.name).is_absolute() and '..' not in pathlib.Path(v.name).parts and sha(t.extractfile(v).read())==m['Files'][v.name] for v in members)
  archives[h]=len(members)
 newcaptures.append({'capture':str(folder.relative_to(D)),'exit':x['ExitCode'],'seconds':x['ElapsedSeconds']})
result['new_captures']=newcaptures;result['new_source_archives']=archives
p=D/'evidence/task3/final/covering/final-consumer-race';x=json.loads((p/'command.json').read_text());events=[json.loads(l) for l in gzip.decompress((p/'stdout.gz').read_bytes()).splitlines()];passed=[e['Test'] for e in events if e.get('Action')=='pass' and e.get('Test')];assert len(passed)==12;assert not any(e.get('Action')=='fail' for e in events);required={f'TestAdaptivePublicConsumerDirectNested/nested={n}/capacity={c}/paired={a}' for n in ['false','true'] for c in [1,2] for a in ['false','true']}|{'TestCLIAdaptiveRequest/direct','TestCLIAdaptiveRequest/nested'};assert required<=set(passed);assert '-race' in x['Command'] and '-timeout=60m' in x['Command'];assert b'WARNING: DATA RACE' not in gzip.decompress((p/'stdout.gz').read_bytes());result['consumer_race']={'exit':x['ExitCode'],'seconds':x['ElapsedSeconds'],'passes':passed,'package_passes':sum(e.get('Action')=='pass' and not e.get('Test') for e in events),'raw_hash':x['RawSHA256']}
# Reviewer outputs were copied verbatim, with no silent rewriting.
for stage in [1,2]:
 p=D/f'evidence/task3/final-review-stage{stage}/provenance.json';v=json.loads(p.read_text())
 for f in v['Files']:
  raw=(R/f['Retained']).read_bytes();assert len(raw)==f['Bytes'] and sha(raw)==f['SHA256'];assert raw==pathlib.Path(f['Original']).read_bytes()
v=json.loads((D/'evidence/task3/controller-rulings-provenance.json').read_text());raw=(R/v['Snapshot']).read_bytes();assert len(raw)==v['Bytes'] and sha(raw)==v['SHA256'];live=(R/v['Original']).read_bytes();assert live.startswith(raw);result['verbatim_review_files']=10;result['controller_snapshot_hash']=sha(raw)
# Local current-report links must resolve to retained files.
for name in ['02-qualification.md','03-review.md']:
 for link in re.findall(r'\]\(([^)]+)\)',(D/name).read_text()):
  if '://' not in link:assert (D/link.split('#')[0]).exists(),(name,link)
path=pathlib.Path('/tmp/alan-phase1h-final-metadata-audit.json');path.write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result,indent=2))
