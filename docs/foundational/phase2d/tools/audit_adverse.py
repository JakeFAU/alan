"""Audit Phase 2D adversarial captures using their original SQLite and independent journal bytes."""
import argparse
from collections import Counter
import json
from pathlib import Path
from urllib.parse import urlsplit

from audit_raw import Tables, check, enc, group_digest, handoff, identity, observed, project, read_json, reports, role_at, select, sha, single, unpack

NAMES = ('competing-original','competing-adopter-query','ledger-replacement','read-only','unsupported','wrong-grant',
         'wrong-role','external-multiplicity','lying-second-snapshot')


def audit(folder):
    name=folder.name
    events=read_json(folder/'transcript.json')
    images={}
    worlds={}
    for path in sorted(folder.glob('*-local.sqlite.gz')):
        label=path.name.removesuffix('-local.sqlite.gz')
        images[label]=Tables(path)
        worlds[label]=Tables(folder/(label+'-world.sqlite.gz'))
    for label in images:
        if label.endswith('-replayed'):
            prev=label.removesuffix('-replayed')
            check(images[label].same(images[prev]) and worlds[label].same(worlds[prev]) and
                  (folder/(label+'-journal.jsonl')).read_bytes()==(folder/(prev+'-journal.jsonl')).read_bytes(),'unchanged replay')
    d,w=images['final'],worlds['final']
    ns=d.row('metadata')['namespace']
    def oid(role):
        inv=enc(['parent',1,1 if role=='l' else 2,0])
        iid=identity('alan/invocation/v1',[ns,inv])
        member=identity('alan/invocation-member/v1',[iid,'source'])
        check(d.row('composition_members',subject=member)['invocation_id']==iid,'bound member')
        return enc([member,1,2,0])
    l,r=oid('l'),oid('r')
    q=lambda role: [ob for _,ob in observed(d,'resolver_'+role) if ob['Provenance']=='runtime-adapter']
    ls,rs=q('l'),q('r')
    results={row['intent_id']:json.loads(row['descriptor'])for row in d['adapter_results']}
    for role,obs in [('l',ls),('r',rs)]:
        for ob in obs:
            result=results[ob['Operation']]
            check(result['Values']==ob['Payload'] and d.row('settlements',intent_id=ob['Operation'])['descriptor']==
                  d.row('adapter_results',intent_id=ob['Operation'])['descriptor'],'actual selected settled query')
    lookup={}
    ing={}
    app={}
    world_journal=[json.loads(x)for x in (folder/'final-journal.jsonl').read_bytes().splitlines()]
    for e in world_journal:
        call=e['PID'],e['Process'],e['Contact']
        if e['Event']=='ingress':
            check(call not in ing and e['Method']=='POST','unique physical ingress');ing[call]=e
        elif e['Event']=='snapshot':
            answer=json.loads(e['Response']);query=answer['Query']; prior=[a for a in app.values() if a['Key']==query['TargetKey']]
            check(query==json.loads(ing[call]['Body']) and answer['Count']==len(prior) and answer==e['Snapshot'] and
                  query['WireSHA256']==sha(query['Wire'].encode()) and query['LedgerID']==w.row('ledger_identity')['id'],'honest endpoint snapshot and physical input')
            ordinal=d.desc('receipt_read_ordinals',intent_id=query['QueryID'],ordinal=query['Ordinal'])
            check(ordinal['Query']==query and query['TargetKey']==enc([2,ns,query['TargetID']]) and
                  (query['TargetID'] in (l,r)) and d.desc('adapter_attempts',intent_id=query['TargetID'])['Prepared']['Wire']==query['Wire'],'eligible bound query')
            if len(prior)==1:
                check(answer['Receipt']==prior[0] and answer['AppliedWire']==query['Wire'] and answer['Status']=='found','unique authentic positive')
            else:
                check(answer['Receipt'] is None and answer['AppliedWire']=='' and answer['Status']==('conflict' if prior else 'not_found'),'no ambiguous evidence')
            wr=w.row('receipt_reads',number=answer['Number'])
            check(wr['response']==e['Response'].encode() and wr['request']==ing[call]['Body'].encode(),'ledger read bytes')
            lookup[query['QueryID']]=e['Response']
        elif e['Event']=='committed':
            rec=e['Receipt'];n=rec['Mutation']
            check(n not in app and rec['Total']==sum(a['Delta'] for a in app.values())+rec['Delta'] and
                  json.loads(w.row('applications',number=n)['receipt'])==rec and
                  w.row('application_wires',number=n)['wire']==ing[call]['Body'].encode(),'actual mutation and original bytes')
            app[n]=rec
        else:check(e['Event']=='rejected' and ing[call]['Path'].endswith('/apply'),'rejected mutation')
    check(len(app)==len(w['applications']) and len(lookup)==len(w['receipt_reads']),'world counts')
    malformed={}
    for key,upstream in lookup.items():
        delivered=results[key]['Candidate']['Response']
        if delivered!=upstream:malformed[key]=(upstream,delivered)
    check(bool(malformed)==(name=='lying-second-snapshot'),'reported endpoint trust boundary')
    decision=unpack(project(d,'parent')['Data']).get('decision')
    if name.startswith('competing'):
        held=images['returns-held']; unpaid=images['published-unpaid']; paid=images['debt-paid']
        twin=single(k for k in results if json.loads(k)[0]=='twin');dup=ls[0]['Operation']
        check(not held.where('adapter_results',intent_id=twin) and not held.where('adapter_results',intent_id=dup) and
              all(held.where('receipt_read_ordinals',intent_id=i) for i in (twin,dup)),'distinct pre-admitted returns')
        proof=unpaid.desc('adapter_results',intent_id=l)['Reconciled']
        check(proof['QueryID']==twin and proof['Receipt']==json.loads(lookup[twin])['Receipt']==json.loads(lookup[dup])['Receipt']
              and unpaid.desc('adapter_results',intent_id=twin)['Read']['Adoption']=='adopted' and
              unpaid.desc('adapter_results',intent_id=dup)['Read']['Adoption']=='duplicate','unique original import and duplicate')
        check(not unpaid.where('settlements',intent_id=twin) and bool(unpaid.where('settlements',intent_id=l))==(name=='competing-adopter-query')
              and unpaid.where('settlements',intent_id=dup) and unpaid.where('settlements',intent_id=r) and
              all(unpaid.where('composition_publications',operation_id=enc(['parent',1,s,0])) for s in (1,2)),'original publication despite precise unpaid settlement')
        sender=single(x for x in unpaid['intents'] if x['subject']=='resolver_l' and json.loads(x['descriptor'])['Obligation']=='l_report')
        admission=single(row for row in unpaid['admissions'] if json.loads(row['descriptor'])['IntentID']==sender['id'])
        check(json.loads(admission['result'])['Accepted'] and
              unpack(unpaid.desc('observations',id=admission['observation_id'])['Payload'])['witness_adoption']=='duplicate','actual duplicate report delivered')
        check(all(paid.where('settlements',intent_id=i) for i in (twin,l)) and worlds['published-unpaid'].same(worlds['debt-paid'])
              and unpaid['adapter_results']==paid['adapter_results'] and decision=='applied','local debt paid on unchanged world')
        faults=[e['Detail']for e in events if e['Event']=='same-owner twin Q']
        check(len(faults)==2 and faults[0]==faults[1] and 'abort' in faults[0],'same owner sticky Q')
    elif name=='wrong-role':
        a=single(e['Detail']for e in events if 'rejected at L receive' in e['Event'])
        seq=select(d,a['RightObservation'])[2]
        eligible=single((s,b) for s,b in observed(d,'parent')if b['Obligation']=='l_report')
        check(a['Error']=='ineligible observation' and a['Parent']['Sequence']+1==eligible[0]<seq and a['RightTarget']==r and
              unpack(d.desc('observations',id=a['RightObservation'])['Payload'])['target']==r and unpack(eligible[1]['Payload'])['target']==l and decision=='applied',
              'real wrong-role refusal then correct selection')
    elif name=='external-multiplicity':
        external=[e['Detail'] for e in events if e['Event']=='external non-idempotent submission']
        leftkey=enc([2,ns,l])
        check(len(external)==2 and all(json.loads(x['Wire'])['Key']==leftkey and json.loads(x['Wire'])['Profile']=='non-idempotent' and
              x['Receipt']==app[x['Receipt']['Mutation']] for x in external) and len(app)==3 and
              Counter(e['Event'] for e in world_journal)['rejected']==1 and unpack(ls[0]['Payload'])['status']=='conflict' and decision=='invalid_evidence','external multiplicity')
    elif name=='lying-second-snapshot':
        key=single(malformed);up,delivered=malformed[key]
        fake=single(e['Detail'] for e in events if 'substituted' in e['Event'])
        check((fake['Upstream'],fake['Delivered'])==(up,delivered) and ls[1]['Operation']==key and
              json.loads(up)['Query']==json.loads(delivered)['Query'] and json.loads(up)['Receipt']!=json.loads(delivered)['Receipt'] and
              decision=='invalid_evidence','true upstream versus substituted positive')
        report=reports(d,role_at(d,'l'),[(o['Operation'],unpack(o['Payload'])) for o in ls])
        fields=unpack(report['Payload'])
        check(fields['conflict'] and fields['witness_query']==ls[0]['Operation'] and fields['query2_mutation']!=fields['witness_mutation'],'stable initial witness plus sticky conflict')
    elif name=='read-only':
        check(not single(g['Adopt'] for g in d.desc('reconciliation_profile')['Grants'] if g['Resolver']['Subject']=='resolver_r') and
              unpack(rs[0]['Payload'])['adoption']=='denied' and r not in results and decision is None,'read-only R')
    elif name=='unsupported':
        check(all(d.desc('adapter_attempts',intent_id=oid)['Prepared']['Model']=='unknown' and oid not in results for oid in (l,r)) and
              all(unpack(o['Payload'])['adoption']=='unsupported_policy' for o in ls+rs) and decision is None,'found under unknown policy')
    elif name=='wrong-grant':
        grants=d.desc('reconciliation_profile')['Grants']
        check(len({g['Invocation'] for g in grants})==1 and results[rs[0]['Operation']]['Disposition']=='read_denied' and
              not d.where('receipt_read_ordinals',intent_id=rs[0]['Operation']) and r not in results and decision is None,'wrong original locator refused before read')
    elif name=='ledger-replacement':
        old=worlds['old-ledger'].row('ledger_identity')['id'];new=w.row('ledger_identity')['id']
        endpoints=[e['Detail']for e in events if e['Event']=='endpoint ready']
        change=single(e['Detail'] for e in events if e['Event']=='same address new ledger')
        check(old!=new==change['LedgerID'] and old==endpoints[0]['LedgerID'] and urlsplit(change['URL']).netloc==urlsplit(endpoints[0]['URL']).netloc
              and len(worlds['old-ledger']['applications'])==2 and not w['applications'] and not lookup and len(ing)==2 and
              all(e['Path']=='/phase2c/'+old+'/lookup' for e in ing.values()) and all(results[o['Operation']]['Candidate']['HTTPStatus']==404 for o in ls+rs)
              and decision is None,'replacement at pinned address')
    return {'cuts':sorted(images),'decision':decision,'applications':len(app),'snapshots':len(lookup),'ingress':len(ing),'substituted':list(malformed)}


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('corpus',type=Path)
    args=parser.parse_args()
    print(json.dumps({'checker_sha256':sha(Path(__file__).read_bytes()),'corpus_sha256':group_digest(args.corpus),
                      'cases':{name:audit(args.corpus/name) for name in NAMES}},sort_keys=True))
