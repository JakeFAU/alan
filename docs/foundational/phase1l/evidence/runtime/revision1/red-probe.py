#!/usr/bin/env python3
"""Expected RED through a temporary Go overlay; original worktree stays unchanged."""
from pathlib import Path
import datetime,gzip,hashlib,json,os,subprocess,tempfile,time
ROOT=Path(__file__).resolve().parents[6];DEST=Path(__file__).resolve().parent
adapter=ROOT/'internal/substitutiontest/invocation_join_evidence_test.go';test=ROOT/'internal/substitutiontest/invocation_join_decoder_test.go'
original_adapter=adapter.read_text();original_test=test.read_text()
probe=original_test.replace('"fact-order",','"extra-charge", "fact-order",')
needle='\n\t\t\tcase "fact-order":'
mutation='''
            case "extra-charge":
                copied:=map[string]any{};for key,value:=range s.Tables["composition_charges"][0]{copied[key]=value}
                copied["operation_id"]="uncommitted-extra-charge"
                charge:=decode[map[string]any](copied,"descriptor");charge["OperationID"]="uncommitted-extra-charge"
                b,e:=json.Marshal(charge);must(t,e);copied["descriptor"]=string(b)
                s.Tables["composition_charges"]=append(s.Tables["composition_charges"],copied)
                s.Roots[0].Charges.Invocations++;s.Roots[0].Charges.CreatedRuns++;changed=true
'''
assert needle in probe
probe=probe.replace(needle,mutation+needle).replace('t.Fatal("raw corruption accepted", name)','retain(t,"accepted-corruption",map[string]any{"mutation":name,"inspection":s,"decoded":decoded});t.Fatal("raw corruption accepted", name)')
settings=dict(GOTOOLCHAIN='local',GOPROXY='off',GOSUMDB='off',GOFLAGS='-mod=readonly',GOCACHE='/tmp/alan-phase1f-go-build')
with tempfile.TemporaryDirectory(prefix='alan-phase1l-charge-red-',dir='/private/tmp') as tmp:
    tmp=Path(tmp);overlaid=tmp/'probe_test.go';overlaid.write_text(probe);overlay=tmp/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(test):str(overlaid)}}));artifacts=tmp/'artifacts';artifacts.mkdir();settings['ALAN_I_ARTIFACTS']=str(artifacts)
    env={k:v for k,v in os.environ.items() if not k.startswith('ALAN_') and k!='OPENAI_API_KEY'};env.update(settings)
    argv=['go','test','-json','-overlay='+str(overlay),'-tags','integration','-run','^TestInvocationJoinDecoder$/extra-charge$','-count=1','./internal/substitutiontest']
    start=time.monotonic();p=subprocess.run(argv,cwd=ROOT,env=env,capture_output=True,text=True)
    assert adapter.read_text()==original_adapter and test.read_text()==original_test
    evidence={str(f.relative_to(artifacts)):json.loads(gzip.decompress(f.read_bytes())) for f in artifacts.rglob('*.json.gz')}
    assert p.returncode==1 and 'raw corruption accepted' in p.stdout and any('accepted-corruption' in k for k in evidence),p.stdout+p.stderr
    record=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),argv=argv,environment_override=settings,exit_code=p.returncode,expected_red=True,seconds=time.monotonic()-start,stdout=p.stdout,stderr=p.stderr,pre_correction_adapter=original_adapter,original_decoder_test=original_test,overlay_probe=probe,source_sha256={str(adapter.relative_to(ROOT)):hashlib.sha256(original_adapter.encode()).hexdigest(),str(test.relative_to(ROOT)):hashlib.sha256(original_test.encode()).hexdigest()},evidence=evidence)
    (DEST/'charge-row-red.json.gz').write_bytes(gzip.compress(json.dumps(record,separators=(',',':')).encode(),mtime=0))
    print('Expected RED: detached extra I/R charge accepted before coverage correction; worktree source unchanged.')
