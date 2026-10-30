"""Audit captured qualification identities and derive the compact census index."""
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import subprocess
import tarfile


def sha(data):
    return hashlib.sha256(data).hexdigest()


root = Path(__file__).parent


def read(label, name):
    return json.loads(gzip.decompress((root/label/(name+'.json.gz')).read_bytes()))


def totals(rows):
    return {k:sum(r[k] for r in rows) for k,v in rows[0].items() if type(v) is int}


commands = {}
for path in sorted(root.glob('*/command.json.gz')):
    c = json.loads(gzip.decompress(path.read_bytes()))
    assert c['exit'] == (1 if c['label'] == 'normal-correspondence-1' else 0), path
    assert not c['failures'] and not c['skips'] and not c['changed_during_run'], path
    if c['leaves']:
        assert all(n in c['passes'] for n in c['leaves']), path
    assert c['environment']['CGO_ENABLED'] == '1' and c['environment']['GOPROXY'] == 'off'
    assert c['environment']['GOSUMDB'] == 'off' and c['environment']['GOFLAGS'] == '-mod=readonly'
    archive = path.parent/'sources.tar.gz'
    assert sha(archive.read_bytes()) == c['source_snapshot_sha256']
    with tarfile.open(fileobj=io.BytesIO(gzip.decompress(archive.read_bytes()))) as t:
        assert sorted(t.getnames()) == sorted(c['sources'])
        for p,h in c['sources'].items():
            assert sha(t.extractfile(p).read()) == h, (path,p)
    for name,ident in c['outputs'].items():
        raw = (path.parent/name).read_bytes()
        assert sha(raw) == ident['sha256'] and sha(gzip.decompress(raw)) == ident['raw_sha256']
    commands[c['label']] = dict(argv=c['argv'],utc=c['utc'],exit=c['exit'],leaves=len(c['leaves']),
        command_sha256=sha(path.read_bytes()),source_archive_sha256=c['source_snapshot_sha256'],outputs=c['outputs'])

for profile in ['normal','race']:
    order = [commands[profile+'-'+stage]['utc'] for stage in ['reference','local','replacements','lifting']]
    assert order == sorted(order)

for stage in ['reference','local','replacements','lifting','artifacts']:
    for path in (root/('normal-'+stage)).glob('*.json.gz'):
        if path.name == 'command.json.gz': continue
        assert gzip.decompress(path.read_bytes()) == gzip.decompress((root/('race-'+stage)/path.name).read_bytes()), path

freeze = json.loads(Path('docs/foundational/phase1n/evidence/model/reference-r1.json').read_text())
for p,h in freeze['sources'].items():
    assert sha(Path(p).read_bytes()) == h, ('frozen source changed',p)

# Earlier census source snapshots precede gofmt and the new artifact-only test.
# Check the actual qualifier bytes are identical after gofmt; do not infer this
# merely from equal results or claim the later snapshot executed those tests.
qualifier = 'research/substitution/callable_join_qualification_test.go'
format_equivalence = {}
for profile in ['normal','race']:
    for stage in ['reference','local','replacements','lifting']:
        label = profile+'-'+stage
        with tarfile.open(root/label/'sources.tar.gz') as t:
            old = t.extractfile(qualifier).read()
        result = subprocess.run(['/opt/homebrew/bin/gofmt'],input=old,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        assert result.returncode == 0 and not result.stderr
        assert result.stdout == Path(qualifier).read_bytes()
        format_equivalence[label] = dict(argv=['/opt/homebrew/bin/gofmt'],exit=result.returncode,
            archived_sha256=sha(old),formatted_sha256=sha(result.stdout))

products, maxima = {}, []
for variant,stage in [('K1','reference'),('local','replacements'),('helper','replacements')]:
    rows = read('normal-'+stage,'callable-join-product-'+variant)
    t = totals(rows)
    for key in ['RankMinimum','RankMaximum','RankCap']: t.pop(key)
    t.update(configurations=len(rows),rank_min=min(r['RankMinimum'] for r in rows),
             rank_max=max(r['RankMaximum'] for r in rows),rank_cap=rows[0]['RankCap'],
             maxima=sum(len(r['Maxima']) for r in rows),counterexamples=sum(len(r['Counterexamples'] or []) for r in rows))
    products[variant] = t
    for r in rows:
        for m in r['Maxima']:
            s = m['State']; inner = s['Inner']['Component']
            maxima.append(dict(variant=variant,request=r['Config']['Request'],kind=m['Kind'],
                charges=[1+inner['I'],1+inner['R'],inner['E']],candidate_choice=inner['Ops'][0]['Choice'],
                candidate_consumed=inner['Ops'][0]['Consumed'],successor_consumed=inner['Ops'][3]['Consumed'],
                join_stage=s['Inner']['ParentStage'],outer_stage=s['OuterStage'],
                diagnostic=s['Report'],outer_decision=s['OuterDecision'],
                trace_length=len(m['Trace']),contract_errors=m['Contract'],outer_errors=m['Outer']))
    assert t['States'] == t['ModelStates']
    assert all(t[k] == 0 for k in ['Rejections','Stuck','Cutoffs','Cycles','StateFailures','RankFailures',
        'PrefixFailures','ComponentFailures','ProjectionFailures','ContractFailures','OuterFailures','counterexamples'])

local, lifting = {}, {}
local_rows = read('normal-local','callable-join-aligned-local')
lift_rows = read('normal-lifting','callable-join-finite-lifting')
for variant in ['local','helper']:
    rows = [r for r in local_rows if r['Variant'] == variant]
    assert all(not r['Failures'] and all(not m['Error'] for m in r['Maxima']) for r in rows)
    local[variant] = dict(configurations=len(rows),relation=totals([r['Relation'] for r in rows]),
        progress=totals([r['Progress'] for r in rows]),rank_min=min(r['RankMinimum'] for r in rows),
        rank_max=max(r['RankMaximum'] for r in rows),rank_cap=rows[0]['RankCap'],maxima=sum(len(r['Maxima']) for r in rows))
    lifting[variant] = totals([r['Relation'] for r in lift_rows if r['Variant'] == variant])

normal = read('normal-correspondence-r2','correspondence')
race = read('race-correspondence-r2','correspondence')
normal.pop('sources');race.pop('sources')
assert normal == race  # artifact compressed bytes are also deterministic
artifacts = read('normal-artifacts','callable-join-artifacts')
for r in artifacts.values():
    assert r['Candidate']['Output'] == r['Successor']['Output']

names = ['TestCallableJoin'+s for s in ['ReferenceQualification','ReplacementQualification',
    'AlignedLocalQualification','FiniteLiftingQualification','ArtifactCorrespondenceInputs']]
discovery = read('discovery-final','command')
assert sorted(line for line in discovery['stdout'].splitlines() if line.startswith('Test')) == sorted(names)
version = subprocess.run(['/opt/homebrew/bin/go','version'],text=True,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
assert version.returncode == 0 and version.stdout == 'go version go1.27.3 darwin/arm64\n' and not version.stderr
result = dict(commands=commands,go_version=dict(argv=['/opt/homebrew/bin/go','version'],exit=version.returncode,stdout=version.stdout),
    normal_race_full_records_equal=True,reference_manifest_sha256=sha(Path('docs/foundational/phase1n/evidence/model/reference-r1.json').read_bytes()),
    unchanged_reference_source_count=len(freeze['sources']),qualifier_gofmt_equivalence=format_equivalence,
    products=products,local=local,lifting=lifting,maxima_index=maxima,correspondence=normal,
    independent_evidence_caution='K_join and G_outer project the same full products. Race repeats and finite artifact policy cases are not additional configurations. ARG-1 is conditional source argument; ARG-2 is open.')
dest = Path(os.environ['ALAN_MODEL_EVIDENCE'])/'audit.json'
dest.write_text(json.dumps(result,sort_keys=True,indent=2)+'\n')
print(json.dumps(dict(commands=len(commands),unchanged_reference_sources=len(freeze['sources']),products=products,
    local=local,lifting=lifting,maxima=len(maxima),output_sha256=sha(dest.read_bytes())),sort_keys=True))
