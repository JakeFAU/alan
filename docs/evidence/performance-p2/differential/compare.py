#!/usr/bin/env python3
"""Fail-closed logical comparison. Never writes an input report or database."""
import argparse
import copy
import hashlib
import json
import pathlib
import sqlite3
import sys
from finite_validation import validate_finite


def read_json(path):
    return json.loads(pathlib.Path(path).read_text())


def sha(path):
    return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()


def canonical(value):
    return json.dumps(value, sort_keys=True, ensure_ascii=False, separators=(",", ":"))


def first_difference(a, b, path="$"):
    if type(a) is not type(b):
        return path + ": type differs"
    if isinstance(a, dict):
        if a.keys() != b.keys():
            return path + ": keys differ"
        for key in sorted(a):
            diff = first_difference(a[key], b[key], path + "." + key)
            if diff:
                return diff
    elif isinstance(a, list):
        if len(a) != len(b):
            return path + ": length differs"
        for index, (left, right) in enumerate(zip(a, b)):
            diff = first_difference(left, right, path + "[" + str(index) + "]")
            if diff:
                return diff
    elif a != b:
        return path + ": value differs"
    return None


def table_dump(path):
    # mode=ro, not immutable: SQLite must honor any retained WAL. Caller supplies
    # closed qualification copies; never read or hash an actively changing DB.
    source = pathlib.Path(path).resolve(strict=True)
    connection = sqlite3.connect(source.as_uri() + "?mode=ro", uri=True)
    try:
        connection.execute("PRAGMA query_only=ON")
        connection.execute("BEGIN")
        schema = connection.execute(
            "SELECT type,name,tbl_name,sql FROM sqlite_master "
            "WHERE name NOT LIKE 'sqlite_%' ORDER BY type,name"
        ).fetchall()
        tables = {}
        for (name,) in connection.execute(
            "SELECT name FROM sqlite_master WHERE type='table' "
            "AND name NOT LIKE 'sqlite_%' ORDER BY name"
        ):
            quoted = '"' + name.replace('"', '""') + '"'
            columns = [row[1] for row in connection.execute("PRAGMA table_info(" + quoted + ")")]
            rows = []
            for row in connection.execute("SELECT * FROM " + quoted):
                # Preserve SQL NULL/integer/text/blob types and exact blob bytes.
                rows.append([
                    ["blob", value.hex()] if isinstance(value, bytes)
                    else ["null", None] if value is None
                    else [type(value).__name__, value]
                    for value in row
                ])
            rows.sort(key=canonical)  # physical row order is not logical history
            tables[name] = {"columns": columns, "rows": rows}
        namespace = connection.execute("SELECT namespace FROM metadata WHERE singleton=1").fetchone()[0]
        return {"schema": [list(row) for row in schema], "tables": tables}, namespace
    finally:
        connection.close()


def validate_report(r):
    validate_finite(r)
    n = r["bound"]
    if type(n) is not int or not 1 <= n <= 256 or r["scenario"] != "success" or r.get("error"):
        raise ValueError("require complete successful finite HTTP report")
    t = r["transport"]
    if t["mode"] != "http" or t["provider_retries"] != 0 or t["post_retries"] or not t["endpoint"]:
        raise ValueError("unexpected transport policy")
    expected = [("before-open", 0), ("after-open", 0)]
    for k in range(1, n+1):
        if k > 1:
            expected.append(("after-approval", k))
        expected.append(("after-run", k))
        if r["reopen_every"] > 0 and k % r["reopen_every"] == 0:
            expected.append(("after-interval-reopen", k))
    expected.append(("final-closed", n))
    if [(c["label"], c["position"]) for c in r["cuts"]] != expected:
        raise ValueError("missing, duplicate, reordered or foreign cut")
    if len(r["contacts"]) != n or r["server_contacts"] != r["contacts"]:
        raise ValueError("adapter/handler contacts do not form a bijection")
    for c in r["cuts"]:
        count = c["position"] - (c["label"] == "after-approval")
        if c["contacts"] != count or c["adapter_contacts"] != r["contacts"][:count] or c["server_contacts"] != r["server_contacts"][:count]:
            raise ValueError("cut journal prefix mismatch")
        rr = c["run_report"]
        if c["label"] in ("after-run", "after-interval-reopen"):
            if not isinstance(rr, dict) or set(rr) != {"Steps", "Dispatches", "Quiescent", "OpenRuns", "UnresolvedIntents", "PendingDeliveries", "UnknownExternal", "UnknownAdapters", "Roots", "RejectedAdmissions"}:
                raise ValueError("missing/incomplete returned RunReport")
            if not rr["Quiescent"] or len(rr["Roots"]) != n or rr["RejectedAdmissions"] != 0 or any(rr[k] for k in ("UnresolvedIntents", "PendingDeliveries", "UnknownExternal", "UnknownAdapters")):
                raise ValueError("RunReport cardinality/pending/unknown mismatch")
            if rr["Roots"] != c["snapshot"]["Roots"]:
                raise ValueError("returned roots differ from retained inspection")
        elif rr is not None:
            raise ValueError("non-Run cut fabricated a returned report")
        if len(c["snapshot"]["Effects"] or []) != count:
            raise ValueError("contact/effect prefix mismatch")
        for j, contact in enumerate(c["adapter_contacts"]):
            prepared = contact["prepared"]
            operation = '["series-%d",1,%d,0]' % (j+1, 1 if j == 0 else 2)
            retained = next(e for e in c["snapshot"]["Effects"] if e["OperationID"] == operation)
            candidate = retained["AdapterResult"]["Candidate"]
            if prepared != retained["AdapterAttempt"]["Prepared"] or prepared["Request"]["ID"] != operation or contact["operation_id"] != operation or contact["ordinal"] != j+1:
                raise ValueError("original Prepared/contact provenance mismatch")
            if contact["key"] != prepared["Request"]["Key"] or candidate["Key"] != contact["key"] or candidate["ID"] != operation or contact["prepared_digest"] != prepared["Digest"]:
                raise ValueError("request identity/key/digest mismatch")
            if contact["endpoint"] != t["endpoint"] or prepared["Endpoint"] != t["endpoint"] or json.loads(contact["request_wire"]) != prepared:
                raise ValueError("HTTP envelope/endpoint mismatch")
            for field in ("request", "response"):
                if hashlib.sha256(contact[field+"_wire"].encode()).hexdigest() != contact[field+"_sha256"]:
                    raise ValueError("HTTP byte digest mismatch")
            response_id = "local-response:"+operation
            response = {"operation_id": operation, "key": contact["key"], "response_id": response_id, "sample": "present"}
            if not contact["response_complete"] or json.loads(contact["response_wire"]) != response or contact["response_id"] != response_id or candidate["ResponseID"] != response_id or candidate["RequestID"] != operation or candidate["Response"] != contact["response_wire"] or candidate["Disposition"] != "validated" or candidate["HTTPStatus"] != 200:
                raise ValueError("original HTTP response identity/result mismatch")


def compare_same_seed(left, right, left_db, right_db):
    for report in (left, right):
        if report.get("version") != "alan/performance-p2-http-differential/v1" or not report.get("ok"):
            raise ValueError("require successful differential runner reports")
        if report.get("mode") != "execute-existing" or not report.get("cuts"):
            raise ValueError("require execute-existing or generic-reader reports with public cuts")
    for report in (left, right):
        validate_report(report)
    namespaces = [r["cuts"][0]["snapshot"]["Namespace"] for r in (left, right)]
    if namespaces[0] != namespaces[1]:
        raise ValueError("exact comparison requires clones of the SAME seed namespace")
    # Only invocation provenance is normalized. No epochs, revisions, IDs,
    # grants, hashes, observations, payloads, contacts, or uncertainty are removed.
    normalized = []
    for report in (left, right):
        item = copy.deepcopy(report)
        for key in ("source_label", "store"):
            item.pop(key)
        normalized.append(item)
    difference = first_difference(*normalized)
    if difference:
        raise ValueError("public report differs: " + difference)
    dumps = []
    for path in (left_db, right_db):
        dump, namespace = table_dump(path)
        if namespace != namespaces[0]:
            raise ValueError("database namespace does not match report")
        dumps.append(dump)
    difference = first_difference(*dumps)
    if difference:
        raise ValueError("durable relational rows differ: " + difference)
    return {
        "coverage": "complete public cuts+History+Replay+SeriesView+returned RunReport+both HTTP journals and exact typed SQL schema/cells",
        "normalization": ["source_label", "store", "physical SQLite row order only"],
        "namespace": namespaces[0],
        "tables": {name: len(value["rows"]) for name, value in dumps[0]["tables"].items()},
        "public_cut_count": len(left["cuts"]),
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("same-seed",))
    parser.add_argument("baseline_report")
    parser.add_argument("candidate_report")
    parser.add_argument("--baseline-db")
    parser.add_argument("--candidate-db")
    args = parser.parse_args()
    left, right = read_json(args.baseline_report), read_json(args.candidate_report)
    result = {"version": "alan/performance-p2-http-differential-comparison/v1", "mode": args.mode,
              "baseline_report_sha256": sha(args.baseline_report), "candidate_report_sha256": sha(args.candidate_report)}
    try:
        if args.mode == "same-seed":
            if not args.baseline_db or not args.candidate_db:
                raise ValueError("same-seed requires both --baseline-db and --candidate-db")
            result.update(compare_same_seed(left, right, args.baseline_db, args.candidate_db))
        result["equal_for_stated_coverage"] = True
    except (ValueError, KeyError, TypeError, IndexError, AttributeError, StopIteration, sqlite3.Error) as error:
        result["equal_for_stated_coverage"] = False
        result["error"] = str(error)
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0 if result["equal_for_stated_coverage"] else 1


if __name__ == "__main__":
    sys.exit(main())
