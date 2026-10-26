#!/usr/bin/env python3
"""Audit retained inputs with the current comparator; never relabel suite SourceIDs.

The additional raw audit enumerates differences, without asserting raw equality.
"""
import base64
from collections import Counter
import difflib
import hashlib
import json
from pathlib import Path
import re
import sys
import tarfile
import shutil
import tempfile

ROOT = Path(__file__).resolve().parents[5]
DOC = ROOT / 'docs/foundational/phase1e'
OUT = DOC / 'evidence/evidence-tools'
sys.path.insert(0, str(DOC / 'tools'))
from comparison import compare_research, rows

MISSING = {'missing': True}

def expanded(value):
    if isinstance(value, str):
        try:
            return expanded(json.loads(value))
        except (ValueError, TypeError):
            return value
    if isinstance(value, dict):
        return {k: expanded(base64.b64decode(v).decode() if k == 'Bytes' and isinstance(v, str) else v)
                for k, v in value.items()}
    if isinstance(value, list):
        return [expanded(v) for v in value]
    return value


def differences(a, b, path=()):
    if path and path[-1] == 'Key' and a != b:
        yield path, a, b
    elif type(a) != type(b):
        yield path, a, b
    elif isinstance(a, dict):
        for key in sorted(a.keys() | b.keys()):
            if key not in a or key not in b:
                yield path + (key,), a.get(key, MISSING), b.get(key, MISSING)
            else:
                yield from differences(a[key], b[key], path + (key,))
    elif isinstance(a, list):
        assert len(a) == len(b), (path, 'raw list length changed')
        for i, (x, y) in enumerate(zip(a, b)):
            yield from differences(x, y, path + (i,))
    elif a != b:
        yield path, a, b


def raw_audit(a, b):
    namespaces = {c[k]['Namespace'] for c in b['checkpoints'] for k in ('runtime', 'snapshot')
                  if isinstance(c.get(k), dict) and 'Namespace' in c[k]}
    assert len(namespaces) == 1
    namespace = namespaces.pop()
    assert re.fullmatch('[0-9a-f]{64}', namespace)
    ids, urls = {}, {}
    for old, new in zip(a['runtime_provision']['Programs'], b['runtime_provision']['Programs']):
        for program in (old, new):
            assert hashlib.sha256(base64.b64decode(program['Bytes'])).hexdigest() == program['ID']
        if old['ID'] != new['ID']:
            ids[old['ID']] = new['ID']
    for old, new in zip(a['runtime_provision']['Permissions'], b['runtime_provision']['Permissions']):
        if old['Target'] != new['Target']:
            assert all(re.fullmatch(r'http://127\.0\.0\.1:[0-9]+', value) for value in (old['Target'], new['Target']))
            urls[old['Target']] = new['Target']
    result = []
    for path, old, new in differences(expanded(a), expanded(b)):
        leaf = path[-1]
        category = None
        if old == MISSING:
            if leaf in ('Namespace', 'namespace') and new == namespace:
                category = 'persistent store namespace added'
            elif leaf in ('KeyVersion', 'key_version') and new == 2:
                category = 'external key version added'
            elif leaf in ('adapter_attempts', 'adapter_results', 'corpus_assets') and new == []:
                category = 'empty retained table added'
            elif path == ('runtime_provision', 'Assets') and new is None:
                category = 'empty provision assets field added'
        elif leaf == 'schema_version' and old == 1 and new == 3:
            category = 'schema version changed'
        elif leaf == 'Key' and new == [2, namespace, old]:
            category = 'external key explicitly encodes same local intent with store namespace'
        elif leaf == 'PID' and isinstance(old, int) and isinstance(new, int) and old > 0 and new > 0:
            category = 'ephemeral process identity'
        elif isinstance(old, str) and urls.get(old) == new:
            category = 'ephemeral loopback endpoint'
        elif leaf in ('ID', 'id', 'ProgramID', 'program_id') and isinstance(old, str) and ids.get(old) == new:
            category = 'verified program content hash (endpoint literal changed)'
        assert category, ('unclassified raw difference', path, old, new)
        result.append(dict(Path=list(path), Before=old, After=new, Classification=category))
    return dict(Test=b['test'], RawEqual=a == b, Differences=result,
                Categories=dict(Counter(d['Classification'] for d in result)))


def main():
    retained = DOC / 'evidence/tooling-correction'
    source = json.loads((retained / 'before-source-manifest.json').read_text())['source_sha256']
    baseline = rows(DOC / 'evidence/baseline/final-normal.records.jsonl.gz')
    comparisons, raw = [], []
    for mode in ('normal', 'race'):
        current = rows(retained / f'final-{mode}.records.jsonl.gz')
        events = rows(retained / f'final-{mode}.go-test.jsonl.gz')
        comparisons += compare_research(baseline, current, ROOT, source, mode, events)
        before = [r['Evidence'] for r in baseline if r['Marker'] == 'CONFORMANCE']
        after = [r['Evidence'] for r in current if r['Marker'] == 'CONFORMANCE']
        raw.append(dict(Mode=mode, Cases=[raw_audit(a, b) for a, b in zip(before, after)]))
    # Portability check: the production comparator only needs current semantic bytes.
    # No Git repository, historical checkout, or retained source archive is copied.
    with tempfile.TemporaryDirectory(prefix='phase1e-comparator-portable-') as directory:
        portable = Path(directory)
        shutil.copyfile(ROOT / 'go.mod', portable / 'go.mod')
        shutil.copytree(ROOT / 'research/semantics', portable / 'research/semantics')
        portable_results = compare_research(baseline, current, portable, source, mode, events)
        assert portable_results == comparisons[-4:]
    files = [DOC / 'tools' / name for name in ('checks.py', 'comparison.py', 'test_comparison.py')]
    hashes = [dict(Path=str(p.relative_to(ROOT)), Bytes=p.stat().st_size,
                   SHA256=hashlib.sha256(p.read_bytes()).hexdigest()) for p in files]
    summary = dict(SuiteSourceID=source, ComparatorFiles=hashes, ResearchComparison=comparisons,
                   PortableCurrentSourceOnlyComparisonPassed=True,
                   Note='Current comparator applied retrospectively to retained inputs. Suite source remains unchanged; these are not fresh Go tests of corrected tools.')
    (OUT / 'retained-comparison.json').write_text(json.dumps(summary, indent=2) + '\n')
    (OUT / 'raw-conformance-differences.json').write_text(json.dumps(raw, indent=2) + '\n')
    with tarfile.open(retained / 'before-source.tar.gz') as archive:
        old = archive.extractfile('docs/foundational/phase1e/tools/checks.py').read().decode()
    patch = ''.join(difflib.unified_diff(old.splitlines(True), files[0].read_text().splitlines(True),
                                       fromfile='before/checks.py', tofile='after/checks.py'))
    for path in files[1:]:
        patch += ''.join(difflib.unified_diff([], path.read_text().splitlines(True), fromfile='/dev/null', tofile=str(path.relative_to(ROOT))))
    (OUT / 'correction.patch').write_text(patch)
    print(json.dumps(dict(InputsSourceID=source, ComparatorFiles=hashes,
                         Modes=['normal', 'race'], ComparatorPassed=True, PortableCurrentSourceOnlyComparisonPassed=True,
                         RawAudit='All decoded differences enumerated and classified; no raw equality claim.'), indent=2))


if __name__ == '__main__':
    main()
