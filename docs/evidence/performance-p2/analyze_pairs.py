#!/usr/bin/env python3
"""Analyze all six predeclared pairs; do not select or discard valid samples."""
import argparse
import hashlib
import json
import math
from pathlib import Path
import statistics
from campaign_validation import (BODY_PHASES, require, sha, tooling_hashes,
                                 validate_meta, validate_report, validate_schedule)


def summary(values):
    return {"median": statistics.median(values), "min": min(values), "max": max(values),
            "values": values}


def window(report):
    rows = [s for s in report["stages"] if s["name"] == "execute_window"]
    if len(rows) != 1:
        raise ValueError("ordinary Execute window required")
    return rows[0]


def phase_times(report):
    # Zero approval time at bound1 follows its declared zero multiplicity.
    # All positive-multiplicity phases were required by validate_report.
    out = {name: 0 for name in BODY_PHASES}
    for stage in report["stages"]:
        if stage["name"] != "execute_window":
            out[stage["name"]] = out.get(stage["name"], 0) + stage["wall_ns"] / 1e9
    return out


def sign_test(differences):
    """Predeclared one-sided six-sign check. Ties cannot satisfy all-six gate."""
    require(len(differences) == 6, "primary endpoint requires six paired differences")
    wins = sum(d > 0 for d in differences)
    losses = sum(d < 0 for d in differences)
    return {"baseline_minus_candidate_seconds": differences, "positive": wins, "negative": losses,
            "ties": 6-wins-losses, "all_six_positive": wins == 6,
            "one_sided_fair_sign_tail_p": sum(math.comb(6, k) for k in range(wins, 7))/64,
            "all_six_null_probability": 1/64,
            "assumption": "six independent fair null signs; ties are not wins; shared-host drift limits inference"}


def fit_growth(bounds, medians):
    # Descriptive finite-range comparisons; no extrapolation/asymptotic claim.
    result = {"adjacent_log_slopes": [math.log(medians[i] / medians[i-1]) / math.log(bounds[i] / bounds[i-1]) for i in range(1, len(bounds))]}
    models = {}
    for power in (1, 2, 3):
        xs = [n**power for n in bounds]
        xbar, ybar = statistics.mean(xs), statistics.mean(medians)
        b = sum((x-xbar)*(y-ybar) for x, y in zip(xs, medians)) / sum((x-xbar)**2 for x in xs)
        a = ybar - b*xbar
        residuals = [y - (a+b*x) for x, y in zip(xs, medians)]
        models["constant_plus_n_to_" + str(power)] = {"constant": a, "coefficient": b, "residual_seconds": residuals,
            "rmse_seconds": math.sqrt(statistics.mean(r*r for r in residuals)), "any_nonpositive_prediction": any(a+b*x <= 0 for x in xs)}
    xs, ys = [math.log(n) for n in bounds], [math.log(y) for y in medians]
    xbar, ybar = statistics.mean(xs), statistics.mean(ys)
    b = sum((x-xbar)*(y-ybar) for x, y in zip(xs, ys)) / sum((x-xbar)**2 for x in xs)
    a = ybar - b*xbar
    models["log_linear_power"] = {"coefficient": math.exp(a), "exponent": b,
        "log_residuals": [y-(a+b*x) for x, y in zip(xs, ys)]}
    result["finite_models"] = models
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("schedule", type=Path)
    parser.add_argument("samples", type=Path)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    schedule_bytes = args.schedule.read_bytes()
    schedule_hash = hashlib.sha256(schedule_bytes).hexdigest()
    rows = json.loads(schedule_bytes)
    variants = validate_schedule(rows)
    current_tools = tooling_hashes()
    require(all(row["expected_tooling_sha256"] == current_tools for row in rows),
            "analysis tooling differs from frozen schedule")
    bounds = [1, 8, 16, 32, 64, 128, 256]
    results = {}
    warmups = 0
    artifacts = []
    for row in rows:
        name = row["name"]
        raw = (args.samples / (name + ".json")).read_bytes()
        report = json.loads(raw)
        meta = json.loads((args.samples / (name + ".meta.json")).read_text())
        progress_hash = sha(args.samples / (name + ".progress"))
        validate_report(report, row)
        validate_meta(meta, report, row, schedule_hash, hashlib.sha256(raw).hexdigest(), progress_hash)
        artifacts.append({"name": name, "report_sha256": hashlib.sha256(raw).hexdigest(),
                          "progress_sha256": progress_hash,
                          "meta_sha256": sha(args.samples / (name + ".meta.json"))})
        if row["warmup"]:
            warmups += 1
            continue
        key = (row["bound"], row["pair"], row["variant"])
        if key in results:
            raise ValueError("duplicate pair member")
        results[key] = (report, meta)
    if warmups != 14 or len(results) != 84:
        raise ValueError("require all 14 excluded warmups and 84 measured processes")
    output = {"schedule_sha256": schedule_hash, "pairs_per_bound": 6,
              "warmups_excluded": warmups, "primary": "256-operation ordinary Execute", "bounds": [],
              "variants": {v: {"binary": identity[0], "expected_binary_sha256": identity[1], "source": identity[2]} for v, identity in variants.items()},
              "runtime_configuration": rows[0]["expected_runtime"],
              "transport_configuration": rows[0]["expected_transport"],
              "report_version": rows[0]["report_version"],
              "environment": rows[0]["expected_environment"],
              "tooling_sha256": rows[0]["expected_tooling_sha256"], "validated_artifacts": artifacts}
    for n in bounds:
        groups = {v: [results[(n, pair, v)] for pair in range(1, 7)] for v in ("baseline", "candidate")}
        metrics = {}
        for variant, reports in groups.items():
            metrics[variant] = {
                "execute_seconds": summary([window(r)["wall_ns"]/1e9 for r, _ in reports]),
                "allocated_bytes": summary([window(r)["total_alloc_bytes"] for r, _ in reports]),
                "allocation_count": summary([window(r)["mallocs"] for r, _ in reports]),
                "cpu_seconds": summary([(window(r)["user_cpu_ns"]+window(r)["system_cpu_ns"])/1e9 for r, _ in reports]),
                "gc_count": summary([window(r)["gc_count"] for r, _ in reports]),
                "gc_pause_ns": summary([window(r)["gc_pause_ns"] for r, _ in reports]),
                "runtime_metrics": {name: summary([window(r)["runtime_metric_deltas"][name] for r, _ in reports]) for name in reports[0][0]["runtime_metric_names"]},
                "prefix_vectors": [r["prefixes"] for r, _ in reports],
                "prefix_residuals": [r["prefix_residual"] for r, _ in reports],
                "process_peak_rss_bytes": summary([m["process_peak_rss"] for _, m in reports]),
            }
        b = metrics["baseline"]["execute_seconds"]
        c = metrics["candidate"]["execute_seconds"]
        ratios = [cv/bv for bv, cv in zip(b["values"], c["values"])]
        phases = {}
        all_names = BODY_PHASES
        for phase in all_names:
            before = summary([phase_times(r)[phase] for r, _ in groups["baseline"]])
            after = summary([phase_times(r)[phase] for r, _ in groups["candidate"]])
            phases[phase] = {"baseline": before, "candidate": after,
                "increase_above_10percent_or_half_ms": after["median"]-before["median"] > max(.1*before["median"], .0005)}
        output["bounds"].append({"bound": n, "metrics": metrics, "phases": phases,
            "median_execution_reduction": 1-c["median"]/b["median"], "paired_ratios": summary(ratios),
            "small_bound_guard_exceeded": n in (1, 8, 32) and c["median"]-b["median"] > max(.1*b["median"], .001),
            "ratio_of_medians_candidate_over_baseline": c["median"]/b["median"],
            "primary_sign_test": sign_test([bv-cv for bv, cv in zip(b["values"], c["values"])]) if n == 256 else None})
    output["descriptive_growth"] = {variant: fit_growth(bounds, [r["metrics"][variant]["execute_seconds"]["median"] for r in output["bounds"]]) for variant in ("baseline", "candidate")}
    args.output.write_text(json.dumps(output, indent=2) + "\n")
    for r in output["bounds"]:
        print(r["bound"], "baseline_s=", r["metrics"]["baseline"]["execute_seconds"]["median"],
              "candidate_s=", r["metrics"]["candidate"]["execute_seconds"]["median"],
              "reduction=", r["median_execution_reduction"], "small_guard=", r["small_bound_guard_exceeded"])


if __name__ == "__main__":
    main()
