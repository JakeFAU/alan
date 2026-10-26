#!/usr/bin/env python3
"""Separate unchanged Phase1E semantic behavior from current source identity."""
from pathlib import Path
import gzip, importlib.util, json, sys
ROOT=Path(__file__).resolve().parents[4]; DOC=ROOT/"docs/foundational/phase1f"
spec=importlib.util.spec_from_file_location("old_comparison",ROOT/"docs/foundational/phase1e/tools/comparison.py")
c=importlib.util.module_from_spec(spec);spec.loader.exec_module(c)
mode=sys.argv[1] if len(sys.argv)>1 else "normal"
base=json.loads((ROOT/"docs/foundational/phase1e/source-manifest.json").read_text())["source_sha256"]
current=json.loads((DOC/"source-manifest.json").read_text())["source_sha256"]
old=c.rows(ROOT/f"docs/foundational/phase1e/evidence/final-{mode}.records.jsonl.gz")
new=c.rows(DOC/f"evidence/final-{mode}.records.jsonl.gz")
go=[json.loads(line) for line in gzip.decompress((DOC/f"evidence/final-{mode}.go-test.jsonl.gz").read_bytes()).splitlines()]
passed={e["Test"] for e in go if e.get("Action")=="pass" and e.get("Package")=="github.com/JakeFAU/alan/research/semantics" and "Test" in e}
results=[]
for marker,count in (("QUALIFICATION",15),("COMPOSITION",20),("SUPERVISION",32),("CONFORMANCE",6)):
 a=[r for r in old if r["Marker"]==marker];b=[r for r in new if r["Marker"]==marker]
 c.require(len(a)==len(b)==count,"count")
 c.require(all(r["SourceID"]==base and r["Mode"]==mode for r in a),"baseline wrapper")
 c.require(all(r["SourceID"]==current and r["Mode"]==mode for r in b),"current wrapper")
 c.require([(r["Package"],r["Test"]) for r in a]==[(r["Package"],r["Test"]) for r in b],"ordered tests")
 x=[r["Evidence"] for r in a];y=[r["Evidence"] for r in b]
 if marker=="CONFORMANCE": result=c.compare_conformance(x,y,passed)
 else:
  c.require(x==y,"exact ordered semantic payloads")
  result=dict(Marker=marker,Records=count,ExactOrderedPayloads=True)
 results.append(result)
report=dict(mode=mode,baseline_source=base,current_source=current,comparisons=results,
 scope="Current-source regression against published Phase1E, separate from historical frozen archive reproduction. Conformance compares declared public projections and model streams, never raw ephemeral ledgers.")
(DOC/f"evidence/comparison-{mode}.json").write_text(json.dumps(report,indent=2)+"\n")
print(json.dumps(report,indent=2))
