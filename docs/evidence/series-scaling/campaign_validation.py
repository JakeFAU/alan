#!/usr/bin/env python3
"""Frozen paired-campaign identities and structural validation (no workloads)."""
from collections import Counter
import hashlib
from pathlib import Path
import random

BOUNDS = (1, 8, 16, 32, 64, 128, 256)
VARIANTS = ("baseline", "candidate")
CONTRACT = "alan/series-scaling/paired-v2"
SEED = 20261104
ENVIRONMENT = {"GOMAXPROCS": "4", "GOGC": "100", "GOMEMLIMIT": "off",
               "CGO_ENABLED": "1", "GOWORK": "off", "GOTOOLCHAIN": "local",
               "GOPROXY": "off", "GODEBUG": "", "GORACE": "", "GOFLAGS": "",
               "GOTRACEBACK": "single"}
SINGLETON_PHASES = ("construct_provision", "create", "execute_window",
                    "final_inspection", "final_series_view", "close_reopen_resume",
                    "resume_inspection_view", "close", "replay")
BODY_PHASES = tuple(sorted((set(SINGLETON_PHASES) - {"execute_window"}) |
                          {"run", "approval_snapshot_basis", "approval_establishment"}))


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def tooling_hashes(directory=None):
    directory = Path(directory or Path(__file__).parent)
    return {name: sha(directory / name) for name in
            ("make_schedule.py", "run_schedule.py", "analyze_pairs.py", "campaign_validation.py")}


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
        require(row.get("campaign_contract") == CONTRACT, "paired-v2 schedule required")
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
                            "schedule": "ordinary", "reopen_every": 0},
                "frozen ordinary runtime configuration required")
        require(row.get("expected_environment") == ENVIRONMENT, "frozen environment required")
        tools = row.get("expected_tooling_sha256", {})
        require(set(tools) == {"make_schedule.py", "run_schedule.py", "analyze_pairs.py", "campaign_validation.py"}
                and all(len(v) == 64 and all(c in "0123456789abcdef" for c in v) for v in tools.values()),
                "frozen tooling hashes required")
        configuration = (runtime, row["expected_environment"], tools)
        if common is None:
            common = configuration
        require(configuration == common, "configuration/tooling changed inside schedule")
    return variants


def validate_report(report, row):
    name, n = row["name"], row["bound"]
    require(report.get("version") == "alan/series-scaling/v1", "wrong report version: " + name)
    require(report.get("invariants_ok") is True and not report.get("error"), "failed proof: " + name)
    require(report.get("bound") == n and report.get("sample") == name and
            report.get("source_label") == row["source"], "report identity mismatch: " + name)
    require(all(report.get(k) == v for k, v in row["expected_runtime"].items()),
            "report runtime configuration mismatch: " + name)
    require(all(report.get(k) == n for k in ("completed", "selected", "spent")) and
            report.get("protected_approvals") == n-1, "workload count mismatch: " + name)
    contacts, evidence = report.get("contacts", []), report.get("evidence", [])
    require(len(contacts) == len(evidence) == n, "contact/evidence count mismatch: " + name)
    require([c.get("ordinal") for c in contacts] == list(range(1, n+1)) and
            [e.get("index") for e in evidence] == list(range(1, n+1)), "position identity mismatch: " + name)
    operations = [e.get("operation_id") for e in evidence]
    require(all(isinstance(op, str) and op for op in operations) and len(set(operations)) == n and
            [c.get("operation_id") for c in contacts] == operations, "operation identity mismatch: " + name)
    require(not evidence[0].get("approval") and all(e.get("approval") for e in evidence[1:]),
            "protected approval evidence mismatch: " + name)
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
                    ("wall_ns", "user_cpu_ns", "system_cpu_ns", "total_alloc_bytes", "mallocs")),
                "invalid phase measurement: " + name + "/" + stage["name"])
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
    store = Path(report.get("store", ""))
    require(store.name == "store.sqlite" and store.parent.name == name + ".store", "store identity mismatch: " + name)
    expected_command = [row["binary"], "-bound", str(row["bound"]), "-dir", str(store.parent),
                        "-sample", name, "-source-label", row["source"]]
    require(meta.get("command") == expected_command, "process arguments mismatch: " + name)
