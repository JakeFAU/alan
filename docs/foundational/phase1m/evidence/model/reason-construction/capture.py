"""Source-bound construction checks only; never a product or local census."""
import gzip, hashlib, json, os, subprocess, sys, time
from pathlib import Path
ROOT = Path(__file__).resolve().parents[6]
OUT = Path(__file__).resolve().parent
ENV = {k: v for k, v in os.environ.items() if not k.startswith('ALAN_') and k != 'OPENAI_API_KEY'}
PINNED = dict(GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off', GOFLAGS='-mod=readonly', CGO_ENABLED='1', GOCACHE='/private/tmp/alan-phase1l-go-build')
ENV.update(PINNED)
# Deliberately excludes all *Qualification tests. No consumer source is read.
SELECTOR = '^(TestConsumerReason.*|TestJoin.*|TestConsumerPolicy(SFirstOracle|RejectsCandidateFirst|FirstSelectionTransition|PendingPrefix|OracleRequiresOriginalJoins|HonestFailureIsNotConformance|AbsentPayloadAndFailedS|DirectedReferenceBranches|DomainAndStateGuards))$'
COMMANDS = [
 ['/opt/homebrew/bin/go','version'],
 ['/opt/homebrew/bin/go','test','-list',SELECTOR,'./research/substitution'],
 ['/opt/homebrew/bin/go','test','-json','-count=1','-timeout=2m','-run',SELECTOR,'./research/substitution'],
 ['/opt/homebrew/bin/go','test','-json','-race','-count=1','-timeout=3m','-run',SELECTOR,'./research/substitution'],
 ['/opt/homebrew/bin/go','vet','./research/substitution'],
 ['/opt/homebrew/bin/gofmt','-l',*[str(p.relative_to(ROOT)) for p in sorted((ROOT/'research/substitution').glob('consumer_reason*.go'))]],
 ['git','diff','--check'],
]
def sources():
 paths = sorted((ROOT/'research/substitution').glob('*.go'))
 paths += [ROOT/p for p in ['go.mod','go.sum','docs/foundational/phase1m/01-contract.md','docs/foundational/phase1m/01-reason-contract.md','docs/foundational/phase1m/assignment.md','docs/foundational/phase1i/01-substitution-contract.md']]
 return {str(p.relative_to(ROOT)):hashlib.sha256(p.read_bytes()).hexdigest() for p in paths}
def main():
 target=OUT/'construction.json.gz'
 if target.exists(): raise SystemExit('refusing to overwrite final construction record')
 hashes=sources(); records=[]
 for argv in COMMANDS:
  start=time.monotonic(); p=subprocess.run(argv,cwd=ROOT,env=ENV,text=True,capture_output=True)
  events=[]
  for line in p.stdout.splitlines():
   try: event=json.loads(line)
   except ValueError: continue
   if isinstance(event,dict): events.append(event)
  runs=[e['Test'] for e in events if e.get('Action')=='run' and 'Test' in e]
  leaves=[n for n in runs if not any(m.startswith(n+'/') for m in runs)]
  record=dict(argv=argv,cwd=str(ROOT),exit=p.returncode,seconds=time.monotonic()-start,stdout=p.stdout,stderr=p.stderr,selected_leaves=leaves,test_results=[{k:e[k] for k in ('Test','Action')} for e in events if 'Test' in e and e.get('Action') in ('pass','fail','skip')])
  records.append(record)
  with gzip.open(target,'wt') as f: json.dump(dict(environment=PINNED,removed_environment=['ALAN_*','OPENAI_API_KEY'],source_sha256=hashes,commands=records),f,indent=2)
  print(json.dumps(dict(argv=argv,exit=p.returncode,leaves=len(leaves),seconds=record['seconds'])),flush=True)
  if p.returncode or argv[0].endswith('gofmt') and p.stdout.strip():
   print(p.stdout);print(p.stderr);raise SystemExit(1)
 if hashes!=sources(): raise SystemExit('source changed during checks')
if __name__=='__main__':main()
