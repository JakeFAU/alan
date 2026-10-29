#!/usr/bin/env python3
"""Scoped RED/GREEN for the two accepted raw-evidence review blockers."""
from pathlib import Path
import gzip,hashlib,json,os,subprocess,sys,tempfile,time,datetime
ROOT=Path(__file__).resolve().parents[6];DEST=Path(__file__).resolve().parent
label=sys.argv[1];expected=1 if label=='red' else 0
adapter=ROOT/'internal/substitutiontest/invocation_join_evidence_test.go'
frozen=json.loads(gzip.decompress((DEST.parent/'rejected-r1/review-freeze.json.gz').read_bytes()))['files'][str(adapter.relative_to(ROOT))]
settings=dict(GOTOOLCHAIN='local',GOPROXY='off',GOSUMDB='off',GOFLAGS='-mod=readonly',GOCACHE='/tmp/alan-phase1f-go-build')
with tempfile.TemporaryDirectory(prefix='alan-phase1l-task3-fix1-',dir='/private/tmp') as tmp:
 tmp=Path(tmp);artifacts=tmp/'artifacts';artifacts.mkdir();settings['ALAN_I_ARTIFACTS']=str(artifacts)
 argv=['go','test','-json','-tags','integration','-run','^TestInvocationJoinDecoder$/(coherent-revision|orphan-intent|unadmitted-revision)$','-count=1','-timeout=2m','./internal/substitutiontest']
 if label=='red':
  prior=tmp/'rejected_adapter_test.go';prior.write_text(frozen);overlay=tmp/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(adapter):str(prior)}}));argv.insert(2,'-overlay='+str(overlay))
 env={k:v for k,v in os.environ.items() if not k.startswith('ALAN_') and k!='OPENAI_API_KEY'};env.update(settings)
 start=time.monotonic();p=subprocess.run(argv,cwd=ROOT,env=env,capture_output=True,text=True)
 events=[]
 for line in p.stdout.splitlines():
  try:events.append(json.loads(line))
  except ValueError:pass
 selected=[e['Test'] for e in events if e.get('Action')=='run' and 'Test' in e]
 failures=[e['Test'] for e in events if e.get('Action')=='fail' and 'Test' in e]
 witnesses={str(f.relative_to(artifacts)):json.loads(gzip.decompress(f.read_bytes())) for f in artifacts.rglob('*.json.gz')}
 record=dict(label=label,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),argv=argv,environment_override=settings,exit_code=p.returncode,expected_exit=expected,seconds=time.monotonic()-start,selected_cases=selected,failed_cases=failures,stdout=p.stdout,stderr=p.stderr,adapter_source_sha256=hashlib.sha256((frozen if label=='red' else adapter.read_text()).encode()).hexdigest(),decoder_source_sha256=hashlib.sha256((ROOT/'internal/substitutiontest/invocation_join_decoder_test.go').read_bytes()).hexdigest(),witnesses=witnesses)
 (DEST/(label+'.json.gz')).write_bytes(gzip.compress(json.dumps(record,separators=(',',':')).encode(),mtime=0))
 assert p.returncode==expected,p.stdout+p.stderr
 if label=='red':assert len([f for f in failures if '/' in f])==3 and sum('accepted-corruption' in k for k in witnesses)==3,p.stdout
 else:assert not failures,p.stdout
 print(label,'exit',p.returncode,'expected',expected,'selected',selected,'failures',failures)
