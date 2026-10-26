"""Compare Phase 1E research behavior with explicitly validated historical provenance.

Raw conformance runtime ledgers are preserved, but are not the equality domain.
"""
import gzip
import hashlib
import json
import struct

BASELINE_SOURCE = '9fef40370632bdf7416d5bdb2e8472f31ae2adc80ae5145cd7c9c90ccfe63e63'
HISTORICAL_SEMANTICS = '6203197c9460a6e8f0e4338225f4fa7a885b1b0f5df67cca8f3949ab81b793d4'
CURRENT_SEMANTICS = 'ad688755b8580924a9604936e9c910f4403c8457ead390024a80ebe2c473f940'
BRIDGE = 'research/semantics/runtime_conformance_test.go'
BRIDGE_BEFORE = '599001e7c6e911d3aa27aa243e5e295f1cce3bbbb6083507333f4ee907b1d7ba'
BRIDGE_AFTER = 'f2f3588736b0e42ad013a3640a3a328c18ca9043454862bdf68c5b08ba418cc5'
PUBLIC_FIELDS = ('actual', 'expected', 'assertions', 'label', 'model_event_count')
MODEL_FIELDS = ('test', 'fixture', 'id_map', 'limits', 'mapping', 'model_config',
                'model_seed', 'model_trace', 'observation_map', 'passed')


def require(condition, detail):
    if not condition:
        raise ValueError(detail)


def rows(path):
    return [json.loads(line) for line in gzip.decompress(path.read_bytes()).splitlines()]


def validate_manifest(manifest, root, historical=False):
    """Verify all semantic file bytes and aggregate, including the exact bridge edit."""
    files = manifest['Files']
    paths = [f['Path'] for f in files]
    expected_paths = sorted(['go.mod'] + [str(p.relative_to(root)) for p in
                                           (root / 'research/semantics').glob('*_test.go')])
    require(paths == expected_paths, 'semantic manifest file set/order')
    digest = hashlib.sha256()
    for entry in files:
        path = entry['Path']
        raw = (root / path).read_bytes()
        if path == BRIDGE:
            require(hashlib.sha256(raw).hexdigest() == BRIDGE_AFTER, 'current bridge bytes')
            if historical:
                for label, tail in [('external request key', 'a.Request.Key'), ('receipt key', 'r.Key')]:
                    after = f'bridgeEqual(b.t, "{label}", ex.ExternalKey(snapshot.Namespace, rid), {tail})'.encode()
                    before = f'bridgeEqual(b.t, "{label}", rid, {tail})'.encode()
                    require(raw.count(after) == 1, 'exact documented bridge assertion')
                    raw = raw.replace(after, before)
                require(hashlib.sha256(raw).hexdigest() == BRIDGE_BEFORE, 'historical bridge bytes')
        require(len(raw) == entry['Bytes'] and hashlib.sha256(raw).hexdigest() == entry['SHA256'],
                f'semantic manifest bytes: {path}')
        name = path.encode()
        digest.update(struct.pack('>Q', len(name)) + name + struct.pack('>Q', len(raw)) + raw)
    expected = HISTORICAL_SEMANTICS if historical else CURRENT_SEMANTICS
    require(digest.hexdigest() == manifest['SHA256'] == expected, 'semantic aggregate identity')


def compare_deterministic(marker, before, after, root):
    count = {'QUALIFICATION': 15, 'COMPOSITION': 20, 'SUPERVISION': 32}[marker]
    require(len(before) == len(after) == count, f'{marker} record count')
    if marker == 'QUALIFICATION':
        require(before == after, 'QUALIFICATION exact ordered payloads')
        return dict(Marker=marker, Records=count, ExactPayloads=count,
                    OrderedBehaviorEqual=True, Scope='Complete payload equality, without normalization.')
    for records, historical in [(before, True), (after, False)]:
        require(set(records[0]) == {'Name', 'Manifest'} and records[0]['Name'] == 'source_manifest',
                f'{marker} manifest header')
        validate_manifest(records[0]['Manifest'], root, historical)
        require(all(r.get('SourceID') == records[0]['Manifest']['SHA256'] for r in records[1:]),
                f'{marker} record SourceID must match manifest')
    old, new = before[0]['Manifest'], after[0]['Manifest']
    require(old['Algorithm'] == new['Algorithm'], 'manifest algorithm')
    changed = [dict(Path=a['Path'], Before=a, After=b) for a, b in zip(old['Files'], new['Files']) if a != b]
    require([c['Path'] for c in changed] == [BRIDGE], 'only documented bridge file may change')
    # These are the only omitted fields; all nested behavior remains untouched and ordered.
    a = [{k: v for k, v in r.items() if k != 'SourceID'} for r in before[1:]]
    b = [{k: v for k, v in r.items() if k != 'SourceID'} for r in after[1:]]
    require(a == b, f'{marker} ordered behavior differs')
    return dict(Marker=marker, Records=count, ExactPayloads=sum(a == b for a, b in zip(before, after)),
                OrderedBehaviorEqual=True, HistoricalManifest=old, CurrentManifest=new, ChangedFiles=changed,
                Scope='Header Manifest and per-record SourceID validated separately; every remaining field equals in original order. Not raw historical payload equality.')


def conformance_projection(record):
    """Declared public checkpoint domain; field presence matters, as does order."""
    require(all(k in record for k in MODEL_FIELDS), 'conformance model fields missing')
    return dict(Model={k: record[k] for k in MODEL_FIELDS}, Checkpoints=[
        {k: checkpoint[k] for k in PUBLIC_FIELDS if k in checkpoint}
        for checkpoint in record['checkpoints']])


def compare_conformance(before, after, passed_tests):
    require(len(before) == len(after) == 6, 'conformance count')
    cases = []
    for a, b in zip(before, after):
        require(a['passed'] is True and b['passed'] is True, 'conformance passed flag')
        require(b['test'] in passed_tests, 'conformance Go test pass event missing')
        require(conformance_projection(a) == conformance_projection(b), 'conformance public projection/model stream differs')
        require([set(c) for c in a['checkpoints']] == [set(c) for c in b['checkpoints']], 'checkpoint field shapes differ')
        for checkpoint in b['checkpoints']:
            if 'actual' in checkpoint:
                require(checkpoint['actual'] == checkpoint['expected'], 'actual/expected checkpoint disagreement')
        cases.append(dict(Test=b['test'], Fixture=b['fixture'], Checkpoints=len(b['checkpoints']),
                          PublicProjectionCheckpoints=sum('actual' in c for c in b['checkpoints']),
                          OtherCheckpoints=[dict(Label=c['label'], Fields=list(c)) for c in b['checkpoints'] if 'actual' not in c],
                          RawPayloadEqual=a == b))
    return dict(Marker='CONFORMANCE', Records=6, AssertionsPass=True,
                ExactPayloads=sum(a == b for a, b in zip(before, after)), Cases=cases,
                PublicCheckpointProjectionEqual=True, ModelEventStreamsEqual=True,
                ComparedCheckpointFields=list(PUBLIC_FIELDS), ComparedModelFields=list(MODEL_FIELDS),
                Scope='Ordered declared public projections and model streams only. Other checkpoints retain labels and field shapes; their raw runtime/snapshot/independent_resource/runtime_outcome data are not equality-checked. Runtime actions/provision and raw ledgers remain preserved; Go assertions cover their local contracts. No raw-payload or whole-trace equivalence claim.')


def compare_research(baseline, current, root, current_source, mode, go_events):
    comparisons = []
    passed_tests = {e['Test'] for e in go_events if e.get('Action') == 'pass'
                    and e.get('Package') == 'github.com/JakeFAU/alan/research/semantics' and 'Test' in e}
    for marker in ('QUALIFICATION', 'COMPOSITION', 'SUPERVISION', 'CONFORMANCE'):
        a = [r for r in baseline if r['Marker'] == marker]
        b = [r for r in current if r['Marker'] == marker]
        require(all(r['SourceID'] == BASELINE_SOURCE and r['Mode'] == 'normal' for r in a), 'baseline wrapper identity')
        require(all(r['SourceID'] == current_source and r['Mode'] == mode for r in b), 'current wrapper identity')
        require([(r['Package'], r['Test']) for r in a] == [(r['Package'], r['Test']) for r in b], 'ordered test identities')
        before, after = [r['Evidence'] for r in a], [r['Evidence'] for r in b]
        result = (compare_conformance(before, after, passed_tests) if marker == 'CONFORMANCE'
                  else compare_deterministic(marker, before, after, root))
        result.update(Mode=mode, BaselineSourceID=BASELINE_SOURCE, CurrentSourceID=current_source)
        comparisons.append(result)
    return comparisons
