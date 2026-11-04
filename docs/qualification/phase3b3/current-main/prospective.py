from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime,timezone
import hashlib,io,json,os,subprocess,tarfile,time
repo=Path('/private/tmp/alan-phase3b3-runtime');root=Path('/tmp/alan-phase3b3-prospective');root.mkdir()
src=root/'source';src.mkdir();out=root/'evidence';out.mkdir()
main='76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315';head='c7fe6d87cdd49878558d676a301bec2d217ae3a6';tree='bdbbae5f89f0c76d48618b3d8891dc40199ca4ed'
assert subprocess.check_output(['git','rev-parse','HEAD'],cwd=repo,text=True).strip()==head
raw=subprocess.check_output(['git','archive',tree],cwd=repo)
with tarfile.open(fileobj=io.BytesIO(raw)) as t:
 for m in t.getmembers():assert not Path(m.name).is_absolute() and '..' not in Path(m.name).parts and not m.issym() and not m.islnk(),m.name
 t.extractall(src)
paths=[p for p in src.rglob('*') if p.is_file() and (p.suffix=='.go' or p.name in ('go.mod','go.sum'))]
h=lambda p:hashlib.sha256(p.read_bytes()).hexdigest(); hashes={str(p.relative_to(src)):h(p) for p in paths}
env=os.environ.copy();env.update(GOTOOLCHAIN='local',GOPROXY='off',GOSUMDB='off',GOCACHE='/tmp/alan-phase3b3-prospective-build',STATICCHECK_CACHE='/tmp/alan-phase3b3-staticcheck',ALAN_PHASE3B3_OLD_READER='/tmp/alan-phase3b3-old-reader/alan')
(out/'source.json').write_text(json.dumps({'Purpose':'Prospective integration only, no branch or PR merged; not the frozen original experiment','MainCommit':main,'TaskCommit':head,'MergeTree':tree,'SourceFiles':hashes,'Go':subprocess.check_output(['go','version'],env=env,text=True).strip(),'RunnerSHA256':h(Path(__file__))},indent=2)+'\n')
tasks=[('core-integration',['go','test','-json','-count=1','-tags=integration','./internal/executor','./runtime','./author','./examples/delegation'],{}),('authority-mode-race',['go','test','-json','-count=1','-race','-tags=integration','./internal/executor','./runtime','./author','-run','^Test(RequestAuthority|InvocationMode)'],{}),('reciprocal-race',['go','test','-json','-count=1','-race','-tags=integration','./internal/durabletest','-run','^TestPhase3B3(Reciprocal|CartesianOperations|GenericUnknown|ActualResultLaundering|SerialSerial|AppliedUnknownRevoked|RetainedMutations)'],{'ALAN_PHASE3B3_EVIDENCE':str(out/'reciprocal-race')}),('staticcheck-integration',['staticcheck','-tags=integration','./...'],{})]
def run(t):
 name,cmd,extra=t;local=env.copy();local.update(extra);start=datetime.now(timezone.utc).isoformat();tick=time.monotonic();log=out/(name+'.log')
 with log.open('wb') as f:r=subprocess.run(cmd,cwd=src,env=local,stdout=f,stderr=subprocess.STDOUT)
 ev=[]
 if cmd[:2]==['go','test']:
  for line in log.read_text().splitlines():
   try:ev.append(json.loads(line))
   except json.JSONDecodeError:pass
 rec={'Name':name,'Command':cmd,'Cwd':str(src),'Environment':{k:local[k] for k in ['GOTOOLCHAIN','GOPROXY','GOSUMDB','GOCACHE','STATICCHECK_CACHE','ALAN_PHASE3B3_OLD_READER']},'AdditionalEnvironment':extra,'StartedUTC':start,'Seconds':time.monotonic()-tick,'Exit':r.returncode,'Log':log.name,'SHA256':h(log),'PassEvents':sum(e.get('Action')=='pass' and 'Test' in e for e in ev),'Failures':[e for e in ev if e.get('Action')=='fail'],'Skips':[e for e in ev if e.get('Action')=='skip']}
 (out/(name+'-command.json')).write_text(json.dumps(rec,indent=2)+'\n');print(json.dumps({k:rec[k] for k in ['Name','Exit','Seconds','PassEvents']}),flush=True);return rec
with ThreadPoolExecutor(max_workers=2) as pool:records=list(pool.map(run,tasks))
assert hashes=={str(p.relative_to(src)):h(p) for p in paths}
(out/'commands.json').write_text(json.dumps(records,indent=2)+'\n')
assert all(r['Exit']==0 for r in records)
print('prospective integration passed; no branch merged',flush=True)
