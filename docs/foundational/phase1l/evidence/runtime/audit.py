#!/usr/bin/env python3
"""Independent record arithmetic/link audit, not a reducer or universal proof."""
from pathlib import Path
import gzip,hashlib,json
ROOT=Path(__file__).resolve().parents[5];DEST=Path(__file__).resolve().parent
sha=lambda raw:hashlib.sha256(raw).hexdigest()
bundle=json.loads(gzip.decompress((DEST/'records.json.gz').read_bytes()))
sources=json.loads((DEST/'sources.json').read_text())
assert all(sha((ROOT/p).read_bytes())==h for p,h in sources['files'].items())
assert sha(json.dumps(sources['files'],sort_keys=True,separators=(',',':')).encode())==sources['aggregate_sha256']==bundle['source_aggregate_sha256']
assert all(c['exit_code']==0 for c in bundle['commands'])
summary={}
def descriptor(row):return json.loads(row['descriptor'])
def decoded(snapshot):return {table:[descriptor(r) for r in rows if 'descriptor' in r] for table,rows in snapshot['Tables'].items()}
def links(snapshot):
    t=snapshot['Tables'];root=next(r for r in t['runs'] if r['subject']=='root');data=json.loads(root['projection'])['Data'];result={}
    for role,register in [('e','candidate_operation'),('s','s_operation')]:
        operation=data[register]['Text']
        intent=descriptor(next(r for r in t['intents'] if r['id']==operation))
        committed=[r for r in t['history'] if r['subject']=='root' and any(i['ID']==operation for i in json.loads(r['intents']))]
        assert len(committed)==1 and json.loads(committed[0]['successor'])['Data'][register]['Text']==operation
        publication=[descriptor(r) for r in t['composition_publications'] if r['operation_id']==operation]
        delivery=[r for r in t['composition_outcome_deliveries'] if r['operation_id']==operation]
        selected=[]
        if delivery:
            assert len(delivery)==len(publication)==1
            observation=next(r for r in t['observations'] if r['id']==delivery[0]['observation_id'])
            original=descriptor(observation);assert original['Operation']==operation and original['Run']=={'Subject':'root','Generation':1}
            selected=[r for r in t['history'] if r['selected_observation']==observation['id']]
            if selected:assert len(selected)==1 and selected[0]['sequence']==observation['consumed_sequence']
        result[role]=dict(operation=operation,committed_sequence=committed[0]['sequence'],consumptions=len(selected),publication=publication[0] if publication else None,intent=intent)
    assert result['e']['operation']!=result['s']['operation'] and result['e']['committed_sequence']<result['s']['committed_sequence']
    return result
def source_authority(snapshot):
    t=snapshot['Tables'];subjects={r['id']:r for r in t['subjects']}
    histories={(r['subject'],r['generation'],r['sequence']):r for r in t['history']}
    intents={r['id']:r for r in t['intents']};committed=[]
    for key,row in histories.items():
        assert row['source_revision']==subjects[key[0]]['revision']
        for original in json.loads(row['intents']):
            committed.append(original['ID']);assert original['ID'] in intents
            actual=intents[original['ID']];assert descriptor(actual)==original
            assert (actual['subject'],actual['generation'],actual['sequence'])==key==(original['Run']['Subject'],original['Run']['Generation'],original['Sequence'])
            assert actual['ordinal']==original['Ordinal'] and actual['source_revision']==row['source_revision']
    assert len(committed)==len(set(committed))==len(intents) and set(committed)==set(intents)
    for row in t['composition_admission_decisions']:
        invocation=descriptor(row)['Invocation'];assert invocation['SourceRevision']==intents[invocation['OperationID']]['source_revision']
    for row in t['attempts']:
        request=descriptor(row);result=json.loads(row['result']);original=intents[result['IntentID']]
        source=subjects[result['Source']['Run']['Subject']]
        assert request['Revision']==result['Source']['Revision']==original['source_revision']==source['revision']
        assert result['Source']['Run']['Generation']==source['generation']
for profile,records in bundle['profiles'].items():
    for record in records.values():
        if isinstance(record,dict) and 'inspection' in record:source_authority(record['inspection'])

    results=[r for key,r in records.items() if key.endswith('/result.json.gz')]
    positives=[r for r in results if 'fixed_runs' in r and 'candidate' in r]
    controls=[r for r in results if 'control' in r]
    orders=[r for r in results if 'order' in r]
    assert len(positives)==8 and len(controls)==8 and len(orders)==2
    for r in positives:
        tr=r['trace'];assert tr['RawValid'] and not tr['Prefix'] and not tr['Component'] and not tr['Guarantee']
        original=links(r['inspection']);assert all(original[k]['consumptions']==1 for k in ['e','s'])
        request=next(e['Fact']['Request'] for e in tr['Events'] if e['Fact']['Kind']=='invoke_commit' and e['Fact']['Operation']=='e')
        for role in ['e','s']:
            p=original[role]['publication'];assert p['Status']=='succeeded' and p['ResponsePresent'] and p['Response']['id']['Number']==request['ID']
            assert p['Response']['value']['Number']==(1-request['Value'] if role=='e' else 0)
        assert r['candidate'] in ['a03135203f8b728a6aa7b732d06952cc87b70b3aced2c0833293e333760eece5','2e10975004e0351d07b237e0360a2b96b50d5884b651cfa5bfc8e88af62da266'] and r['successor']=='a6bfb72f823541b21e8780faf1eaab779254f5a563e5352cd631521c7ab6c61c'
        charges=r['inspection']['Roots'][0]['Charges'];assert charges['Invocations']==charges['CreatedRuns'] in [2,3] and charges['Effects']==0 and r['fixed_runs']==1
    wrong=next(r for r in controls if r['control']=='wrong-id');assert wrong['trace']['RawValid'] and not wrong['trace']['Prefix'] and wrong['trace']['Component'] and wrong['trace']['Guarantee']
    original=links(wrong['inspection']);assert original['e']['publication']['Response']['id']['Number']==2 and all(original[k]['consumptions']==1 for k in ['e','s'])
    scarce=next(r for r in controls if r['control']=='budget2');assert len(scarce['actual_refused'])==1 and scarce['actual_refused'][0]['Reason']=='invocation_limit'
    before=next(r for k,r in records.items() if k.endswith('/before.json.gz'));after=next(r for k,r in records.items() if k.endswith('/after.json.gz'))
    old=before['inspection'];new=after['inspection'];bl=links(old);al=links(new)
    assert bl['e']['consumptions']==1 and bl['s']['consumptions']==0 and bl['s']['publication'] is None
    assert all(bl[k]['operation']==al[k]['operation'] and al[k]['consumptions']==1 for k in ['e','s'])
    admission=next(descriptor(r) for r in old['Tables']['composition_admission_decisions'] if r['operation_id']==bl['s']['operation']);member=admission['Members'][0]
    srun=next(r for r in old['Tables']['runs'] if r['subject']==member['Subject']);assert srun['sequence']==0 and json.loads(srun['projection'])['Status']=='open'
    request=next(r for r in old['Tables']['composition_requests'] if r['invocation_id']==admission['Invocation']['ID']);assert next(r for r in old['Tables']['observations'] if r['id']==request['observation_id'])['consumed_sequence'] is None
    for row in old['Tables']['observations']:
        if row['consumed_sequence'] is not None:assert row==next(r for r in new['Tables']['observations'] if r['id']==row['id'])
    assert old['Tables']['composition_charges']==new['Tables']['composition_charges'] and old['Tables']['composition_admission_decisions']==new['Tables']['composition_admission_decisions']
    starts=[r for k,r in records.items() if '/start-' in k];exits=[r for k,r in records.items() if '/exit-' in k]
    assert len(starts)==2 and len(exits)==2 and starts[0]['binary_sha256']==starts[1]['binary_sha256'] and all(r['race_instrumented']==(profile=='race') for r in starts)
    killed=next(r for r in exits if r['signal']=='killed');fresh=next(r for r in exits if r['exit_code']==0);assert killed['pid']!=fresh['pid'] and before['ack']['PID']==killed['pid']
    summary[profile]=dict(positive_cases=8,controls=8,completion_orders=2,decoder_mutations=len([k for k in records if k.endswith('/rejection.json.gz')]),actual_budget_refused_role=scarce['actual_refused'][0]['Role'],recovery=dict(killed_pid=killed['pid'],fresh_pid=fresh['pid'],race_built_child=profile=='race',before_epoch=after['before_root_epoch'],after_epoch=after['after_root_epoch'],s_unproduced=True,old_consumed_rows_preserved=True,charges_preserved=True))
(DEST/'audit.json').write_text(json.dumps(summary,indent=2)+'\n')
index=json.loads((DEST/'index.json').read_text())
index['audit']=summary
index['fix_round']=1
index['rejected_review_revision']=dict(path='rejected-r1/index.json',source_aggregate_sha256=json.loads((DEST/'rejected-r1/sources.json').read_text())['aggregate_sha256'],verdict='REJECT R1 source revision binding; R2 inverse intent/commitment linkage')
index['previous_capture']=dict(path='revision1/index.json',source_aggregate_sha256=json.loads((DEST/'revision1/sources.json').read_text())['aggregate_sha256'],default_suite='watchdog failure after10m in examples/adaptive/TestAdaptiveWorstCaseBytes/nested=false/sixteen; README bound60m; retained, not relabeled')
index['retained_counterexample']=dict(path='revision1/charge-row-red.json.gz',classification='pre-correction raw decoder accepted unlinked extra I/R charge and matching inflated root total; no runtime/reference semantic change')
for f in sorted(DEST.rglob('*')):
    if f.is_file() and f!=DEST/'index.json':index['files'][str(f.relative_to(DEST))]=dict(sha256=sha(f.read_bytes()),bytes=f.stat().st_size)
for name in ['task-3-r1.md','task-3-r1-probes.json.gz','task-3-r1-archive-index.json']:
    f=DEST.parent/'reviews'/name;index['files']['../reviews/'+name]=dict(sha256=sha(f.read_bytes()),bytes=f.stat().st_size)
(DEST/'index.json').write_text(json.dumps(index,indent=2)+'\n')
print(json.dumps(summary,indent=2))
