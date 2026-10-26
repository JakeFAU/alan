from pathlib import Path
import collections,datetime,gzip,hashlib,importlib.util,json,os,shutil,subprocess,sys,time
repo=Path('/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan');out=repo/'docs/foundational/phase1g/evidence/baseline'
identity=json.loads((out/'identity.json').read_text());frozen=Path(identity['FrozenDirectory']);doc=frozen/'docs/foundational/phase1f'
env=dict(os.environ);env.pop('OPENAI_API_KEY',None);env.pop('ALAN_BARRIER',None);env.update(GOTOOLCHAIN='local',GOFLAGS='-mod=readonly',GOCACHE='/tmp/alan-phase1f-go-build',GOPROXY='off',GOSUMDB='off',PYTHONDONTWRITEBYTECODE='1');os.environ.clear();os.environ.update(env)
def sha(b):return hashlib.sha256(b).hexdigest()
def save(n,v):(out/n).write_text(json.dumps(v,indent=2)+'\n')
shutil.copyfile('/tmp/alan-phase1g-capture.py',out/'capture-baseline.py');shutil.copyfile(__file__,out/'run-baseline.py')
invocations=[]
for mode in ('default','normal'):
 original=doc/'tools/qualify.py';code=original.read_text().replace('OUT = DOC / "evidence"','OUT = Path('+repr(str(out))+')').replace('str(raw_path.relative_to(ROOT))','str(raw_path)')
 cmd=[sys.executable,'-c','import sys;sys.argv=["qualify.py",'+repr(mode)+'];exec(compile('+repr(code)+','+repr(str(original))+',"exec"),{"__file__":'+repr(str(original))+'})']
 start=time.monotonic();p=subprocess.run(cmd,cwd=frozen,env=env,capture_output=True)
 for stream,b in [('stdout',p.stdout),('stderr',p.stderr)]: (out/f'collector-{mode}.{stream}.gz').write_bytes(gzip.compress(b,mtime=0))
 invocations.append({'Mode':mode,'Tool':str(original),'OriginalToolSHA256':sha(original.read_bytes()),'Transformation':'Only OUT redirected to Phase1G baseline evidence and raw log path rendered absolute; source verification, Go commands, parser and assertions unchanged.','ExitCode':p.returncode,'ElapsedSeconds':round(time.monotonic()-start,3),'StdoutSHA256':sha(p.stdout),'StderrSHA256':sha(p.stderr)})
 save('collector-invocations.json',invocations);print(mode,p.returncode,round(time.monotonic()-start,3),flush=True)
 assert p.returncode==0,(mode,p.stdout[-4000:],p.stderr)
# Preserve the published collector, and compare separate historical/model streams with its existing comparison library.
spec=importlib.util.spec_from_file_location('comparison',frozen/'docs/foundational/phase1e/tools/comparison.py');c=importlib.util.module_from_spec(spec);spec.loader.exec_module(c)
comparisons=[]
for mode in ('default','normal'):
 new=c.rows(out/f'final-{mode}.records.jsonl.gz');published=c.rows(doc/f'evidence/final-{mode}.records.jsonl.gz');old=c.rows(frozen/f'docs/foundational/phase1e/evidence/final-{mode}.records.jsonl.gz')
 events=c.rows(out/f'final-{mode}.go-test.jsonl.gz');passed={e['Test'] for e in events if e.get('Action')=='pass' and e.get('Package')=='github.com/JakeFAU/alan/research/semantics' and 'Test' in e}
 for label,base in [('published-phase1f',published),('historical-phase1e',old)]:
  for marker,count in [('QUALIFICATION',15),('COMPOSITION',20),('SUPERVISION',32),('CONFORMANCE',6)]:
   a=[r for r in base if r['Marker']==marker];b=[r for r in new if r['Marker']==marker];c.require(len(a)==len(b)==count,'count');c.require([(r['Package'],r['Test']) for r in a]==[(r['Package'],r['Test']) for r in b],'ordered tests')
   x=[r['Evidence'] for r in a];y=[r['Evidence'] for r in b]
   if marker=='CONFORMANCE': result=c.compare_conformance(x,y,passed)
   else:c.require(x==y,'exact ordered semantic payloads');result={'Marker':marker,'Records':count,'ExactOrderedPayloads':True}
   comparisons.append({'Mode':mode,'Baseline':label,'Comparison':result,'Scope':'Published historical payload reproduction, with current-main source equivalence established separately by identity.json.'})
 save('historical-comparisons.json',comparisons)
commands=[]
for cmd in [[sys.executable,'-m','unittest','discover','-s','docs/foundational/phase1e/tools','-p','test_comparison.py','-v'],[sys.executable,'-m','unittest','discover','-s','docs/foundational/phase1f/tools','-p','test_*.py','-v'],['go','vet','./...'],['go','vet','-tags','integration','./...'],['gofmt','-l','author','schema','runtime','effects','adapters','cmd','internal','research','examples'],['go','mod','verify']]:
 start=time.monotonic();p=subprocess.run(cmd,cwd=frozen,env=env,capture_output=True);commands.append({'Command':cmd,'Directory':str(frozen),'ExitCode':p.returncode,'ElapsedSeconds':round(time.monotonic()-start,3),'Stdout':p.stdout.decode(),'Stderr':p.stderr.decode()});save('auxiliary.json',commands);assert p.returncode==0,commands[-1]
 if cmd[0]=='gofmt':assert not p.stdout,commands[-1]
 print('auxiliary',cmd,'PASS',flush=True)
before=json.loads((out/'prior-artifacts-before.json').read_text());after={p:sha((repo/p).read_bytes()) for p in before};save('prior-artifacts-after.json',after);assert before==after
summary={'Result':'PASS','StartingGitCommit':identity['GitCommit'],'SourceID':identity['FrozenSource']['SourceSHA256'],'Modes':{mode:json.loads((out/f'final-{mode}.command.json').read_text()) for mode in ('default','normal')},'HistoricalComparisons':len(comparisons),'PriorArtifactsUnchanged':True,'PriorArtifactFiles':len(before),'Race':'Deferred to final full current-source verification','ProviderCalls':'None; credential removed; controlled loopback only'};save('summary.json',summary);print('COMPLETE',json.dumps({k:v for k,v in summary.items() if k!='Modes'}),flush=True)
