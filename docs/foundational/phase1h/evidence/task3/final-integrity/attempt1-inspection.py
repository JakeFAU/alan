"""Generated post-capture inspection, not an input of frozen Source770."""

import collections, gzip, hashlib, json, sys, tarfile
from pathlib import Path

R = Path("/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan")
D = R / "docs/foundational/phase1h"
sys.path.insert(0, str(D / "tools"))
from common import inventory, source_id, parse

sha = lambda b: hashlib.sha256(b).hexdigest()
m = json.loads((D / "source-manifest.json").read_text())
frozen = m["SourceID"]
assert inventory() == m["Files"] and source_id(m["Files"]) == frozen
am = json.loads((D / "archive-manifest.json").read_text())
ar = (D / "source.tar.gz").read_bytes()
assert sha(ar) == am["SHA256"] and len(ar) == am["Bytes"] and am["SourceID"] == frozen
with tarfile.open(D / "source.tar.gz") as t:
    members = t.getmembers()
    names = [x.name for x in members]
    assert len(names) == len(set(names)) == len(am["Files"])
    assert names == [x["Path"] for x in am["Files"]]
    for v in am["Files"]:
        x = t.getmember(v["Path"])
        p = Path(x.name)
        assert x.isfile() and not p.is_absolute() and ".." not in p.parts
        b = t.extractfile(x).read()
        assert len(b) == v["Bytes"] and sha(b) == v["SHA256"]
    for p, h in m["Files"].items():
        assert sha(t.extractfile(p).read()) == h
captures = []
retained = 0
for f in sorted((D / "evidence/task3/final").rglob("command.json")):
    c = json.loads(f.read_text())
    p = f.parent
    s = json.loads((p / "sources.json").read_text())
    assert source_id(s["Files"]) == s["SourceID"] == c["SourceID"]
    assert (
        c["SourceUnchanged"]
        and c["SourceAfter"] == c["SourceID"]
        and c["CredentialAbsent"]
    )
    assert sha((p / "source.tar.gz").read_bytes()) == c["SourceArchiveSHA256"]
    with tarfile.open(p / "source.tar.gz") as t:
        names = [x.name for x in t.getmembers()]
        assert len(names) == len(set(names)) and set(names) == set(s["Files"])
        for name, h in s["Files"].items():
            x = t.getmember(name)
            assert x.isfile() and sha(t.extractfile(x).read()) == h
    raw = gzip.decompress((p / "stdout.gz").read_bytes())
    err = gzip.decompress((p / "stderr.gz").read_bytes())
    assert sha(raw) == c["RawSHA256"] and sha(err) == c["StderrSHA256"]
    if "-json" in c["Command"]:
        rows = [
            json.loads(x)
            for x in gzip.decompress((p / "records.jsonl.gz").read_bytes()).splitlines()
        ]
        mode = rows[0]["Mode"] if rows else "focused"
        parsed = parse(raw, c["SourceID"], mode)
        assert parsed["records"] == rows and all(
            parsed[k] == c[k] for k in ("passes", "failures", "packages")
        )
        for row in rows:
            v = row["Evidence"]
            if isinstance(v, dict) and v.get("kind") == "retained_artifact":
                q = Path(v["path"])
                assert q.is_relative_to(R)
                b = q.read_bytes()
                assert (
                    sha(b) == v["gzip_sha256"]
                    and sha(gzip.decompress(b)) == v["value_sha256"]
                )
                retained += 1
    captures.append(
        dict(
            Capture=str(p.relative_to(D)),
            ExitCode=c["ExitCode"],
            SourceID=c["SourceID"],
        )
    )
expected_failures = {
    "final-build-old-consumer",
    "final-consumer-race",
    "guard-compare-default-original-failure",
}
assert {Path(x["Capture"]).name for x in captures if x["ExitCode"]} == expected_failures
for mode in ("normal", "race"):
    p = D / "evidence/task3/final" / ("final-" + mode)
    c = json.loads((p / "command.json").read_text())
    assert c["SourceID"] == frozen and c["ExitCode"] == 0 and not c["failures"]
    assert (
        c["Markers"]["ADAPTIVE_BYTES"] == 12
        and c["Markers"]["ADAPTIVE_ACCEPTANCE_BOUNDARY"] == 2
    )
    assert b"WARNING: DATA RACE" not in gzip.decompress(
        (p / "stdout.gz").read_bytes()
    ) + gzip.decompress((p / "stderr.gz").read_bytes())
    q = json.loads((p / "comparison.json").read_text())
    assert q["SourceID"] == frozen and q["Mode"] == mode
    assert [v["Records"] for v in q["Comparisons"]] == [15, 20, 32, 6]
c = json.loads((D / "evidence/task3/final/final-default/command.json").read_text())
assert (
    c["SourceID"] == "4a92f31979d30e322b2afa195d8da6eb71333ab70c02029e8870189f73e0f5af"
    and c["ExitCode"] == 0
)
c = json.loads(
    (D / "evidence/task3/final/covering/final-consumer-race/command.json").read_text()
)
assert c["SourceID"] == frozen and c["ExitCode"] == 0 and len(c["passes"]) == 12
p = D / "evidence/task3/extracted"
e = json.loads((p / "reproduction.json").read_text())
assert (
    e["Result"] == "PASS"
    and e["SourceID"] == frozen
    and e["ArchiveSHA256"] == am["SHA256"]
)
assert (
    e["SourceUnchanged"]
    and e["InitialModuleCacheEmpty"]
    and e["CredentialAbsent"]
    and e["OriginalSemanticComparisons"] == "PASS"
)
assert len(e["Commands"]) == 6
for i, c in enumerate(e["Commands"], 1):
    assert c["ExitCode"] == 0
    assert (
        sha(gzip.decompress((p / f"{i:02d}.stdout.gz").read_bytes())) == c["RawSHA256"]
    )
    assert (
        sha(gzip.decompress((p / f"{i:02d}.stderr.gz").read_bytes()))
        == c["StderrSHA256"]
    )
for stem, n in [("04", 7), ("05", 8)]:
    actual = [
        json.loads(x)
        for x in gzip.decompress((p / f"{stem}.stdout.gz").read_bytes()).splitlines()
    ]
    passed = {x.get("Test") for x in actual if x.get("Action") == "pass"}
    assert (
        len(e["RequiredActualPasses"][stem]) == n
        and set(e["RequiredActualPasses"][stem]) <= passed
    )
    assert not [x for x in actual if x.get("Action") == "fail"]
reviews = 0
for p in (D / "evidence/task3").glob("final-review-*/provenance.json"):
    for v in json.loads(p.read_text())["Files"]:
        b = (R / v["Retained"]).read_bytes()
        assert len(b) == v["Bytes"] and sha(b) == v["SHA256"]
        reviews += 1
print(
    json.dumps(
        dict(
            Result="PASS",
            FrozenSourceID=frozen,
            Inputs=len(m["Files"]),
            ArchiveSHA256=am["SHA256"],
            ArchiveMembers=len(am["Files"]),
            CapturedCommands=len(captures),
            PreservedFailedCommands=sorted(expected_failures),
            RetainedArtifactReferencesVerified=retained,
            ExtractedCommands=6,
            ExtractedRequiredPasses=[7, 8],
            GeneratedReviewArtifactsVerified=reviews,
            Scope="Post-capture byte/hash/source/actual-event integrity inspection; no new Go execution and no universal semantic proof",
        ),
        indent=2,
    )
)
