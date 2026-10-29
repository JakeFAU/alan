from pathlib import Path
import datetime,hashlib,json,os,re,subprocess,sys,time
root=Path('/Users/jacob/.codex/worktrees/phase1j-retained-joins/alan')
out=root/'docs/foundational/phase1l/evidence/model/task2b'
out.mkdir(parents=True,exist_ok=True)
label,evidence=sys.argv[1:3]; argv=sys.argv[3:]
env=os.environ.copy()
for key in ['OPENAI_API_KEY','ALAN_MODEL_EVIDENCE']:env.pop(key,None)
settings={'GOTOOLCHAIN':'local','GOPROXY':'off','GOSUMDB':'off','GOFLAGS':'-mod=readonly','GOCACHE':'/tmp/alan-phase1f-go-build'}
if evidence!='-':settings['ALAN_MODEL_EVIDENCE']=evidence
if evidence!='-':Path(evidence).mkdir(parents=True,exist_ok=True)
env.update(settings)
start=time.monotonic();p=subprocess.run(argv,cwd=root,env=env,text=True,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
selected=re.findall(r'^=== RUN\s+(.+)$',p.stdout,re.M)
record={'label':label,'utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'cwd':str(root),'argv':argv,'environment_unset':['OPENAI_API_KEY','ALAN_MODEL_EVIDENCE'],'environment_override':settings,'exit_code':p.returncode,'duration_seconds':round(time.monotonic()-start,6),'selected_cases':selected,'selected_leaves':[case for case in selected if not any(other.startswith(case+'/') for other in selected)],'test_results':re.findall(r'^\s*--- (PASS|FAIL|SKIP): ([^ ]+) \(([^)]+)\)$',p.stdout,re.M),'output':p.stdout,'source_sha256':{str(f.relative_to(root)):hashlib.sha256(f.read_bytes()).hexdigest() for f in sorted((root/'research/substitution').glob('*.go'))}}
with (out/(label+'.json')).open('x') as f:json.dump(record,f,indent=2);f.write('\n')
print(p.stdout,end='');print('Recorded:',out/(label+'.json'))
sys.exit(p.returncode)
