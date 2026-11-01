"""Reproduce one independent frozen qualification batch. All outputs live outside the checkout."""
import argparse
from pathlib import Path
import subprocess
import sys

from gate import ROOT
from process_cases import CASES

TOOLS = ROOT / "docs/foundational/phase2d/tools"
CONSUMER = "examples/effectfulcomposition/consumer"
ADVERSE = "^TestPhase2D(CompetingAccounting|BindingControls|Multiplicity|IncompatiblePositive)$"
PRIOR = "^TestPhase2(B(RevokedCrashPair|RecordedRecovery|DelayedApplication|LiveOrders|NestedDrainCoversDescendant|ExactPriorStores)|C(Process.*|ResolverAuthority|CompetingResolvers|OriginalBindingControls|RefreshAfterAdoption))$"
PURE = "^TestOwningTree(TightLimits|DelayedFrontier|LateReply|Invalid|FaultIsNotPending)$"
PACKAGES = ["./author","./internal/executor","./adapters/receiptquery","./internal/fixture","./cmd/alan-resource",
            "./examples/owningtree","./examples/reconciliation","./examples/effectfulcomposition",
            "./research/effectfulcomposition/...","./research/owningtree","./research/reconciliation"]


def main():
    ap=argparse.ArgumentParser(description=__doc__)
    ap.add_argument("output",type=Path)
    ap.add_argument("mode",choices=("normal","race"))
    ap.add_argument("group",choices=("broad","build","focus","adverse","prior","grid","cases-a","cases-b","cases-c","static","model","dep-start","dep-end"))
    a=ap.parse_args()
    out=a.output.resolve()
    mode=a.mode
    group=a.group
    flags=["-race"] if mode=="race" else []
    binaries=out/(mode+"-bin")
    def run(label,argv,cwd=".",env=()):
        record=out/"commands"/(mode+"-"+label+".json.gz")
        cmd=[sys.executable,"-B",str(TOOLS/"gate.py"),str(record),"--cwd",cwd,"--env","GORACE=atexit_sleep_ms=0"]
        for k,v in env:
            cmd.extend(["--env",k+"="+str(v)])
        subprocess.run(cmd+["--",*map(str,argv)],cwd=ROOT,check=True)
    def gotest(label,packages,select=None,cwd=".",store=None):
        opts=["go","test",*flags,"-p=2","-tags=integration","-count=1","-v","-timeout=35m"]
        if select:
            opts+=["-run",select]
        env=[("ALAN_PHASE2_EVIDENCE",store)] if store else []
        run(label,opts+packages,cwd,env)
    if group=="broad":
        assert mode=="normal"
        run("full-root",["go","test","-p=2","-count=1","-timeout=60m","./..."])
    elif group=="build":
        binaries.mkdir(parents=True,exist_ok=True)
        for name in ("consumer","owner","resource","alan"):
            path=binaries/name
            if name=="owner":
                run("build-"+name,["go","test","-c",*flags,"-tags=integration","-o",path,"./internal/durabletest"])
            elif name=="consumer":
                run("build-"+name,["go","build","-buildvcs=false",*flags,"-o",path,"."],CONSUMER)
            else:
                package="./cmd/alan-resource" if name=="resource" else "./cmd/alan"
                run("build-"+name,["go","build","-buildvcs=false",*flags,"-o",path,package])
    elif group=="focus":
        gotest("unit",PACKAGES)
        gotest("consumer",["."],cwd=CONSUMER)
        gotest("pure",["./internal/substitutiontest"],PURE)
    elif group=="adverse":
        gotest("adverse",["./internal/durabletest"],ADVERSE,store=out/(mode+"-adverse"))
    elif group=="prior":
        gotest("prior",["./internal/durabletest"],PRIOR,store=out/(mode+"-prior"))
    elif group=="grid":
        run(group,[sys.executable,"-B",TOOLS/"grid.py",binaries,out/(mode+"-grid"),"--jobs","4"])
    elif group.startswith("cases-"):
        cases=[*CASES,"paired"]
        split={"cases-a":0,"cases-b":1,"cases-c":2}[group]
        run(group,[sys.executable,"-B",TOOLS/"process_cases.py",binaries,out/(mode+"-process"),*cases[split::3]])
    elif group=="model":
        dest=out/(mode+"-model")
        dest.mkdir(parents=True,exist_ok=False)
        run("model-export",["go","run",*flags,"./research/effectfulcomposition/cmd/export",dest])
    elif group=="static":
        assert mode=="normal"
        run("vet",["go","vet","-tags=integration","./..."])
        run("consumer-vet",["go","vet","./..."],CONSUMER)
        run("module",["go","mod","verify"])
        run("consumer-module",["go","mod","verify"],CONSUMER)
        run("format-scope", [sys.executable,"-B",TOOLS/"delivery.py","--scope"])
    elif group in ("dep-start","dep-end"):
        run(group,[sys.executable,"-B",TOOLS/"dependencies.py",out/"dependencies.json.gz",*(["--check"] if group=="dep-end" else [])])


if __name__=="__main__":
    main()
