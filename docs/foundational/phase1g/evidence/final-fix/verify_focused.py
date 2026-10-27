#!/usr/bin/env python3
"""Verify the metadata-tool archive and only its focused extracted commands."""
from pathlib import Path
import gzip
import hashlib
import json
import os
import struct
import subprocess
import tarfile
import tempfile
import time

ROOT = Path(__file__).resolve().parents[5]
DOC = ROOT / 'docs/foundational/phase1g'
OUT = DOC / 'evidence/final-fix/extracted'
m = json.loads((DOC / 'archive-manifest.json').read_text())
archive = DOC / m['Archive']
assert hashlib.sha256(archive.read_bytes()).hexdigest() == m['SHA256']
assert not OUT.exists(), 'refuse to replace focused evidence'
OUT.mkdir(parents=True)


def verify_source(folder):
    source = json.loads((folder / 'docs/foundational/phase1g/source-manifest.json').read_text())
    digest = hashlib.sha256()
    for f in source['files']:
        raw = (folder / f['path']).read_bytes()
        assert len(raw) == f['bytes'] and hashlib.sha256(raw).hexdigest() == f['sha256'], f['path']
        name = f['path'].encode()
        digest.update(struct.pack('>Q', len(name)) + name + struct.pack('>Q', len(raw)) + raw)
    assert digest.hexdigest() == source['source_sha256'] == m['SourceID']
    return len(source['files'])


records = []
with tempfile.TemporaryDirectory(prefix='alan-phase1g-metadata-offline-') as tmp:
    folder = Path(tmp).resolve()
    with tarfile.open(archive) as tar:
        assert len(tar.getmembers()) == len(m['Files'])
        for f in m['Files']:
            member = tar.getmember(f['Path'])
            assert member.isfile() and not Path(member.name).is_absolute() and '..' not in Path(member.name).parts
            raw = tar.extractfile(member).read()
            assert len(raw) == f['Bytes'] and hashlib.sha256(raw).hexdigest() == f['SHA256'], f['Path']
        tar.extractall(folder, filter='data')
    source_count = verify_source(folder)
    env = dict(os.environ, GOTOOLCHAIN='local', GOMODCACHE=str(folder / 'module-cache'),
               GOCACHE='/tmp/alan-phase1f-go-build', GOPROXY='file://' + str(folder / 'dependencies'),
               GOSUMDB='off', GOFLAGS='-mod=readonly', PYTHONDONTWRITEBYTECODE='1',
               ALAN_QUALIFICATION_OUT=str(OUT))
    env.pop('OPENAI_API_KEY', None)
    env.pop('ALAN_METADATA_TEST_TOOL', None)
    env.pop('ALAN_METADATA_TEST_EVIDENCE', None)
    assert not Path(env['GOMODCACHE']).exists()
    commands = [
        ['python3', '-m', 'unittest', 'discover', '-s', 'docs/foundational/phase1g/tools', '-p', 'test_qualify.py', '-v'],
        ['python3', 'docs/foundational/phase1g/tools/qualify.py', 'consumer'],
        ['go', '-C', 'examples/worktree/consumer', 'vet', './...'],
        ['go', '-C', 'examples/worktree/consumer', 'mod', 'verify'],
    ]
    result = dict(SourceID=m['SourceID'], ArchiveSHA256=m['SHA256'], VerifiedMembers=len(m['Files']),
                  VerifiedSourceFiles=source_count, InitialModuleCacheEmpty=True, CredentialAbsent=True,
                  Scope='Metadata tooling regression and real consumer only; original full Go qualification retained separately.',
                  Directory=str(folder), Environment={k: env[k] for k in
                    ('GOTOOLCHAIN', 'GOMODCACHE', 'GOCACHE', 'GOPROXY', 'GOSUMDB', 'GOFLAGS')}, Commands=records)
    for i, cmd in enumerate(commands, 1):
        start = time.monotonic()
        p = subprocess.run(cmd, cwd=folder, env=env, capture_output=True, timeout=400)
        stem = f'{i:02d}'
        (OUT / (stem + '.stdout.gz')).write_bytes(gzip.compress(p.stdout, mtime=0))
        (OUT / (stem + '.stderr.gz')).write_bytes(gzip.compress(p.stderr, mtime=0))
        verify_source(folder)
        records.append(dict(Command=cmd, ExitCode=p.returncode, ElapsedSeconds=round(time.monotonic()-start, 3),
                            Stdout=stem+'.stdout.gz', Stderr=stem+'.stderr.gz', SourceUnchanged=True))
        (OUT / 'verification.json').write_text(json.dumps(result, indent=2) + '\n')
        print(stem, cmd, p.returncode, flush=True)
        assert p.returncode == 0, records[-1]
    verify_source(folder)
    result.update(Result='PASS', SourceUnchanged=True)
    (OUT / 'verification.json').write_text(json.dumps(result, indent=2) + '\n')
verify_source(ROOT)
