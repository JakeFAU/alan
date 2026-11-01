"""Extract complete profiles and shortest adverse traces from the recorded export."""
from collections import deque
import gzip
import json
from pathlib import Path

EVIDENCE = Path(__file__).resolve().parents[2] / "docs/foundational/phase2c/evidence"
CASES = {
    'post-Q packet': ('held-packet-refresh', lambda s: s['Quiescent'] and s['Held'] and not s['Applications']),
    'absence then found': ('held-packet-refresh', lambda s: s['Reads'][0]['Outcome']=='not_found' and s['Reads'][1]['Outcome']=='found' and s['Used']),
    'lost absence': ('lost-negative-later-found', lambda s: s['Log'][0]['Status']=='not_found' and s['Log'][1]['Status']=='found' and s['Reads'][0]['Retained']==2 and s['Used']),
    'read without adopt': ('read-without-import', lambda s: s['Reads'][0]['Selected'] and s['Reads'][0]['Adoption']=='denied'),
    'unsupported adapter success': ('selected-error-policy', lambda s: s['Reads'][0]['Selected'] and s['Reads'][0]['Adoption']=='unsupported_policy'),
    'query exhausted': ('spent-before-send', lambda s: s['Losses']==2 and not s['SnapshotCount'] and s['Reads'][0]['Adoption']=='exhausted'),
    'query completion alone': ('raw-keyed', lambda s: s['Reads'][0]['Selected'] and s['Reads'][0]['Outcome']=='found' and not s['Published']),
    'immutable historical failure': ('existing-interaction-failure', lambda s: s['Used'] and s['Interaction']=='transport_error'),
    'concurrent equivalent evidence': ('two-resolvers', lambda s: s['Reads'][0]['Adoption']=='duplicate' and s['Reads'][1]['Adoption']=='adopted' and s['Used']),
}


def trace(rows, predicate):
    pending = deque([0])
    previous = {0: None}
    while pending:
        i = pending.popleft()
        if predicate(rows[i]['State']):
            result = []
            end = i
            while previous[i] is not None:
                src, action = previous[i]
                result.append([src, action, i])
                i = src
            return {'steps': result[::-1], 'last': rows[end]['State']}
        for edge in rows[i]['Edges']:
            target = edge['To']
            if target not in previous:
                previous[target] = i, edge['Action']
                pending.append(target)
    raise ValueError('missing witness')


if __name__ == '__main__':
    out = EVIDENCE/'model'
    with gzip.open(EVIDENCE/'commands/model-export.json.gz') as f:
        record = json.load(f)
    assert record['exit']==0
    raw = record['output'].encode()
    (out/'graphs.jsonl.gz').write_bytes(gzip.compress(raw, mtime=0))
    graphs = [json.loads(line) for line in raw.splitlines()]
    summary, examples = [], {}
    for graph in graphs:
        name = graph['Profile']['Name']
        rows = graph['Rows']
        sinks = [r['Class'] for r in rows if not r['Edges']]
        summary.append(dict(name=name, states=len(rows), edges=sum(len(r['Edges']) for r in rows),
                            sinks=len(sinks), outcomes=sorted(set(sinks))))
        for label, (profile, pred) in CASES.items():
            if profile==name:
                examples[label] = {'profile': name, **trace(rows, pred)}
    assert set(examples)==set(CASES)
    totals = {k: sum(p[k] for p in summary) for k in ('states','edges','sinks')}
    (out/'summary.json').write_text(json.dumps(dict(profiles=summary, totals=totals),indent=2)+'\n')
    (out/'traces.json').write_text(json.dumps(examples,indent=2)+'\n')
    print(totals)
