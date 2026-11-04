from pathlib import Path
from datetime import datetime,timezone
import hashlib,json,os,subprocess,time
root=Path('/tmp/alan-phase3b3-prospective');src=root/'source';out=root/'evidence';prior=json.loads((out/'commands.json').read_text())[2]
h=lambda p:hashlib.sha256(p.read_bytes()).hexdigest();expected=json.loads((out/'source.json').read_text())['SourceFiles']
assert expected=={p:h(src/p) for p in expected}
env=os.environ.copy();env.update(prior['Environment']);env['ALAN_PHASE3B3_EVIDENCE']=str(out/'reciprocal-race-corrected')
log=out/'reciprocal-race-corrected.log';tick=time.monotonic();start=datetime.now(timezone.utc).isoformat()
with log.open('wb') as f:r=subprocess.run(prior['Command'],cwd=src,env=env,stdout=f,stderr=subprocess.STDOUT)
ev=[]
for line in log.read_text().splitlines():
 try:ev.append(json.loads(line))
 except json.JSONDecodeError:pass
rec={**prior,'Name':'reciprocal-race-corrected','AdditionalEnvironment':{'ALAN_PHASE3B3_EVIDENCE':env['ALAN_PHASE3B3_EVIDENCE']},'StartedUTC':start,'Seconds':time.monotonic()-tick,'Exit':r.returncode,'Log':log.name,'SHA256':h(log),'PassEvents':sum(e.get('Action')=='pass' and 'Test' in e for e in ev),'Failures':[e for e in ev if e.get('Action')=='fail'],'Skips':[e for e in ev if e.get('Action')=='skip'],'ScratchSource':json.loads((out/'scratch-source.json').read_text()),'RunnerSHA256':h(Path(__file__))}
(out/'reciprocal-race-corrected-command.json').write_text(json.dumps(rec,indent=2)+'\n')
assert expected=={p:h(src/p) for p in expected}
print(json.dumps({k:rec[k] for k in ['Name','Exit','Seconds','PassEvents']}),flush=True)
assert r.returncode==0
