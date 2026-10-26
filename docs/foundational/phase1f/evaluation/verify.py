#!/usr/bin/env python3
"""Verify frozen fixtures; never print question, gold, or witness-query contents."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import tarfile


def digest(data):
    return hashlib.sha256(data).hexdigest()


def read(path):
    return json.loads(path.read_text(encoding="utf-8"))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--sealed", action="store_true", help="Independent custodian only before unseal")
    args = parser.parse_args()
    public = Path(__file__).resolve().parent
    root = public.parents[3]
    sealed = root / ".superpowers/sdd/00-plan/sealed-evaluation"
    manifest = read(public / "corpus/manifest.json")
    documents = {}
    lines = []
    for item in manifest["documents"]:
        raw = (public / item["path"]).read_bytes()
        assert digest(raw) == item["sha256"]
        assert item["version"] == digest(raw), "Runtime asset version must equal content SHA-256"
        assert item["editorial_revision"] == "1"
        assert len(raw) == item["bytes"] and raw.endswith(b"\n")
        assert item["id"] not in documents
        documents[item["id"]] = (item["version"], raw)
        offset = 0
        for number, line in enumerate(raw.splitlines(keepends=True), 1):
            assert line.endswith(b"\n") and b"\r" not in line
            snippet = line[:-1]
            span = {"line": number, "start": offset, "end": offset + len(snippet)}
            assert span == item["line_spans"][number - 1]
            assert 0 < len(snippet) <= 2048
            lines.append((item["id"], offset, snippet.decode("utf-8")))
            offset += len(line)
        assert number == len(item["line_spans"])
    # ASCII fixtures make this tokenizer identical to the adapter's Unicode
    # letter/number tokenizer for these bytes. This is a reachability check,
    # not an execution test of the Go adapter or a prediction of model queries.
    assert all(text.isascii() for _, _, text in lines)
    tokens = lambda text: re.findall(r"[a-z0-9]+", text.lower())
    ordered = sorted(lines, key=lambda line: (line[0], line[1]))
    reachable = 0
    for doc, start, text in lines:
        query = list(dict.fromkeys(tokens(text)))[:32]
        assert len(" ".join(query).encode()) <= 512
        scored = [(len(set(query) & set(tokens(line[2]))), line) for line in ordered]
        best = max(score for score, _ in scored)
        winner = next(line for score, line in scored if score == best)
        assert winner[:2] == (doc, start), "Corpus line lacks canonical lexical witness"
        reachable += 1
    splits = [(public / "development-inputs.json", public / "development-gold.json")]
    if args.sealed:
        splits.append((sealed / "inputs.json", sealed / "gold.json"))
    checked = 0
    for inputs_path, gold_path in splits:
        inputs, gold = read(inputs_path), read(gold_path)
        cases, answers = inputs["cases"], gold["cases"]
        assert len(cases) == len(answers) == 4
        assert [x["id"] for x in cases] == [x["id"] for x in answers]
        assert len(set(x["id"] for x in cases)) == 4
        assert sorted(x["category"] for x in answers) == ["ambiguity", "contradiction", "missing_evidence", "supported"]
        for case, answer in zip(cases, answers):
            assert set(case) == {"id", "question"}
            assert len(case["question"].encode()) <= 4096
            assert len(answer["aspects"]) == 2
            assert answer["stopping_rule"] and answer["forbidden_claims"]
            fact_ids = [x["id"] for x in answer["required_facts"]]
            assert len(fact_ids) == len(set(fact_ids))
            for fact in answer["required_facts"] + answer["optional_facts"]:
                assert fact["evidence"] and fact["claim"]
                assert set(fact.get("required_for_modes", [])) <= set(answer["acceptable_semantic_modes"])
                for ref in fact["evidence"]:
                    version, raw = documents[ref["document"]]
                    assert ref["version"] == version
                    assert 0 <= ref["start"] < ref["end"] <= len(raw)
                    assert raw[ref["start"]:ref["end"]].decode() == ref["snippet"]
                    assert raw[:ref["start"]].count(b"\n") + 1 == ref["line"]
            checked += 1
    # Prove the correction is confined to metadata, never semantic tuning.
    with tarfile.open(public / "history/v1-public.tar.gz", "r:gz") as archive:
        original = {member.name: archive.extractfile(member).read()
                    for member in archive.getmembers() if member.isfile()}
    original_freeze = json.loads(original["freeze-manifest.json"])
    correction = read(public / "history/v2-correction.json")
    assert digest(original["freeze-manifest.json"]) == correction["original_freeze_identity"]
    for entry in original_freeze["public_files"]:
        assert digest(original[entry["path"]]) == entry["sha256"]
    assert (public / "development-inputs.json").read_bytes() == original["development-inputs.json"]
    for doc in manifest["documents"]:
        assert (public / doc["path"]).read_bytes() == original[doc["path"]]
    def without_reference_versions(value):
        value = json.loads(json.dumps(value))
        for case in value["cases"]:
            for fact in case["required_facts"] + case["optional_facts"]:
                for ref in fact["evidence"]:
                    ref.pop("version")
        return value
    assert without_reference_versions(read(public / "development-gold.json")) == without_reference_versions(json.loads(original["development-gold.json"]))
    if args.sealed:
        retained = root / ".superpowers/sdd/00-plan/evaluation-freeze-v1/sealed-evaluation"
        assert digest((retained / "manifest.json").read_bytes()) == correction["original_sealed_commitment"]
        for entry in read(retained / "manifest.json")["files"]:
            assert digest((retained / entry["path"]).read_bytes()) == entry["sha256"]
        assert (sealed / "inputs.json").read_bytes() == (retained / "inputs.json").read_bytes()
        assert without_reference_versions(read(sealed / "gold.json")) == without_reference_versions(read(retained / "gold.json"))
    frozen = public / "freeze-manifest.json"
    identity = None
    if frozen.exists():
        freeze = read(frozen)
        for entry in freeze["public_files"]:
            assert digest((public / entry["path"]).read_bytes()) == entry["sha256"], "Public fixture drift"
        if args.sealed:
            sealed_manifest = sealed / "manifest.json"
            assert digest(sealed_manifest.read_bytes()) == freeze["sealed_commitment"]["sha256"]
            for entry in read(sealed_manifest)["files"]:
                assert digest((sealed / entry["path"]).read_bytes()) == entry["sha256"], "Sealed fixture drift"
        identity = digest(frozen.read_bytes())
    print(json.dumps({"ok": True, "documents": len(documents), "lines": len(lines), "lexically_reachable_lines": reachable, "checked_cases": checked, "sealed_checked": args.sealed, "freeze_identity": identity}, sort_keys=True))


if __name__ == "__main__":
    main()
