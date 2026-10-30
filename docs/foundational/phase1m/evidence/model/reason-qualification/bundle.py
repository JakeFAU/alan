"""Package C_reason qualification without changing the approved reference."""
from pathlib import Path
import datetime, gzip, hashlib, json, sys
root=Path.cwd();incoming=Path(sys.argv[1]).resolve();out=Path(__file__).resolve().parent
sha=lambda raw:hashlib.sha256(raw).hexdigest()
read=lambda p:json.loads(p.read_text())
freeze_path=root/'docs/foundational/phase1m/evidence/model/reason-reference-r1.json.gz'
freeze=json.loads(gzip.decompress(freeze_path.read_bytes()))
assert sha(freeze_path.read_bytes())=='a18845104f33dc1950795ae0d48c1d880456f3c7a2ecd24ac78442653224aeaa'
assert freeze['source_aggregate_sha256']=='b6745667b6c5c14aebf9100f9b0b50ea729ea5d8ed4c34c779676efdd1dbca9b'
assert all(sha((root/p).read_bytes())==h for p,h in freeze['source_sha256'].items())
commands={p.stem:read(p) for p in incoming.glob('*.json') if 'label' in read(p)}
for label,r in commands.items():
    if label=='correspondence':
        assert r['exit']==1 and "KeyError: 'Members'" in r['stderr']
        failed_checker=incoming/'check-correspondence-r1.py'
        key='docs/foundational/phase1m/evidence/model/reason-qualification/check_correspondence.py'
        assert sha(failed_checker.read_bytes())==r['extra_source_sha256'][key]
    else: assert r['exit']==0 and not r['failures'] and not r['skips'],label
    assert all(sha((root/p).read_bytes())==h for p,h in r['source_sha256'].items()),label
assert commands['format']['stdout']==''
for profile in ['normal','race']:
    ref,repl=commands['reference-'+profile],commands['replacements-'+profile]
    assert datetime.datetime.fromisoformat(ref['utc']) <= datetime.datetime.fromisoformat(repl['utc'])-datetime.timedelta(seconds=repl['elapsed_seconds'])
    for label,leaves in [('reference',16),('replacements',32),('local',32)]:
        r=commands[label+'-'+profile];assert len(r['selected_leaves'])==leaves
        assert all(n in r['passes'] for n in r['selected_leaves'])
        assert r['environment']['ALAN_MODEL_EVIDENCE']==str(incoming/(label+'-'+profile+'-census'))
records={str(p.relative_to(incoming)):read(p) for p in incoming.glob('*-census/*.json')}
for name,data in records.items():
    if '-normal-' in name: assert data==records[name.replace('-normal-','-race-')],name
summary={'products':{},'local_all_modes':{}}
bad=['RankFailures','StateFailures','PrefixFailures','ComponentFailures','GuaranteeFailures','Stuck','Cycles','Cutoffs','Rejections']
for variant in ['K1','local','helper']:
    folder='reference-normal' if variant=='K1' else 'replacements-normal'
    rows=records[folder+'-census/consumer-reason-census-'+variant+'.json']
    assert len(rows)==16
    assert {tuple(r['Config']['Request'][k] for k in ['ID','Value','Mode']) for r in rows}==set((i,v,m) for i in [1,2] for v in [0,1] for m in ['answer','fail','absent','pending'])
    for r in rows:
        assert r['Entitled'] and not r['Counterexamples'] and not any(r[k] for k in bad)
        assert r['GAnswerChecked']==(r['Config']['Request']['Mode']=='answer') and r['GAnswerFailures']==0
        charges=set()
        for maximum in r['Maxima']:
            o=maximum['Observation'];mode=r['Config']['Request']['Mode'];charges.add(o['I'])
            assert not maximum['Guarantee'] and o['I']==o['R'] and o['E']==0 and o['FixedRuns']==1
            s=o['S'];assert o['SConsumed'] and s['Status']=='succeeded' and s['Reason']=='member_succeeded' and s['Present'] and s['ID']==r['Config']['Request']['ID'] and s['Value']==0
            e=o['Candidate']
            if mode=='pending':
                assert maximum['Kind']=='pending' and o['ParentStage']==3 and o['ParentStatus']=='open' and o['RootStatus']=='' and o['Classification']=='' and not o['CandidateConsumed']
                e_state=maximum['State']['Component']['Ops'][0]
                assert e_state['PC']==8 and not e_state['Published'] and e_state['Freeze']==0
                assert any(ev['Fact']['Kind']=='pending_receive' and ev['Fact']['Operation']=='e' for ev in maximum['Trace'])
            else:
                tag,status,reason={'answer':('answer','succeeded','member_succeeded'),'fail':('candidate_failed','failed','member_failed'),'absent':('candidate_missing_reply','failed','missing_reply')}[mode]
                assert maximum['Kind']=='terminal' and o['CandidateConsumed'] and o['Classification']==tag and o['ParentStatus']==status and o['RootStatus']==status
                assert e['Status']==status and e['Reason']==reason and e['Present']==(mode=='answer')
                if mode=='answer':assert e['ID']==r['Config']['Request']['ID'] and e['Value']==1-r['Config']['Request']['Value']
                assert maximum['Trace'][-1]['Fact']['Reason']=='member_'+status
        assert charges==({'K1':{2,3},'local':{2},'helper':{3}}[variant])
    summary['products'][variant]=dict(configurations=len(rows),states=sum(r['States'] for r in rows),model_states=sum(r['ModelStates'] for r in rows),edges=sum(r['Edges'] for r in rows),revisits=sum(r['Revisits'] for r in rows),terminal=sum(r['Terminal'] for r in rows),pending=sum(r['Pending'] for r in rows),maxima=sum(len(r['Maxima']) for r in rows),rank_min=min(r['RankMinimum'] for r in rows),rank_max=max(r['RankMaximum'] for r in rows),rank_cap=rows[0]['RankCap'],failure_counts={k:0 for k in bad},g_answer_configurations=sum(r['GAnswerChecked'] for r in rows),g_outcome_configurations=len(rows),by_mode={m:{k:sum(r[k] for r in rows if r['Config']['Request']['Mode']==m) for k in ['States','Edges','Terminal','Pending']} for m in ['answer','fail','absent','pending']})
local=records['local-normal-census/consumer-reason-local-all-modes.json']
assert len(local)==32
for variant in ['local','helper']:
    rows=[r for r in local if r['Variant']==variant];assert len(rows)==16
    for r in rows:
        assert r['Entitled'] and not r['Failures'] and not r['Relation']['Counterexamples'] and r['Relation']['Failures']==0 and r['Relation']['Initials']==1
        assert not r['Progress']['Counterexamples'] and not any(r['Progress'][k] for k in ['Stuck','Cycles','RankFailures','Cutoffs'])
        assert r['Maxima'] and all(not m['Error'] for m in r['Maxima'])
        for m in r['Maxima']:
            s=m['State'];e=s['Ops'][0];h=s['Ops'][1];mode=r['Config']['Request']['Mode']
            assert not s['Closed'] and s['PeerTicket']==2 and s['I']==s['R']==(2 if variant=='local' else 3) and s['E']==0
            assert e['Admission']==1 and e['RequestStage']==2 and not e['FailureReady']
            if variant=='helper': assert h['Consumed'] and h['Present'] and h['Status']=='succeeded' and h['Reason']=='member_succeeded'
            if mode=='pending':assert e['PC']==8 and not e['Published'] and not e['Consumed'] and e['Freeze']==0 and s['RootStatus']==''
            else:
                assert e['Consumed'] and e['Published']
                assert e['Reason']=={'answer':'member_succeeded','fail':'member_failed','absent':'missing_reply'}[mode]
                assert e['Present']==(mode=='answer')
    summary['local_all_modes'][variant]=dict(configurations=16,relation={k:sum(r['Relation'][k] for r in rows) for k in ['Pairs','ComponentChecks','EnvironmentChecks','Stutters','Failures']},progress={k:sum(r['Progress'][k] for r in rows) for k in ['States','Transitions','Terminal','Pending','Stuck','Cycles','RankFailures','Cutoffs']},maxima=sum(len(r['Maxima']) for r in rows),rank_min=min(r['RankMinimum'] for r in rows),rank_max=max(r['RankMaximum'] for r in rows),rank_cap=rows[0]['RankCap'])
artifacts=read(incoming/'artifacts-inputs.json');correspondence=read(incoming/'correspondence-final-output.json')
assert correspondence['artifact_input_sha256']==sha((incoming/'artifacts-inputs.json').read_bytes())
assert all(sha((root/p).read_bytes())==h for p,h in correspondence['source_sha256'].items())
preflight_path=root/'docs/foundational/phase1m/evidence/reason-preflight/evidence-index.json';preflight=read(preflight_path)
for p,h in preflight['source_hashes'].items():
    if p.startswith(('examples/consumingpolicy/','examples/substitution/','author/','internal/executor/','runtime/','schema/')) and p.endswith('.go'):
        assert sha((root/p).read_bytes())==h,p
source_paths=[str(p.relative_to(root)) for p in sorted(out.iterdir()) if p.suffix in ['.py','.txt','.md']]
source_paths+=['research/substitution/consumer_reason_qualification_test.go']
sources={p:dict(sha256=sha((root/p).read_bytes()),text=(root/p).read_text()) for p in source_paths}
source_hashes={p:v['sha256'] for p,v in sources.items()}
record=dict(schema='alan/phase1m/reason-qualification/v1',created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),packaging_argv=sys.argv,cwd=str(root),reference=dict(path=str(freeze_path.relative_to(root)),archive_sha256=sha(freeze_path.read_bytes()),source_aggregate=freeze['source_aggregate_sha256'],frozen_inputs_unchanged=len(freeze['sources'])),summary=summary,commands=commands,censuses=records,artifact_inputs=artifacts,correspondence=correspondence,sources=sources,source_aggregate=sha(json.dumps(source_hashes,sort_keys=True,separators=(',',':')).encode()),preflight_source_link=dict(path=str(preflight_path.relative_to(root)),index_sha256=sha(preflight_path.read_bytes()),scope='Read after reference APPROVE; relevant consumer/original-template/compiler/runtime source hashes agree. No Task6 raw-runtime authentication claimed.'),capture_history=dict(failed_correspondence_source=(incoming/'check-correspondence-r1.py').read_text(),failed_correspondence_sha256=sha((incoming/'check-correspondence-r1.py').read_bytes()),disposition='Initial checker used template.Members instead of template.Descriptor.Members and exited1. Corrected lookup only; final command exits0. No reference, guarantee or artifact changed. Failed stdout/stderr/argv/hash preserved in commands.correspondence.'),extraction_source=dict(text=(incoming/'extract.go').read_text(),sha256=sha((incoming/'extract.go').read_bytes())))
raw=json.dumps(record,sort_keys=True,separators=(',',':')).encode();archive=gzip.compress(raw,mtime=0)
(out/'records.json.gz').write_bytes(archive)
index=dict(schema=record['schema'],archive='records.json.gz',archive_sha256=sha(archive),archive_bytes=len(archive),uncompressed_bytes=len(raw),reference=record['reference'],source_aggregate=record['source_aggregate'],summary=summary,command_summary={k:dict(argv=r['argv'],environment=r['environment'],exit=r['exit'],leaves=len(r['selected_leaves']),utc=r['utc']) for k,r in sorted(commands.items())},artifact_input_sha256=correspondence['artifact_input_sha256'],correspondence_output_sha256=sha((incoming/'correspondence-final-output.json').read_bytes()),instruction_coverage=correspondence['instruction_coverage_total'],all_maxima_preserved=True,normal_race_census_identity=True,scope='Bounded C_reason reference/replacement/local qualification and postfreeze finite artifact mapping; Task6 raw runtime qualification remains gated.')
(out/'index.json').write_text(json.dumps(index,indent=2)+'\n')
print(json.dumps(dict(archive_sha256=index['archive_sha256'],archive_bytes=len(archive),commands=len(commands),frozen_inputs_unchanged=len(freeze['sources']),summary=summary),indent=2))
