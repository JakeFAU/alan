#!/usr/bin/env python3
"""Tiny synthetic tests only; never starts an ALAN/Go workload."""
import copy
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import analyze_pairs
from campaign_validation import (BOUNDS, ENVIRONMENT, SINGLETON_PHASES, sha,
                                 validate_meta, validate_report, validate_schedule)
from make_schedule import build_schedule

HERE = Path(__file__).resolve().parent
RUNTIME = {"go_version": "go1.27.3", "goos": "darwin", "goarch": "arm64",
           "gomaxprocs": 4, "mem_profile_rate": 524288, "profiled": False,
           "instrumented": False, "schedule": "ordinary", "reopen_every": 0}


class CampaignProof(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(prefix="alan-campaign-proof.")
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name)
        self.binaries = []
        for name in ("baseline", "candidate"):
            binary = self.root / name
            binary.write_bytes((name + " frozen executable bytes").encode())
            self.binaries.append(binary)
        self.rows = build_schedule(*self.binaries, "baseline-source", "candidate-source", runtime=RUNTIME)
        self.schedule = self.root / "schedule.json"
        self.schedule.write_text(json.dumps(self.rows))
        self.samples = self.root / "samples"
        self.samples.mkdir()

    def report(self, row):
        n = row["bound"]
        report = dict(RUNTIME, version="alan/series-scaling/v1", invariants_ok=True,
                      bound=n, sample=row["name"], source_label=row["source"],
                      store=str(self.samples / (row["name"] + ".store") / "store.sqlite"),
                      completed=n, selected=n, spent=n, protected_approvals=n-1,
                      contacts=[{"ordinal": i, "operation_id": "operation-" + str(i)} for i in range(1, n+1)],
                      evidence=[dict(index=i, operation_id="operation-" + str(i),
                                     **({"approval": {"present": True}} if i > 1 else {})) for i in range(1, n+1)])
        stages = []
        for phase, positions in [(p, [0]) for p in SINGLETON_PHASES] + [
                ("run", range(1, n+1)), ("approval_snapshot_basis", range(2, n+1)),
                ("approval_establishment", range(2, n+1))]:
            for position in positions:
                stages.append(dict(name=phase, position=position, wall_ns=1_000_000,
                                   user_cpu_ns=1000, system_cpu_ns=1000,
                                   total_alloc_bytes=1000, mallocs=10))
        execution = next(s for s in stages if s["name"] == "execute_window")
        seconds = n * (row["pair"] + 10) * (1 if row["variant"] == "baseline" else .5)
        execution["wall_ns"] = int(seconds * 1e9) if not row["warmup"] else int(1000*n*1e9)
        report["stages"] = stages
        return report

    def write(self, row, report=None, edit_meta=None):
        name = row["name"]
        report = report if report is not None else self.report(row)
        raw = self.samples / (name + ".json")
        raw.write_text(json.dumps(report))
        progress = self.samples / (name + ".progress")
        progress.write_text("synthetic progress " + name)
        meta = {"exit": 0, "schedule_sha256": sha(self.schedule), "schedule_row": row,
                "report_sha256": sha(raw), "progress_sha256": sha(progress),
                "binary_sha256": row["expected_binary_sha256"],
                "binary_sha256_after": row["expected_binary_sha256"],
                "runner_sha256": row["expected_tooling_sha256"]["run_schedule.py"],
                "validation_sha256": row["expected_tooling_sha256"]["campaign_validation.py"],
                "environment": ENVIRONMENT.copy(), "process_peak_rss": 1234,
                "process_peak_rss_unit": "bytes",
                "command": [row["binary"], "-bound", str(row["bound"]), "-dir", str(Path(report["store"]).parent),
                            "-sample", name, "-source-label", row["source"]]}
        if edit_meta:
            edit_meta(meta)
        (self.samples / (name + ".meta.json")).write_text(json.dumps(meta))
        return meta

    def analyze(self):
        directory = Path(os.environ.get("CAMPAIGN_ANALYZER_UNDER_TEST", str(HERE)))
        output = self.root / "analysis.json"
        result = subprocess.run([sys.executable, str(directory / "analyze_pairs.py"),
                                 str(self.schedule), str(self.samples), str(output)],
                                capture_output=True, text=True)
        return result, json.loads(output.read_text()) if result.returncode == 0 else None

    def write_all(self):
        for row in self.rows:
            self.write(row)

    def measured(self):
        return next(r for r in self.rows if not r["warmup"] and r["bound"] == 8 and r["variant"] == "baseline")

    def test_schedule_coverage_and_adjacent_ab_ba(self):
        validate_schedule(self.rows)
        self.assertEqual(len(self.rows), 98)
        self.assertEqual(len({r["name"] for r in self.rows}), 98)
        self.assertEqual(sum(r["warmup"] for r in self.rows), 14)
        self.assertEqual({(r["bound"], r["variant"]) for r in self.rows[:14]},
                         {(n, v) for n in BOUNDS for v in ("baseline", "candidate")})
        for i in range(14, 98, 2):
            a, b = self.rows[i:i+2]
            self.assertEqual((a["bound"], a["pair"]), (b["bound"], b["pair"]))
            self.assertEqual([a["variant"], b["variant"]],
                             ["baseline", "candidate"] if a["pair"] % 2 else ["candidate", "baseline"])

    def test_missing_pair_and_bad_order_rejected(self):
        with self.assertRaises(ValueError):
            validate_schedule(self.rows[:-1])
        rows = copy.deepcopy(self.rows)
        rows[14], rows[15] = rows[15], rows[14]
        with self.assertRaises(ValueError):
            validate_schedule(rows)
        rows = copy.deepcopy(self.rows)
        rows[2] = rows[0]
        with self.assertRaises(ValueError):
            validate_schedule(rows)

    def test_runtime_and_variant_schedule_consistency(self):
        for key, value in [("source", "different-source"), ("expected_binary_sha256", "0"*64),
                           ("expected_environment", dict(ENVIRONMENT, GOGC="off")),
                           ("expected_runtime", dict(RUNTIME, mem_profile_rate=1))]:
            with self.subTest(key=key):
                rows = copy.deepcopy(self.rows)
                rows[20][key] = value
                with self.assertRaises(ValueError):
                    validate_schedule(rows)

    def test_changed_executable_rejected_before_launch(self):
        self.binaries[0].write_bytes(b"replacement")
        result = subprocess.run([sys.executable, str(HERE / "run_schedule.py"),
                                 str(self.schedule), str(self.samples)], capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("executable changed before sample", result.stderr)
        self.assertEqual(list(self.samples.iterdir()), [])

    def test_binary_changed_during_sample_rejected_and_retained(self):
        binary = self.binaries[0]
        binary.write_text("#!/usr/bin/env python3\nimport pathlib\np=pathlib.Path(__file__)\np.write_text(p.read_text()+'\\n# replacement')\nprint('{}')\n")
        binary.chmod(0o755)
        self.rows = build_schedule(*self.binaries, "baseline-source", "candidate-source", runtime=RUNTIME)
        self.schedule.write_text(json.dumps(self.rows))
        result = subprocess.run([sys.executable, str(HERE / "run_schedule.py"),
                                 str(self.schedule), str(self.samples)], capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        meta = json.loads((self.samples / (self.rows[0]["name"]+".meta.json")).read_text())
        self.assertIn("executable changed during sample", meta["validation_error"])
        self.assertEqual(meta["binary_sha256"], self.rows[0]["expected_binary_sha256"])
        self.assertNotEqual(meta["binary_sha256_after"], meta["binary_sha256"])

    def assert_postrun_binary_read_failure_retained(self, action, error_text):
        binary = self.binaries[0]
        binary.write_text("#!/usr/bin/env python3\nimport pathlib\np=pathlib.Path(__file__)\n" + action + "\nprint('{}')\n")
        binary.chmod(0o755)
        self.rows = build_schedule(*self.binaries, "baseline-source", "candidate-source", runtime=RUNTIME)
        self.schedule.write_text(json.dumps(self.rows))
        result = subprocess.run([sys.executable, str(HERE / "run_schedule.py"),
                                 str(self.schedule), str(self.samples)], capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        name = self.rows[0]["name"]
        path = self.samples / (name + ".meta.json")
        self.assertTrue(path.exists(), "post-run executable read failure discarded completed process metadata")
        meta = json.loads(path.read_text())
        self.assertEqual(meta["exit"], 0)
        self.assertIn(error_text, meta["validation_error"])
        self.assertEqual(meta["binary_sha256"], self.rows[0]["expected_binary_sha256"])
        self.assertNotIn("binary_sha256_after", meta)
        for field in ("seconds", "user_cpu_seconds", "system_cpu_seconds", "process_peak_rss"):
            self.assertGreaterEqual(meta[field], 0)
        self.assertEqual(meta["report_sha256"], sha(self.samples / (name + ".json")))
        self.assertEqual(meta["progress_sha256"], sha(self.samples / (name + ".progress")))
        self.assertEqual(json.loads((self.samples / (name + ".json")).read_text()), {})
        self.assertIn('"invariants_ok": false', result.stdout)
        self.assertFalse((self.samples / (self.rows[1]["name"] + ".json")).exists())

    def test_binary_deleted_during_sample_retains_invalid_process_metadata(self):
        self.assert_postrun_binary_read_failure_retained("p.unlink()", "No such file")

    @unittest.skipIf(os.geteuid() == 0, "root can read mode-000 files")
    def test_binary_unreadable_after_sample_retains_invalid_process_metadata(self):
        self.assert_postrun_binary_read_failure_retained("p.chmod(0)", "Permission denied")

    def test_changed_progress_artifact_rejected(self):
        self.write_all()
        (self.samples / (self.measured()["name"]+".progress")).write_text("changed after metadata")
        result, _ = self.analyze()
        self.assertNotEqual(result.returncode, 0, "changed progress artifact was silently accepted")

    def test_valid_full_campaign_warmups_excluded_and_ratios(self):
        self.write_all()
        result, output = self.analyze()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(output["warmups_excluded"], 14)
        for item in output["bounds"]:
            self.assertEqual(item["metrics"]["baseline"]["execute_seconds"]["median"], 13.5*item["bound"])
            self.assertEqual(item["median_execution_reduction"], .5)
            self.assertEqual(item["paired_ratios"]["values"], [.5]*6)
        self.assertEqual(output["bounds"][-1]["primary_sign_test"]["two_sided_exact_p"], .03125)

    def test_missing_required_phase_rejected(self):
        self.write_all()
        row = self.measured()
        report = self.report(row)
        report["stages"] = [s for s in report["stages"] if s["name"] != "final_inspection"]
        self.write(row, report)
        result, _ = self.analyze()
        self.assertNotEqual(result.returncode, 0, "missing final_inspection was silently accepted")

    def test_changed_metadata_binary_rejected(self):
        self.write_all()
        self.write(self.measured(), edit_meta=lambda m: m.update(binary_sha256="0"*64))
        result, _ = self.analyze()
        self.assertNotEqual(result.returncode, 0, "changed executable metadata was silently accepted")

    def test_wrong_source_label_rejected(self):
        self.write_all()
        row = self.measured()
        report = self.report(row)
        report["source_label"] = "different-source"
        self.write(row, report)
        result, _ = self.analyze()
        self.assertNotEqual(result.returncode, 0, "wrong source identity was silently accepted")

    def test_duplicate_wrong_error_and_unknown_phases(self):
        row = self.measured()
        for mutation in ("duplicate", "position", "error", "unknown"):
            with self.subTest(mutation=mutation):
                report = self.report(row)
                if mutation == "duplicate": report["stages"].append(report["stages"][0])
                if mutation == "position": next(s for s in report["stages"] if s["name"] == "run")["position"] = 8
                if mutation == "error": report["stages"][0]["error"] = "failed"
                if mutation == "unknown": report["stages"].append(dict(report["stages"][0], name="interval_close_reopen_resume"))
                with self.assertRaises(ValueError): validate_report(report, row)

    def test_report_identity_configuration_and_counts(self):
        row = self.measured()
        for key, value in [("sample", "wrong"), ("bound", 1), ("source_label", "wrong"),
                           ("go_version", "go1.28"), ("goos", "linux"), ("goarch", "amd64"),
                           ("gomaxprocs", 1), ("mem_profile_rate", 1), ("profiled", True),
                           ("instrumented", True), ("schedule", "recovery"), ("reopen_every", 1),
                           ("selected", 0), ("protected_approvals", 0), ("contacts", [])]:
            with self.subTest(key=key):
                report = self.report(row)
                report[key] = value
                with self.assertRaises(ValueError): validate_report(report, row)

    def test_metadata_integrity_environment_runner_progress_and_rss(self):
        row = self.measured()
        report = self.report(row)
        original = self.write(row, report)
        for key, value in [("report_sha256", "0"*64), ("progress_sha256", "0"*64),
                           ("runner_sha256", "0"*64), ("binary_sha256_after", "0"*64),
                           ("environment", dict(ENVIRONMENT, GODEBUG="memprofilerate=1")),
                           ("process_peak_rss_unit", "unknown")]:
            with self.subTest(key=key):
                meta = copy.deepcopy(original)
                meta[key] = value
                with self.assertRaises(ValueError):
                    validate_meta(meta, report, row, sha(self.schedule),
                                  sha(self.samples/(row["name"]+".json")), sha(self.samples/(row["name"]+".progress")))

    def test_sign_test_and_growth_math(self):
        self.assertEqual(analyze_pairs.sign_test([.5]*6)["two_sided_exact_p"], .03125)
        self.assertEqual(analyze_pairs.sign_test([.5]*3+[2]*3)["two_sided_exact_p"], 1)
        self.assertEqual(analyze_pairs.sign_test([1]*6)["two_sided_exact_p"], 1)
        self.assertEqual(analyze_pairs.sign_test([.5]*5+[1])["two_sided_exact_p"], .0625)
        model = analyze_pairs.fit_growth(BOUNDS, [5+3*n*n for n in BOUNDS])["finite_models"]["constant_plus_n_to_2"]
        self.assertAlmostEqual(model["constant"], 5)
        self.assertAlmostEqual(model["coefficient"], 3)
        self.assertLess(max(abs(r) for r in model["residual_seconds"]), 1e-7)

    def test_guardrails_just_below_and_above_half_ms(self):
        for increase, expected in [(.00049, False), (.00051, True)]:
            self.write_all()
            for row in self.rows:
                if row["bound"] != 1 or row["warmup"]: continue
                report = self.report(row)
                for stage in report["stages"]:
                    if stage["name"] == "execute_window": stage["wall_ns"] = int((.01+(increase if row["variant"] == "candidate" else 0))*1e9)
                    if stage["name"] == "final_inspection": stage["wall_ns"] = int((.002+(increase if row["variant"] == "candidate" else 0))*1e9)
                self.write(row, report)
            result, output = self.analyze()
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(output["bounds"][0]["small_bound_guard_exceeded"], expected)
            self.assertEqual(output["bounds"][0]["phases"]["final_inspection"]["increase_above_10percent_or_half_ms"], expected)


if __name__ == "__main__":
    unittest.main()
