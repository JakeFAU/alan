import os,sys,subprocess,json,hashlib,pathlib,datetime
root=pathlib.Path(__file__).resolve().parents[6]
label=sys.argv[1];cmd=sys.argv[2:];base=root/'docs/foundational/phase1k/evidence/runtime/revision2';checks=base/'checks';checks.mkdir(parents=True,exist_ok=True)
env=os.environ.copy();env.pop('OPENAI_API_KEY',None)
env.update(GOTOOLCHAIN='local',GOPROXY='off',GOSUMDB='off',GOFLAGS='-mod=readonly',CGO_ENABLED='1',GOCACHE='/private/tmp/alan-phase1k-runtime-cache',ALAN_I_ARTIFACTS=str(base/label))
if label.startswith('red-send'):env['ALAN_K_RED']='send-is-enough'
else:env.pop('ALAN_K_RED',None)
paths=sorted(root.glob('internal/substitutiontest/communication_*_test.go'))+[root/'research/communication/oracle.go']+sorted((base.parent/'public-consumer').glob('*'))
record={'label':label,'utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'cwd':str(root),'command':cmd,'environment':{k:env[k] for k in ['GOTOOLCHAIN','GOPROXY','GOSUMDB','GOFLAGS','CGO_ENABLED','GOCACHE','ALAN_I_ARTIFACTS']},'provider_credential_absent':'OPENAI_API_KEY' not in env,'source_sha256':{str(p.relative_to(root)):hashlib.sha256(p.read_bytes()).hexdigest() for p in paths}}
if 'ALAN_K_RED'in env:record['environment']['ALAN_K_RED']=env['ALAN_K_RED']
with open(checks/(label+'.stdout'),'wb') as out,open(checks/(label+'.stderr'),'wb') as err:r=subprocess.run(cmd,cwd=root,env=env,stdout=out,stderr=err)
record['exit']=r.returncode
for suffix in ['stdout','stderr']:record[suffix+'_sha256']=hashlib.sha256((checks/(label+'.'+suffix)).read_bytes()).hexdigest()
(checks/(label+'.json')).write_text(json.dumps(record,indent=2)+'\n')
print(json.dumps({'label':label,'exit':r.returncode,'stdout':str(checks/(label+'.stdout'))}))
print((checks/(label+'.stdout')).read_text()[-4000:]);print((checks/(label+'.stderr')).read_text()[-4000:])
sys.exit(r.returncode)
