"""Build immutable inputs, attest candidate to a Git revision, or time four B256 arms."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import statistics
import subprocess
import sys
import tempfile

sys.dont_write_bytecode = True
P = argparse.ArgumentParser(description=__doc__)
P.add_argument("action", choices=["prepare", "attest", "time"])
P.add_argument("--baseline", type=Path, required=True)
P.add_argument("--candidate", type=Path, required=True)
P.add_argument("--out", type=Path, required=True)
P.add_argument("--revision", help="exact candidate commit SHA for attest")
a = P.parse_args()
out = a.out.resolve()
base = a.baseline.resolve()
candidate = a.candidate.resolve()
ev = base / "docs/evidence"
spec = importlib.util.spec_from_file_location("progress_measure", ev/"progress-scaling/measure.py")
measure = importlib.util.module_from_spec(spec)
spec.loader.exec_module(measure)
env = dict(os.environ, CGO_ENABLED="1", GOGC="100", GOMAXPROCS="4", GOMEMLIMIT="off",
           GOWORK="off", GOTOOLCHAIN="local", GOENV="off", GOPROXY="off", GOFLAGS="")
build = out/"build"
sha = lambda b: hashlib.sha256(b).hexdigest()
def obj(path, value):
    path.write_text(json.dumps(value, sort_keys=True, indent=2)+"\n")
def cmd(*args, cwd):
    return subprocess.check_output([str(x) for x in args], cwd=cwd, env=env).decode().strip()
def sources(root, destination):
    paths = [Path("go.mod"), Path("go.sum")]
    dep_format = '{{if and .Module (eq .Module.Path "github.com/JakeFAU/alan")}}{{.Dir}}{{end}}'
    dirs = cmd("go","list","-deps","-f",dep_format,"./author","./runtime",cwd=root)
    for pkg in sorted(filter(None,dirs.splitlines())):
        info = json.loads(cmd("go","list","-json",cwd=pkg))
        for group in ["GoFiles","CgoFiles","EmbedFiles","CFiles","CXXFiles","MFiles",
                      "HFiles","FFiles","SFiles","SwigFiles","SwigCXXFiles","SysoFiles"]:
            paths.extend((Path(pkg)/f).relative_to(root) for f in info.get(group,[]))
    inventory = {}
    for rel in sorted(set(paths)):
        data = (root/rel).read_bytes()
        new = destination/rel
        new.parent.mkdir(parents=True, exist_ok=True)
        new.write_bytes(data)
        inventory[str(rel)] = sha(data)
    return inventory

arms = ("base-full","candidate-full","base-progress","candidate-progress")
def prepare():
    build.mkdir(parents=True)
    bsha = cmd("git","rev-parse","HEAD",cwd=base)
    if bsha!="64a692b6f82ae10353f50c8741377166bc6cf854":
        raise ValueError("baseline is not the audited runtime commit")
    manifests = {}
    binaries = {}
    for lab,root in [("base",base),("candidate",candidate)]:
        snap = build/lab/"source"
        inventory = sources(root,snap)
        if lab=="base":
            verify_commit(root,bsha,inventory)
        manifests[lab]=inventory
        for mode in ["full","progress"]:
            arm = lab+"-"+mode
            d = build/arm
            d.mkdir()
            for name in measure.DRIVER_FILES:
                shutil.copyfile(measure.DRIVER/name,d/name)
            measure.make_driver(d,mode)
            cmd("gofmt","-w",*(str(f) for f in sorted(d.glob("*.go"))),cwd=d)
            cmd("go","mod","edit","-replace=github.com/JakeFAU/alan="+str(snap),cwd=d)
            binary = d/"driver"
            cmd("go","build","-buildvcs=false","-mod=readonly","-o",binary,".",cwd=d)
            binaries[arm] = {"sha256":sha(binary.read_bytes()),
                             "go_version_m":cmd("go","version","-m",binary,cwd=d),
                             "driver_sha256":{f.name:sha(f.read_bytes()) for f in sorted(d.glob("*.go"))}}
    data = {"base_commit":bsha,"sources":manifests,"binaries":binaries,
            "go":cmd("go","version",cwd=base),"host":cmd("uname","-srm",cwd=base),
            "sdk":cmd("xcrun","--show-sdk-version",cwd=base), "cc":cmd("cc","--version",cwd=base),
            "environment":{k:env[k] for k in ["CGO_ENABLED","GOGC","GOMAXPROCS","GOMEMLIMIT","GOWORK","GOTOOLCHAIN","GOENV","GOPROXY","GOFLAGS"]}}
    obj(build/"manifest.json",data)

def verify_commit(repo,revision,inventory):
    # Git hashes the shipped bytes for every ALAN build input. Track the tree
    # separately so docs-only follow-up commits can be independently attested.
    objects = cmd("git","ls-tree","-r",revision,cwd=repo)
    blobs={s.split("\t",1)[1]:s.split("\t",1)[0].split()[-1] for s in objects.splitlines()}
    for path,want in inventory.items():
        if path not in blobs:
            raise ValueError("missing committed input: "+path)
        b=subprocess.check_output(["git","cat-file","blob",blobs[path]],cwd=repo)
        if sha(b)!=want:
            raise ValueError("different committed input: "+path)

def calc_summary(rows):
    report = {}
    stages = ["construct_provision","create","execute_window","open_completed","replay","lifecycle"]
    for stage in stages:
        by={arm:[r["resources"][stage]["wall_ns"]/1e6 for r in rows if r["arm"]==arm] for arm in arms}
        report[stage]={k:{"median_ms":statistics.median(v),"range_ms":[min(v),max(v)]} for k,v in by.items()}
        for mode in ("full","progress"):
            ds=[(v/q-1)*100 for q,v in zip(by["base-"+mode],by["candidate-"+mode])]
            report[stage][mode+"_paired_percent"]={"median":statistics.median(ds),"values":ds}
    return report

def timing():
    manifest=json.loads((build/"manifest.json").read_text())
    samples=out/"samples"
    samples.mkdir()
    # Balanced Williams design: each arm appears in every position and
    # follows each other arm exactly once over the four measured blocks.
    orders=((0,1,3,2),(1,2,0,3),(2,3,1,0),(3,0,2,1))
    rows=[]
    for block,order in enumerate([(0,1,2,3),*orders]):
        for index in order:
            arm=arms[index]; lab,mode=arm.split("-")
            kind="warmup" if block==0 else "timing"
            name=f"{kind}-{block}-{arm}"
            d=samples/name
            d.mkdir()
            binary=build/arm/"driver"
            if sha(binary.read_bytes())!=manifest["binaries"][arm]["sha256"]:
                raise ValueError("binary changed: "+arm)
            label=manifest["base_commit"] if lab=="base" else "source-sha256:"+sha(json.dumps(manifest["sources"][lab],sort_keys=True).encode())
            store=d/"store/store.sqlite"
            command=[sys.executable,"-B",str(measure.P5/"sample_usage.py"),str(binary),"-bound","256",
                     "-dir",str(store.parent),"-sample",name,"-source-label",label,"-timeout","3m"]
            with (d/"report.json").open("wb") as f,(d/"log").open("wb") as e:
                p=subprocess.run(command,stdout=f,stderr=e,env=env,timeout=200,cwd=d)
            p.check_returncode()
            raw=json.loads((d/"report.json").read_text())
            spec={"name":name,"bound":256,"kind":kind,"arm":arm,"block":block}
            row=measure.validated_summary(raw,spec,label,mode,store,measure.usage_from_log(d/"log",0))
            row.update(arm=arm,block=block,command=command)
            obj(d/"checks.json",row)
            print(f"{name} passed {row['resources']['execute_window']['wall_ns']/1e9:.3f}s",flush=True)
            if block:
                rows.append(row)
    obj(out/"summary.json",calc_summary(rows))
    obj(out/"rows.json",rows)

if a.action=="prepare":
    prepare()
elif a.action=="attest":
    revision=cmd("git","rev-parse",a.revision+"^{commit}",cwd=candidate)
    manifest=json.loads((build/"manifest.json").read_text())
    # Re-enumerate every package's build inputs in the exact committed tree,
    # including imports or files added or removed after our initial snapshot.
    with tempfile.TemporaryDirectory(dir=out) as temp:
        exported = Path(temp)/"export"
        exported.mkdir()
        archive = Path(temp)/"source.tar"
        cmd("git","archive","--output",archive,revision,"go.mod","go.sum","author","schema","effects","runtime","internal","adapters","compose",cwd=candidate)
        cmd("tar","-xf",archive,"-C",exported,cwd=candidate)
        if sources(exported,Path(temp)/"selected") != manifest["sources"]["candidate"]:
            raise ValueError("release commit has different build inputs; build and time a fresh comparison")
    obj(out/"commit.json",{"commit":revision,
          "tree":cmd("git","rev-parse",revision+"^{tree}",cwd=candidate),
          "source_manifest_sha256":sha(json.dumps(manifest["sources"]["candidate"],sort_keys=True).encode())})
else:
    timing()
