"""Independent checks for this fixed success-only capture contract, not generic ALAN validation."""
import hashlib
import ipaddress
import json
from urllib.parse import urlsplit

# Copied from reviewed ../campaign_validation.py: fixed integral/string workload,
# Go encoding/json HTML and U+2028/U+2029 escaping. Dicts below are built in Go
# struct order; the few fixed map keys are explicitly in lexical order.
def go_json(value):
    return json.dumps(value, ensure_ascii=False, separators=(",", ":")).replace("<", r"\u003c").replace(">", r"\u003e").replace("&", r"\u0026").replace("\u2028", r"\u2028").replace("\u2029", r"\u2029")


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def equal(a, b):
    if type(a) is not type(b):
        return False
    if isinstance(a, dict):
        return a.keys() == b.keys() and all(equal(a[k], b[k]) for k in a)
    if isinstance(a, list):
        return len(a) == len(b) and all(equal(x, y) for x, y in zip(a, b))
    return a == b


def exact(a, b, reason):
    require(equal(a, b), reason)


def digest(value):
    return hashlib.sha256(go_json(value).encode()).hexdigest()


def text(value):
    return {"Kind": "string", "Number": 0, "Text": value, "Flag": False}


def run(k):
    return {"Subject": "series-"+str(k), "Generation": 1}


def operation(k):
    return go_json(["series-"+str(k), 1, 1 if k == 1 else 2, 0])


def retained_schema(name):
    record = {name: {"Kind": "string", "MaxBytes": 64, "Enum": None}}
    return {"ID": digest(record), "Record": record}


def fixed_transport(t):
    # Same frozen constants as reviewed campaign_validation.TRANSPORT. Only the
    # originally controlled nonzero listen address differs from timing's :0.
    endpoint = urlsplit(t["endpoint"])
    address = t["listen_address"]
    require(endpoint.scheme == "http" and endpoint.hostname is not None and
            ipaddress.ip_address(endpoint.hostname).is_loopback and endpoint.port is not None and
            endpoint.port > 0 and endpoint.username is None and endpoint.password is None and
            endpoint.path == "/measure" and not endpoint.query and not endpoint.fragment,
            "numeric fixed loopback endpoint required")
    # Canonical equality rules out userinfo, port aliases, alternate paths and
    # host spelling changes. IPv6 brackets are retained exactly when present.
    host = endpoint.hostname
    fixed_address = ("["+host+"]" if ":" in host else host)+":"+str(endpoint.port)
    exact(address, fixed_address, "fixed listen address differs from endpoint")
    exact(t["endpoint"], "http://"+address+"/measure", "endpoint/listener identity mismatch")
    expected = {"http_version": "HTTP/1.1", "dial_timeout_ns": 30_000_000_000,
                "tcp_keepalive_ns": 30_000_000_000, "idle_timeout_ns": 90_000_000_000,
                "response_header_timeout_ns": 30_000_000_000, "server_read_timeout_ns": 30_000_000_000,
                "server_write_timeout_ns": 30_000_000_000, "max_conns_per_host": 1,
                "max_header_bytes": 8192, "expect_continue_timeout_ns": 1_000_000_000,
                "proxy": False, "mode": "http", "listen_address": address, "endpoint": t["endpoint"],
                "protocol": "alan-p2-http/v1", "request_format": "json-encoding-go-effects.Prepared-v1",
                "response_format": "json-operation-key-response-sample-v1", "max_request_bytes": 65536,
                "max_response_bytes": 4096, "timeout_ns": 30_000_000_000, "reuse_connections": True,
                "redirects": False, "provider_retries": 0, "post_retries": False,
                "resource_scope": "same-process-server-included"}
    exact(t, expected, "complete frozen HTTP configuration mismatch")


def fixed_exchange(contact, namespace, endpoint, k):
    op = operation(k)
    request = {"ID": op, "Key": go_json([2, namespace, op]), "Capability": "meter", "Effect": "read",
               "Input": retained_schema("station"), "Output": retained_schema("sample"),
               "Payload": {"station": text("north")}, "Assets": []}
    scoped = dict(request, ID="", Key="")
    wire = go_json({"request": scoped, "protocol": "alan-p2-http/v1", "endpoint": endpoint})
    prepared = {"Request": request, "Digest": "", "Protocol": "alan-p2-http/v1", "Endpoint": endpoint,
                "Model": "", "MaxOutputTokens": 0, "MaxResponseBytes": 4096, "Wire": wire}
    prepared["Digest"] = digest(prepared)
    raw = go_json(prepared)
    response_id = "local-response:"+op
    response = go_json({"operation_id": op, "key": request["Key"], "response_id": response_id, "sample": "present"})
    require(len(raw.encode()) <= 65536 and len(response.encode()) <= 4096, "finite wire bounds exceeded")
    expected = {"prepared": prepared, "request_wire": raw, "request_sha256": hashlib.sha256(raw.encode()).hexdigest(),
                "response_wire": response, "response_sha256": hashlib.sha256(response.encode()).hexdigest(),
                "response_complete": True, "ordinal": k, "operation_id": op, "key": request["Key"],
                "prepared_digest": prepared["Digest"], "capability": "meter", "effect": "read",
                "protocol": prepared["Protocol"], "endpoint": endpoint, "response_id": response_id}
    exact(contact, expected, "canonical HTTP bytes/authentic Prepared digest/fixed request mismatch")
    candidate = {"ID": op, "Key": request["Key"], "Disposition": "validated", "Payload": {"sample": text("present")},
                 "HTTPStatus": 200, "RequestID": op, "ResponseID": response_id, "ReturnedModel": "", "Response": response, "Usage": None}
    values = {"disposition": text("validated"), "sample": text("present"),
              "valid": {"Kind": "bool", "Number": 0, "Text": "", "Flag": True}}
    result = {"Digest": "", "Candidate": candidate, "Values": values, "Disposition": "validated"}
    result["Digest"] = digest(result)
    return prepared, result


def ref_key(ref):
    require(isinstance(ref, dict), "missing selected reduction reference")
    require(type(ref["Sequence"]) is int and ref["Sequence"] > 0, "invalid selected sequence")
    exact(ref["Run"], run(int(ref["Run"]["Subject"].split("-")[1])), "foreign selected RunID")
    return (ref["Run"]["Subject"], ref["Run"]["Generation"], ref["Sequence"])


def roots_at_cut(roots, membership, completed, original_outcomes):
    require(type(roots) is list and len(roots) == len(membership), "complete root membership required")
    require(len({r["RootID"] for r in roots}) == len(roots), "duplicate root identity")
    by_run = {r["Run"]["Subject"]: r for r in roots}
    require(len(by_run) == len(roots), "duplicate root RunID")
    for k in range(1, len(membership)+1):
        root = by_run[run(k)["Subject"]]
        exact(root["Run"], run(k), "root generation/RunID mismatch")
        exact(root["RootID"], membership[k-1], "original root identity changed")
        exact(root["Limits"], {"MaxInvocations": 0, "MaxCreatedRuns": 0, "MaxDepth": 0, "MaxEffects": 1, "MaxInFlight": 1}, "root limits changed")
        exact(root["Charges"], {"RootID": root["RootID"], "OperationID": "", "Kind": "", "Invocations": 0,
                                "CreatedRuns": 0, "Effects": int(k <= completed), "DecisionOrder": 0}, "root debit mismatch")
        for name in ("PendingInvocations", "UnresolvedExternal", "UnknownExternal"):
            exact(root[name], [], "root pending/unknown mismatch")
        exact(root["LocalStatus"], "succeeded" if k <= completed else "open", "root local state mismatch")
        exact(root["Pending"], "" if k <= completed else "members", "root pending members mismatch")
        outcome = root["Outcome"]
        if k > completed:
            exact(outcome, None, "future root has outcome")
        else:
            require(isinstance(outcome, dict) and isinstance(outcome["ID"], str) and bool(outcome["ID"]), "missing root outcome")
            require(type(outcome["DecisionOrder"]) is int and outcome["DecisionOrder"] > 0, "invalid root decision order")
            exact(outcome, {"ID": outcome["ID"], "InvocationID": "root:"+root["RootID"], "OperationID": "root:"+root["RootID"],
                            "TemplateID": "", "Caller": run(k), "Status": "succeeded", "Reason": "member_succeeded",
                            "ResponsePresent": False, "Response": None, "DecisionOrder": outcome["DecisionOrder"]}, "original root outcome mismatch")
            exact(outcome, original_outcomes.setdefault(k, outcome), "prior original outcome changed")
    require(len({v["ID"] for v in original_outcomes.values()}) == len(original_outcomes), "aliased root outcomes")


def validate_finite(r):
    n = r["bound"]
    require(type(n) is int and 1 <= n <= 256, "finite bound required")
    exact(r["ok"], True, "unsuccessful report")
    exact(r["scenario"], "success", "fixed success scenario required")
    require(type(r["reopen_every"]) is int and r["reopen_every"] >= 0, "invalid reopen schedule")
    require(not r.get("error"), "failed capture")
    fixed_transport(r["transport"])
    namespace = r["cuts"][0]["snapshot"]["Namespace"]
    require(isinstance(namespace, str) and len(namespace) == 64, "missing original namespace")
    initial = r["cuts"][0]["snapshot"]["Roots"]
    membership = []
    for k in range(1, n+1):
        roots = [v for v in initial if equal(v["Run"], run(k))]
        require(len(roots) == 1 and isinstance(roots[0]["RootID"], str) and len(roots[0]["RootID"]) == 64, "invalid initial root registry")
        membership.append(roots[0]["RootID"])
    expected_cuts = [("before-open", 0), ("after-open", 0)]
    for k in range(1, n+1):
        if k > 1:
            expected_cuts.append(("after-approval", k))
        expected_cuts.append(("after-run", k))
        if r["reopen_every"] > 0 and k % r["reopen_every"] == 0:
            expected_cuts.append(("after-interval-reopen", k))
    expected_cuts.append(("final-closed", n))
    exact([{ "label":c["label"], "position":c["position"]} for c in r["cuts"]],
          [{"label":label, "position":position} for label, position in expected_cuts], "finite cut order/position/type mismatch")
    exact(len(r["contacts"]), n, "complete original contact count required")
    original_outcomes = {}
    exchanges = [fixed_exchange(c, namespace, r["transport"]["endpoint"], k) for k, c in enumerate(r["contacts"], 1)]
    for c in r["cuts"]:
        completed = c["position"] - int(c["label"] == "after-approval")
        exact(c["contacts"], completed, "finite cut contact count/type mismatch")
        if completed == 0:
            exact(c["history"], None, "unexecuted capture History must be nil")
        else:
            require(type(c["history"]) is list, "executed capture History must be a list")
        s, replay, history, view = c["snapshot"], c["replay"], c["history"] or [], c["series_view"]
        for value in (s["Namespace"], replay["Namespace"], view["Namespace"]):
            exact(value, namespace, "capture namespace mismatch")
        exact(replay["HistoryCount"], len(history), "Replay/captured History count mismatch")
        exact(len(history), 4*completed-1 if completed else 0, "finite History reduction count mismatch")
        exact(replay["IntentCount"], completed, "Replay original intent count mismatch")
        require(type(replay["Runs"]) is list and len(replay["Runs"]) == n and len(s["Runs"]) == n, "Replay/capture must retain all runs")
        expected_runs = sorted([run(k) for k in range(1, n+1)], key=lambda v: v["Subject"])
        for identity, rr in zip(expected_runs, replay["Runs"]):
            exact(rr["ID"], identity, "Replay run membership mismatch")
            source = next(v for v in s["Runs"] if equal(v["ID"], identity))
            exact(rr, {key: source[key] for key in rr}, "Replay run differs from snapshot")
        roots_at_cut(s["Roots"], membership, completed, original_outcomes)
        exact(replay["Roots"], s["Roots"], "Replay roots differ from snapshot")
        for name, value in {"Bound": n, "Selected": completed, "Completed": completed, "Failed": 0,
                            "Pending": n-completed, "Unresolved": 0, "Spent": completed, "BoundExhausted": completed == n}.items():
            exact(view[name], value, "finite SeriesView count/state mismatch")
        rr = c["run_report"]
        if rr is not None:
            ordinary = c["label"] == "after-run"
            exact(rr["Steps"], (3 if completed == 1 else 4) if ordinary else 0, "finite RunReport Steps mismatch")
            exact(rr["Dispatches"], int(ordinary), "finite RunReport Dispatches mismatch")
            exact(rr["Quiescent"], True, "RunReport quiescence type/value mismatch")
            exact(rr["OpenRuns"], sorted([run(k) for k in range(completed+1, n+1)], key=lambda v: v["Subject"]), "finite RunReport OpenRuns mismatch")
            exact(rr["RejectedAdmissions"], 0, "RunReport rejected admission mismatch")
            for field in ("UnresolvedIntents", "PendingDeliveries", "UnknownExternal", "UnknownAdapters"):
                exact(rr[field], [], "RunReport pending/unknown list mismatch")
        require(type(view["Positions"]) is list and len(view["Positions"]) == n, "complete SeriesView positions required")
        steps = {}
        for h in history:
            key = ref_key({"Run": h["Run"], "Sequence": h["Sequence"]})
            require(key not in steps, "duplicate History reduction")
            steps[key] = h
        expected_steps = [(run(k), seq) for k in range(1, completed+1) for seq in range(1, 4 if k == 1 else 5)]
        exact([{"Run": h["Run"], "Sequence": h["Sequence"]} for h in history],
              [{"Run": identity, "Sequence": seq} for identity, seq in expected_steps], "finite History order/membership mismatch")
        for k in range(1, n+1):
            p = view["Positions"][k-1]
            exact(p["Index"], k, "position index/type mismatch")
            source = next(v for v in s["Runs"] if equal(v["ID"], run(k)))
            exact(p["Run"], source, "SeriesView run detached from snapshot")
            root = next(v for v in s["Roots"] if equal(v["Run"], run(k)))
            exact(p["Root"], root, "SeriesView root detached from snapshot")
            expected_state = "completed" if k <= completed else "not_eligible"
            if k == completed+1:
                expected_state = "approved" if c["label"] == "after-approval" else ("awaiting_request" if k == 1 else "awaiting_approval")
            exact(p["State"], expected_state, "finite position state mismatch")
            exact(source["Projection"]["Status"], "succeeded" if k <= completed else "open", "finite local run state mismatch")
            if k > completed:
                exact(p["Effect"], None, "future position has effect")
                exact(p["Observation"], None, "future position has selected observation")
                continue
            e = next(v for v in s["Effects"] if v["OperationID"] == operation(k))
            exact(p["Effect"], e, "SeriesView effect detached from snapshot")
            exact(e["Source"], run(k), "original effect RunID mismatch")
            for field, value in {"State": "settled", "ResultDisposition": "validated", "Kind": "adapter", "Capability": "meter", "Name": "read"}.items():
                exact(e[field], value, "finite retained effect state mismatch")
            request_sequence = 1 if k == 1 else 2
            exact(e["Sequence"], request_sequence, "original effect sequence mismatch")
            instructions = ["emit", "receive", "end"] if k == 1 else ["receive", "emit", "receive", "end"]
            for seq, instruction in enumerate(instructions, 1):
                h = steps[(run(k)["Subject"], 1, seq)]
                exact(h["Instruction"], instruction, "finite History instruction mismatch")
                require(type(h["Epoch"]) is int and h["Epoch"] > 0 and type(h["SourceRevision"]) is int and h["SourceRevision"] > 0, "invalid historical authority scalar")
                exact(h["Epoch"], e["Authorization"]["Source"]["Epoch"], "historical operation epoch mismatch")
                exact(h["SourceRevision"], e["SourceRevision"], "historical operation revision mismatch")
            exact(e["AdapterAttempt"]["Allowed"], True, "original attempt not allowed")
            exact(e["AdapterAttempt"]["Prepared"], exchanges[k-1][0], "retained Prepared differs from authentic exchange")
            exact(e["AdapterResult"], exchanges[k-1][1], "retained Candidate/result digest differs from authentic exchange")
            exact(e["Dispatch"], {"Minimum": 1, "Maximum": 1}, "original dispatch bounds mismatch")
            exact(source["EffectIDs"], [operation(k)], "original run/effect relation mismatch")
            obs = p["Observation"]
            exact(obs, next(v for v in s["Observations"] if v["ID"] == obs["ID"]), "SeriesView observation detached from snapshot")
            exact(obs["SourceOperationID"], operation(k), "selected observation source mismatch")
            exact(obs["SelectedAt"]["Run"], run(k), "selection RunID mismatch")
            exact(e["Settlement"]["ObservationID"], obs["ID"], "settlement/selection observation mismatch")
            exact(steps[ref_key(obs["SelectedAt"])]["SelectedObservationID"], obs["ID"], "selected History reduction mismatch")
            if k == 1:
                exact(p["Approval"], None, "first position must use provisioned approval")
        approvals = s.get("Continuations", [])
        wanted_approvals = max(0, completed-1)+int(c["label"] == "after-approval")
        exact(len(approvals), wanted_approvals, "protected approval count mismatch")
        # Public Continuations use runLess: lexical Subject, then Generation.
        # Sort only the expected registry; retain actual evidence order exactly.
        expected_targets = sorted([run(k) for k in range(2, wanted_approvals+2)],
                                  key=lambda identity: (identity["Subject"], identity["Generation"]))
        exact([a["Approval"]["Target"] for a in approvals], expected_targets, "protected approval target membership/order mismatch")
        for approval in approvals:
            a = approval["Approval"]
            k = int(a["Target"]["Subject"].split("-")[1])
            require(2 <= k <= n and k <= completed+int(c["label"] == "after-approval"), "foreign approval target")
            prev = exchanges[k-2]
            for field, value in {"Namespace": namespace, "Target": run(k), "PredecessorID": operation(k-1),
                                 "PreparedDigest": prev[0]["Digest"], "ResultDigest": prev[1]["Digest"],
                                 "Admitted": True, "Reason": "operator"}.items():
                exact(a[field], value, "protected approval predecessor/scope mismatch")
            require(isinstance(a["ScopeDigest"], str) and len(a["ScopeDigest"]) == 64, "missing protected scope digest")
            require(type(a["TargetRevision"]) is int and a["TargetRevision"] > 0, "invalid target revision")
            p = view["Positions"][k-1]
            exact(p["Approval"], approval, "SeriesView approval detached from snapshot")
            if k <= completed:
                exact(approval["SuccessorOperationID"], operation(k), "approval/successor identity mismatch")
                ref = approval["SelectedAt"]
                exact(ref["Run"], run(k), "approval selection run mismatch")
                require(ref["Sequence"] < p["Effect"]["Sequence"], "approval selected after successor")
                ob = next(v for v in s["Observations"] if v["ID"] == approval["ObservationID"])
                exact(ob["SelectedAt"], ref, "approval observation selection mismatch")
                exact(steps[ref_key(ref)]["SelectedObservationID"], approval["ObservationID"], "approval selected History mismatch")
            else:
                exact(approval["SelectedAt"], None, "unrun approval unexpectedly selected")
                exact(approval["SuccessorOperationID"], "", "unrun approval has successor")
