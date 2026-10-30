"""Focused, source-bound reused-boundary gates; no historical campaign rerun."""
import gzip, hashlib, json, os, subprocess, time
from pathlib import Path
ROOT = Path(__file__).resolve().parents[5]
OUT = Path(__file__).resolve().parent
ENV = {k: v for k, v in os.environ.items() if not k.startswith('ALAN_') and k != 'OPENAI_API_KEY'}
ENV.update(GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off', GOFLAGS='-mod=readonly', GOCACHE='/private/tmp/alan-phase1l-go-build')
SELECTOR = '^(TestInvocation.*|TestCompositionRootJoinWaitsAndPublishes|TestCompositionNestedAutomaticRunAndConsumption|TestCompositionFailedChildCannotHideUnresolvedGrandchild|TestCompositionNestedCapacityOneExecutesLeafTransport|TestCompositionReopenPreservesAdmissionBudget)$'
COMMANDS = [
 ['go','test','-json','-count=1','-timeout=10m','./author','./schema','./compose'],
 ['go','test','-json','-count=1','-race','-timeout=15m','./author','./schema','./compose'],
 ['go','test','-json','-count=1','-tags','integration','-run',SELECTOR,'-timeout=10m','./internal/executor'],
 ['go','test','-json','-count=1','-race','-tags','integration','-run',SELECTOR,'-timeout=15m','./internal/executor'],
 ['go','mod','verify'],
]
def sources():
 paths = subprocess.check_output(['git','ls-files'],cwd=ROOT,text=True).splitlines()
 return {p:hashlib.sha256((ROOT/p).read_bytes()).hexdigest() for p in paths if p.endswith('.go') or p in ('go.mod','go.sum')}
def main():
 records=[]; hashes=sources()
 for argv in COMMANDS:
  start=time.monotonic(); p=subprocess.run(argv,cwd=ROOT,env=ENV,text=True,capture_output=True)
  tests=[]
  for line in p.stdout.splitlines():
   try: item=json.loads(line)
   except ValueError: continue
   if item.get('Action') in ('pass','fail','skip') and 'Test' in item: tests.append({k:item[k] for k in ('Package','Test','Action')})
  record=dict(argv=argv,cwd=str(ROOT),exit=p.returncode,seconds=time.monotonic()-start,stdout=p.stdout,stderr=p.stderr,test_results=tests)
  records.append(record)
  with gzip.open(OUT/'checks.json.gz','wt') as f: json.dump(dict(env={k:ENV[k] for k in ('GOTOOLCHAIN','GOPROXY','GOSUMDB','GOFLAGS','GOCACHE')},source_sha256=hashes,commands=records),f,indent=2)
  print(json.dumps(dict(argv=argv,exit=p.returncode,seconds=record['seconds'],test_results=len(tests))),flush=True)
  if p.returncode: raise SystemExit(p.returncode)
 print('focused reused-boundary gates complete',flush=True)
if __name__=='__main__':main()
