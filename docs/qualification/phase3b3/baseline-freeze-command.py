from pathlib import Path
import subprocess,hashlib,json,os,time,datetime
root=Path('/private/tmp/alan-phase3b3-runtime')
out=Path('/tmp/alan-phase3b3-baseline-frozen');out.mkdir(exist_ok=True)
head=subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip()
assert head.startswith('8319a90')
assert not subprocess.check_output(['git','status','--porcelain'],cwd=root).strip()
files=subprocess.check_output(['git','ls-files','*.go','go.mod','go.sum'],cwd=root,text=True).splitlines()
manifest=[]
for name in files:
 b=(root/name).read_bytes();manifest.append({'path':name,'bytes':len(b),'sha256':hashlib.sha256(b).hexdigest()})
(out/'source.json').write_text(json.dumps({'Commit':head,'ProductionBase':'af6943a5805a9a6c0e8fe77f804264f4765d07ec','Files':manifest},indent=2)+'\n')
env=os.environ.copy();env.update(GOTOOLCHAIN='local',GOCACHE='/tmp/alan-phase3a-go-build',GOPROXY='off',GOSUMDB='off',STATICCHECK_CACHE='/tmp/alan-phase3b3-staticcheck')
commands=[('normal',['go','test','-tags=integration','./internal/durabletest','-run','^TestPhase3B3Baseline','-count=1','-v'],0,{}),('race',['go','test','-race','-tags=integration','./internal/durabletest','-run','^TestPhase3B3Baseline','-count=1','-v'],0,{}),('expected-s2-failure',['go','test','-tags=integration','./internal/durabletest','-run','^TestPhase3B3BaselineS2CompletionExpectedFailure$','-count=1','-v'],1,{'ALAN_PHASE3B3_EXPECT_S2_COMPLETION':'1'})]
records=[]
for label,args,want,extra in commands:
 e=env.copy();e.update(extra);e['ALAN_PHASE3B3_EVIDENCE']=str(out/label)
 start=time.monotonic();utc=datetime.datetime.now(datetime.timezone.utc).isoformat()
 with (out/(label+'.log')).open('wb') as log:
  p=subprocess.run(args,cwd=root,env=e,stdout=log,stderr=subprocess.STDOUT)
 raw=(out/(label+'.log')).read_bytes()
 rec={'Label':label,'Command':args,'CWD':str(root),'SourceCommit':head,'StartedUTC':utc,'ElapsedSeconds':time.monotonic()-start,'ExitCode':p.returncode,'ExpectedExitCode':want,'Environment':{k:e[k] for k in ['GOTOOLCHAIN','GOCACHE','GOPROXY','GOSUMDB','STATICCHECK_CACHE','ALAN_PHASE3B3_EVIDENCE']},'LogSHA256':hashlib.sha256(raw).hexdigest()}
 rec['Environment'].update(extra);records.append(rec)
 (out/(label+'.json')).write_text(json.dumps(rec,indent=2)+'\n')
 print(label,'exit',p.returncode,'expected',want,flush=True)
 assert p.returncode==want,raw[-4000:].decode(errors='replace')
 if want==1:assert b'EXPECTED FAIL: desired S2 completion unavailable' in raw
for r in manifest:
 b=(root/r['path']).read_bytes();assert len(b)==r['bytes'] and hashlib.sha256(b).hexdigest()==r['sha256'],r['path']
checks=0
for p in out.rglob('digests.json'):
 for name,digest in json.loads(p.read_text()).items():
  assert hashlib.sha256((p.parent/name).read_bytes()).hexdigest()==digest,(p,name)
  checks+=1
(out/'commands.json').write_text(json.dumps(records,indent=2)+'\n')
print('Verified',checks,'untouched regenerated artifact digests and',len(manifest),'frozen source inputs.',flush=True)
