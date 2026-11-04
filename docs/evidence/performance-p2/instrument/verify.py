#!/usr/bin/env python3
"""Run focused proofs serially in a fresh disposable SQLite copy."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--sqlite',type=Path,required=True,help='generated instrumented SQLite copy')
    p.add_argument('--out',type=Path,required=True,help='new proof output directory')
    a=p.parse_args();here=Path(__file__).resolve().parent
    if a.out.exists():raise SystemExit('output must be new')
    a.out.mkdir(parents=True)
    sqlite=a.out/'sqlite';shutil.copytree(a.sqlite,sqlite)
    (sqlite/'alan_measurement_proof_test.go').write_bytes((here/'proof_test.go.txt').read_bytes())
    env=dict(os.environ,GOWORK='off',GOTOOLCHAIN='local',GOPROXY='off',GOSUMDB='off',GOMAXPROCS='4')
    commands=[]
    for label,race in [('actual-green',[]),('actual-race-green',['-race'])]:
        command=['go','test',*race,'-count=1','-run','^TestMeasurement','-v']
        with (a.out/(label+'.log')).open('w') as log:
            result=subprocess.run(command,cwd=sqlite,env=env,stdout=log,stderr=subprocess.STDOUT)
        commands.append({'command':command,'returncode':result.returncode})
        if result.returncode:raise SystemExit(label+' failed; see preserved log')
    mutant=a.out/'synthetic-mutant';mutant.mkdir()
    lines=(here/'sqlite_metrics.go.txt').read_text().splitlines(True)
    for i,line in enumerate(lines):
        if 'regexp.MustCompile' in line:lines[i]=line.replace('\\','\\\\')
    (mutant/'metrics.go').write_text(''.join(lines))
    tests=(here/'proof_test.go.txt').read_text().split('func measurementReset')[0]
    tests=re.sub(r'import \(.*?\)', 'import "testing"',tests,flags=re.S)
    (mutant/'normalize_test.go').write_text(tests)
    (mutant/'go.mod').write_text('module example.org/measurement-mutant\n\ngo 1.27.3\n')
    command=['go','test','-count=1','-run','^TestMeasurementNormalize$','-v']
    with (a.out/'synthetic-mutant-red.log').open('w') as log:
        result=subprocess.run(command,cwd=mutant,env=env,stdout=log,stderr=subprocess.STDOUT)
    commands.append({'command':command,'returncode':result.returncode,'expected':'failure in TestMeasurementNormalize'})
    if result.returncode==0:raise SystemExit('synthetic double-escape mutant unexpectedly passed')
    if '--- FAIL: TestMeasurementNormalize' not in (a.out/'synthetic-mutant-red.log').read_text():raise SystemExit('mutant failed for an unrelated reason')
    manifest=[{'path':str(p.relative_to(here)),'sha256':hashlib.sha256(p.read_bytes()).hexdigest()} for p in sorted(here.iterdir()) if p.is_file()]
    (a.out/'proof-commands.json').write_text(json.dumps({'commands':commands,'tool_manifest':manifest},indent=2)+'\n')
    print('Focused ordinary/race proofs passed; synthetic normalizer mutant rejected. Workload reconciliation remains separate.')

if __name__=='__main__':main()
