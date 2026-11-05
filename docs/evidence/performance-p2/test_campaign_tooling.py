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
from campaign_validation import (BOUNDS, ENVIRONMENT, SINGLETON_PHASES, METRIC_NAMES, RESOURCE_KEYS, TRANSPORT, PREFIX_CONFIGURATION, go_json, sha,
                                 validate_meta, validate_report, validate_schedule, reported_configuration)
from make_schedule import build_schedule

HERE = Path(__file__).resolve().parent
RUNTIME = {"go_version": "go1.27.3", "goos": "darwin", "goarch": "arm64",
           "gomaxprocs": 4, "mem_profile_rate": 524288, "profiled": False,
           "instrumented": False, "schedule": "ordinary", "reopen_every": 0, "prefix_enabled": True}


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
        endpoint = "http://127.0.0.1:12345/measure"
        report = dict(RUNTIME, version="alan/performance-p2/v1", invariants_ok=True,
                      bound=n, sample=row["name"], source_label=row["source"],
                      store=str(self.samples / (row["name"] + ".store") / "store.sqlite"),
                      completed=n, selected=n, spent=n, protected_approvals=n-1, reductions=4*n-1,
                      transport=dict(TRANSPORT, endpoint=endpoint), prefix_configuration=PREFIX_CONFIGURATION.copy(), runtime_metric_names=METRIC_NAMES,
                      resource_scope="process-wide including same-process HTTP server; GC/CPU asynchronous; RSS cumulative high-water gauge",
                      process_wall_ns=123456789, contacts=[], evidence=[])
        value = {"Kind": "string", "Number": 0, "Text": "north", "Flag": False}
        for i in range(1, n+1):
            operation = "operation-"+str(i)
            request = {"ID": operation, "Key": "key-"+str(i), "Capability": "meter", "Effect": "read",
                       "Input": {}, "Output": {}, "Payload": {"station": value}, "Assets": None}
            scoped = dict(request, ID="", Key="")
            prepared = {"Request": request, "Digest": "", "Protocol": TRANSPORT["protocol"], "Endpoint": endpoint,
                        "Model": "", "MaxOutputTokens": 0, "MaxResponseBytes": 4096,
                        "Wire": go_json({"request": scoped, "protocol": TRANSPORT["protocol"], "endpoint": endpoint})}
            prepared["Digest"] = hashlib.sha256(go_json(prepared).encode()).hexdigest()
            raw = go_json(prepared)
            response_id = "local-response:"+operation
            response = go_json({"operation_id": operation, "key": request["Key"], "response_id": response_id, "sample": "present"})
            candidate = {"ID": operation, "Key": request["Key"], "Disposition": "validated", "HTTPStatus": 200,
                         "RequestID": operation, "ResponseID": response_id, "Response": response, "ReturnedModel": "", "Usage": None,
                         "Payload": {"sample": dict(value, Text="present")}}
            contact = {"ordinal": i, "operation_id": operation, "key": request["Key"], "prepared_digest": prepared["Digest"],
                       "capability": request["Capability"], "effect": "read", "protocol": prepared["Protocol"], "endpoint": endpoint,
                       "response_id": response_id, "prepared": prepared, "request_wire": raw, "response_wire": response,
                       "request_sha256": hashlib.sha256(raw.encode()).hexdigest(), "response_sha256": hashlib.sha256(response.encode()).hexdigest(),
                       "response_complete": True}
            run = {"Subject": "series-"+str(i), "Generation": 1}
            item = {"index": i, "operation_id": operation, "prepared": prepared, "candidate": candidate,
                    "prepared_digest": prepared["Digest"], "result_digest": "result-"+str(i), "response_id": response_id,
                    "root_id": "root-"+str(i), "observation_id": "observation-"+str(i), "run": run,
                    "selected_at": {"Run": run, "Sequence": 3}}
            item["root_outcome"] = {"ID": "outcome-"+str(i), "InvocationID": "root:"+item["root_id"],
                                    "OperationID": "root:"+item["root_id"], "TemplateID": "", "Caller": run,
                                    "Status": "succeeded", "Reason": "member_succeeded", "ResponsePresent": False,
                                    "Response": None, "DecisionOrder": i}
            if i > 1:
                prev = report["evidence"][-1]
                item["approval"] = {"Approval": {"Target": run, "PredecessorID": prev["operation_id"],
                    "PreparedDigest": prev["prepared_digest"], "ResultDigest": prev["result_digest"], "Admitted": True,
                    "Reason": "operator", "ScopeDigest": "scope"}, "SelectedAt": {"Run": run, "Sequence": 1},
                    "SuccessorOperationID": operation}
            report["contacts"].append(contact)
            report["evidence"].append(item)
        report["server_contacts"] = copy.deepcopy(report["contacts"])
        stages = []
        for phase, positions in [(p, [0]) for p in SINGLETON_PHASES] + [
                ("run", range(1, n+1)), ("approval_snapshot_basis", range(2, n+1)),
                ("approval_establishment", range(2, n+1))]:
            for position in positions:
                stages.append(dict(self.vector(), name=phase, position=position, wall_ns=1_000_000,
                                   user_cpu_ns=1000, system_cpu_ns=1000, total_alloc_bytes=1000, mallocs=1000))
        execution = next(s for s in stages if s["name"] == "execute_window")
        seconds = n * (row["pair"] + 10) * (1 if row["variant"] == "baseline" else .5)
        execution["wall_ns"] = int(seconds * 1e9) if not row["warmup"] else int(1000*n*1e9)
        report["stages"] = stages
        report["prefixes"] = [{"position": i, "declaration_bound": n, "retained_positions": i, "contacts": i, "server_contacts": i,
                              "direct": self.vector(1), "cumulative": self.vector(i)} for i in range(1, n+1)]
        report["root_membership"] = [{"position": i, "run": item["run"], "root_id": item["root_id"]}
                                     for i, item in enumerate(report["evidence"], 1)]
        membership_sha = hashlib.sha256(go_json(report["root_membership"]).encode()).hexdigest()
        for k, prefix in enumerate(report["prefixes"], 1):
            item = report["evidence"][k-1]
            prefix["cut"] = {"root_membership_sha256": membership_sha, "root_count": n, "completed": k,
                             "selected_positions": k, "prior_completed": k-1, "future_open": n-k,
                             "quiescent": True, "steps": 3 if k == 1 else 4, "dispatches": 1, "open_runs": n-k,
                             "unresolved_intents": 0, "pending_deliveries": 0, "unknown_external": 0, "unknown_adapters": 0,
                             "rejected_admissions": 0, "new_root": {"root_id": item["root_id"], "run": item["run"],
                             "local_status": "succeeded", "outcome": copy.deepcopy(item["root_outcome"]),
                             "effect_charges": 1, "invocation_charges": 0, "created_run_charges": 0}}
        self.residual(report)
        return report

    @staticmethod
    def vector(k=0):
        vector = {key: 0 for key in RESOURCE_KEYS}
        vector.update(wall_ns=k, user_cpu_ns=k, system_cpu_ns=k, total_alloc_bytes=k, mallocs=k,
                      process_maxrss_cumulative=123, runtime_metric_deltas=dict.fromkeys(METRIC_NAMES, 0))
        return vector

    def residual(self, report):
        outer = next(s for s in report["stages"] if s["name"] == "execute_window")
        final = report["prefixes"][-1]["cumulative"]
        residual = self.vector()
        for key in RESOURCE_KEYS: residual[key] = outer[key]-final[key]
        residual["name"] = "execute_window_minus_final_prefix"
        report["prefix_residual"] = residual

    def write(self, row, report=None, edit_meta=None):
        name = row["name"]
        report = report if report is not None else self.report(row)
        raw = self.samples / (name + ".json")
        raw.write_text(json.dumps(report))
        progress = self.samples / (name + ".progress")
        progress.write_text("synthetic progress " + name)
        meta = {"exit": 0, "reported_configuration": reported_configuration(report), "schedule_sha256": sha(self.schedule), "schedule_row": row,
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
        self.assertEqual(output["bounds"][-1]["primary_sign_test"]["one_sided_fair_sign_tail_p"], 1/64)

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
                           ("selected", 0), ("protected_approvals", 0), ("contacts", []), ("version", "alan/series-scaling/v1"),
                           ("prefix_enabled", False), ("runtime_metric_names", []), ("transport", dict(TRANSPORT, mode="direct"))]:
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
        self.assertEqual(analyze_pairs.sign_test([1]*6)["one_sided_fair_sign_tail_p"], 1/64)
        self.assertTrue(analyze_pairs.sign_test([1]*6)["all_six_positive"])
        self.assertFalse(analyze_pairs.sign_test([1]*5+[0])["all_six_positive"])
        self.assertEqual(analyze_pairs.sign_test([0]*6)["ties"], 6)
        self.assertEqual(analyze_pairs.sign_test([-1]*6)["one_sided_fair_sign_tail_p"], 1)
        self.assertEqual(analyze_pairs.sign_test([1]*5+[0])["one_sided_fair_sign_tail_p"], 7/64)
        model = analyze_pairs.fit_growth(BOUNDS, [5+3*n*n for n in BOUNDS])["finite_models"]["constant_plus_n_to_2"]
        self.assertAlmostEqual(model["constant"], 5)
        self.assertAlmostEqual(model["coefficient"], 3)
        self.assertLess(max(abs(r) for r in model["residual_seconds"]), 1e-7)

    def test_guardrails_just_below_and_above_one_ms(self):
        for increase, expected in [(.00099, False), (.00101, True)]:
            self.write_all()
            for row in self.rows:
                if row["bound"] != 1 or row["warmup"]: continue
                report = self.report(row)
                for stage in report["stages"]:
                    if stage["name"] == "execute_window": stage["wall_ns"] = int((.005+(increase if row["variant"] == "candidate" else 0))*1e9)
                    if stage["name"] == "final_inspection": stage["wall_ns"] = int((.002+(increase if row["variant"] == "candidate" else 0))*1e9)
                self.residual(report)
                self.write(row, report)
            result, output = self.analyze()
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(output["bounds"][0]["small_bound_guard_exceeded"], expected)
            self.assertEqual(output["bounds"][0]["phases"]["final_inspection"]["increase_above_10percent_or_half_ms"], True)

    def test_server_journal_mutants(self):
        row = self.measured()
        for journal in ("contacts", "server_contacts"):
            for field in ("operation_id", "key", "capability", "effect", "protocol", "endpoint", "prepared_digest",
                          "request_wire", "request_sha256", "response_id", "response_wire", "response_sha256"):
                with self.subTest(journal=journal, field=field):
                    report = self.report(row)
                    report[journal][0][field] = "wrong"
                    with self.assertRaises(ValueError): validate_report(report, row)
            for kind in ("missing", "duplicate", "reordered"):
                with self.subTest(journal=journal, kind=kind):
                    report = self.report(row)
                    if kind == "missing": report[journal].pop()
                    if kind == "duplicate": report[journal][1] = report[journal][0]
                    if kind == "reordered": report[journal][0], report[journal][1] = report[journal][1], report[journal][0]
                    with self.assertRaises(ValueError): validate_report(report, row)

    def test_paired_journals_require_canonical_envelope(self):
        row = self.measured()
        unchanged = self.report(row)
        validate_report(unchanged, row)
        for mutation in ("whitespace", "duplicate_key"):
            with self.subTest(mutation=mutation):
                report = copy.deepcopy(unchanged)
                for journal in ("contacts", "server_contacts"):
                    for contact in report[journal]:
                        raw = contact["request_wire"]
                        raw = raw + " " if mutation == "whitespace" else raw[:-1] + ',"MaxResponseBytes":4096}'
                        contact["request_wire"] = raw
                        contact["request_sha256"] = hashlib.sha256(raw.encode()).hexdigest()
                with self.assertRaises(ValueError): validate_report(report, row)

    def test_prefix_mutants(self):
        row = self.measured()
        for kind in ("missing", "duplicate", "reordered", "nonmonotone", "negative", "overcount", "arithmetic", "residual"):
            with self.subTest(kind=kind):
                report = self.report(row)
                prefixes = report["prefixes"]
                if kind == "missing": prefixes.pop()
                if kind == "duplicate": prefixes[1] = prefixes[0]
                if kind == "reordered": prefixes[0], prefixes[1] = prefixes[1], prefixes[0]
                if kind == "nonmonotone": prefixes[1]["cumulative"]["total_alloc_bytes"] = 0
                if kind == "negative": prefixes[1]["direct"]["wall_ns"] = -1
                if kind == "overcount": prefixes[1]["contacts"] = 3
                if kind == "arithmetic": prefixes[1]["direct"]["mallocs"] += 1
                if kind == "residual": report["prefix_residual"]["wall_ns"] += 1
                with self.assertRaises(ValueError): validate_report(report, row)

    def test_prefix_returned_cut_proof_mutants(self):
        row = self.measured()
        for field, value in (("completed", 2), ("selected_positions", 2), ("root_count", 1), ("future_open", 0),
                             ("prior_completed", 2), ("root_membership_sha256", "foreign"), ("quiescent", False),
                             ("steps", 4), ("dispatches", 2), ("open_runs", 0), ("unknown_adapters", 1)):
            report = self.report(row)
            report["prefixes"][0]["cut"][field] = value
            with self.subTest(field=field), self.assertRaises(ValueError): validate_report(report, row)
        for field, value in (("root_id", "foreign"), ("run", {"Subject": "series-1", "Generation": 2}),
                             ("local_status", "open"), ("outcome", None), ("effect_charges", 0)):
            report = self.report(row)
            report["prefixes"][0]["cut"]["new_root"][field] = value
            with self.subTest(field=field), self.assertRaises(ValueError): validate_report(report, row)
        report = self.report(row)
        report["prefixes"][0]["cut"]["new_root"]["outcome"]["Status"] = "failed"
        with self.assertRaises(ValueError): validate_report(report, row)
        for mutation in ("missing", "duplicate", "foreign", "generation"):
            report = self.report(row)
            roots = report["root_membership"]
            if mutation == "missing": roots.pop()
            if mutation == "duplicate": roots[1] = roots[0]
            if mutation == "foreign": roots[0]["root_id"] = "foreign"
            if mutation == "generation": roots[0]["run"]["Generation"] = 2
            with self.subTest(mutation=mutation), self.assertRaises(ValueError): validate_report(report, row)

    def test_frozen_transport_report_and_tool_drift(self):
        for field, value in (("report_version", "old"), ("expected_transport", dict(TRANSPORT, post_retries=True))):
            rows = copy.deepcopy(self.rows)
            rows[0][field] = value
            with self.assertRaises(ValueError): validate_schedule(rows)
        rows = copy.deepcopy(self.rows)
        rows[0]["expected_tooling_sha256"]["driver/main.go"] = "0"*64
        with self.assertRaises(ValueError): validate_schedule(rows)


if __name__ == "__main__":
    unittest.main()
