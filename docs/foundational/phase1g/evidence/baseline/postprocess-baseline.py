from pathlib import Path
import collections,datetime,gzip,hashlib,importlib.util,json,os,shutil,subprocess,sys,time,traceback
repo=Path('/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan');out=repo/'docs/foundational/phase1g/evidence/baseline';identity=json.loads((out/'identity.json').read_text());frozen=Path(identity['FrozenDirectory']);doc=frozen/'docs/foundational/phase1f'
env=dict(os.environ);env.pop('OPENAI_API_KEY',None);env.pop('ALAN_BARRIER',None);env.update(GOTOOLCHAIN='local',GOFLAGS='-mod=readonly',GOCACHE='/tmp/alan-phase1f-go-build',GOPROXY='off',GOSUMDB='off',PYTHONDONTWRITEBYTECODE='1')
def sha(b):return hashlib.sha256(b).hexdigest()
def save(n,v):(out/n).write_text(json.dumps(v,indent=2)+'\n')
original=(out/'run-baseline.py').read_text();tail=original[original.index('# Preserve the published collector'):]
# Reproduce the post-collection failure without rerunning passing suites; retain original wrapper.
try:exec(compile(tail,str(out/'run-baseline.py'),'exec'),globals())
except FileNotFoundError:
 failure=traceback.format_exc();(out/'postprocess-original-error.log.gz').write_bytes(gzip.compress(failure.encode(),mtime=0));print(failure,flush=True)
 save('postprocess-correction.json',{'Failure':'Published Phase1F generated qualification logs are intentionally absent from the source archive; original wrapper addressed them under the frozen tree.','FailureEvidence':'postprocess-original-error.log.gz','EvidenceKind':'Exact post-collection failure reproduced from unchanged original wrapper tail; original passing Go logs retained without rerun.','Correction':'Read immutable published Phase1F record logs from repository docs/foundational/phase1f/evidence; prior-artifact SHA256 checks bind these bytes.','SourceOrTestsChanged':False})
else:raise AssertionError('Expected retained postprocessing path failure')
corrected=tail.replace("published=c.rows(doc/f'evidence/final-{mode}.records.jsonl.gz')","published=c.rows(repo/f'docs/foundational/phase1f/evidence/final-{mode}.records.jsonl.gz')")
shutil.copyfile(__file__,out/'postprocess-baseline.py')
exec(compile(corrected,str(out/'run-baseline.py')+' (corrected historical evidence path)','exec'),globals())
