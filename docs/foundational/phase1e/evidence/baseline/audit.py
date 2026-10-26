from pathlib import Path
import collections,gzip,hashlib,json,subprocess
root=Path('/tmp/alan-phase1e-baseline-source')
out=Path('/tmp/alan-phase1e-baseline-evidence')
hist=root/'docs/foundational/phase1d1/evidence'
def rows(p): return [json.loads(x) for x in gzip.decompress(p.read_bytes()).splitlines()]
def selected(a,m): return [x['Evidence'] for x in a if x['Marker']==m]
result={'current_source_id':'9fef40370632bdf7416d5bdb2e8472f31ae2adc80ae5145cd7c9c90ccfe63e63','historical_source_id':'9fef40370632bdf7416d5bdb2e8472f31ae2adc80ae5145cd7c9c90ccfe63e63','comparison_source':'retained Phase 1D1 final records','modes':{}}
for mode in ('normal','race'):
 current=rows(out/f'final-{mode}.records.jsonl.gz')
 prior=rows(hist/f'final-{mode}.records.jsonl.gz')
 meta=json.loads((out/f'final-{mode}.command.json').read_text())
 assert meta['ExitCode']==0 and meta['PassingEvents']==357 and not meta['Failures']
 assert meta['SourceID']==result['current_source_id']
 d={'pass_events':meta['PassingEvents'],'markers':dict(collections.Counter(x['Marker'] for x in current)),'comparisons':{},'conformance':{},'scenario_tests':{},'process_evidence':{}}
 for marker,n in [('QUALIFICATION',15),('COMPOSITION',20),('SUPERVISION',32)]:
  a,b=selected(current,marker),selected(prior,marker)
  d['comparisons'][marker]={'count':len(a),'expected':n,'exact_ordered_payload_equal_to_phase1d1_final':a==b}
  assert len(a)==len(b)==n and a==b,(mode,marker)
 a,b=selected(current,'CONFORMANCE'),selected(prior,'CONFORMANCE')
 d['conformance']={'count':len(a),'all_passed':all(x.get('passed') is True for x in a),'test_fixture_order_equal':[(x['test'],x['fixture']) for x in a]==[(x['test'],x['fixture']) for x in b],'exact_payloads_equal':sum(x==y for x,y in zip(a,b)),'fixtures':[{'test':x['test'],'fixture':x['fixture'],'passed':x['passed'],'checkpoints':len(x.get('checkpoints',[]))} for x in a]}
 assert len(a)==len(b)==6 and d['conformance']['all_passed'] and d['conformance']['test_fixture_order_equal']
 raw=rows(out/f'final-{mode}.go-test.jsonl.gz')
 passed={x.get('Test','') for x in raw if x.get('Action')=='pass' and x.get('Test')}
 names=['TestAsyncHeldRequestPeerProgress','TestAsyncSaturationLeavesBacklogAndDeliversPeer','TestAsyncReorderedReceiptsConsumeWithoutHeadOfLineAndReplay','TestAsyncCancelAccountsReceivedCandidateAndPreservesUnknown','TestAsyncLimitsWithHeldWork','TestAsyncLateReceiptValidatesRetainedEpochAndRejectsForgery','TestAsyncOrdinarySupervisorAdvancesWhileExternalHeld','TestAsyncKnownReceiptSurvivesSettlementFailure','TestAsyncAuthorizedTransportFinishesAfterOrdinarySourceClosure','TestAutomaticAsyncHeldProgressAndConcurrentCrash','TestRuntimeConformanceAcceptedLossAndSuccessor','TestRuntimeConformanceRejectionSurvivesFailure','TestRuntimeConformanceExternalGenerationsAndLateReceipt']
 for name in names:
  matched=sorted(x for x in passed if x==name or x.startswith(name+'/'))
  d['scenario_tests'][name]=matched
  assert matched,(mode,name)
 durable=selected(current,'DURABLE')
 processes=[x for x in durable if x.get('type')=='process']
 d['process_evidence']={'process_records':len(processes),'race_children_values':dict(collections.Counter(str(x.get('race_children')) for x in processes)),'sigkill_exit_signal_count':sum(x.get('exit_signal')=='killed' for x in processes),'explicit_sigkill_records':sum(x.get('exit')=='SIGKILL' for x in durable),'async_durable_records':sum(x.get('type')=='ASYNC' for x in durable)}
 assert d['process_evidence']['process_records']>=80
 assert all(x.get('race_children')==(mode=='race') for x in processes)
 result['modes'][mode]=d
(out/'comparison.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps({mode:{'passes':r['pass_events'],'markers':r['markers'],'comparisons':r['comparisons'],'conformance':{k:v for k,v in r['conformance'].items() if k!='fixtures'},'process_evidence':r['process_evidence']} for mode,r in result['modes'].items()},indent=2))
