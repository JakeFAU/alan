import json,gzip,hashlib,tarfile,pathlib,collections
root=pathlib.Path('/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan');doc=root/'docs/foundational/phase1h';sha=lambda x:hashlib.sha256(x).hexdigest()
summary={'captures':[],'failures':[],'artifacts':0}
for cp in sorted((doc/'evidence/task3').rglob('command.json')):
 m=json.loads(cp.read_text());d=cp.parent
 if not (d/'sources.json').exists():continue
 f=json.loads((d/'sources.json').read_text());sid=sha(json.dumps(f['Files'],sort_keys=True,separators=(',',':')).encode());assert sid==f['SourceID']==m['SourceID'],str(cp)
 for label,key in [('stdout','RawSHA256'),('stderr','StderrSHA256')]:
  raw=gzip.decompress((d/(label+'.gz')).read_bytes());assert sha(raw)==m[key],str(cp)
 archive=d/'source.tar.gz';assert sha(archive.read_bytes())==m['SourceArchiveSHA256'],str(cp)
 with tarfile.open(archive) as t:
  members=t.getmembers();assert len(members)==len(f['Files']);assert set(x.name for x in members)==set(f['Files'])
  for x in members:
   assert x.isfile() and not pathlib.Path(x.name).is_absolute() and '..' not in pathlib.Path(x.name).parts
   assert sha(t.extractfile(x).read())==f['Files'][x.name],(str(cp),x.name)
 row={'capture':str(d.relative_to(doc)),'source':sid,'exit':m['ExitCode'],'seconds':m['ElapsedSeconds']}
 if (d/'records.jsonl.gz').exists():
  events=[json.loads(x) for x in gzip.decompress((d/'stdout.gz').read_bytes()).splitlines()]
  row['pass_count']=sum(e.get('Action')=='pass' and bool(e.get('Test')) for e in events);row['failed_tests']=[e.get('Test') for e in events if e.get('Action')=='fail'];row['race_diagnostic']=b'WARNING: DATA RACE' in gzip.decompress((d/'stdout.gz').read_bytes())
  assert row['pass_count']==len(m['passes'])
  records=[json.loads(x) for x in gzip.decompress((d/'records.jsonl.gz').read_bytes()).splitlines()]
  assert all(x['SourceID']==sid for x in records)
  row['markers']=dict(collections.Counter(x['Marker'] for x in records))
  for r in records:
   e=r['Evidence']
   if isinstance(e,dict) and e.get('kind')=='retained_artifact':
    tail=e['path'].split('/artifacts/',1)[1];p=d/'artifacts'/tail
    raw=p.read_bytes();assert sha(raw)==e['gzip_sha256'];json.loads(gzip.decompress(raw))
    summary['artifacts']+=1
 summary['captures'].append(row)
manifest=json.loads((doc/'source-manifest.json').read_text());assert all(sha((root/p).read_bytes())==h for p,h in manifest['Files'].items());summary['frozen_source']=manifest['SourceID'];summary['frozen_inputs']=len(manifest['Files'])
path=pathlib.Path('/tmp/alan-phase1h-final-evidence-audit.json');path.write_text(json.dumps(summary,indent=2)+'\n');print(json.dumps({'captures':len(summary['captures']),'retained_artifacts_compressed_hash_verified':summary['artifacts'],'source':summary['frozen_source'],'inputs':summary['frozen_inputs']}));print('nonpassing',[(x['capture'],x['exit'],x.get('failed_tests')) for x in summary['captures'] if x['exit']]);print('output',path)
