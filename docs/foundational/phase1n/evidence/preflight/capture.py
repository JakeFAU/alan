import os,sys,subprocess,pathlib,json,gzip,hashlib
root=pathlib.Path(__file__).resolve().parents[5]
label,cwd,*argv=sys.argv[1:]
env={k:v for k,v in os.environ.items() if not k.startswith('ALAN_') and k!='OPENAI_API_KEY'}
fixed=dict(GOTOOLCHAIN='local',GOPROXY='off',GOSUMDB='off',GOFLAGS='-mod=readonly',CGO_ENABLED='1',GOCACHE='/private/tmp/alan-phase1l-go-build')
env.update(fixed)
sample_dir=root/'docs/foundational/phase1n/evidence/preflight'/label
sample_dir.mkdir(exist_ok=True)
env['ALAN_CALLABLE_PREFLIGHT_DIR']=str(sample_dir)
version=subprocess.check_output(['go','version'],env=env,text=True).strip()
if version!='go version go1.27.3 darwin/arm64': raise SystemExit('unexpected toolchain: '+version)
effective=json.loads(subprocess.check_output(['go','env','-json','GOVERSION','GOOS','GOARCH','CC','GOTOOLCHAIN','GOPROXY','GOSUMDB','GOFLAGS','CGO_ENABLED','GOCACHE','GOMODCACHE'],cwd=root/cwd,env=env))
sqlite=subprocess.check_output(['go','list','-m','github.com/mattn/go-sqlite3'],cwd=root/cwd,env=env,text=True).strip()
if sqlite!='github.com/mattn/go-sqlite3 v1.14.52': raise SystemExit('unexpected sqlite: '+sqlite)
p=subprocess.run(argv,cwd=root/cwd,env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
files=list((root/'examples/callablejoin').rglob('*.go'))+list((root/'examples/callablejoin').rglob('go.mod'))+list((root/'examples/callablejoin').rglob('go.sum'))
record=dict(argv=argv,cwd=cwd,environment=fixed,effective_go_environment=effective,toolchain=version,sqlite=sqlite,environment_exclusion=['ALAN_*','OPENAI_API_KEY'],evidence_override=env['ALAN_CALLABLE_PREFLIGHT_DIR'],exit=p.returncode,output=p.stdout.decode(),sources={str(f.relative_to(root)):hashlib.sha256(f.read_bytes()).hexdigest() for f in files})
events=[]
for line in p.stdout.decode().splitlines():
 try: event=json.loads(line)
 except ValueError: continue
 if isinstance(event,dict) and event.get('Test') and event.get('Action') in ['pass','fail','skip']: events.append(event)
record['test_leaves']=[e for e in events if not any(o['Test'].startswith(e['Test']+'/') for o in events)]
if label=='red': record['source_witness']={str(f.relative_to(root)):f.read_text() for f in files}
with gzip.GzipFile(str(root/'docs/foundational/phase1n/evidence/preflight'/f'{label}.json.gz'),'wb',mtime=0) as z:z.write(json.dumps(record,sort_keys=True,indent=2).encode())
print(p.stdout.decode());print('exit:',p.returncode)
sys.exit(p.returncode)
