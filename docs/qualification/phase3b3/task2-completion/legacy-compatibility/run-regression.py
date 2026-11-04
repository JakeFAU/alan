import subprocess,pathlib,os,json,sys,datetime,hashlib
out=pathlib.Path(__file__).parent
cwd=pathlib.Path(sys.argv[2]); label=sys.argv[1]
cmd=['go','test','./internal/executor','-run','^TestRequestAuthorityLegacyUnknownOperationCompatibility$','-count=1','-v']
env=os.environ.copy(); explicit=dict(GOTOOLCHAIN='local',GOCACHE='/tmp/alan-phase3a-go-build',GOPROXY='off',GOSUMDB='off');env.update(explicit)
p=subprocess.run(cmd,cwd=cwd,env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
(out/(label+'.log')).write_bytes(p.stdout)
record=dict(Command=cmd,CWD=str(cwd),Environment=explicit,ExitCode=p.returncode,TimeUTC=datetime.datetime.now(datetime.timezone.utc).isoformat(),TestSHA256=hashlib.sha256((cwd/'internal/executor/request_authority_legacy_test.go').read_bytes()).hexdigest())
(out/(label+'.json')).write_text(json.dumps(record,indent=2)+'\n');print(json.dumps(record));print(p.stdout.decode())
