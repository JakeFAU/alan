from pathlib import Path
import collections,gzip,hashlib,json,shutil
REPO=Path('/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan')
OUT=REPO/'docs/foundational/phase1f/evidence/baseline'
assert json.loads((OUT/'summary.json').read_text())['Result']=='PASS'
shutil.copyfile(__file__,OUT/'audit-baseline.py')
def rows(name):return [json.loads(x) for x in gzip.decompress((OUT/name).read_bytes()).splitlines()]
causal={};regressions={}
for mode in ('default','normal','race'):
 records=rows(f'final-{mode}.records.jsonl.gz');events=rows(f'final-{mode}.go-test.jsonl.gz')
 e=[r['Evidence'] for r in records if isinstance(r['Evidence'],dict)]
 builds=[x for x in e if x.get('type')=='build'];processes=[x for x in e if x.get('type')=='process'];marked=[x for x in e if 'race_children' in x]
 assert all(x['race_children']==(mode=='race') for x in marked)
 for b in builds:
  assert ('-race' in b['command'])==(mode=='race')
  assert len(b['binary_sha256'])==64
 cases=[]
 for r in records:
  x=r['Evidence']
  if not isinstance(x,dict) or 'causal_events' not in x:continue
  names=[v['Name'] for v in x['causal_events']]
  expected=['endpoint-held','peer-committed','peer-delivered','held-live']+(['response-released','result-settled'] if x['cut']=='settle' else [])
  assert names==expected,(mode,names)
  endpoint=x['endpoint'];assert len(endpoint['Applications'])==1 and endpoint['Total']==5
  want_contacts=2 if x['profile']=='idempotent' and x['cut']!='settle' else 1
  assert endpoint['Contacts']==want_contacts
  cases.append(dict(Test=r['Test'],Profile=x['profile'],Cut=x['cut'],OrderedEvents=names,Contacts=endpoint['Contacts'],Applications=len(endpoint['Applications']),Total=endpoint['Total'],ExecutorBinarySHA256=x['executor_binary_sha256'],ResourceBinarySHA256=x['resource_binary_sha256'],RawRecordIndex=r['RecordIndex']))
 assert len(cases)==6
 names=['TestSupervisionInterruptionWitnessCorrelation','TestSupervisionInterruptionWitnessOrdering','TestAsyncOracleRejectsFalseOrderAndIdentity']
 if mode!='default':names+=['TestFinalReviewOutputItemStatus','TestReviewR1NoReversibleCredentialEvidence','TestReviewR2RejectsProtocolCaseAliases','TestReviewR3InspectRequiresReverseAdapterObservationLinks','TestFaultyReplayDispatchIsDetected']
 passed={x['Test'] for x in events if x.get('Action')=='pass' and x.get('Test')}
 regressions[mode]={name:sorted(p for p in passed if p==name or p.startswith(name+'/')) for name in names}
 assert all(regressions[mode].values())
 controls=[]
 for x in e:
  if x.get('type') in ('cancellation-durable-identity-control','durable-sequence-control'):
   assert x['rejected']==(x['case']!='original')
   controls.append({k:x[k] for k in ('type','case','rejected','result')})
  if x.get('type')=='negative-control':assert x['actual_extra_contacts']==1 and x['detected']
 causal[mode]=dict(DurableProcessRecords=len(processes),SIGKILLRecords=sum(x.get('exit_signal')=='killed' for x in processes),RaceChildrenRecordCount=len(marked),RaceChildrenValue=(mode=='race'),EvidenceBuildCommands=builds,AsyncCausalCases=cases,RetainedSubstitutionControls=controls)
(OUT/'causal-audit.json').write_text(json.dumps(causal,indent=2)+'\n')
(OUT/'review-and-phase1c-controls.json').write_text(json.dumps(regressions,indent=2)+'\n')
s=json.loads((OUT/'summary.json').read_text());i=json.loads((OUT/'identity.json').read_text())
lines=['# Phase 1F immutable merged Phase 1E baseline reproduction','',f"**PASS.** New offline reproduction from merged Git commit `{i['MergedGitCommit']}` (tree `{i['MergedGitTree']}`). This Git identity is distinct from the published executable/input identity `{i['SourceID']}`. All 133 published files matched their lengths, hashes and aggregate before and after in both the merged worktree and the isolated frozen source.",'',f"The published `source.tar.gz` (SHA-256 `{i['ArchiveSHA256']}`) was copied here and all {i['ArchiveVerifiedMembers']} members verified before extraction to `{i['FrozenDirectory']}`. Tests ran only in that extraction. The archive carries the published source identity; it is not a newly generated Git-merge archive. Qualification and review reports are copied alongside it.",'',f"Platform: `{i['Platform']}`, `{i['GoVersion']['Stdout'].strip()}`, CGO enabled, `{i['CCompiler']['Stdout'].splitlines()[0]}`. Environment: `GOTOOLCHAIN=local GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build GOPROXY=off GOSUMDB=off`, with `OPENAI_API_KEY` unset. No `.env` file was read and no live provider call was made. Controlled HTTP providers use loopback, fixture credentials and child processes.",'','| Command in frozen source | Result |','| --- | --- |']
for mode,m in s['Modes'].items():lines.append(f"| `{' '.join(m['Command'])}` | PASS, {m['PassingEvents']} test/subtest pass events, zero failures, {m['ElapsedSeconds']}s |")
lines+=['| `python3 -m unittest discover -s docs/foundational/phase1e/tools -p test_comparison.py -v` | PASS, seven comparison controls |','| `go vet ./...`; `go vet -tags integration ./...` | PASS |','| `gofmt -l author schema runtime effects adapters cmd internal research examples` | PASS, empty output |','| `go mod verify` | PASS, all modules verified |','','`final-{default,normal,race}.command.json` records exact commands, environment, wall times, package results, raw hashes and source checks; `identity.json` retains platform/toolchain details. Raw compressed Go JSON and extracted records retain their original order. Pass counts are test/subtest events, not independent properties. ASYNC marker counts vary with concurrently completed work; deterministic research counts are 15 QUALIFICATION, 20 COMPOSITION, 32 SUPERVISION and six CONFORMANCE; each mode retained 119 DURABLE records. Integration and race each retained 120 EVIDENCE records.','','For all three modes, comparison with the published Phase 1E final records preserves exact ordered payloads for all 15 QUALIFICATION, 20 COMPOSITION and 32 SUPERVISION records. The six conformance fixtures preserve ordered public projections and model streams; raw snapshots include ephemeral namespace, endpoint and process data and are not asserted equal. Historical Phase 1D.1 comparisons independently pass using the unchanged Phase 1E comparator: exact finite semantics, the documented validated namespace bridge/provenance change in composition/supervision, and the declared conformance projection. See `published-current-comparison.json` and `historical-comparison.json`.','','`causal-audit.json` verifies six held/crash cases per mode: `endpoint-held → peer-committed → peer-delivered → held-live`, followed by release/settlement for the settle cases. Each has one application and total 5; idempotent recovery has two contacts under the retained identity and non-idempotent recovery one. Each mode retained 85 durable subprocess records and 20 SIGKILL signal records. All race-marked evidence agrees with its mode; race EVIDENCE build commands actually contain `-race` and retain binary hashes. Full raw records provide the snapshots and original boundary identities.','','The full runs explicitly include the Phase 1C interruption correlation and false-order witness tests, async false-order/identity oracle controls, dequeue cancellation accounting, actual CLI cancellation, wildcard epoch/sequence substitution controls, final review output-status regressions, credential/protocol/Inspect review cases, and real extra-dispatch replay negative control. The retained original substitution cases pass, epoch/sequence mutations are rejected, and the faulty replay creates one extra provider contact which the unchanged monitor rejects. Named pass events are in `selected-controls.json` and `review-and-phase1c-controls.json`. No source mutation experiment was rerun beyond the existing selected regression/negative-control tests.','','The first collector invocation completed its default Go run but failed while formatting an output path outside the frozen root. Its raw JSON, wrapper and traceback remain under `initial-collector-path-error-*`. The new wrapper changes only the collector output directory and absolute RawLog metadata; frozen qualification-tool bytes remain unchanged. A complete source-matched default/integration/race execution then passed. A subsequent reporting audit initially expected integration-tagged cancellation tests in default mode; source build tags established the correct selections. `audit-correction.md` records that correction, and the unchanged successful raw runs were re-audited without rerunning Go suites. The current runner includes both wrapper corrections; its original selection version is retained.','','`old-artifacts-before.json` and `old-artifacts-after.json` match for all '+str(s['OldArtifactFiles'])+' prior phase files. `identity.json` confirms the 133-input source before/after checks. No product source, historical evidence, commit or live gate was changed. This is finite controlled reproduction, not a claim of unbounded correctness or production readiness. Live provider gate: **NOT RUN**.','']
report='\n'.join(lines)
(OUT/'report.md').write_text(report)
(REPO/'.superpowers/sdd/00-plan/baseline-report.md').write_text(report)
manifest=[]
for p in sorted(OUT.iterdir()):
 if p.is_file() and p.name!='evidence-manifest.json':
  raw=p.read_bytes();manifest.append(dict(Path=p.name,Bytes=len(raw),SHA256=hashlib.sha256(raw).hexdigest()))
(OUT/'evidence-manifest.json').write_text(json.dumps(dict(SourceID=i['SourceID'],Files=manifest),indent=2)+'\n')
print(json.dumps(dict(Result='PASS',Report=str(OUT/'report.md'),SourceID=i['SourceID'],PassEvents={mode:m['PassingEvents'] for mode,m in s['Modes'].items()},EvidenceFiles=len(manifest)),indent=2))
