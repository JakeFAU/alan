"""Two-original recovery and ACK schedules using real process owners and ledger."""
import argparse
import gzip
import json
from pathlib import Path
import tempfile
from urllib.parse import urlsplit
from process_lab import Lab, count, projections, rowset, sha, value, wait_for, write


def advance(p):
    p.stdin.write(b'continue\n')
    p.stdin.flush()


def apply(packet):
    packet.accept.set()
    assert packet.upstream_done.wait(80) and packet.result[0]==200,packet.result
    return json.loads(packet.result[1])


def targets(lab):
    p=projections(lab.db)['parent']
    return {role:value(p,role+'_target') for role in 'lr'}


def closed(lab):
    t=targets(lab)
    if not all(t.values()):
        return False
    s=projections(lab.db)
    return all(s[json.loads(id)[0]]['Status'] in ('succeeded','failed') for id in t.values())


def original_loss(lab, order='lr', mixed=False, capacity=2, late=False):
    lab.relay.held=True
    p=lab.owner('issuer-before-end',capacity=capacity)
    # In the mixed case the observing L needs its dropped connection before End.
    # R's staged End or the two HTTP arrivals may be observed independently.
    packets=[lab.relay.next() for _ in range(capacity)]
    assert all(x.path.endswith('/apply') for x in packets)
    src={}
    for role,seq in [('l',1),('r',2)]:
        handle=json.dumps(['parent',1,seq,0],separators=(',',':'))
        desc=json.loads(rowset(lab.db,'SELECT descriptor FROM composition_admission_decisions WHERE operation_id=?',(handle,))[0][0])
        src[desc['Members'][0]['Subject']]=role
    by_role={src[json.loads(json.loads(json.loads(x.body)['Key'])[2])[0]]:x for x in packets}
    assert set(by_role)==set('lr' if capacity==2 else 'l'),set(by_role)
    receipts={}
    if late:
        packet=by_role['r']
        packet.accept.set()
        event=json.loads(lab.barrier[0].readline())
        assert event['PID']==lab.ep.pid and event['Boundary']=='resource.ingress' and event['Intent']==json.loads(packet.body)['Key']
        lab.event('endpoint-holds-right',event)
    for role in order:
        if role not in by_role or (late and role=='r'):
            continue
        packet=by_role[role]
        receipts[role]=apply(packet)
        if mixed and role=='l':
            packet.drop=True
            packet.respond.set()
    first=lab.ack(p,'issuer-before-end')
    lab.event('originals-before-first-end',dict(staged=first,receipts=receipts,members=src))
    advance(p)
    second=lab.ack(p,'issuer-before-end')
    assert second['Subject']!=first['Subject']
    advance(p)
    wait_for(lambda:closed(lab),'both genuine issuer Ends')
    lab.save('before-death')
    lab.stop(p,kill=True)
    lab.event('original-loss',{'targets':targets(lab),'order':order,'mixed':mixed})
    for role,packet in by_role.items():
        if not (late and role=='r'):
            packet.respond.set()
    lab.relay.held=False
    return by_role


def read(lab,role,port='start'):
    lab.service(role,port)
    lab.run()


def scalar(lab,sql,arg=()):
    return rowset(lab.db,sql,arg)[0][0]


def business(lab):
    p=projections(lab.db)['parent']
    keys=['decision','tree_class','l_class','r_class','l_source_status','r_source_status','l_issued_target','r_issued_target']
    for role in 'lr':
        keys.extend(role+'_report_witness_'+k for k in ('mutation','total'))
    return {k:value(p,k) for k in keys}


def finish(lab,expected='applied'):
    b=business(lab)
    assert b['decision']==expected,b
    lab.save('final')
    lab.replay()
    write(lab.out/'business.json',b)
    return b


def release_right(lab,packet):
    lab.barrier[1].write(b'c')
    receipt=apply(packet)
    packet.respond.set()
    assert receipt['Delta']==2
    lab.event('right-applied',receipt)


def check_sole(lab, role):
    pending=rowset(lab.db,"SELECT i.id,i.descriptor FROM intents i LEFT JOIN message_acknowledgments a ON a.intent_id=i.id WHERE json_extract(i.descriptor,'$.Kind')='message' AND a.intent_id IS NULL")
    assert len(pending)==1
    ack=json.loads(pending[0][1])
    assert ack['Run']['Subject']=='parent' and ack['Obligation']==role+'_report'
    assert scalar(lab,"SELECT count(*) FROM intents i LEFT JOIN composition_outcome_deliveries d ON d.operation_id=i.id WHERE json_extract(i.descriptor,'$.Kind')='invocation' AND d.operation_id IS NULL")==0
    assert scalar(lab,"SELECT count(*) FROM intents WHERE json_extract(descriptor,'$.Kind')='lifecycle'")==0
    for dest,status in rowset(lab.db,'select destinations,delivery_status from terminal_obligations'):
        assert not json.loads(dest) or len(json.loads(dest))==sum(json.loads(status).values())
    p=projections(lab.db)['parent']
    assert p['Status']=='open' and value(p,'decision')=='applied'
    lab.event('sole-internal-delivery',{'operation':pending[0][0],'obligation':ack['Obligation'],'decision':value(p,'decision')})

def case(lab,name):
    mixed = name=='mixed'
    late = name in ('lost-absence','retained-absence','finite-reads')
    two = name in ('stable-duplicate','stable-error','retained-absence','finite-reads')
    cap = 1 if name=='capacity-one' else 2
    args = ['-queries','1','-capacity',str(cap)]
    if mixed:
        args+=['-left-observe=true']
    if name=='read-only':
        args+=['-right-adopt=false']
    if name=='equal-reverse':
        args+=['-right-delta','1']
    if two:
        args+=['-right-queries','2']
    lab.cli('init',*args)
    packets=original_loss(lab,'rl' if name in ('reverse','equal-reverse') else 'lr',mixed,cap,late)
    ids=targets(lab)
    if cap==1:
        lab.run(capacity=1)
        lab.save('final')
        assert count(lab.world,'applications')==1 and count(lab.world,'contacts')==1
        assert scalar(lab,"SELECT count(*) FROM adapter_attempts WHERE intent_id=? AND json_extract(descriptor,'$.Allowed')=0",(ids['r'],))==1
        lab.replay()
        return
    if not mixed:
        lab.cli('frontier')
    if name=='original-selected':
        lab.service('l')
        p=lab.owner('left-original-selected','parent')
        lab.ack(p,'left-original-selected')
        lab.save('left-original-selected')
        lab.stop(p,kill=True)
        lab.run()
    else:
        read(lab,'l')
    lab.save('left-only')
    assert scalar(lab,'SELECT count(*) FROM settlements WHERE intent_id=?',(ids['l'],))==1
    if mixed:
        raw=scalar(lab,'SELECT descriptor FROM adapter_results WHERE intent_id=?',(ids['l'],))
        assert json.loads(raw)['Disposition']=='transport_error'
    lab.service('r')
    if name in ('pre-retention','atomic','spent','lost-absence'):
        cut={'pre-retention':'adapter.response','atomic':'adapter.result-recorded','spent':'dispatch.admitted','lost-absence':'adapter.response'}[name]
        for i in range(2 if name=='spent' else 1):
            p=lab.owner(cut,'resolver_r')
            lab.ack(p,cut)
            lab.save('ordinal-'+str(i+1))
            lab.stop(p,kill=True)
        if name=='lost-absence':
            release_right(lab,packets['r'])
        lab.run()
    elif name in ('manual-ack','fresh-ack','auto-acks'):
        cut='both-final-acks' if name=='auto-acks' else 'last-final-ack'
        p=lab.owner(cut,'parent')
        if name=='auto-acks':
            lab.ack(p,cut)
            check_sole(lab,'l')
            lab.save('first-ack-committed')
            advance(p)
        lab.ack(p,cut)
        check_sole(lab,'r')
        lab.save('ack-committed')
        if name=='manual-ack':
            p.stdin.write(b'manual\n')
            p.stdin.flush()
            lab.stop(p)
        elif name=='fresh-ack':
            lab.stop(p,kill=True)
            lab.run()
        else:
            advance(p)
            lab.stop(p)
    else:
        lab.run()
    if two:
        if name=='retained-absence':
            release_right(lab,packets['r'])
        lab.service('r','refresh')
        if name=='stable-error':
            while not lab.relay.incoming.empty():
                lab.relay.next()
            lab.relay.held=True
            p=lab.owner()
            query=lab.relay.next()
            assert query.path.endswith('/lookup')
            apply(query)
            query.drop=True
            query.respond.set()
            lab.stop(p)
            lab.relay.held=False
        else:
            lab.run()
        if name=='finite-reads':
            release_right(lab,packets['r'])
            lab.run()
    unresolved=name in ('spent','finite-reads','read-only')
    b=finish(lab,None if unresolved else 'applied')
    rs=projections(lab.db)
    rv=lambda k:value(rs['resolver_r'],k)
    assert count(lab.world,'applications')==2
    if unresolved:
        assert rs['parent']['Status']=='open' and value(rs['parent'],'l_source_status')=='succeeded'
        assert value(rs['parent'],'r_source_status') is None
    if name=='atomic':
        assert count(lab.world,'receipt_reads')==2
    if name=='spent':
        assert scalar(lab,"SELECT count(*) FROM receipt_read_ordinals WHERE json_extract(intent_id,'$[0]')='resolver_r'")==2
        assert count(lab.world,'receipt_reads')==1
    if name=='manual-ack':
        assert rs['resolver_r']['Status']=='open' and rv('report_ack') is None
    if name=='fresh-ack':
        assert rs['resolver_r']['Status']=='succeeded' and rv('report_ack')==ids['r']
    if name in ('stable-duplicate','stable-error'):
        assert rv('witness_query')==rv('query1') and rv('query1_adopted')
        if name=='stable-duplicate':
            assert rv('query2_adoption')=='duplicate' and not rv('query2_adopted')
        else:
            assert rv('query2_disposition')=='transport_error' and not rv('query2_valid')
    if mixed:
        assert raw==scalar(lab,'SELECT descriptor FROM adapter_results WHERE intent_id=?',(ids['l'],))
        assert b['l_class']=='applied_after_original_failure'



def paired(bins,out,work):
    # Fork a durable prefix without an in-flight packet. All historical
    # applications (with PID, process token, wire and receipt) precede the fork.
    prefix=Lab(bins,out/'prefix',work/'prefix')
    try:
        prefix.cli('init','-queries','1')
        original_loss(prefix)
        prefix.cli('frontier')
        read(prefix,'l')
        prefix.service('r')
        assert count(prefix.world,'applications')==2 and count(prefix.world,'receipt_reads')==1
        prefix.save('fork')
        address=urlsplit(prefix.url).port
        endpoint=prefix.url
    finally:
        prefix.close()
    records={}
    for variant in ('continuous','lost-response'):
        lab=Lab(bins,out/variant,work/variant,prefix.out/'fork-world.sqlite.gz',address)
        try:
            assert lab.url==endpoint
            lab.db.write_bytes(gzip.decompress((prefix.out/'fork-local.sqlite.gz').read_bytes()))
            lab.event('forked',{'local':sha(prefix.out/'fork-local.sqlite.gz'),'world':sha(prefix.out/'fork-world.sqlite.gz'),'prefix_journal':sha(prefix.log)})
            if variant=='lost-response':
                owner=lab.owner('adapter.response','resolver_r')
                lab.ack(owner,'adapter.response')
                lab.save('lost')
                lab.stop(owner,kill=True)
            lab.run()
            result=finish(lab)
            records[variant]=dict(business=result,ordinals=count(lab.db,'receipt_read_ordinals'),
                applications=[json.loads(x[0]) for x in rowset(lab.world,'select receipt from applications order by number')],
                contacts=count(lab.world,'receipt_reads'),local_archive=sha(lab.out/'final-local.sqlite.gz'),world_archive=sha(lab.out/'final-world.sqlite.gz'))
        finally:
            lab.close()
    x,y=records.values()
    assert x['business']==y['business'] and x['applications']==y['applications'] and len(x['applications'])==2
    assert (x['ordinals'],y['ordinals'],x['contacts'],y['contacts'])==(2,3,2,3)
    write(out/'comparison.json',records)

CASES=('forward','reverse','equal-reverse','stable-duplicate','stable-error','mixed','capacity-one','read-only',
       'retained-absence','finite-reads','lost-absence','spent','atomic','pre-retention','original-selected','fresh-ack','manual-ack','auto-acks')


def main():
    ap=argparse.ArgumentParser(description=__doc__)
    ap.add_argument('bins',type=Path)
    ap.add_argument('output',type=Path)
    ap.add_argument('cases',nargs='+',choices=CASES+('paired',))
    a=ap.parse_args()
    bins={name:(a.bins/name).resolve() for name in ('consumer','resource','owner')}
    for name in a.cases:
        with tempfile.TemporaryDirectory(prefix='phase2d-case-') as temp:
            if name=='paired':
                paired(bins,a.output/name,Path(temp))
                print(name,'PASS',flush=True)
                continue
            late=name in ('lost-absence','retained-absence','finite-reads')
            lab=Lab(bins,a.output/name,Path(temp)/'work',hold_ingress=late)
            try:
                case(lab,name)
            finally:
                lab.close()
        print(name,'PASS',flush=True)


if __name__=='__main__':
    main()
