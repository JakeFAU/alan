import pathlib,json,subprocess,hashlib,difflib
R=pathlib.Path('/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan');D=R/'docs/foundational/phase1h';T=D/'evidence/task3'
sha=lambda b:hashlib.sha256(b).hexdigest()
git=lambda *a:subprocess.check_output(['git',*a],cwd=R)
head=git('rev-parse','HEAD').decode().strip();assert head=='ed3006859dcdf075e92192b0722afc6f29ec2c43'
assert not git('status','--porcelain')
m=json.loads((D/'source-manifest.json').read_text());changes=git('diff','--name-only','564b4a4..HEAD').decode().splitlines()
assert not set(changes)&set(m['Files'])
for p,h in m['Files'].items():assert sha((R/p).read_bytes())==h,p
preserved={}
for old,new in [('report.md','report-checkpoints.md'),('controller-rulings-snapshot.md','controller-rulings-snapshot-before-race.md'),('controller-rulings-provenance.json','controller-rulings-provenance-before-race.json')]:
 b=git('show','564b4a4:'+str((T/old).relative_to(R)));assert b==(T/new).read_bytes();preserved[new]=sha(b)
review_files=0
for p in T.glob('final-review-*/provenance.json'):
 for v in json.loads(p.read_text())['Files']:
  b=(R/v['Retained']).read_bytes();assert len(b)==v['Bytes'] and sha(b)==v['SHA256']
  if pathlib.Path(v['Original']).exists():assert pathlib.Path(v['Original']).read_bytes()==b
  review_files+=1
p=T/'final-integrity'
for v in json.loads((p/'provenance.json').read_text())['Files']:
 b=(p/v['Path']).read_bytes();assert len(b)==v['Bytes'] and sha(b)==v['SHA256']
a=(p/'attempt1-inspection.py').read_text();b=(p/'inspection.py').read_text();assert a.replace('sha(gzip.decompress(b))','sha(gzip.decompress(b).removesuffix(b"\\n"))')==b
index=[json.loads(x) for x in (D/'qualification.jsonl').read_text().splitlines()];captures=list((T/'final').rglob('command.json'));assert len(index)==len(captures)==41
seen=set()
for v in index:
 c=json.loads((D/v['Metadata']).read_text());seen.add((D/v['Metadata']).resolve())
 for k in ['SourceID','HeadAtStart','Command','ExitCode','ElapsedSeconds','SourceUnchanged','Markers','RawSHA256','StderrSHA256','SourceArchiveSHA256']:assert v[k]==c.get(k),(v['Metadata'],k)
 assert v['GoPassEvents']==(len(c['passes']) if 'passes' in c else None)
assert seen=={x.resolve() for x in captures}
failures=[v['Capture'] for v in index if v['ExitCode']];assert len(failures)==3
archive=(D/'source.tar.gz').read_bytes();assert sha(archive)=='2e9efb6136f3de728cff2a09c3a7a3d0686375e6054577e4bb384414102d1d36'
r=dict(Result='PASS',ReviewedHead=head,NewGeneratedPaths=len(changes),FrozenInputsVerified=len(m['Files']),FrozenSourceID=m['SourceID'],ArchiveSHA256=sha(archive),PreservedCheckpointBytes=preserved,ReviewFilesVerified=review_files,IndexCapturesMatched=len(index),PreservedFailedCaptures=failures,InspectorCorrection='Exactly one terminal Encoder newline removed for Marshal value digest; compressed hash and all other assertions unchanged',WorktreeClean=True)
print(json.dumps(r,indent=2))
