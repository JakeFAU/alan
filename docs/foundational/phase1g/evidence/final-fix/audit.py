#!/usr/bin/env python3
"""Recompute source, archive and retained-evidence identities without Go execution."""
from pathlib import Path
import hashlib
import json
import struct
import subprocess
import tarfile

ROOT = Path(__file__).resolve().parents[5]
DOC = ROOT / 'docs/foundational/phase1g'
OUT = DOC / 'evidence/final-fix'
QUALIFIED = OUT / 'qualified-source-43b86781'


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def check_source(manifest, folder):
    digest = hashlib.sha256()
    for f in manifest['files']:
        raw = (folder / f['path']).read_bytes()
        assert len(raw) == f['bytes'] and sha(raw) == f['sha256'], f['path']
        name = f['path'].encode()
        digest.update(struct.pack('>Q', len(name)) + name + struct.pack('>Q', len(raw)) + raw)
    assert digest.hexdigest() == manifest['source_sha256']


current = json.loads((DOC / 'source-manifest.json').read_text())
old = json.loads((QUALIFIED / 'source-manifest.json').read_text())
check_source(current, ROOT)
old_map = {f['path']: f for f in old['files']}
new_map = {f['path']: f for f in current['files']}
changed = sorted(p for p in old_map.keys() & new_map.keys() if old_map[p] != new_map[p])
added = sorted(new_map.keys() - old_map.keys())
assert changed == ['docs/foundational/phase1g/tools/qualify.py']
assert added == ['docs/foundational/phase1g/tools/test_qualify.py']
assert not old_map.keys() - new_map.keys()
archives = []
for folder in (QUALIFIED, DOC):
    m = json.loads((folder / 'archive-manifest.json').read_text())
    archive = folder / m['Archive']
    assert len(archive.read_bytes()) == m['Bytes'] and sha(archive.read_bytes()) == m['SHA256']
    with tarfile.open(archive) as tar:
        assert len(tar.getmembers()) == len(m['Files'])
        for f in m['Files']:
            member = tar.getmember(f['Path'])
            assert member.isfile() and not Path(member.name).is_absolute() and '..' not in Path(member.name).parts
            raw = tar.extractfile(member).read()
            assert len(raw) == f['Bytes'] and sha(raw) == f['SHA256'], f['Path']
        source = json.loads(tar.extractfile('docs/foundational/phase1g/source-manifest.json').read())
        assert source['source_sha256'] == m['SourceID']
        digest = hashlib.sha256()
        for f in source['files']:
            raw = tar.extractfile(f['path']).read()
            assert len(raw) == f['bytes'] and sha(raw) == f['sha256'], f['path']
            name = f['path'].encode()
            digest.update(struct.pack('>Q', len(name)) + name + struct.pack('>Q', len(raw)) + raw)
        assert digest.hexdigest() == source['source_sha256']
        dependency = tar.extractfile('dependencies/github.com/mattn/go-sqlite3/@v/v1.14.52.zip').read()
        assert sha(dependency) == source['dependency']['zip_sha256']
    archives.append(dict(SourceID=m['SourceID'], SHA256=m['SHA256'], VerifiedMembers=len(m['Files'])))

preserved = json.loads((OUT / 'qualified-source-preservation.json').read_text())
for f in preserved:
    raw = (OUT / f['Path']).read_bytes()
    assert len(raw) == f['Bytes'] and sha(raw) == f['SHA256'], f['Path']
before = json.loads((OUT / 'before-inventory.json').read_text())
allowed = {'README.md', 'docs/foundational/05-decisions.md',
           'docs/foundational/phase1g/02-qualification.md', 'docs/foundational/phase1g/04-runbook.md',
           'docs/foundational/phase1g/source-manifest.json', 'docs/foundational/phase1g/archive-manifest.json',
           'docs/foundational/phase1g/source.tar.gz', 'docs/foundational/phase1g/tools/qualify.py'}
actual_changes = []
for f in before:
    raw = (ROOT / f['Path']).read_bytes()
    if len(raw) != f['Bytes'] or sha(raw) != f['SHA256']:
        actual_changes.append(f['Path'])
assert set(actual_changes) == allowed, actual_changes
prior = json.loads((DOC / 'evidence/baseline/prior-artifacts-before.json').read_text())
for path, digest in prior.items():
    assert sha((ROOT / path).read_bytes()) == digest, path
base = '10f6d8c9766c41d2b9b5129c3eeaf5d5614c55ba'
git_changes = subprocess.check_output(['git', 'diff', '--name-status', base], cwd=ROOT, text=True).splitlines()
modified, additions = [], []
for line in git_changes:
    status, path = line.split('\t')
    if status == 'A':
        assert path.startswith('docs/foundational/phase1g/evidence/final-fix/') or path == 'docs/foundational/phase1g/tools/test_qualify.py', path
        additions.append(path)
    else:
        assert status == 'M', line
        modified.append(path)
assert set(modified) == allowed, modified
# Bind every original input to git's original bytes independently of the manifest.
for f in old['files']:
    original = subprocess.check_output(['git', 'show', base + ':' + f['path']], cwd=ROOT)
    assert len(original) == f['bytes'] and sha(original) == f['sha256'], f['path']
    if f['path'] not in changed:
        assert (ROOT / f['path']).read_bytes() == original, f['path']
result = dict(Result='PASS', CurrentSourceID=current['source_sha256'], QualifiedSourceID=old['source_sha256'],
              SourceInputsVerified=len(current['files']), UnchangedOriginalInputs=len(old['files'])-len(changed),
              ChangedSourceInputs=changed, AddedSourceInputs=added, AllGoTestsModulesFixturesUnchanged=True,
              Archives=archives, QualifiedSnapshotFilesVerified=len(preserved),
              OriginalArtifactsInventoried=len(before), ChangedOriginalFiles=actual_changes,
              UnchangedOriginalArtifacts=len(before)-len(actual_changes), PriorPhaseArtifactsUnchanged=len(prior),
              GitBase=base, OriginalSourceInputsIndependentlyComparedToGit=len(old['files']))
(OUT / 'final-audit.json').write_text(json.dumps(result, indent=2) + '\n')
print(json.dumps(result, indent=2))
