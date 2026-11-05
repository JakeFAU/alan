#!/usr/bin/env python3
"""P3 strict direct proof; adapted from unchanged P2 common vector/cut checks."""
from collections import Counter
import hashlib
import json
import math

PREFIX_CONFIGURATION = {"boundary": "contiguous-after-run-contact-and-returned-cut-proof-including-approval",
                        "snapshot_method": "ReadMemStats/Getrusage/runtime.metrics", "preallocated": True,
                        "extra_inspection": False, "forced_gc": False, "serialization": "after-Execute"}
TRANSPORT = {"http_version": "none", "dial_timeout_ns": 0, "tcp_keepalive_ns": 0,
             "idle_timeout_ns": 0, "response_header_timeout_ns": 0, "server_read_timeout_ns": 0,
             "server_write_timeout_ns": 0, "max_conns_per_host": 0, "max_header_bytes": 0,
             "expect_continue_timeout_ns": 0, "proxy": False, "mode": "direct", "listen_address": "",
             "protocol": "measurement/v1", "endpoint": "deterministic-local-no-http",
             "request_format": "station-text", "response_format": "historical-candidate-v1",
             "max_request_bytes": 64, "max_response_bytes": 256, "timeout_ns": 0,
             "reuse_connections": False, "redirects": False, "provider_retries": 0,
             "post_retries": False, "resource_scope": "process-no-server"}
RESOURCE_KEYS = ("wall_ns", "user_cpu_ns", "system_cpu_ns", "total_alloc_bytes", "mallocs",
                 "gc_count", "forced_gc_count", "gc_pause_ns")
METRIC_NAMES = sorted(("/cpu/classes/gc/mark/assist:cpu-seconds", "/cpu/classes/gc/mark/dedicated:cpu-seconds",
 "/cpu/classes/gc/mark/idle:cpu-seconds", "/cpu/classes/gc/pause:cpu-seconds", "/cpu/classes/gc/total:cpu-seconds",
 "/gc/cycles/automatic:gc-cycles", "/gc/cycles/forced:gc-cycles", "/gc/cycles/total:gc-cycles",
 "/sched/goroutines-created:goroutines"))
ENVIRONMENT = {"GOMAXPROCS": "4", "GOGC": "100", "GOMEMLIMIT": "off",
               "CGO_ENABLED": "1", "GOWORK": "off", "GOTOOLCHAIN": "local",
               "GOPROXY": "off", "GOSUMDB": "off", "GODEBUG": "", "GORACE": "", "GOFLAGS": "",
               "GOTRACEBACK": "single"}
SINGLETON_PHASES = ("construct_provision", "create", "execute_window",
                    "original_state_verification", "final_inspection", "final_series_view", "final_series_proof", "close_reopen_resume",
                    "resume_inspection_view", "close", "replay")
BODY_PHASES = tuple(sorted((set(SINGLETON_PHASES) - {"execute_window"}) |
                          {"run", "approval_snapshot_basis", "approval_establishment"}))


def require(ok, message):
    if not ok:
        raise ValueError(message)


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
    require(report.get("transport") == TRANSPORT, "direct transport configuration mismatch: " + name)
    require(report.get("server_contacts") == [], "direct mode must have no HTTP journal")
    require(report.get("maxrss_unit") == "bytes" and report.get("storage_version") == (9 if n == 1 else 11),
            "storage/RSS configuration drift")
    require(report.get("replay_history") == 4*n-1 and report.get("replay_intents") == n,
            "replay logical counts mismatch")
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
    validate_vector(report.get("execute_totals", {}), METRIC_NAMES)
    body = [s for s in stages if s["name"] in ("run", "approval_snapshot_basis", "approval_establishment")]
    require(report["execute_totals"].get("name") == "execute_sum" and
            all(report["execute_totals"][key] == sum(s[key] for s in body) for key in RESOURCE_KEYS),
            "Execute sum mismatch")
    require(next(s for s in stages if s["name"] == "execute_window")["wall_ns"] > 0,
            "nonpositive Execute duration: " + name)


def validate_vector(vector, metric_names):
    require(all(type(vector.get(k)) is int and vector[k] >= 0 for k in RESOURCE_KEYS), "negative/missing resource measurement")
    require(type(vector.get("process_maxrss_cumulative")) is int and vector["process_maxrss_cumulative"] >= 0,
            "invalid RSS gauge")
    metrics = vector.get("runtime_metric_deltas", {})
    require(set(metrics) == set(metric_names) and all(type(v) in (int, float) and math.isfinite(v) and v >= 0 for v in metrics.values()),
            "invalid runtime metric deltas")
    counters = vector.get("counter_deltas", {})
    require(counters == {}, "counter contamination in primary report")


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
                     "contacts": k, "server_contacts": 0}.items()), "prefix count/order mismatch")
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
    clients, evidence = report["contacts"], report["evidence"]
    require(report.get("server_contacts") == [], "direct mode has no server contacts")
    seen_keys, seen_prepared, seen_result, seen_observations, seen_roots = set(), set(), set(), set(), set()
    for index, (client, item) in enumerate(zip(clients, evidence), 1):
        prepared, candidate = item.get("prepared", {}), item.get("candidate", {})
        request = prepared.get("Request", {})
        require(request.get("ID") == item["operation_id"] == client.get("operation_id") and
                prepared.get("Digest") == item.get("prepared_digest") == client.get("prepared_digest"), "retained Prepared identity mismatch")
        require(prepared.get("Protocol") == TRANSPORT["protocol"] and prepared.get("Endpoint") == TRANSPORT["endpoint"] and
                prepared.get("MaxResponseBytes") == 256 and prepared.get("Wire") == "north", "retained direct transport mismatch")
        require(request.get("Capability") == "meter" and request.get("Effect") == "read" and
                request.get("Payload") == {"station": {"Kind": "string", "Number": 0, "Text": "north", "Flag": False}} and
                prepared.get("Model") == "" and prepared.get("MaxOutputTokens") == 0,
                "fixed workload request mismatch")
        require(request.get("Assets") == [], "unexpected request assets")
        for field, name in (("Input", "station"), ("Output", "sample")):
            schema = {name: {"Kind":"string", "MaxBytes":64, "Enum":None}}
            require(request.get(field, {}).get("Record") == schema and
                    request[field].get("ID") == hashlib.sha256(go_json(schema).encode()).hexdigest(), "request schema drift")
        expected_op = go_json(["series-" + str(index), 1, 1 if index == 1 else 2, 0])
        require(request["ID"] == expected_op and request.get("Key") == go_json([2, report.get("namespace"), expected_op]),
                "request operation/namespace identity mismatch")
        require(item.get("observation_id") == go_json(["observation", "adapter", expected_op]), "observation identity mismatch")
        zero_request = {"ID": "", "Key": "", "Capability": "", "Effect": "", "Input": {"ID":"", "Record":None}, "Output": {"ID":"", "Record":None}, "Payload": None, "Assets": None}
        zero_prepared = {"Request": zero_request, "Protocol": "", "Endpoint": "", "Model": "", "Wire": "",
                         "MaxOutputTokens": 0, "MaxResponseBytes": 0, "Digest": ""}
        expected_contact = {"prepared": zero_prepared, "request_wire": "", "request_sha256": "",
                            "response_wire": "", "response_sha256": "", "response_complete": False,
                            "ordinal": index, "operation_id": request.get("ID"), "key": request.get("Key"),
                            "prepared_digest": prepared.get("Digest"), "capability": "meter", "effect": "read",
                            "protocol": TRANSPORT["protocol"], "endpoint": TRANSPORT["endpoint"],
                            "response_id": "local-response:" + request["ID"]}
        require(client == expected_contact, "direct scalar journal/provenance mismatch")
        freeze = dict(prepared, Digest="")
        require(hashlib.sha256(go_json(freeze).encode()).hexdigest() == prepared["Digest"], "Prepared digest mismatch")
        expected_candidate = {"ID": request["ID"], "Key": request["Key"], "Disposition": "validated",
                              "Payload": {"sample": {"Kind": "string", "Number": 0, "Text": "present", "Flag": False}},
                              "HTTPStatus": 0, "RequestID": "", "ResponseID": expected_contact["response_id"],
                              "Response": "", "ReturnedModel": "", "Usage": None}
        require(candidate == expected_candidate and item.get("response_id") == expected_candidate["ResponseID"],
                "direct retained Candidate mismatch")
        values = {"disposition":{"Kind":"string", "Number":0, "Text":"validated", "Flag":False},
                  "sample":candidate["Payload"]["sample"], "valid":{"Kind":"bool", "Number":0, "Text":"", "Flag":True}}
        retained_result = {"Digest":"", "Candidate":candidate, "Values":values, "Disposition":"validated"}
        require(item.get("result_digest") == hashlib.sha256(go_json(retained_result).encode()).hexdigest(), "result digest mismatch")
        require(item.get("run") == {"Subject": "series-" + str(index), "Generation": 1}, "evidence run identity mismatch")
        for value, seen in ((request.get("Key"), seen_keys), (prepared.get("Digest"), seen_prepared),
                            (item.get("result_digest"), seen_result), (item.get("observation_id"), seen_observations),
                            (item.get("root_id"), seen_roots)):
            require(isinstance(value, str) and value and value not in seen, "missing/duplicate retained identity")
            seen.add(value)
        selected = item.get("selected_at", {})
        require(selected.get("Run") == item.get("run") and selected.get("Sequence") == (2 if index == 1 else 3),
                "detached selection")
        if index > 1:
            approval = item.get("approval", {})
            protected = approval.get("Approval", {})
            prev = evidence[index-2]
            scoped = dict(prev["prepared"], Digest="", Request=dict(prev["prepared"]["Request"], ID="", Key=""))
            scope_digest = hashlib.sha256(go_json(scoped).encode()).hexdigest()
            require(protected.get("Namespace") == report.get("namespace") and protected.get("TargetRevision") == 1 and protected.get("Target") == item.get("run") and protected.get("PredecessorID") == prev["operation_id"] and
                    protected.get("PreparedDigest") == prev["prepared_digest"] and protected.get("ResultDigest") == prev["result_digest"] and
                    protected.get("Admitted") is True and protected.get("Reason") == "operator" and protected.get("ScopeDigest") == scope_digest and
                    approval.get("SuccessorOperationID") == item["operation_id"] and approval.get("SelectedAt") == {"Run":item.get("run"), "Sequence":1} and approval.get("ObservationID") == go_json(["observation", "input", "series-"+str(index)+"-approval"]),
                    "approval predecessor/selection mismatch")


def reported_configuration(report):
    return {k: report.get(k) for k in ("version", "transport", "prefix_enabled", "prefix_configuration",
                                      "runtime_metric_names", "resource_scope", "profiled", "instrumented", "reopen_every")}
