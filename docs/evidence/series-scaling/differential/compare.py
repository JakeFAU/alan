#!/usr/bin/env python3
"""Fail-closed logical comparison. Never writes an input report or database."""
import argparse
import copy
import hashlib
import json
import pathlib
import sqlite3
import sys


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


def compare_same_seed(left, right, left_db, right_db):
    for report in (left, right):
        if report.get("version") != "alan/series-differential/v1" or not report.get("ok"):
            raise ValueError("require successful differential runner reports")
        if report.get("mode") not in ("execute-existing", "generic-reader") or not report.get("cuts"):
            raise ValueError("require execute-existing or generic-reader reports with public cuts")
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
        "coverage": "complete exported public cuts+history+Replay+contacts and exact durable relational rows",
        "normalization": ["source_label", "store", "physical SQLite row order only"],
        "namespace": namespaces[0],
        "tables": {name: len(value["rows"]) for name, value in dumps[0]["tables"].items()},
        "public_cut_count": len(left["cuts"]),
    }


def normalize_driver(report):
    if report.get("version") != "alan/series-scaling/v1" or not report.get("invariants_ok"):
        raise ValueError("require successful timing-driver report")
    n = report["bound"]
    if len(report["evidence"]) != n or len(report["contacts"]) != n:
        raise ValueError("timing report identity counts differ")
    mapping = {report["namespace"]: "<namespace>"}
    manifest = []
    for index, (evidence, contact) in enumerate(zip(report["evidence"], report["contacts"]), 1):
        run = {"Subject": "series-" + str(index), "Generation": 1}
        if evidence["index"] != index or evidence["run"] != run or contact["ordinal"] != index:
            raise ValueError("identity must be mapped by original series position")
        operation = evidence["operation_id"]
        decoded = json.loads(operation)
        if decoded != [run["Subject"], 1, 1 if index == 1 else 2, 0]:
            raise ValueError("unexpected original operation identity")
        if contact["operation_id"] != operation or contact["prepared_digest"] != evidence["prepared_digest"]:
            raise ValueError("contact is detached from original operation")
        if json.loads(contact["key"]) != [2, report["namespace"], operation]:
            raise ValueError("request key is not namespace-bound to original operation")
        if contact["response_id"] != evidence["response_id"]:
            raise ValueError("response identity disagrees")
        for field in ("root_id", "prepared_digest", "result_digest"):
            original = evidence[field]
            if not isinstance(original, str) or len(original) != 64 or any(c not in "0123456789abcdef" for c in original):
                raise ValueError("invalid retained hash shape")
            if original in mapping:
                raise ValueError("aliased namespace-derived identities")
            mapping[original] = "<" + field + ":position=" + str(index) + ">"
        mapping[contact["key"]] = "<request-key:position=" + str(index) + ">"
        manifest.append({"position": index, "operation_id": operation, "root_id": evidence["root_id"], "prepared_digest": evidence["prepared_digest"], "result_digest": evidence["result_digest"], "request_key": contact["key"]})
    def replace(value):
        if isinstance(value, str):
            return mapping.get(value, value)
        if isinstance(value, list):
            return [replace(v) for v in value]
        if isinstance(value, dict):
            return {k: replace(v) for k, v in value.items()}
        return value
    fields = ("bound", "schedule", "reopen_every", "storage_version", "selected", "completed", "spent", "protected_approvals", "replay_history", "replay_intents", "contacts", "evidence")
    return replace({key: report[key] for key in fields}), manifest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("same-seed", "driver-relations"))
    parser.add_argument("baseline_report")
    parser.add_argument("candidate_report")
    parser.add_argument("--baseline-db")
    parser.add_argument("--candidate-db")
    args = parser.parse_args()
    left, right = read_json(args.baseline_report), read_json(args.candidate_report)
    result = {"version": "alan/series-differential-comparison/v1", "mode": args.mode,
              "baseline_report_sha256": sha(args.baseline_report), "candidate_report_sha256": sha(args.candidate_report)}
    try:
        if args.mode == "same-seed":
            if not args.baseline_db or not args.candidate_db:
                raise ValueError("same-seed requires both --baseline-db and --candidate-db")
            result.update(compare_same_seed(left, right, args.baseline_db, args.candidate_db))
        else:
            if args.baseline_db or args.candidate_db:
                raise ValueError("driver-relations cannot claim full database comparison")
            a, am = normalize_driver(left)
            b, bm = normalize_driver(right)
            difference = first_difference(a, b)
            if difference:
                raise ValueError("normalized retained driver relations differ: " + difference)
            result.update({"coverage": "limited timing-report identity relations only; NOT full semantic qualification",
                           "not_covered": ["full grants", "full charges", "authority epochs", "full prepared/result preimages", "full uncertainty/pending objects", "durable rows"],
                           "baseline_identity_mapping": am, "candidate_identity_mapping": bm})
        result["equal_for_stated_coverage"] = True
    except (ValueError, KeyError, TypeError, sqlite3.Error) as error:
        result["equal_for_stated_coverage"] = False
        result["error"] = str(error)
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0 if result["equal_for_stated_coverage"] else 1


if __name__ == "__main__":
    sys.exit(main())
