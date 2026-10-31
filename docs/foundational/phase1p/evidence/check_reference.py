import gzip, hashlib, itertools, json, subprocess
base = 'd0def139632c73936a3e821d93271a62267fa46d'
path = 'docs/foundational/phase1n/evidence/model/qualification/'
def read(name):
    return subprocess.check_output(['git', 'show', base+':'+path+name])
def record(folder, name):
    cmd = json.loads(gzip.decompress(read(folder+'/command.json.gz')))
    data = read(folder+'/'+name)
    raw = gzip.decompress(data)
    digest = lambda b: hashlib.sha256(b).hexdigest()
    assert cmd['exit'] == 0 and not cmd['failures'] and not cmd['changed_during_run']
    assert cmd['outputs'][name] == {'sha256': digest(data), 'raw_sha256': digest(raw)}
    print(folder, cmd['exit'], digest(data))
    return json.loads(raw)
rows = record('normal-reference', 'callable-join-product-K1.json.gz')
domain = set(itertools.product((1,2), (0,1), ('answer','fail','absent','pending')))
requests = [(r['Config']['Request']['ID'], r['Config']['Request']['Value'], r['Config']['Request']['Mode']) for r in rows]
assert len(requests) == len(domain) and set(requests) == domain
assert all((r['Variant'], r['Config']['I'], r['Config']['R'], r['Config']['Depth']) == ('K1',4,4,3) for r in rows)
bad = [k for k in rows[0] if k.endswith('Failures')] + ['Rejections','Stuck','Cutoffs','Cycles']
assert all(r[k] == 0 for r in rows for k in bad)
print('K1', len(rows), *(sum(r[k] for r in rows) for k in ('States','Edges','Terminal','Pending')), 'all zero', bad)
art = record('normal-correspondence-r2','correspondence.json.gz')
print('mapped', art['mapped_instructions'], [(k, v['join_policy']['cases'], v['outer_policy']['cases']) for k,v in sorted(art['bindings'].items())])
