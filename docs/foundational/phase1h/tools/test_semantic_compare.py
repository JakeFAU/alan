"""Negative controls for the explicit provenance/behavior comparison boundary."""

import copy
import gzip
import hashlib
import json
import unittest

from common import ROOT
from semantic_compare import NEW_AGGREGATE, compare_deterministic


def paired(marker):
    path = ROOT / "docs/foundational/phase1e/evidence/final-default.records.jsonl.gz"
    before = [
        r["Evidence"]
        for r in [
            json.loads(line) for line in gzip.decompress(path.read_bytes()).splitlines()
        ]
        if r["Marker"] == marker
    ]
    after = copy.deepcopy(before)
    manifest = after[0]["Manifest"]
    # Build only the documented source mapping, never change any model result.
    for entry in manifest["Files"]:
        raw = (ROOT / entry["Path"]).read_bytes()
        entry["Bytes"], entry["SHA256"] = len(raw), hashlib.sha256(raw).hexdigest()
    manifest["SHA256"] = NEW_AGGREGATE
    for record in after[1:]:
        record["SourceID"] = NEW_AGGREGATE
    return before, after


class ProvenanceBoundary(unittest.TestCase):
    def test_exact_pinned_mapping_accepts_unchanged_behavior(self):
        for marker in ("COMPOSITION", "SUPERVISION"):
            before, after = paired(marker)
            result = compare_deterministic(marker, before, after, ROOT)
            self.assertTrue(result["OrderedBehaviorEqual"])
            self.assertTrue(result["ProvenanceValidatedSeparately"])

    def test_manifest_aggregate_tamper_rejected(self):
        before, after = paired("COMPOSITION")
        after[0]["Manifest"]["SHA256"] = "0" * 64
        with self.assertRaisesRegex(ValueError, "aggregate"):
            compare_deterministic("COMPOSITION", before, after, ROOT)

    def test_manifest_file_digest_tamper_rejected(self):
        before, after = paired("COMPOSITION")
        after[0]["Manifest"]["Files"][0]["SHA256"] = "0" * 64
        with self.assertRaisesRegex(ValueError, "file bytes"):
            compare_deterministic("COMPOSITION", before, after, ROOT)

    def test_inner_source_forgery_rejected(self):
        before, after = paired("COMPOSITION")
        after[1]["SourceID"] = "forged"
        with self.assertRaisesRegex(ValueError, "SourceID"):
            compare_deterministic("COMPOSITION", before, after, ROOT)

    def test_behavior_order_and_count_changes_rejected(self):
        for change in ("states", "order", "count"):
            with self.subTest(change=change):
                before, after = paired("COMPOSITION")
                if change == "states":
                    after[1]["States"] += 1
                elif change == "order":
                    after[1], after[2] = after[2], after[1]
                else:
                    after.pop()
                with self.assertRaises(ValueError):
                    compare_deterministic("COMPOSITION", before, after, ROOT)


if __name__ == "__main__":
    unittest.main()
