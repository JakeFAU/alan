"""Reproduce F1 and restore its frozen audit inputs only in a disposable copy."""
import argparse
import gzip
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import shutil
import subprocess
import sys
import tarfile
import tempfile


MANIFEST = 'docs/foundational/phase1n/evidence/model/reference-r1.json'
ARCHIVE = 'docs/foundational/phase1n/evidence/model/reference-r1.tar.gz'
AUDIT = 'docs/foundational/phase1n/evidence/model/qualification/audit.py'
QUALIFIER = 'research/substitution/callable_join_qualification_test.go'
TASK_REPORT = '.superpowers/sdd/00-plan-phase1n/task-3b-report.md'
MANIFEST_SHA = '4603206f1eccf3bb471ca1df821aa56c613633d98dfdd9d84016786db137023e'
ARCHIVE_SHA = '8d2a2cfb47c37f8f4a666e95c5dfa8f386fffef5af46e5ff12a32de22fb1b0b3'
REPORT_SHA = '6d89366d36b61fe45a69a734a3872995a6cde01e0c8e232013f35c80475b1abf'


def sha(data):
    return hashlib.sha256(data).hexdigest()


def safe_relative(name):
    path = PurePosixPath(name)
    assert not path.is_absolute() and '..' not in path.parts, name
    return Path(*path.parts)


def put(copy, name, data):
    path = copy / safe_relative(name)
    assert path.resolve().is_relative_to(copy.resolve()), name
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(data)


def mismatches(copy, sources):
    result = []
    for name, expected in sources.items():
        path = copy / safe_relative(name)
        actual = sha(path.read_bytes()) if path.exists() else None
        if actual != expected:
            result.append(dict(path=name, expected=expected, actual=actual))
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, default=Path.cwd())
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--task-records', type=Path)
    args = parser.parse_args()
    source = args.source.resolve()
    output = args.output.resolve()
    assert not output.exists(), 'Use a fresh output filename; never replace evidence.'
    manifest_bytes = (source / MANIFEST).read_bytes()
    archive_bytes = (source / ARCHIVE).read_bytes()
    assert sha(manifest_bytes) == MANIFEST_SHA
    assert sha(archive_bytes) == ARCHIVE_SHA
    frozen = json.loads(manifest_bytes)['sources']
    assert len(frozen) == 66
    with tarfile.open(source / ARCHIVE) as archive:
        assert sorted(archive.getnames()) == sorted(frozen)
        archived = {}
        for name, expected in frozen.items():
            safe_relative(name)
            member = archive.getmember(name)
            assert member.isfile(), name
            data = archive.extractfile(member).read()
            assert sha(data) == expected, name
            archived[name] = data
    protected = sorted(set(frozen) | {MANIFEST, ARCHIVE, AUDIT, QUALIFIER,
        'docs/foundational/phase1n/evidence/model/qualification/index.json',
        'docs/foundational/phase1n/evidence/reviews/final-r1.md'})
    before = {name: sha((source / name).read_bytes()) if (source / name).exists()
              else None for name in protected}
    record = dict(format='phase1n-final-review-fix-v1', argv=sys.argv,
                  cwd=str(Path.cwd()), source=str(source), python=sys.version,
                  script_sha256=sha(Path(__file__).read_bytes()),
                  manifest_sha256=MANIFEST_SHA, archive_sha256=ARCHIVE_SHA,
                  archived_sources=frozen, source_before=before,
                  current_manifest_mismatches=mismatches(source, frozen))
    env = {k: v for k, v in os.environ.items()
           if not k.startswith('ALAN_') and k != 'OPENAI_API_KEY'}
    overrides = dict(GOTOOLCHAIN='local', CGO_ENABLED='1', GOPROXY='off',
                     GOSUMDB='off', GOFLAGS='-mod=readonly',
                     PYTHONDONTWRITEBYTECODE='1')
    env.update(overrides)
    record['environment'] = overrides
    record['removed_environment_names'] = sorted(set(os.environ) - set(env))
    with tempfile.TemporaryDirectory(prefix='alan-phase1n-restoration-', dir='/private/tmp') as temp:
        copy = Path(temp) / 'copy'
        copy.mkdir()
        model = Path('docs/foundational/phase1n/evidence/model')
        shutil.copytree(source / model, copy / model)
        # Simulate checkout after ignored scratch cleanup, retaining current paper.
        for name in frozen:
            if name.startswith('.superpowers/'):
                continue
            put(copy, name, (source / name).read_bytes())
        put(copy, QUALIFIER, (source / QUALIFIER).read_bytes())
        scratch = [name for name in frozen if name.startswith('.superpowers/')]
        assert len(scratch) == 3
        for name in scratch:
            put(copy, name, archived[name])
        record['scratch_only_mismatches'] = mismatches(copy, frozen)
        assert [x['path'] for x in record['scratch_only_mismatches']] == [
            'paper/alan-dissertation.md', 'paper/alan-dissertation.tex']

        def audit(label):
            destination = Path(temp) / label
            destination.mkdir()
            run_env = dict(env, ALAN_MODEL_EVIDENCE=str(destination))
            argv = [sys.executable, AUDIT]
            result = subprocess.run(argv, cwd=copy, env=run_env, text=True,
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=120)
            entry = dict(argv=argv, cwd=str(copy), environment=dict(overrides,
                         ALAN_MODEL_EVIDENCE=str(destination)), exit=result.returncode,
                         stdout=result.stdout, stderr=result.stderr)
            saved = destination / 'audit.json'
            if saved.exists():
                entry['output_sha256'] = sha(saved.read_bytes())
            return entry

        record['scratch_only_audit'] = audit('scratch-only')
        assert record['scratch_only_audit']['exit'] != 0
        assert 'frozen source changed' in record['scratch_only_audit']['stderr']
        # Restore every manifest entry, including old paper; never write source.
        for name, data in archived.items():
            put(copy, name, data)
        record['full_restoration_mismatches'] = mismatches(copy, frozen)
        assert not record['full_restoration_mismatches']
        record['restored_manifest_count'] = len(frozen)
        record['task_report_restoration'] = dict(status='pending final task-records archive',
                                               path=TASK_REPORT, expected_sha256=REPORT_SHA)
        if args.task_records:
            raw = args.task_records.read_bytes()
            tasks = json.loads(gzip.decompress(raw))
            assert tasks['format'] == 'phase1n-task-records-v1'
            entry = tasks['files']['task-3b-report.md']
            data = entry['text'].encode('utf-8')
            assert entry['sha256'] == REPORT_SHA and sha(data) == REPORT_SHA
            put(copy, TASK_REPORT, data)
            record['task_report_restoration'].update(status='verified and restored',
                                                     archive_sha256=sha(raw))
        record['restored_audit'] = audit('restored')
        assert record['restored_audit']['exit'] == 0 and not record['restored_audit']['stderr']
    after = {name: sha((source / name).read_bytes()) if (source / name).exists()
             else None for name in protected}
    assert after == before, 'Source files changed during the disposable-copy check.'
    record.update(source_after=after, source_unchanged=True,
                  disposable_copy_removed=True, status='PASS')
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_bytes(gzip.compress((json.dumps(record, sort_keys=True, indent=2)+'\n').encode(), mtime=0))
    print(json.dumps(dict(status='PASS', output=str(output), output_sha256=sha(output.read_bytes()),
                         restored_manifest_count=len(frozen), negative_exit=record['scratch_only_audit']['exit'],
                         positive_exit=record['restored_audit']['exit'],
                         task_report_restoration=record['task_report_restoration']), sort_keys=True))


if __name__ == '__main__':
    main()
