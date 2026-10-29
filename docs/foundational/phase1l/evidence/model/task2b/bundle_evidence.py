from pathlib import Path
import gzip, hashlib, json
root=Path.cwd(); temp=Path('/tmp/alan-phase1l-task2b'); dest=root/'docs/foundational/phase1l/evidence/model/task2b'
sha=lambda raw:hashlib.sha256(raw).hexdigest()
freeze=json.loads((root/'.superpowers/sdd/00-plan-phase1l/task-2a-freeze.json').read_text())
for path,digest in (freeze['source_sha256'] | freeze['inherited_phase1i_sources_unchanged']).items():
    assert sha((root/path).read_bytes())==digest,path
assert sha((dest.parent/'reference-r1.json.gz').read_bytes())=='a1c7a10f8ebc7a59dba5abc80af2580bbc497ef45e301daf92204981fd17972d'
assert sha((dest.parent/'reference-r2.json.gz').read_bytes())=='adebc8c4dccce81f8656ef4746680825b354f4bf4115354989b5742de2e45937'
census={}; output_hashes={}
for path in sorted((temp/'normal').glob('*.json')):
    raw=path.read_bytes(); assert raw==(temp/'race'/path.name).read_bytes()
    census[path.name]=json.loads(raw);output_hashes[path.name]=sha(raw)
artifacts=(temp/'artifacts-inputs.json').read_bytes()
bundle={'scope':'Task2b model census and post-freeze artifact correspondence; no new runtime qualification', 'reference_aggregate_sha256':freeze['aggregate_sha256'],'census':census,'artifact_extraction_json':artifacts.decode(),'correspondence':json.loads((temp/'correspondence.json').read_text())}
raw=(json.dumps(bundle,separators=(',',':'))+'\n').encode(); packed=gzip.compress(raw,mtime=0)
(dest/'qualification.json.gz').write_bytes(packed)
source_paths=set(freeze['source_sha256'])|set(freeze['inherited_phase1i_sources_unchanged'])|{'go.mod','go.sum','docs/foundational/phase1l/assignment.md'}
for folder in ['author','compose','runtime','schema','internal/executor','examples/substitution','examples/invocationjoin','research/substitution']:
    source_paths.update(str(p.relative_to(root)) for p in (root/folder).rglob('*.go'))
source_manifest={p:sha((root/p).read_bytes()) for p in sorted(source_paths)}
(dest/'sources.json').write_text(json.dumps(source_manifest,indent=2)+'\n')
commands={}
for p in sorted(dest.glob('*.json')):
    record=json.loads(p.read_text())
    if isinstance(record,dict) and 'argv' in record:
        assert record['exit_code']==0,(p,record['exit_code'])
        commands[p.name]={'sha256':sha(p.read_bytes()),'argv':record['argv'],'utc':record['utc'],'exit_code':record['exit_code'],'selected_leaf_count':len(record['selected_leaves'])}
summary={}
for key,rows in census.items():
    if 'census' not in key:continue
    variant=rows[0]['Variant']
    fields=['States','ModelStates','Edges','Revisits','Rejections','Terminal','Pending','Stuck','Cycles','Cutoffs','RankFailures','StateFailures','PrefixFailures','ComponentFailures','GuaranteeFailures']
    summary[variant]={'configurations':len(rows),**{k:sum(r[k] for r in rows) for k in fields},'maxima':sum(len(r['Maxima']) for r in rows),'rank_min':min(r['RankMinimum'] for r in rows),'rank_max':max(r['RankMaximum'] for r in rows),'rank_cap':10621,'maxima_charges':sorted({(m['Observation']['I'],m['Observation']['R'],m['Observation']['E'],m['Observation']['FixedRuns']) for r in rows for m in r['Maxima']}),'counterexamples':sum(len(r['Counterexamples']) for r in rows)}
index={'status':'Task2b qualification review candidate; no Task3 runtime qualification','reference_aggregate_sha256':freeze['aggregate_sha256'],'reference_files_unchanged':True,'inherited_source_manifest_files_unchanged':len(freeze['inherited_phase1i_sources_unchanged']),'prior_archives_unchanged':True,'bundle':{'path':'qualification.json.gz','sha256':sha(packed),'uncompressed_bytes':len(raw),'compressed_bytes':len(packed)},'source_manifest':{'path':'sources.json','sha256':sha((dest/'sources.json').read_bytes()),'files':len(source_manifest)},'normal_race_outputs_byte_identical':output_hashes,'artifact_output_sha256':sha(artifacts),'first_artifact_output_sha256':sha((temp/'artifacts.json').read_bytes()),'aggregate_census':summary,'commands':commands,'reproducer_sources':{p.name:sha(p.read_bytes()) for p in sorted(dest.iterdir()) if p.name.endswith(('.py','.go.txt'))}}
(dest/'index.json').write_text(json.dumps(index,indent=2)+'\n')
print(json.dumps({'bundle':index['bundle'],'source_files':len(source_manifest),'commands':len(commands),'census':summary},indent=2))
