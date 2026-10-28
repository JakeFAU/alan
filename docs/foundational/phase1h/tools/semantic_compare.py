"""Validate the exact merged bridge edit before comparing historical behavior."""

import hashlib
import struct

BRIDGE = "research/semantics/runtime_conformance_test.go"
BEFORE = "f2f3588736b0e42ad013a3640a3a328c18ca9043454862bdf68c5b08ba418cc5"
AFTER = "ac85028e1360e2b0ee9c8a1b7ddd925454ce4b900dd3a0843a0b04cbd1be4dfc"
OLD_AGGREGATE = "ad688755b8580924a9604936e9c910f4403c8457ead390024a80ebe2c473f940"
NEW_AGGREGATE = "161b8874d0f75300f77f5a5c408bb8f38a25c5356918f74344e2f75b68f47343"
FIXTURE = "docs/foundational/phase1h/evidence/task3/comparison-inputs/runtime_conformance-before.go.txt"
ALGORITHM = "SHA-256 of sorted repository paths: uint64-BE path byte length, UTF-8 path bytes, uint64-BE file byte length, raw file bytes; go.mod and research/semantics/*_test.go only; evidence excluded"
OLD_LINE = b'\tp.Permissions = append(p.Permissions, ex.Permission{Kind: "external", Source: "worker", Target: server.URL, Operation: "apply", Revision: 7}, ex.Permission{Kind: "input", Source: "local", Target: "worker", Operation: "input"})\n'
NEW_LINE = OLD_LINE.replace(
    b'Operation: "input"}', b'Operation: "input", Obligation: "control"}'
)


def require(condition, detail):
    if not condition:
        raise ValueError(detail)


def bridge_bytes(root):
    before, after = (root / FIXTURE).read_bytes(), (root / BRIDGE).read_bytes()
    require(
        hashlib.sha256(before).hexdigest() == BEFORE, "pinned historical bridge bytes"
    )
    require(hashlib.sha256(after).hexdigest() == AFTER, "pinned current bridge bytes")
    require(before.count(OLD_LINE) == 1, "unique original permission line")
    require(
        before.replace(OLD_LINE, NEW_LINE, 1) == after,
        "only exact Obligation control addition",
    )
    return before, after


def validate_manifest(manifest, root, historical):
    require(
        set(manifest) == {"Algorithm", "SHA256", "Files"}, "semantic manifest fields"
    )
    require(manifest["Algorithm"] == ALGORITHM, "semantic manifest algorithm")
    paths = sorted(
        ["go.mod"]
        + [
            str(p.relative_to(root))
            for p in (root / "research/semantics").glob("*_test.go")
        ]
    )
    require(
        [e["Path"] for e in manifest["Files"]] == paths,
        "semantic manifest exact path set/order",
    )
    before, after = bridge_bytes(root)
    aggregate = hashlib.sha256()
    for entry in manifest["Files"]:
        require(
            set(entry) == {"Path", "SHA256", "Bytes"}, "semantic file descriptor fields"
        )
        path = entry["Path"]
        raw = (
            (before if historical else after)
            if path == BRIDGE
            else (root / path).read_bytes()
        )
        require(
            len(raw) == entry["Bytes"]
            and hashlib.sha256(raw).hexdigest() == entry["SHA256"],
            "semantic file bytes: " + path,
        )
        name = path.encode()
        aggregate.update(
            struct.pack(">Q", len(name)) + name + struct.pack(">Q", len(raw)) + raw
        )
    expected = OLD_AGGREGATE if historical else NEW_AGGREGATE
    require(
        aggregate.hexdigest() == manifest["SHA256"] == expected,
        "pinned semantic aggregate",
    )


def compare_deterministic(marker, before, after, root):
    count = {"COMPOSITION": 20, "SUPERVISION": 32}[marker]
    require(len(before) == len(after) == count, "semantic count")
    for records, historical in ((before, True), (after, False)):
        require(
            set(records[0]) == {"Name", "Manifest"}
            and records[0]["Name"] == "source_manifest",
            "semantic manifest header",
        )
        manifest = records[0]["Manifest"]
        validate_manifest(manifest, root, historical)
        require(
            all(r.get("SourceID") == manifest["SHA256"] for r in records[1:]),
            "every semantic record SourceID",
        )
    old, new = before[0]["Manifest"], after[0]["Manifest"]
    changed = [
        dict(Path=a["Path"], Before=a, After=b)
        for a, b in zip(old["Files"], new["Files"])
        if a != b
    ]
    require(
        [e["Path"] for e in changed] == [BRIDGE],
        "only pinned upstream bridge may differ",
    )
    # Only now omit the separately verified header and per-record source identity.
    a = [{k: v for k, v in r.items() if k != "SourceID"} for r in before[1:]]
    b = [{k: v for k, v in r.items() if k != "SourceID"} for r in after[1:]]
    require(a == b, marker + " exact ordered behavior")
    return dict(
        Marker=marker,
        Records=count,
        BehaviorRecords=count - 1,
        OrderedBehaviorEqual=True,
        HistoricalManifest=old,
        CurrentManifest=new,
        ChangedFiles=changed,
        ProvenanceValidatedSeparately=True,
        Scope="Exact ordered behavior after independent validation of both complete source manifests and every inner SourceID. Only the pinned one-line upstream control-obligation addition is mapped. Not historical raw-payload or source-byte reproduction.",
    )
