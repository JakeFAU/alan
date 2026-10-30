"""Package current source-bound qualification records, preserving every maximum."""
import datetime, gzip, hashlib, json, pathlib, sys
root = pathlib.Path.cwd()
incoming = pathlib.Path(sys.argv[1]).resolve()
out = pathlib.Path(__file__).resolve().parent
sha = lambda raw: hashlib.sha256(raw).hexdigest()
read = lambda p: json.loads(p.read_text())
freeze_path = root/'docs/foundational/phase1m/evidence/model/reference-r1.json.gz'
freeze = json.loads(gzip.decompress(freeze_path.read_bytes()))
assert sha(freeze_path.read_bytes()) == '3663adea727e91f714e3257dae60331d43c194b074b7bb7bfe1c7ccc5dde329f'
assert freeze['source_aggregate'] == 'cdfe2a191641129dc591b92aabd680d238a80a1e027cfb77a82edc84fd465cd1'
assert all(sha((root/p).read_bytes()) == r['sha256'] for p,r in freeze['sources'].items())
commands = {p.stem:read(p) for p in incoming.glob('*.json') if 'label' in read(p)}
assert all(r['exit'] == 0 and not r['failures'] and not r['skips'] for r in commands.values())
for profile in ['normal','race']:
    ref, repl = commands['reference-'+profile], commands['replacements-'+profile]
    assert datetime.datetime.fromisoformat(ref['utc']) <= datetime.datetime.fromisoformat(repl['utc'])-datetime.timedelta(seconds=repl['elapsed_seconds'])
    for label, leaves in [('reference',4),('replacements',8),('local-historical',23),('construction',50)]:
        assert len(commands[label+'-'+profile]['selected_leaves']) == leaves
assert commands['format']['stdout'] == ''
records = {str(p.relative_to(incoming)):read(p) for p in incoming.glob('*-census/*.json')}
for name, data in records.items():
    if '-normal-' in name:
        assert data == records[name.replace('-normal-','-race-')], name
summary = {}
for label, prefix in [('C_se','consumer-policy'),('C_es-regression','invocation-join')]:
    summary[label] = {}
    for variant in ['K1','local','helper']:
        stem = 'reference-normal' if label == 'C_se' and variant == 'K1' else 'replacements-normal' if label == 'C_se' else 'local-historical-normal'
        rows = records[stem+'-census/'+prefix+'-census-'+variant+'.json']
        assert len(rows) == 4
        bad = ['RankFailures','StateFailures','PrefixFailures','ComponentFailures','GuaranteeFailures','Stuck','Pending','Cycles','Cutoffs','Rejections']
        assert not any(row[k] or row['Counterexamples'] for row in rows for k in bad)
        summary[label][variant] = dict(configurations=len(rows), states=sum(r['States'] for r in rows), edges=sum(r['Edges'] for r in rows), maxima=sum(len(r['Maxima']) for r in rows), rank_min=min(r['RankMinimum'] for r in rows), rank_max=max(r['RankMaximum'] for r in rows), rank_cap=rows[0]['RankCap'], failure_counts={k:0 for k in bad})
local = records['local-historical-normal-census/invocation-join-local-subset.json']
summary['local-answer'] = {}
for variant in ['local','helper']:
    rows = [r for r in local if r['Variant']==variant]
    assert len(rows) == 4
    assert all(r['Relation']['Failures']==0 and not r['Relation']['Counterexamples'] and not r['OpenProgress']['Counterexamples'] for r in rows)
    summary['local-answer'][variant] = dict(configurations=4,relation={k:sum(r['Relation'][k] for r in rows) for k in ['Pairs','ComponentChecks','EnvironmentChecks','Stutters','Failures']},open_progress={k:sum(r['OpenProgress'][k] for r in rows) for k in ['States','Transitions','Terminal','Pending','Stuck','Cycles','RankFailures','Cutoffs']})
artifacts = read(incoming/'artifacts-inputs.json')
if (incoming/'artifacts.json').exists():
    assert (incoming/'artifacts.json').read_bytes() == (incoming/'artifacts-inputs.json').read_bytes()
correspondence = read(incoming/'correspondence-final-output.json')
assert correspondence['artifact_input_sha256'] == sha((incoming/'artifacts-inputs.json').read_bytes())
source_paths = [str(p.relative_to(root)) for p in out.glob('*') if p.suffix in ['.py','.txt']]
source_paths += ['research/substitution/consumer_policy_qualification_test.go']
sources = {p:dict(sha256=sha((root/p).read_bytes()),text=(root/p).read_text()) for p in sorted(source_paths)}
source_hashes = {p:r['sha256'] for p,r in sources.items()}
record = dict(schema='alan/phase1m/se-qualification/v1',created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),packaging_argv=sys.argv,cwd=str(root),reference=dict(path=str(freeze_path.relative_to(root)),archive_sha256=sha(freeze_path.read_bytes()),source_aggregate=freeze['source_aggregate'],frozen_inputs_unchanged=len(freeze['sources'])),summary=summary,commands=commands,censuses=records,artifact_inputs=artifacts,correspondence=correspondence,sources=sources,source_aggregate=sha(json.dumps(source_hashes,sort_keys=True,separators=(',',':')).encode()),capture_history=dict(initial_correspondence=read(incoming/'correspondence-output.json') if (incoming/'correspondence-output.json').exists() else None,initial_correspondence_checker=(incoming/'check_correspondence-initial.py').read_text() if (incoming/'check_correspondence-initial.py').exists() else None,artifact_capture_collision=dict(label='artifacts',error='FileExistsError: capture target artifacts.json already existed after the exporter wrote that same output path',scope='Exporter completed; capture wrapper then failed before saving its command record. Repeated with separate artifact-extraction.json and artifacts-inputs.json names. Initial and repeated artifact bytes are identical. No model/source failure and no runtime execution.',initial_artifact_sha256=sha((incoming/'artifacts.json').read_bytes()) if (incoming/'artifacts.json').exists() else None,failed_child_argv=['/opt/homebrew/bin/go','run','/private/tmp/phase1m-se-extract.go',str(incoming/'artifacts.json')],limitation='The failed wrapper did not save stdout/stderr/exit metadata; this failure is preserved from the observed tool result, not a fabricated command record.') if (incoming/'artifacts.json').exists() else None))
raw=json.dumps(record,sort_keys=True,separators=(',',':')).encode()
archive=gzip.compress(raw,mtime=0)
(out/'records.json.gz').write_bytes(archive)
index=dict(schema=record['schema'],archive='records.json.gz',archive_sha256=sha(archive),archive_bytes=len(archive),uncompressed_bytes=len(raw),reference=record['reference'],source_aggregate=record['source_aggregate'],summary=summary,command_summary={k:dict(argv=r['argv'],environment=r['environment'],exit=r['exit'],leaves=len(r['selected_leaves']),utc=r['utc']) for k,r in sorted(commands.items())},artifact_input_sha256=correspondence['artifact_input_sha256'],correspondence_output_sha256=sha((incoming/'correspondence-final-output.json').read_bytes()),instruction_coverage=correspondence['instruction_coverage_total'],all_maxima_preserved=True,normal_race_census_identity=True,scope='Bounded reference/replacement/local model qualification and finite artifact mapping only; no Task3 runtime qualification, C_reason, production changes, or paper claims.')
(out/'index.json').write_text(json.dumps(index,indent=2)+'\n')
print(json.dumps(dict(archive_sha256=index['archive_sha256'],archive_bytes=len(archive),command_records=len(commands),frozen_inputs_unchanged=len(freeze['sources']),summary=summary),indent=2))
