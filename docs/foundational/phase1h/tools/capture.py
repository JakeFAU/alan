#!/usr/bin/env python3
"""Capture one exact source/command, retaining failures and refusing overwrite/drift."""

from pathlib import Path
import collections, datetime, gzip, io, json, os, platform, subprocess, sys, tarfile, time
from common import ROOT, DOC, inventory, source_id, environment, parse, sha

name = sys.argv[1]
command = sys.argv[2:]
if command and command[0] == "--":
    command = command[1:]
assert command and "/" not in name and name not in (".", "..")
out = Path(os.environ.get("ALAN_H_OUT", str(DOC / "evidence/task3/development"))) / name
out.mkdir(parents=True, exist_ok=False)
env = environment()
env["ALAN_H_ARTIFACTS"] = str(out / "artifacts")
files = inventory()
source = source_id(files)
(out / "sources.json").write_text(
    json.dumps(dict(SourceID=source, Files=files), indent=2) + "\n"
)
# Each development attempt keeps exact source bytes, not merely a hash of lost inputs.
with (out / "source.tar.gz").open("wb") as f:
    with gzip.GzipFile(fileobj=f, mode="wb", mtime=0, filename="") as z:
        with tarfile.open(fileobj=z, mode="w") as t:
            for name in sorted(files):
                raw = (ROOT / name).read_bytes()
                info = tarfile.TarInfo(name)
                info.size = len(raw)
                info.mode = 0o644
                info.mtime = 0
                t.addfile(info, io.BytesIO(raw))
started = datetime.datetime.now(datetime.timezone.utc).isoformat()
start = time.monotonic()
head = subprocess.run(
    ["git", "rev-parse", "HEAD"], cwd=ROOT, capture_output=True, text=True
)
meta = dict(
    SourceID=source,
    Command=command,
    Directory=str(ROOT),
    HeadAtStart=head.stdout.strip() if head.returncode == 0 else None,
    StartedUTC=started,
    Platform=platform.platform(),
    GoVersion=subprocess.check_output(
        ["go", "version"], cwd=ROOT, env=env, text=True
    ).strip(),
    GoEnvironment=json.loads(
        subprocess.check_output(
            ["go", "env", "-json", "GOOS", "GOARCH", "GOVERSION", "CGO_ENABLED", "CC"],
            cwd=ROOT,
            env=env,
        )
    ),
    Environment={
        k: env[k] for k in ("GOTOOLCHAIN", "GOFLAGS", "GOCACHE", "GOPROXY", "GOSUMDB")
    },
    CredentialAbsent="OPENAI_API_KEY" not in env,
)
try:
    with (
        (out / "stdout.running").open("wb") as stdout,
        (out / "stderr.running").open("wb") as stderr,
    ):
        p = subprocess.run(command, cwd=ROOT, env=env, stdout=stdout, stderr=stderr)
    raw, errors, code = (
        (out / "stdout.running").read_bytes(),
        (out / "stderr.running").read_bytes(),
        p.returncode,
    )
except Exception as e:
    raw, errors, code = b"", repr(e).encode(), 127
for label, value in (("stdout", raw), ("stderr", errors)):
    (out / (label + ".gz")).write_bytes(gzip.compress(value, mtime=0))
    (out / (label + ".running")).unlink(missing_ok=True)
after = inventory()
meta.update(
    ElapsedSeconds=round(time.monotonic() - start, 3),
    ExitCode=code,
    SourceUnchanged=files == after,
    SourceAfter=source_id(after),
    RawSHA256=sha(raw),
    StderrSHA256=sha(errors),
    SourceArchiveSHA256=sha((out / "source.tar.gz").read_bytes()),
)
if "-json" in command:
    try:
        result = parse(raw, source, os.environ.get("ALAN_H_MODE", "focused"))
        records = result.pop("records")
        meta.update(result)
        meta["Markers"] = dict(collections.Counter(r["Marker"] for r in records))
        (out / "records.jsonl.gz").write_bytes(
            gzip.compress(
                "".join(
                    json.dumps(r, separators=(",", ":")) + "\n" for r in records
                ).encode(),
                mtime=0,
            )
        )
    except Exception as e:
        meta["ParseError"] = repr(e)
(out / "command.json").write_text(json.dumps(meta, indent=2) + "\n")
print(json.dumps(meta, indent=2), flush=True)
if code or files != after or meta.get("ParseError"):
    sys.exit(code or 1)
