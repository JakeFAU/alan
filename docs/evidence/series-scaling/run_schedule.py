#!/usr/bin/env python3
"""Run an explicit serial schedule; each row is a fresh driver process/store."""
import datetime
import json
import os
from pathlib import Path
import sys
import time
from campaign_validation import (ENVIRONMENT, require, sha, tooling_hashes,
                                 validate_meta, validate_report, validate_schedule)


def main():
    schedule_path, output = map(Path, sys.argv[1:])
    schedule = json.loads(schedule_path.read_text())
    paired = any("variant" in row or "pair" in row for row in schedule)
    tools = tooling_hashes()
    if paired:
        validate_schedule(schedule)
        require(all(row["expected_tooling_sha256"] == tools for row in schedule),
                "campaign tooling differs from frozen schedule")
    output.mkdir(parents=True, exist_ok=True)
    schedule_hash = sha(schedule_path)
    runner_hash = tools["run_schedule.py"]
    env = os.environ.copy()
    for key in ("OPENAI_API_KEY", "ALAN_BARRIER", "GOMEMPROFILERATE"):
        env.pop(key, None)
    env.update(ENVIRONMENT)
    for row in schedule:
        name = row["name"]
        store = output / (name + ".store")
        report = output / (name + ".json")
        progress = output / (name + ".progress")
        meta = output / (name + ".meta.json")
        if any(p.exists() for p in (store, report, progress, meta)):
            raise RuntimeError("refusing to overwrite sample " + name)
        binary = str(Path(row["binary"]).resolve())
        before_hash = sha(binary)
        if paired:
            require(before_hash == row["expected_binary_sha256"], "executable changed before sample " + name)
        command = [binary, "-bound", str(row["bound"]), "-dir", str(store),
                   "-sample", name, "-source-label", row["source"]]
        command += row.get("args", [])
        record = {"schedule_sha256": schedule_hash, "schedule_row": row, "runner_sha256": runner_hash,
                  "validation_sha256": tools["campaign_validation.py"],
                  "command": command, "cwd": os.getcwd(), "binary_sha256": before_hash,
                  "environment": {k: env[k] for k in ENVIRONMENT},
                  "started_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                  "load_before": os.getloadavg()}
        print("START " + name, flush=True)
        with report.open("xb") as out, progress.open("xb") as err:
            start = time.monotonic()
            pid = os.fork()
            if pid == 0:
                os.dup2(out.fileno(), 1)
                os.dup2(err.fileno(), 2)
                os.execve(binary, command, env)
            _, status, usage = os.wait4(pid, 0)
        record.update(exit=os.waitstatus_to_exitcode(status), seconds=time.monotonic() - start,
                      user_cpu_seconds=usage.ru_utime, system_cpu_seconds=usage.ru_stime,
                      process_peak_rss=usage.ru_maxrss,
                      process_peak_rss_unit="bytes" if sys.platform == "darwin" else "KiB",
                      load_after=os.getloadavg(), report_sha256=sha(report), progress_sha256=sha(progress),
                      binary_sha256_after=sha(binary))
        result, good, windows = {}, False, []
        try:
            result = json.loads(report.read_text())
            if paired:
                require(record["binary_sha256_after"] == row["expected_binary_sha256"],
                        "executable changed during sample " + name)
                validate_report(result, row)
                validate_meta(record, result, row, schedule_hash, sha(report), sha(progress))
                good = True
                windows = [s for s in result["stages"] if s["name"] == "execute_window"]
            else:
                # Legacy diagnostic rows lack pair/config hashes. They never enter
                # analyze_pairs; the strict paired path above is mandatory there.
                windows = [s for s in result["stages"] if s["name"].endswith("window") or s["name"].startswith("execution_window")]
                good = record["exit"] == 0 and result.get("invariants_ok") is True and not result.get("error")
                good = good and len(windows) == 1 and result["bound"] == row["bound"]
                good = good and result["gomaxprocs"] == 4 and result["sample"] == name
                good = good and all(result[k] == row["bound"] for k in ("completed", "selected", "spent"))
                good = good and result["protected_approvals"] == row["bound"] - 1
                good = good and len(result["contacts"]) == row["bound"] and len(result["evidence"]) == row["bound"]
                good = good and result["instrumented"] == row.get("instrumented", False)
                good = good and result["profiled"] == row.get("profiled", False)
        except (ValueError, KeyError, TypeError) as error:
            record["validation_error"] = str(error)
        with meta.open("x") as out:
            out.write(json.dumps(record, indent=2) + "\n")
        print(json.dumps({"sample": name, "exit": record["exit"], "invariants_ok": good,
                          "process_seconds": record["seconds"],
                          "execute_seconds": windows[0]["wall_ns"] / 1e9 if windows else None,
                          "execute_alloc_bytes": windows[0]["total_alloc_bytes"] if windows else None}), flush=True)
        if not good:
            raise RuntimeError("sample failed: " + name + ": " + record.get("validation_error", "proof/configuration failure"))


if __name__ == "__main__":
    main()
