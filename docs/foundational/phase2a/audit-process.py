#!/usr/bin/env python3
"""Reducer-free second decoding of Phase 2A process SQLite and ingress evidence.

Authored separately from the test verdict code but by the apparatus implementer;
this is an independent mechanism, not independent-author review. It never imports
ALAN or the reference model. SQLite, gzip and recorded process attestations are
trusted. An independent reviewer must still challenge boundary placement/source.
"""
import argparse
import gzip
import hashlib
import json
import pathlib
import sqlite3
import tempfile


def require(value, message):
    if not value:
        raise AssertionError(message)


def database(directory, recorded):
    path = directory / pathlib.Path(recorded).name
    # A coherent backup can retain the WAL file-header mode. SQLite deserialize
    # cannot open that image in memory; read an immutable temporary file instead.
    with tempfile.NamedTemporaryFile() as temporary:
        temporary.write(gzip.decompress(path.read_bytes())); temporary.flush()
        connection = sqlite3.connect("file:" + temporary.name + "?mode=ro&immutable=1", uri=True)
        require(connection.execute("PRAGMA integrity_check").fetchone()[0] == "ok", str(path))
    connection.execute("PRAGMA query_only=on")
    return connection


def dump(connection):
    names = [r[0] for r in connection.execute("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")]
    return {name: connection.execute('SELECT * FROM "' + name + '" ORDER BY rowid').fetchall() for name in names}


def number(connection, table):
    return connection.execute('SELECT count(*) FROM "' + table + '"').fetchone()[0]


def documents(connection, table, column="descriptor"):
    return [json.loads(row[0]) for row in connection.execute('SELECT "' + column + '" FROM "' + table + '" ORDER BY rowid')]


def world_state(connection):
    receipts = documents(connection, "applications", "receipt")
    contacts = documents(connection, "contacts", "request")
    for index, receipt in enumerate(receipts, 1):
        require(receipt["Mutation"] == index, "nonsequential application")
        expected = sum(r["Delta"] for r in receipts[:index])
        require(receipt["Total"] == expected, "counter total false")
        require(any(r["Key"] == receipt["Key"] and r["Profile"] == receipt["Profile"] and r["Delta"] == receipt["Delta"] for r in contacts), "application lacks request")
    return contacts, receipts


def inspect_consistent(connection, inspection):
    raw = dump(connection)
    require(set(raw) == set(inspection["Tables"]), "inspection omits SQL table")
    for name in raw:
        columns = [r[1] for r in connection.execute('PRAGMA table_info("' + name + '")')]
        expected = []
        for values in raw[name]:
            expected.append(dict(zip(columns, [v.decode() if isinstance(v, bytes) else v for v in values])))
        require(expected == inspection["Tables"][name], "inspection differs from raw SQL: " + name)


def inspect_world_consistent(connection, snapshot):
    contacts, receipts = world_state(connection)
    require(len(contacts) == snapshot["Contacts"], "inspection contacts false")
    require(receipts == (snapshot["Applications"] or []), "inspection applications false")
    require(contacts == (snapshot["Requests"] or []), "inspection requests false")


def verify_binding(local, world, route):
    table = "adapter_attempts" if route == "adapter" else "external_attempts"
    attempts = documents(local, table)
    require(len(attempts) == 1 and attempts[0]["Allowed"], "not a unique allowed reservation")
    attempt = attempts[0]
    intent = attempt["Intent"]
    authorizations = documents(local, "attempts", "result")
    require(any(a["ID"] == attempt["AuthorizationID"] and a["IntentID"] == intent["ID"] and a["Allowed"] for a in authorizations), "reservation lacks original authority")
    request = attempt["Prepared"]["Request"] if route == "adapter" else attempt["Request"]
    key = request["Key"]
    contacts, receipts = world_state(world)
    require(all(c["Key"] == key for c in contacts), "world used another operation key")
    if route != "adapter":
        require(all(c == request for c in contacts), "raw request bytes/fields changed")
    else:
        require(all(c["Delta"] == request["Payload"]["delta"]["Number"] and c["Profile"] == "idempotent" for c in contacts), "adapter payload differs")
        require(all(c["Authorization"] == "" and c["Epoch"] == 0 for c in contacts), "adapter fabricated private identity")
    require(all(r["Key"] == key for r in receipts), "receipt identity mismatch")
    return key


def check_directory(directory):
    directory = pathlib.Path(directory)
    data = {path.name: json.loads(path.read_text()) for path in directory.glob("*.json")}
    counts = {"paired_branches": 0, "pairs": 0, "known_crashes": 0, "parent_crashes": 0, "replays": 0, "result_modes": 0, "endpoint_controls": 0}
    pairs = {}
    for name, record in data.items():
        if "cut_local_store" in record and "branch" in record:
            cut = database(directory, record["cut_local_store"])
            after = database(directory, record["after_local_store"])
            wc = database(directory, record["cut_world_store"])
            wa = database(directory, record["after_world_store"])
            inspect_consistent(cut, record["cut"])
            inspect_consistent(after, record["after"])
            inspect_world_consistent(wc, record["world_cut"])
            inspect_world_consistent(wa, record["world_after"])
            route, branch = record["route"], record["branch"]
            verify_binding(cut, wc, route)
            verify_binding(after, wa, route)
            results = "adapter_results" if route == "adapter" else "external_receipts"
            require(number(cut, results) == 0 and number(cut, "settlements") == 0, "cut contains result")
            contacts, applications = world_state(wc)
            final_contacts, final_applications = world_state(wa)
            index = int(branch[1])
            require(len(applications) == index and len(contacts) == index, "world cuts not U0/U1")
            require(record["signal"] == "SIGKILL" and record["crash_PID"] != record["recovery_PID"], "not distinct crash/recovery")
            status = json.loads(after.execute("SELECT projection FROM runs WHERE subject='client' AND generation=1").fetchone()[0])["Status"]
            if route == "idempotent":
                require(len(final_contacts) == index + 1 and len(final_applications) == 1, "idempotent recovery activity differs")
                require(number(after, results) == 1 and status == "succeeded", "idempotent did not progress")
                if index:
                    require(applications == final_applications, "original result changed")
            else:
                require(contacts == final_contacts and applications == final_applications, "conservative recovery repeated")
                require(number(after, results) == 0 and status == "open", "conservative recovery invented continuation")
            pairs.setdefault(route, {})[branch] = dump(cut)
            counts["paired_branches"] += 1
        elif name.endswith("-known.json"):
            cut = database(directory, record["cut_local_store"])
            after = database(directory, record["after_local_store"])
            wc = database(directory, record["cut_world_store"])
            wa = database(directory, record["after_world_store"])
            inspect_consistent(cut, record["stopped"]); inspect_consistent(after, record["after"])
            require(dump(wc) == dump(wa), "known recovery contacted world")
            table = "adapter_results" if record["route"] == "adapter" else "external_receipts"
            require(number(cut, table) == number(after, table) == 1, "result lost/duplicated")
            require(number(cut, "settlements") == (record["stage"] == "settled") and number(after,"settlements") == 1, "settlement cut false")
            observation = after.execute("SELECT observation_id FROM settlements").fetchone()[0]
            require(after.execute("SELECT count(*) FROM history WHERE selected_observation=?", (observation,)).fetchone()[0] == 1, "settlement not consumed exactly once")
            require(record["signal"] == "SIGKILL" and record["crash_PID"] != record["recovery_PID"], "known not actual fresh recovery")
            counts["known_crashes"] += 1
        elif name.endswith("-parent-crash.json"):
            cut = database(directory, record["cut_local_store"]);after = database(directory, record["after_local_store"])
            inspect_consistent(cut, record["stopped"]);inspect_consistent(after, record["after"])
            wc=database(directory,record["cut_world_store"]);wa=database(directory,record["after_world_store"])
            require(dump(wc)==dump(wa),"parent recovery changed world")
            root=json.loads(after.execute("SELECT projection FROM runs WHERE subject='root' AND generation=1").fetchone()[0])
            if record["stage"] == "unknown":
                require(root["Status"]=="open" and "child_status" not in root["Data"],"parent invented outcome")
                require(number(after,"adapter_results")==0 and number(wa,"applications")==0,"unknown parent mismatch")
            else:
                require(root["Status"]=="succeeded" and number(after,"adapter_results")==1,"parent known failed progress")
            require(record["signal"]=="SIGKILL" and record["crash_PID"]!=record["recovery_PID"],"parent not actual fresh recovery")
            counts["parent_crashes"]+=1
        elif "before_world_store" in record:
            require(dump(database(directory,record["before_local_store"]))==dump(database(directory,record["after_local_store"])),"replay changed SQL local")
            require(dump(database(directory,record["before_world_store"]))==dump(database(directory,record["after_world_store"])),"replay changed SQL world")
            require(record["ingress_before"]==record["ingress_after"],"replay made invalid contact")
            counts["replays"]+=1
        elif name.endswith("-result.json") and "adapter_mode" in record:
            local=database(directory,record["local_store"]);world=database(directory,record["world_store"])
            verify_binding(local,world,"adapter");inspect_consistent(local,record["after"])
            require(number(world,"applications")==1,"result mode lacks remote mutation")
            results=documents(local,"adapter_results")
            if record["adapter_mode"]=="unknown":require(not results,"unknown result persisted")
            elif record["endpoint_mode"]=="drop-after-commit":require(results[0]["Candidate"]["Disposition"]=="transport_error","not actual transport error")
            elif record["adapter_mode"]=="selected-error":require(results[0]["Candidate"]["Disposition"]=="selected_error","not selected unsuccessful")
            counts["result_modes"]+=1
        elif name.endswith("-endpoint-controls.json"):
            world=database(directory,record["world_store"]);contacts,receipts=world_state(world)
            require(len(contacts)==4 and len(receipts)==2,"endpoint controls count false")
            require(record["first"]==record["after_restart"]==record["historical_receipt_after_interference"],"original receipt not preserved")
            require(receipts[-1]["Total"]!=record["first"]["Total"],"no later interference")
            journal=[json.loads(line) for line in (directory/pathlib.Path(record["journal"]).name).read_text().splitlines()]
            require(sum(r["Event"]=="ingress" for r in journal)==5,"physical contacts should include rejected altered request")
            require(sum(r["Event"]=="rejected" and r["Status"]==409 for r in journal)==1,"changed request lacks actual rejection")
            require(receipts[0]["Process"]!=receipts[1]["Process"],"endpoint restart not reflected in new receipt")
            counts["endpoint_controls"]+=1
    for route, branches in pairs.items():
        require(set(branches)=={"U0","U1"}, "incomplete pair " + route)
        require(branches["U0"]==branches["U1"],"raw SQL local cuts differ: "+route)
        counts["pairs"]+=1
    expected={"paired_branches":6,"pairs":3,"known_crashes":4,"parent_crashes":2,"replays":28,"result_modes":4,"endpoint_controls":1}
    require(counts==expected, "coverage incomplete: "+repr(counts)+" expected "+repr(expected))
    return {"directory":str(directory),"counts":counts,"verdict":"PASS","mechanism":"stdlib SQLite raw backups and ingress JSONL; no ALAN/model imports","auditor_sha256":hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest()}


if __name__ == "__main__":
    parser=argparse.ArgumentParser();parser.add_argument("directories",nargs="+");args=parser.parse_args()
    print(json.dumps([check_directory(directory) for directory in args.directories],indent=2))
