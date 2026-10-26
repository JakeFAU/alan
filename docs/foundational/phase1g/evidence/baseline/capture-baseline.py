from pathlib import Path
import datetime,hashlib,json,os,platform,struct,subprocess,tarfile,tempfile
root=Path('/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan')
out=root/'docs/foundational/phase1g/evidence/baseline';out.mkdir(parents=True,exist_ok=True)
doc=root/'docs/foundational/phase1f'
def sha(b):return hashlib.sha256(b).hexdigest()
m=json.loads((doc/'source-manifest.json').read_text())
def verify(base):
 d=hashlib.sha256()
 for f in m['files']:
  b=(base/f['path']).read_bytes();n=f['path'].encode();assert len(b)==f['bytes'] and sha(b)==f['sha256'],f['path'];d.update(struct.pack('>Q',len(n))+n+struct.pack('>Q',len(b))+b)
 assert d.hexdigest()==m['source_sha256'];return {'Files':len(m['files']),'SourceSHA256':d.hexdigest()}
status=subprocess.check_output(['git','status','--porcelain'],cwd=root,text=True)
assert not any(line[3:].startswith(('author/','schema/','runtime/','effects/','adapters/','cmd/','internal/','research/','examples/','go.mod','go.sum')) for line in status.splitlines()),status
current=verify(root)
a=json.loads((doc/'archive-manifest.json').read_text());raw=(doc/'source.tar.gz').read_bytes();assert sha(raw)==a['SHA256']
frozen=Path(tempfile.mkdtemp(prefix='alan-phase1g-baseline-source-'))
with tarfile.open(doc/'source.tar.gz') as t:
 assert len(t.getmembers())==len(a['Files'])
 for f in a['Files']:
  member=t.getmember(f['Path']);b=t.extractfile(member).read();assert member.isfile() and not Path(member.name).is_absolute() and '..' not in Path(member.name).parts;assert len(b)==f['Bytes'] and sha(b)==f['SHA256']
 t.extractall(frozen,filter='data')
def cmd(args):
 p=subprocess.run(args,cwd=root,capture_output=True,text=True);return {'Command':args,'ExitCode':p.returncode,'Stdout':p.stdout,'Stderr':p.stderr}
prior={str(p.relative_to(root)):sha(p.read_bytes()) for folder in sorted((root/'docs/foundational').glob('phase1*')) if folder.name!='phase1g' for p in sorted(folder.rglob('*')) if p.is_file() and '__pycache__' not in p.parts}
(out/'prior-artifacts-before.json').write_text(json.dumps(prior,indent=2)+'\n')
identity={'CapturedUTC':datetime.datetime.now(datetime.timezone.utc).isoformat(),'GitCommit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip(),'GitTree':subprocess.check_output(['git','rev-parse','HEAD^{tree}'],cwd=root,text=True).strip(),'Branch':subprocess.check_output(['git','branch','--show-current'],cwd=root,text=True).strip(),'InitialStatus':status,'CurrentSource':current,'FrozenSource':verify(frozen),'FrozenDirectory':str(frozen),'ArchivePath':'docs/foundational/phase1f/source.tar.gz','ArchiveSHA256':a['SHA256'],'ArchiveVerifiedMembers':len(a['Files']),'Platform':platform.platform(),'GoVersion':cmd(['go','version']),'GoEnvironment':cmd(['go','env','-json','GOOS','GOARCH','CGO_ENABLED','CC','GOVERSION']),'CCompiler':cmd(['cc','--version']),'Method':'Starting main contributing sources verified byte-for-byte against published Phase1F archive; suites execute isolated frozen copy. Historical reproduction and current-main equivalence are separate claims.','CredentialPolicy':'OPENAI_API_KEY and ALAN_BARRIER removed before execution; no provider/live calls.'}
(out/'identity.json').write_text(json.dumps(identity,indent=2)+'\n')
(out/'published-source-manifest.json').write_bytes((doc/'source-manifest.json').read_bytes())
(out/'published-archive-manifest.json').write_bytes((doc/'archive-manifest.json').read_bytes())
print(json.dumps(identity,indent=2))
