"""Verify the integration boundary and inventory a delivered packet or its frozen command records."""
import argparse
from collections import Counter
import gzip
import hashlib
import json
from pathlib import Path
import re
import subprocess

from gate import ROOT, git, inputs, sha

BASE = "2612a85b287d95aa95f11e35c57b8b7b8dc26c6d"
PHASE = "docs/foundational/phase2d/"
INDEX = PHASE+"evidence/index.sha256"
REVIEW = PHASE+"evidence/whole-review.md"
EXACT = {
    "author/invocations.go","author/paired_handoff_test.go","internal/executor/composition_binding.go",
    "internal/executor/composition_types.go","internal/executor/invocation_handoff.go",
    "internal/durabletest/phase2d_owner_test.go","internal/durabletest/phase2d_adversarial_test.go",
    "internal/durabletest/phase2d_adversarial_world_test.go","paper/alan-dissertation.md",
    "paper/alan-dissertation.tex","paper/alan-dissertation.pdf"}
PREFIX=(PHASE,"examples/effectfulcomposition/","research/effectfulcomposition/")


def names(*args):
    return set(git(*args,"-z").decode().split("\0"))-{""}


def scope():
    paths=names("diff","--name-only",BASE) | names("ls-files","--others","--exclude-standard")
    extra=sorted(p for p in paths if p not in EXACT and not p.startswith(PREFIX))
    assert not extra,extra
    go=sorted(p for p in paths if p.endswith(".go"))
    if go:
        output=subprocess.check_output(["gofmt","-l",*go],cwd=ROOT,text=True)
        assert not output,output
    subprocess.run(["git","diff","--check",BASE],cwd=ROOT,check=True)
    assert git("rev-parse",BASE+"^{tree}")==git("rev-parse","089c1b6e61630f3e7db713eda2883b7b725d9f78^{tree}")
    return paths


def inventory(write):
    files=names("ls-files","-co","--exclude-standard")
    selected=set(inputs())|scope()|{p for p in files if p.startswith(PREFIX)}|{
        "AGENTS.md","paper/references.bib", "docs/foundational/phase2c/evidence/index.sha256",
        "docs/foundational/phase1t/evidence/identity-index.json"}
    selected-= {INDEX,REVIEW}
    lines="".join(sha(ROOT/p)+"  "+p+"\n" for p in sorted(selected))
    if write:
        (ROOT/INDEX).write_text(lines)
    else:
        assert lines==(ROOT/INDEX).read_text(),"source or evidence bytes changed"
    print("PASS inventory",len(selected),hashlib.sha256(lines.encode()).hexdigest())


def command_summary(directory):
    baseline=inputs()
    heads=set()
    summary={}
    for path in sorted(directory.glob("*.json.gz")):
        rec=json.loads(gzip.decompress(path.read_bytes()))
        assert rec["exit"]==0 and not rec["end_differences"] and not rec["end_status"],path
        assert rec["head"]==rec["end_head"] and rec["inputs"]==baseline,path
        heads.add(rec["head"])
        status=Counter(re.findall(r"^\s*--- (PASS|FAIL|SKIP): ",rec["output"],re.M))
        summary[path.name]=dict(argv=rec["argv"],seconds=round((rec["end_ns"]-rec["start_ns"])/1e9,2),
                                reported_go_statuses=dict(status),manifest_entries=len(baseline))
    assert len(heads)==1 and summary
    print(json.dumps(dict(head=heads.pop(),commands=summary),sort_keys=True,indent=2))


if __name__=="__main__":
    ap=argparse.ArgumentParser(description=__doc__)
    g=ap.add_mutually_exclusive_group(required=True)
    g.add_argument("--scope",action="store_true")
    g.add_argument("--write",action="store_true")
    g.add_argument("--verify",action="store_true")
    g.add_argument("--commands",type=Path)
    a=ap.parse_args()
    if a.scope:
        print("PASS scope, gofmt, whitespace, integration and delivery tree identity;",len(scope()),"changed paths")
    elif a.commands:
        command_summary(a.commands)
    else:
        inventory(a.write)
