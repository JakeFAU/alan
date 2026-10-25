import json, pathlib, hashlib, gzip, collections, re, struct, tarfile
root=pathlib.Path('/Users/jacob/dev/personal/alan');d=root/'docs/foundational/phase1d'
m=json.loads((d/'source-manifest.json').read_text()); h=hashlib.sha256()
for f in m['files']:
 b=(root/f['path']).read_bytes(); assert len(b)==f['bytes'];assert hashlib.sha256(b).hexdigest()==f['sha256'];p=f['path'].encode();h.update(struct.pack('>Q',len(p)));h.update(p);h.update(struct.pack('>Q',len(b)));h.update(b)
print('all source entries match:',len(m['files']), 'digest:',h.hexdigest());assert h.hexdigest()==m['source_sha256']
a=json.loads((d/'archive-manifest.json').read_text());print('archive manifest keys', list(a));print('archive sha256',hashlib.sha256((d/'source.tar.gz').read_bytes()).hexdigest())
with tarfile.open(d/'source.tar.gz') as t:
 entries=a.get('files',a.get('members',[]))
 for f in entries:
  b=t.extractfile(f['path']).read();assert hashlib.sha256(b).hexdigest()==f['sha256'];assert len(b)==f['bytes']
 print('archive entries verified',len(entries))
q=[json.loads(l) for l in (d/'qualification.jsonl').read_text().splitlines()]
for mode in ['normal','race']:
 meta=json.loads((d/f'evidence/final-{mode}.command.json').read_text());raw=gzip.decompress((d/f'evidence/final-{mode}.go-test.jsonl.gz').read_bytes());assert hashlib.sha256(raw).hexdigest()==meta['RawUncompressedSHA256']
 evs=[json.loads(l) for l in raw.splitlines()];assert not [e for e in evs if e['Action']=='fail'];print(mode,'passing events',len([e for e in evs if e['Action']=='pass' and e.get('Test')]))
 buffers={};parsed=[]
 for e in evs:
  if e['Action']!='output':continue
  key=(e.get('Package',''),e.get('Test',''));v=buffers.get(key,'')+e.get('Output','')
  while '\n' in v:
   line,_,v=v.partition('\n');match=re.match(r'^\s*(?:[^\s:]+:\d+:\s*)?(QUALIFICATION|COMPOSITION|SUPERVISION|CONFORMANCE|DURABLE) (.*)$',line)
   if match:parsed.append((key[0],key[1],match[1],json.loads(match[2])))
  buffers[key]=v
 stored=[(x['Package'],x['Test'],x['Marker'],x['Evidence']) for x in q if x.get('Mode')==mode and x.get('Marker')]
 assert stored==parsed
 durable=[x[3] for x in parsed if x[2]=='DURABLE'];kills=[x for x in durable if x.get('exit_signal')]
 assert len(kills)==16
 barriers=[]
 for x in durable:
  for k in ['barrier','second_barrier','endpoint_barrier','executor_barrier']:
   if k in x: barriers.append(x[k])
 for kill in kills:
  cfg=kill['barrier_config']; matches=[b for b in barriers if all(b[k]==v for k,v in cfg.items() if k!='PID') and b['PID']==kill['pid']]
  assert matches,kill
 print(mode,'exact extracted records',len(parsed),'durable',len(durable),'matched killed PID/config',len(kills),'fault detections',len([x for x in durable if x.get('unchanged_oracle_failures')]))
print('PASS')
