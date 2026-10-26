from pathlib import Path
import collections, datetime, gzip, hashlib, importlib.util, json, os, platform, shutil, struct, subprocess, sys, tarfile, tempfile, time
REPO=Path('/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan')
DOC=REPO/'docs/foundational/phase1e'
OUT=REPO/'docs/foundational/phase1f/evidence/baseline'
OUT.mkdir(parents=True,exist_ok=True)
shutil.copyfile(__file__,OUT/'run-baseline.py')
ENV=dict(os.environ,GOTOOLCHAIN='local',GOFLAGS='-mod=readonly',GOCACHE='/tmp/alan-phase1f-go-build',GOPROXY='off',GOSUMDB='off',PYTHONDONTWRITEBYTECODE='1')
ENV.pop('OPENAI_API_KEY',None)
os.environ.clear();os.environ.update(ENV)
def save(name,data): (OUT/name).write_text(json.dumps(data,indent=2)+'\n')
def sha(raw):return hashlib.sha256(raw).hexdigest()
def command(cmd,cwd):
 start=time.monotonic();p=subprocess.run(cmd,cwd=cwd,env=ENV,text=True,capture_output=True)
 return dict(Command=cmd,Directory=str(cwd),ExitCode=p.returncode,ElapsedSeconds=round(time.monotonic()-start,3),Stdout=p.stdout,Stderr=p.stderr)
m=json.loads((DOC/'source-manifest.json').read_text())
def verify(root):
 digest=hashlib.sha256()
 for f in m['files']:
  raw=(root/f['path']).read_bytes();name=f['path'].encode()
  assert len(raw)==f['bytes'] and sha(raw)==f['sha256'],f['path']
  digest.update(struct.pack('>Q',len(name))+name+struct.pack('>Q',len(raw))+raw)
 assert digest.hexdigest()==m['source_sha256']
 return dict(Files=len(m['files']),SHA256=digest.hexdigest())
def old_hashes():
 return {str(p.relative_to(REPO)):sha(p.read_bytes()) for folder in sorted((REPO/'docs/foundational').glob('phase1*')) if folder.name!='phase1f' for p in sorted(folder.rglob('*')) if p.is_file() and '__pycache__' not in p.parts}
before=old_hashes();save('old-artifacts-before.json',before)
identity=dict(SourceID=m['source_sha256'],MergedGitCommit=subprocess.check_output(['git','rev-parse','HEAD'],cwd=REPO,text=True).strip(),MergedGitTree=subprocess.check_output(['git','rev-parse','HEAD^{tree}'],cwd=REPO,text=True).strip(),Before=verify(REPO),Platform=platform.platform(),Python=sys.version,GoVersion=command(['go','version'],REPO),GoEnvironment=command(['go','env','-json','GOOS','GOARCH','CGO_ENABLED','CC','GOVERSION'],REPO),CCompiler=command(['cc','--version'],REPO),Environment={k:ENV[k] for k in ('GOTOOLCHAIN','GOFLAGS','GOCACHE','GOPROXY','GOSUMDB')},OPENAI_API_KEY='unset')
a=json.loads((DOC/'archive-manifest.json').read_text());archive=DOC/'source.tar.gz'
assert sha(archive.read_bytes())==a['SHA256']
shutil.copyfile(archive,OUT/'source.tar.gz');shutil.copyfile(DOC/'source-manifest.json',OUT/'published-source-manifest.json');shutil.copyfile(DOC/'archive-manifest.json',OUT/'published-archive-manifest.json')
for name in ('02-qualification.md','03-review.md'):shutil.copyfile(DOC/name,OUT/('published-'+name))
FROZEN=Path(tempfile.mkdtemp(prefix='alan-phase1f-baseline-source-'))
with tarfile.open(archive) as tar:
 assert len(tar.getmembers())==len(a['Files'])
 for f in a['Files']:
  member=tar.getmember(f['Path']);raw=tar.extractfile(member).read()
  assert member.isfile() and not Path(member.name).is_absolute() and '..' not in Path(member.name).parts
  assert len(raw)==f['Bytes'] and sha(raw)==f['SHA256']
 tar.extractall(FROZEN,filter='data')
identity.update(FrozenDirectory=str(FROZEN),FrozenBefore=verify(FROZEN),ArchiveSHA256=a['SHA256'],ArchiveVerifiedMembers=len(a['Files']),ArchiveMethod='Published source archive; every member verified; current merged source matches same 133-file identity')
save('identity.json',identity)
print('MILESTONE frozen published archive: 224 members verified; 133 merged input files match '+m['source_sha256'],flush=True)
for mode in ('default','normal','race'):
 original=FROZEN/'docs/foundational/phase1e/tools/qualify.py'
 code=original.read_text().replace('OUT = DOC / "evidence"', 'OUT = Path('+repr(str(OUT))+')')
 start=time.monotonic()
 p=subprocess.run([sys.executable,'-c','import sys;sys.argv=["qualify.py",'+repr(mode)+'];exec(compile('+repr(code)+','+repr(str(original))+',"exec"),{"__file__":'+repr(str(original))+'})'],cwd=FROZEN,env=ENV,text=True,capture_output=True,timeout=900)
 (OUT/f'collector-{mode}.log').write_text(p.stdout+p.stderr)
 print('MILESTONE '+mode+' collector exit '+str(p.returncode)+' elapsed '+str(round(time.monotonic()-start,3)),flush=True)
 assert p.returncode==0,p.stdout[-4000:]+p.stderr
commands=[]
for cmd in [[sys.executable,'-m','unittest','discover','-s','docs/foundational/phase1e/tools','-p','test_comparison.py','-v'],['go','vet','./...'],['go','vet','-tags','integration','./...'],['gofmt','-l','author','schema','runtime','effects','adapters','cmd','internal','research','examples'],['go','mod','verify']]:
 r=command(cmd,FROZEN);commands.append(r);save('static-checks.json',commands)
 assert r['ExitCode']==0,r
 if cmd[0]=='gofmt':assert not r['Stdout'],r
print('MILESTONE static checks and seven comparison controls PASS',flush=True)
sys.path.insert(0,str(FROZEN/'docs/foundational/phase1e/tools'))
from comparison import compare_research,rows,conformance_projection
historical=rows(FROZEN/'docs/foundational/phase1e/evidence/baseline/final-normal.records.jsonl.gz')
comparisons=[];current_comparison=[];causal={};selected={}
required=['TestSupervisionInterruptionWitnessOrdering','TestAsyncOracleRejectsFalseOrderAndIdentity','TestBarrierRejectsCrossExecutionAndUnorderedEvidence','TestCancellationWildcardBoundaryRejectsDurableSubstitutions','TestAcknowledgmentRejectsIdentitySubstitution','TestWildcardAcknowledgmentRejectsRetainedSequenceSubstitution','TestAdapterCancellationAccountsReceivedCandidate','TestCancellationAfterDequeueAccountsCompletion','TestCancellationDuringAdapterSettlementPreservesEvidence','TestCancellationAccountingErrorRetainsKnownPendingResult','TestCancellationRetainsRealReceivedModelCandidate','TestCancellationAfterDequeueRetainsRealModelCandidate']
for mode in ('default','normal','race'):
 records=rows(OUT/f'final-{mode}.records.jsonl.gz');events=rows(OUT/f'final-{mode}.go-test.jsonl.gz');meta=json.loads((OUT/f'final-{mode}.command.json').read_text())
 assert meta['PassingEvents']==(462 if mode=='default' else 655)
 comparisons.extend(compare_research(historical,records,FROZEN,m['source_sha256'],mode,events))
 published=rows(DOC/f'evidence/final-{mode}.records.jsonl.gz')
 for marker in ('QUALIFICATION','COMPOSITION','SUPERVISION','CONFORMANCE'):
  before_payload=[x['Evidence'] for x in published if x['Marker']==marker];after_payload=[x['Evidence'] for x in records if x['Marker']==marker]
  equal=before_payload==after_payload
  if marker=='CONFORMANCE':assert [conformance_projection(x) for x in before_payload]==[conformance_projection(x) for x in after_payload]
  else:assert equal,(mode,marker)
  current_comparison.append(dict(Mode=mode,Marker=marker,Records=len(after_payload),ExactOrderedPayloadsEqual=equal,ExactPayloadCount=sum(a==b for a,b in zip(before_payload,after_payload)),PublicProjectionAndModelStreamsEqual=True if marker=='CONFORMANCE' else None))
 passed={e['Test'] for e in events if e.get('Action')=='pass' and e.get('Test')}
 expected=required if mode!='default' else [x for x in required if x not in ('TestCancellationWildcardBoundaryRejectsDurableSubstitutions','TestAcknowledgmentRejectsIdentitySubstitution','TestWildcardAcknowledgmentRejectsRetainedSequenceSubstitution','TestCancellationRetainsRealReceivedModelCandidate','TestCancellationAfterDequeueRetainsRealModelCandidate')]
 selected[mode]={name:sorted(p for p in passed if p==name or p.startswith(name+'/')) for name in expected}
 assert all(selected[mode].values()),selected[mode]
 processes=[r for r in records if isinstance(r['Evidence'],dict) and r['Evidence'].get('type')=='process']
 assert processes and all(r['Evidence'].get('race_children')==(mode=='race') for r in processes)
 cases=[r for r in records if isinstance(r['Evidence'],dict) and any(k in r['Evidence'] for k in ('causal','causal_order','events','barrier','negative_control','control'))]
 causal[mode]=dict(ProcessRecords=len(processes),RaceChildren=dict(collections.Counter(str(r['Evidence'].get('race_children')) for r in processes)),SIGKILLSignalRecords=sum(r['Evidence'].get('exit_signal')=='killed' for r in processes),CausalRecords=cases)
identity.update(After=verify(REPO),FrozenAfter=verify(FROZEN));save('identity.json',identity)
after=old_hashes();save('old-artifacts-after.json',after);assert before==after
save('historical-comparison.json',comparisons);save('published-current-comparison.json',current_comparison);save('selected-controls.json',selected);save('causal-cases.json',causal)
save('summary.json',dict(Result='PASS',SourceID=m['source_sha256'],MergedGitCommit=identity['MergedGitCommit'],Modes={mode:json.loads((OUT/f'final-{mode}.command.json').read_text()) for mode in ('default','normal','race')},OldArtifactsUnchanged=True,OldArtifactFiles=len(before),SourceUnchanged=True,LiveProviderGate='NOT RUN'))
print('MILESTONE baseline complete; frozen and merged 133-file identities unchanged, historical and published-current comparisons PASS',flush=True)
