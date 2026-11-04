#!/usr/bin/env python3
"""Frozen paired-campaign identities and structural validation (no workloads)."""
from collections import Counter
import hashlib
import json
import math
import ipaddress
from urllib.parse import urlsplit
from pathlib import Path
import random

BOUNDS = (1, 8, 16, 32, 64, 128, 256)
VARIANTS = ("baseline", "candidate")
CONTRACT = "alan/performance-p2/paired-v1"
SEED = 20261104
REPORT_VERSION = "alan/performance-p2/v1"
PREFIX_CONFIGURATION = {"boundary": "contiguous-after-run-contact-and-returned-cut-proof-including-approval",
                        "snapshot_method": "ReadMemStats/Getrusage/runtime.metrics", "preallocated": True,
                        "extra_inspection": False, "forced_gc": False, "serialization": "after-Execute"}
TRANSPORT = {"http_version": "HTTP/1.1", "dial_timeout_ns": 30_000_000_000, "tcp_keepalive_ns": 30_000_000_000,
             "idle_timeout_ns": 90_000_000_000, "response_header_timeout_ns": 30_000_000_000,
             "server_read_timeout_ns": 30_000_000_000, "server_write_timeout_ns": 30_000_000_000,
             "max_conns_per_host": 1, "max_header_bytes": 8192, "expect_continue_timeout_ns": 1_000_000_000,
             "proxy": False, "mode": "http", "listen_address": "127.0.0.1:0", "protocol": "alan-p2-http/v1",
             "request_format": "json-encoding-go-effects.Prepared-v1",
             "response_format": "json-operation-key-response-sample-v1",
             "max_request_bytes": 65536, "max_response_bytes": 4096, "timeout_ns": 30_000_000_000,
             "reuse_connections": True, "redirects": False, "provider_retries": 0,
             "post_retries": False, "resource_scope": "same-process-server-included"}
RESOURCE_KEYS = ("wall_ns", "user_cpu_ns", "system_cpu_ns", "total_alloc_bytes", "mallocs",
                 "gc_count", "forced_gc_count", "gc_pause_ns")
METRIC_NAMES = sorted(("/cpu/classes/gc/mark/assist:cpu-seconds", "/cpu/classes/gc/mark/dedicated:cpu-seconds",
 "/cpu/classes/gc/mark/idle:cpu-seconds", "/cpu/classes/gc/pause:cpu-seconds", "/cpu/classes/gc/total:cpu-seconds",
 "/gc/cycles/automatic:gc-cycles", "/gc/cycles/forced:gc-cycles", "/gc/cycles/total:gc-cycles",
 "/sched/goroutines-created:goroutines"))
TOOL_PATHS = ("make_schedule.py", "run_schedule.py", "analyze_pairs.py", "campaign_validation.py",
              "driver/main.go", "driver/transport.go", "driver/resources.go", "driver/prefix_proof.go", "driver/go.mod",
              "driver/go.sum", "driver/build.sh", "driver/README.md")
ENVIRONMENT = {"GOMAXPROCS": "4", "GOGC": "100", "GOMEMLIMIT": "off",
               "CGO_ENABLED": "1", "GOWORK": "off", "GOTOOLCHAIN": "local",
               "GOPROXY": "off", "GODEBUG": "", "GORACE": "", "GOFLAGS": "",
               "GOTRACEBACK": "single"}
SINGLETON_PHASES = ("construct_provision", "create", "execute_window",
                    "original_state_verification", "final_inspection", "final_series_view", "final_series_proof", "close_reopen_resume",
                    "resume_inspection_view", "close", "replay")
BODY_PHASES = tuple(sorted((set(SINGLETON_PHASES) - {"execute_window"}) |
                          {"run", "approval_snapshot_basis", "approval_establishment"}))


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def tooling_hashes(directory=None):
    directory = Path(directory or Path(__file__).parent)
    return {name: sha(directory / name) for name in
            TOOL_PATHS}


def layout():
    """The original frozen warmup and six serial AB/BA pair ordering."""
    rows = []
    def add(variant, bound, pair, warmup):
        rows.append({"name": ("warmup" if warmup else "pair" + str(pair)) + "-" + variant + "-" + str(bound),
                     "bound": bound, "variant": variant, "pair": pair,
                     "warmup": warmup, "schedule_seed": SEED})
    for index, bound in enumerate(BOUNDS):
        order = VARIANTS if index % 2 == 0 else tuple(reversed(VARIANTS))
        for variant in order:
            add(variant, bound, 0, True)
    rng = random.Random(SEED)
    for pair in range(1, 7):
        order = list(BOUNDS)
        rng.shuffle(order)
        for bound in order:
            for variant in (VARIANTS if pair % 2 else tuple(reversed(VARIANTS))):
                add(variant, bound, pair, False)
    return rows


def require(ok, message):
    if not ok:
        raise ValueError(message)


def validate_schedule(rows):
    require(isinstance(rows, list) and len(rows) == 98,
            "require exactly 14 warmups and 84 members of six pairs per bound")
    variants, common = {}, None
    for row, expected in zip(rows, layout()):
        require(all(row.get(k) == v for k, v in expected.items()),
                "schedule membership/order mismatch: " + str(row.get("name")))
        require(row.get("campaign_contract") == CONTRACT, "P2 paired-v1 schedule required")
        require(not row.get("args") and not row.get("instrumented") and not row.get("profiled"),
                "paired schedule permits no profiling/recovery/extra flags")
        require(isinstance(row.get("source"), str) and bool(row["source"]), "source label required")
        require(Path(row.get("binary", "")).is_absolute(), "absolute executable path required")
        digest = row.get("expected_binary_sha256", "")
        require(len(digest) == 64 and all(c in "0123456789abcdef" for c in digest),
                "frozen executable SHA256 required")
        identity = (row["binary"], digest, row["source"])
        require(variants.setdefault(row["variant"], identity) == identity,
                "variant executable/source changed inside schedule")
        runtime = row.get("expected_runtime", {})
        require(runtime == {"go_version": "go1.27.3", "goos": "darwin", "goarch": "arm64",
                            "gomaxprocs": 4, "mem_profile_rate": 524288,
                            "profiled": False, "instrumented": False,
                            "schedule": "ordinary", "reopen_every": 0, "prefix_enabled": True},
                "frozen ordinary runtime configuration required")
        require(row.get("expected_transport") == TRANSPORT, "frozen HTTP configuration required")
        require(row.get("report_version") == REPORT_VERSION, "frozen P2 report version required")
        require(row.get("expected_environment") == ENVIRONMENT, "frozen environment required")
        tools = row.get("expected_tooling_sha256", {})
        require(set(tools) == set(TOOL_PATHS)
                and all(len(v) == 64 and all(c in "0123456789abcdef" for c in v) for v in tools.values()),
                "frozen tooling hashes required")
        configuration = (runtime, row["expected_environment"], row["expected_transport"], tools)
        if common is None:
            common = configuration
        require(configuration == common, "configuration/tooling changed inside schedule")
    return variants


def validate_report(report, row):
    name, n = row["name"], row["bound"]
    require(report.get("version") == "alan/performance-p2/v1", "wrong report version: " + name)
    require(report.get("invariants_ok") is True and not report.get("error"), "failed proof: " + name)
    require(report.get("bound") == n and report.get("sample") == name and
            report.get("source_label") == row["source"], "report identity mismatch: " + name)
    require(all(report.get(k) == v for k, v in row["expected_runtime"].items()),
            "report runtime configuration mismatch: " + name)
    require(all(report.get(k) == n for k in ("completed", "selected", "spent")) and
            report.get("protected_approvals") == n-1, "workload count mismatch: " + name)
    require(report.get("prefix_configuration") == PREFIX_CONFIGURATION, "prefix configuration drift: " + name)
    require(report.get("reductions") == 4*n-1, "reduction count mismatch: " + name)
    transport = report.get("transport", {})
    require(all(transport.get(k) == v for k, v in row["expected_transport"].items()) and
            set(transport) == set(TRANSPORT) | {"endpoint"}, "transport configuration mismatch: " + name)
    endpoint = urlsplit(transport.get("endpoint", ""))
    require(endpoint.scheme == "http" and endpoint.hostname is not None and
            ipaddress.ip_address(endpoint.hostname).is_loopback and endpoint.port is not None and
            endpoint.path == "/measure" and not endpoint.query and not endpoint.fragment and
            endpoint.username is None, "loopback endpoint identity mismatch: " + name)
    require(report.get("runtime_metric_names") == METRIC_NAMES, "runtime metrics changed/unavailable: " + name)
    require(report.get("resource_scope") == "process-wide including same-process HTTP server; GC/CPU asynchronous; RSS cumulative high-water gauge",
            "resource scope mismatch: " + name)
    require(type(report.get("process_wall_ns")) is int and report["process_wall_ns"] > 0, "process time missing: " + name)
    contacts, evidence = report.get("contacts", []), report.get("evidence", [])
    require(len(contacts) == len(evidence) == n, "contact/evidence count mismatch: " + name)
    require([c.get("ordinal") for c in contacts] == list(range(1, n+1)) and
            [e.get("index") for e in evidence] == list(range(1, n+1)), "position identity mismatch: " + name)
    operations = [e.get("operation_id") for e in evidence]
    require(all(isinstance(op, str) and op for op in operations) and len(set(operations)) == n and
            [c.get("operation_id") for c in contacts] == operations, "operation identity mismatch: " + name)
    require(not evidence[0].get("approval") and all(e.get("approval") for e in evidence[1:]),
            "protected approval evidence mismatch: " + name)
    validate_transport_proof(report, n)
    stages = report.get("stages", [])
    required = Counter((phase, 0) for phase in SINGLETON_PHASES)
    required.update(("run", i) for i in range(1, n+1))
    for phase in ("approval_snapshot_basis", "approval_establishment"):
        required.update((phase, i) for i in range(2, n+1))
    require(isinstance(stages, list) and Counter((s.get("name"), s.get("position", 0)) for s in stages) == required,
            "required phase multiplicity/positions mismatch: " + name)
    for stage in stages:
        require(not stage.get("error"), "stage error: " + name + "/" + stage["name"])
        require(all(type(stage.get(k)) is int and stage[k] >= 0 for k in
                    RESOURCE_KEYS),
                "invalid phase measurement: " + name + "/" + stage["name"])
    for stage in stages:
        validate_vector(stage, METRIC_NAMES)
    validate_prefixes(report, n)
    require(next(s for s in stages if s["name"] == "execute_window")["wall_ns"] > 0,
            "nonpositive Execute duration: " + name)


def validate_meta(meta, report, row, schedule_hash, raw_hash, progress_hash):
    name = row["name"]
    require(meta.get("exit") == 0 and not meta.get("validation_error"), "failed process: " + name)
    require(meta.get("schedule_sha256") == schedule_hash and meta.get("schedule_row") == row,
            "sample/schedule integrity mismatch: " + name)
    require(meta.get("report_sha256") == raw_hash and meta.get("progress_sha256") == progress_hash,
            "raw report/progress integrity mismatch: " + name)
    require(meta.get("binary_sha256") == meta.get("binary_sha256_after") == row["expected_binary_sha256"],
            "executable hash mismatch: " + name)
    require(meta.get("runner_sha256") == row["expected_tooling_sha256"]["run_schedule.py"] and
            meta.get("validation_sha256") == row["expected_tooling_sha256"]["campaign_validation.py"],
            "runner/validation hash mismatch: " + name)
    require(meta.get("environment") == row["expected_environment"], "process environment mismatch: " + name)
    require(meta.get("process_peak_rss_unit") == "bytes", "unexpected RSS unit for frozen darwin runtime: " + name)
    require(type(meta.get("process_peak_rss")) is int and meta["process_peak_rss"] >= 0,
            "invalid process RSS: " + name)
    require(meta.get("reported_configuration") == reported_configuration(report), "recorded configuration mismatch: " + name)
    store = Path(report.get("store", ""))
    require(store.name == "store.sqlite" and store.parent.name == name + ".store", "store identity mismatch: " + name)
    expected_command = [row["binary"], "-bound", str(row["bound"]), "-dir", str(store.parent),
                        "-sample", name, "-source-label", row["source"]]
    require(meta.get("command") == expected_command, "process arguments mismatch: " + name)


def validate_vector(vector, metric_names):
    require(all(type(vector.get(k)) is int and vector[k] >= 0 for k in RESOURCE_KEYS), "negative/missing resource measurement")
    require(type(vector.get("process_maxrss_cumulative")) is int and vector["process_maxrss_cumulative"] >= 0,
            "invalid RSS gauge")
    metrics = vector.get("runtime_metric_deltas", {})
    require(set(metrics) == set(metric_names) and all(type(v) in (int, float) and math.isfinite(v) and v >= 0 for v in metrics.values()),
            "invalid runtime metric deltas")
    counters = vector.get("counter_deltas", {})
    require(isinstance(counters, dict) and all(type(v) is int and v >= 0 for v in counters.values()), "invalid counters")


def validate_prefixes(report, n):
    prefixes = report.get("prefixes", [])
    require(report.get("prefix_enabled") is True and len(prefixes) == n, "missing/extra prefixes")
    membership = report.get("root_membership", [])
    require(len(membership) == n and [r.get("position") for r in membership] == list(range(1, n+1)), "root membership count/order mismatch")
    require([r.get("run") for r in membership] == [{"Subject": "series-"+str(k), "Generation": 1} for k in range(1, n+1)] and
            [r.get("root_id") for r in membership] == [e["root_id"] for e in report["evidence"]] and
            len({r["root_id"] for r in membership}) == n, "root membership identity mismatch")
    membership_sha = hashlib.sha256(go_json(membership).encode()).hexdigest()
    outcomes = set()
    previous = {k: 0 for k in RESOURCE_KEYS}
    previous_metrics = dict.fromkeys(METRIC_NAMES, 0)
    previous_counters = {}
    previous_rss = 0
    for k, prefix in enumerate(prefixes, 1):
        require(all(prefix.get(field) == value for field, value in
                    {"position": k, "declaration_bound": n, "retained_positions": k,
                     "contacts": k, "server_contacts": k}.items()), "prefix count/order mismatch")
        cut = prefix.get("cut", {})
        expected_counts = {"root_membership_sha256": membership_sha, "root_count": n, "completed": k,
                           "selected_positions": k, "prior_completed": k-1, "future_open": n-k,
                           "quiescent": True, "steps": 3 if k == 1 else 4, "dispatches": 1, "open_runs": n-k,
                           "unresolved_intents": 0, "pending_deliveries": 0, "unknown_external": 0,
                           "unknown_adapters": 0, "rejected_admissions": 0}
        require(all(type(cut.get(field)) is type(value) and cut[field] == value for field, value in expected_counts.items()),
                "prefix returned-cut completion/work proof mismatch")
        item, identity = report["evidence"][k-1], membership[k-1]
        outcome = item.get("root_outcome", {})
        require(isinstance(outcome, dict) and outcome.get("ID") and outcome["ID"] not in outcomes and
                outcome.get("Caller") == identity["run"] and outcome.get("InvocationID") == "root:"+identity["root_id"] and
                outcome.get("OperationID") == "root:"+identity["root_id"] and outcome.get("TemplateID") == "" and
                outcome.get("Status") == "succeeded" and outcome.get("Reason") == "member_succeeded" and
                outcome.get("ResponsePresent") is False and not outcome.get("Response") and
                type(outcome.get("DecisionOrder")) is int and outcome["DecisionOrder"] > 0,
                "prefix original root outcome mismatch")
        outcomes.add(outcome["ID"])
        require(cut.get("new_root") == {"root_id": identity["root_id"], "run": identity["run"], "local_status": "succeeded",
                                        "outcome": outcome, "effect_charges": 1, "invocation_charges": 0, "created_run_charges": 0},
                "prefix new root/debit proof mismatch")
        direct, cumulative = prefix.get("direct", {}), prefix.get("cumulative", {})
        for vector in (direct, cumulative): validate_vector(vector, METRIC_NAMES)
        require(all(cumulative[key] >= previous[key] and cumulative[key]-previous[key] == direct[key]
                    for key in RESOURCE_KEYS), "prefix direct/cumulative arithmetic mismatch")
        current_metrics = cumulative["runtime_metric_deltas"]
        require(all(current_metrics[key] >= previous_metrics[key] and
                    math.isclose(current_metrics[key]-previous_metrics[key], direct["runtime_metric_deltas"][key], rel_tol=1e-12, abs_tol=1e-12)
                    for key in METRIC_NAMES), "prefix metric arithmetic mismatch")
        counters = cumulative.get("counter_deltas", {})
        delta = direct.get("counter_deltas", {})
        require(set(previous_counters) <= set(counters) and set(delta) == set(counters) and
                all(counters[key] >= previous_counters.get(key, 0) and counters[key]-previous_counters.get(key, 0) == delta[key]
                    for key in counters), "prefix counter arithmetic mismatch")
        require(cumulative["process_maxrss_cumulative"] >= previous_rss and
                direct["process_maxrss_cumulative"] == cumulative["process_maxrss_cumulative"], "prefix RSS gauge mismatch")
        previous, previous_metrics, previous_counters, previous_rss = cumulative, current_metrics, counters, cumulative["process_maxrss_cumulative"]
    outer = next(s for s in report["stages"] if s["name"] == "execute_window")
    residual = report.get("prefix_residual", {})
    validate_vector(residual, METRIC_NAMES)
    require(residual.get("name") == "execute_window_minus_final_prefix", "prefix residual absent")
    require(all(outer[key] >= previous[key] and outer[key]-previous[key] == residual[key] for key in RESOURCE_KEYS),
            "prefix exceeds window/residual arithmetic mismatch")
    require(all(outer["runtime_metric_deltas"][key] >= previous_metrics[key] and
                math.isclose(outer["runtime_metric_deltas"][key]-previous_metrics[key], residual["runtime_metric_deltas"][key], rel_tol=1e-12, abs_tol=1e-12)
                for key in METRIC_NAMES), "prefix residual metric arithmetic mismatch")
    outer_counters, residual_counters = outer.get("counter_deltas", {}), residual.get("counter_deltas", {})
    require(set(previous_counters) <= set(outer_counters) and set(residual_counters) == set(outer_counters) and
            all(outer_counters[key]-previous_counters.get(key, 0) == residual_counters[key] for key in outer_counters),
            "prefix residual counter arithmetic mismatch")
    require(residual["process_maxrss_cumulative"] == outer["process_maxrss_cumulative"] and
            outer["process_maxrss_cumulative"] >= previous_rss, "residual RSS gauge mismatch")


def go_json(value):
    # Exact fixed workload: string keys, integral fields, no custom JSON types.
    return json.dumps(value, ensure_ascii=False, separators=(",", ":")).replace("<", r"\u003c").replace(">", r"\u003e").replace("&", r"\u0026").replace("\u2028", r"\u2028").replace("\u2029", r"\u2029")


def validate_transport_proof(report, n):
    clients, servers, evidence = report["contacts"], report.get("server_contacts", []), report["evidence"]
    require(len(servers) == n, "missing/extra server contacts")
    seen_keys, seen_prepared, seen_result, seen_observations, seen_roots = set(), set(), set(), set(), set()
    for index, (client, server, item) in enumerate(zip(clients, servers, evidence), 1):
        require(client == server, "adapter/server journal bijection mismatch")
        prepared, candidate = item.get("prepared", {}), item.get("candidate", {})
        request = prepared.get("Request", {})
        require(client.get("prepared") == prepared and client.get("ordinal") == index and
                request.get("ID") == item["operation_id"] == client.get("operation_id") and
                prepared.get("Digest") == item.get("prepared_digest") == client.get("prepared_digest"), "retained Prepared identity mismatch")
        require(prepared.get("Protocol") == report["transport"]["protocol"] and prepared.get("Endpoint") == report["transport"]["endpoint"] and
                prepared.get("MaxResponseBytes") == 4096, "retained transport mismatch")
        require(request.get("Capability") == "meter" and request.get("Effect") == "read" and
                request.get("Payload") == {"station": {"Kind": "string", "Number": 0, "Text": "north", "Flag": False}} and
                prepared.get("Model") == "" and prepared.get("MaxOutputTokens") == 0,
                "fixed workload request mismatch")
        for field, key in (("Key", "key"), ("Capability", "capability"), ("Effect", "effect")):
            require(request.get(field) == client.get(key), "journal request field mismatch")
        require(client.get("protocol") == prepared["Protocol"] and client.get("endpoint") == prepared["Endpoint"], "journal protocol/endpoint mismatch")
        raw, response = client.get("request_wire", ""), client.get("response_wire", "")
        require(len(raw.encode()) <= 65536 and raw == go_json(prepared) and client.get("response_complete") is True,
                "request bytes/incomplete response mismatch")
        require(client.get("request_sha256") == hashlib.sha256(raw.encode()).hexdigest() and
                client.get("response_sha256") == hashlib.sha256(response.encode()).hexdigest(), "wire digest mismatch")
        freeze = json.loads(raw); freeze["Digest"] = ""
        require(hashlib.sha256(go_json(freeze).encode()).hexdigest() == prepared["Digest"], "Prepared digest mismatch")
        scoped_request = dict(request, ID="", Key="")
        wire = {"request": scoped_request, "protocol": prepared["Protocol"], "endpoint": prepared["Endpoint"]}
        require(prepared.get("Wire") == go_json(wire), "Prepared wire mismatch")
        expected_response = {"operation_id": request["ID"], "key": request["Key"],
                             "response_id": "local-response:" + request["ID"], "sample": "present"}
        require(response == go_json(expected_response) and len(response.encode()) <= 4096,
                "deterministic response bytes mismatch")
        require(candidate.get("ID") == request["ID"] and candidate.get("Key") == request["Key"] and
                candidate.get("Disposition") == "validated" and candidate.get("HTTPStatus") == 200 and
                candidate.get("RequestID") == request["ID"] and candidate.get("ResponseID") == expected_response["response_id"] ==
                client.get("response_id") == item.get("response_id") and candidate.get("Response") == response,
                "retained Candidate mismatch")
        require(candidate.get("Payload") == {"sample": {"Kind": "string", "Number": 0, "Text": "present", "Flag": False}} and
                candidate.get("ReturnedModel") == "" and candidate.get("Usage") is None,
                "Candidate payload/policy mismatch")
        for value, seen in ((request.get("Key"), seen_keys), (prepared.get("Digest"), seen_prepared),
                            (item.get("result_digest"), seen_result), (item.get("observation_id"), seen_observations),
                            (item.get("root_id"), seen_roots)):
            require(isinstance(value, str) and value and value not in seen, "missing/duplicate retained identity")
            seen.add(value)
        selected = item.get("selected_at", {})
        require(selected.get("Run") == item.get("run") and selected.get("Sequence", 0) > 0,
                "detached selection")
        if index > 1:
            approval = item.get("approval", {})
            protected = approval.get("Approval", {})
            prev = evidence[index-2]
            require(protected.get("Target") == item.get("run") and protected.get("PredecessorID") == prev["operation_id"] and
                    protected.get("PreparedDigest") == prev["prepared_digest"] and protected.get("ResultDigest") == prev["result_digest"] and
                    protected.get("Admitted") is True and protected.get("Reason") == "operator" and protected.get("ScopeDigest") and
                    approval.get("SuccessorOperationID") == item["operation_id"] and approval.get("SelectedAt", {}).get("Run") == item.get("run"),
                    "approval predecessor/selection mismatch")


def reported_configuration(report):
    return {k: report.get(k) for k in ("version", "transport", "prefix_enabled", "prefix_configuration",
                                      "runtime_metric_names", "resource_scope", "profiled", "instrumented", "reopen_every")}
