#!/usr/bin/env python3
"""Run focused proofs serially in fresh copies, never editing generated inputs."""
import argparse, pathlib, shutil, subprocess, os
p=argparse.ArgumentParser()
p.add_argument('--sqlite',type=pathlib.Path,required=True,help='generated instrumented SQLite copy')
p.add_argument('--out',type=pathlib.Path,required=True,help='new scratch output directory')
a=p.parse_args();here=pathlib.Path(__file__).resolve().parent
if a.out.exists():raise SystemExit('output must be new')
a.out.mkdir(parents=True)
sqlite=a.out/'sqlite';shutil.copytree(a.sqlite,sqlite)
(sqlite/'alan_measurement_proof_test.go').write_bytes((here/'proof_test.go.txt').read_bytes())
env=dict(os.environ,GOWORK='off',GOTOOLCHAIN='local',GOPROXY='off',GOSUMDB='off',GOMAXPROCS='4')
with (a.out/'actual-green.log').open('w') as log:
 result=subprocess.run(['go','test','-count=1','-run','^TestMeasurement(Normalize|SQLStepAccounting)$','-v'],cwd=sqlite,env=env,stdout=log,stderr=subprocess.STDOUT)
if result.returncode:raise SystemExit('actual source proof failed; see actual-green.log')
mutant=a.out/'synthetic-mutant';mutant.mkdir()
lines=(here/'sqlite_metrics.go.txt').read_text().splitlines(True)
for i,line in enumerate(lines):
 if 'regexp.MustCompile' in line:lines[i]=line.replace('\\','\\\\')
(mutant/'metrics.go').write_text(''.join(lines))
t=(here/'proof_test.go.txt').read_text().split('func measurementReset')[0]
t=t.replace(' "context"\n','').replace(' "database/sql"\n','')
(mutant/'normalize_test.go').write_text(t)
(mutant/'go.mod').write_text('module example.org/measurement-mutant\n\ngo 1.27.3\n')
with (a.out/'synthetic-mutant-red.log').open('w') as log:
 result=subprocess.run(['go','test','-count=1','-run','^TestMeasurementNormalize$','-v'],cwd=mutant,env=env,stdout=log,stderr=subprocess.STDOUT)
if result.returncode==0:raise SystemExit('synthetic double-escape mutant unexpectedly passed')
if '--- FAIL: TestMeasurementNormalize' not in (a.out/'synthetic-mutant-red.log').read_text():raise SystemExit('mutant failed for an unrelated reason')
print('Actual source GREEN; deliberately double-escaped synthetic mutant RED.')
