"""Build and run the independent reference; the saved argv arrays are reproducible."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[4]
MODEL = ROOT / "research/effectprotocol"
outcomes = []


def digest(file):
    return hashlib.sha256(Path(file).read_bytes()).hexdigest()


def run(argv, env, out=None):
    result = subprocess.run([str(x) for x in argv], cwd=ROOT, env=env,
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    data = dict(argv=[str(x) for x in argv], executable_sha256=digest(argv[0]),
                exit=result.returncode, stdout_sha256=hashlib.sha256(result.stdout).hexdigest(),
                stderr=result.stderr.decode())
    if out:
        (HERE / out).write_bytes(result.stdout)
        data["stdout_file"] = out
    else:
        data["stdout"] = result.stdout.decode()
    outcomes.append(data)
    if result.returncode:
        raise RuntimeError(data)


def main():
    go = Path(shutil.which("go")).resolve()
    gofmt = Path(shutil.which("gofmt")).resolve()
    env = {k: os.environ[k] for k in ("PATH", "HOME", "TMPDIR") if k in os.environ}
    env.update({"CGO_ENABLED": "1", "GOWORK": "off", "GOFLAGS": "-mod=readonly",
                "GOPROXY": "off", "GOSUMDB": "off", "GOTOOLCHAIN": "local",
                "GOCACHE": "/private/tmp/alan-phase2a-model-cache"})
    sources = sorted(MODEL.rglob("*.go"))
    inputs = [*sources, ROOT/"go.mod", ROOT/"go.sum",
              HERE.parent.parent/"02-protocol.md", Path(__file__)]
    expected = {str(p.relative_to(ROOT)): digest(p) for p in inputs}
    report = {"cwd": str(ROOT), "environment": env, "sources": expected, "commands": outcomes}
    package = "./research/effectprotocol"
    with tempfile.TemporaryDirectory(prefix="alan-phase2a-model-") as tmp:
        try:
            run([go, "version"], env)
            run([go, "env", "GOOS", "GOARCH", "CGO_ENABLED", "GOROOT", "GOMODCACHE", "CC"], env)
            run([gofmt, "-l", *sources], env)
            assert outcomes[-1]["stdout"] == ""
            run([go, "vet", package+"/..."], env)
            for name, flags in [("normal", []), ("race", ["-race"])]:
                binary = Path(tmp) / ("reference-"+name)
                run([go,"test","-c",*flags,"-o",binary,package],env)
                run([go,"version","-m",binary],env)
                run([binary,"-test.v","-test.count=1"],env,name+".txt")
            exporter = Path(tmp)/"export"
            run([go,"build","-o",exporter,package+"/cmd/export"],env)
            run([exporter,HERE/"graphs.jsonl.gz"],env,"summary.json")
            report["graphs_sha256"] = digest(HERE/"graphs.jsonl.gz")
        finally:
            (HERE/"tool-record.json").write_text(json.dumps(report, indent=2)+"\n")


if __name__ == "__main__":
    main()
