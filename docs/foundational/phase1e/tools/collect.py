#!/usr/bin/env python3
"""Index final commands and causal records under their exact working-tree identity."""
from pathlib import Path
import gzip, hashlib, json
ROOT=Path(__file__).resolve().parents[4]; DOC=ROOT/'docs/foundational/phase1e'
m=json.loads((DOC/'source-manifest.json').read_text()); source=m['source_sha256']
for f in m['files']:
    raw=(ROOT/f['path']).read_bytes()
    assert len(raw)==f['bytes'] and hashlib.sha256(raw).hexdigest()==f['sha256'],f['path']
rows=[dict(Kind='source_identity',SourceID=source,Manifest='source-manifest.json',BaselineCommit=m['baseline_commit'],HeadAtFreeze=m['head_at_freeze'],Platform=m['platform'],GoVersion=m['go_version'],Live='NOT RUN',LiveReason='No explicitly configured model and authorized credential available; controlled protocol evidence is not live inference.')]
for mode in ('default','normal','race'):
    meta=json.loads((DOC/f'evidence/final-{mode}.command.json').read_text())
    assert meta['SourceID']==source and meta['ExitCode']==0 and not meta['Failures']
    rows.append(meta)
    rows.extend(json.loads(line) for line in gzip.decompress((DOC/f'evidence/final-{mode}.records.jsonl.gz').read_bytes()).splitlines())
for name in ('final-checks.json','archive-reproduction.json'):
    row=json.loads((DOC/'evidence'/name).read_text());assert row['SourceID']==source
    rows.append(dict(Kind=name.removesuffix('.json'),**row))
rows.append(dict(Kind='historical_baseline',SourceID='9fef40370632bdf7416d5bdb2e8472f31ae2adc80ae5145cd7c9c90ccfe63e63',Commit=m['baseline_commit'],Evidence='evidence/baseline/report.md',Scope='Reproduced historical source separately. Current-source records above do not replace its identity.'))
(DOC/'qualification.jsonl').write_text(''.join(json.dumps(r,separators=(',',':'))+'\n' for r in rows))
print(json.dumps(dict(SourceID=source,Records=len(rows),File='qualification.jsonl',SHA256=hashlib.sha256((DOC/'qualification.jsonl').read_bytes()).hexdigest()),indent=2))
