"""Run the same retained workload on stock and counted Go selectors."""

import json
import os
from pathlib import Path
import subprocess
import tempfile


HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
EXECUTOR = ROOT / "internal" / "executor"

# Insert counters beside the current owners of the work. Each anchor must
# occur exactly once. All instrumented files live in the temporary overlay.
PROBES = {
    "runner.go": [
        ("if s.guardExecution(ctx) {\n\t\treturn s.selectExecutionReady(ctx, last)",
         "defer recordSelection()()\n\t", ""),
        ("func (s *Store) selectReadyBaselineLocked(ctx context.Context, last *RunID) (*readyRun, error) {",
         "", "\n\tactiveSelection.FallbackCalls++"),
        ("id := ids[(start+i)%len(ids)]", "", "\n\t\tactiveSelection.CandidateVisits++"),
        ("for _, observationID := range observationIDs {", "", "\n\t\t\tactiveSelection.ObservationChecks++"),
    ],
    "static_execution.go": [
        ("func (s *Store) selectExecutionReady(ctx context.Context, last *RunID) (*readyRun, error) {",
         "", "\n\tactiveSelection.GuardedCalls++"),
        ("ordered := append(slices.Clone(x.candidates[at:]), x.candidates[:at]...)",
         "", "\n\tactiveSelection.OrderedRanks += len(ordered)"),
        ("for _, rank := range ordered {", "", "\n\t\tactiveSelection.CandidateVisits++"),
    ],
    "replay.go": [
        ("out[id] = o", "", "\n\t\tif activeSelection != nil { activeSelection.ObservationRows++ }"),
    ],
}


def replace_once(source, anchor, replacement):
    if source.count(anchor) != 1:
        raise ValueError(f"expected exactly one source anchor: {anchor}")
    return source.replace(anchor, replacement)


def run(directory, counted):
    files = {str(EXECUTOR / "selector_work_test.go"): str(HERE / "fixture_test.go.txt")}
    probe = (HERE / "probe.go.txt").read_text()
    if counted:
        probe = replace_once(probe, "selectorInstrumented = false", "selectorInstrumented = true")
        for name, edits in PROBES.items():
            source = (EXECUTOR / name).read_text()
            for anchor, before, after in edits:
                source = replace_once(source, anchor, before + anchor + after)
            (directory / name).write_text(source)
            files[str(EXECUTOR / name)] = str(directory / name)
    (directory / "probe.go").write_text(probe)
    files[str(EXECUTOR / "selector_work_probe.go")] = str(directory / "probe.go")
    overlay = directory / "overlay.json"
    overlay.write_text(json.dumps({"Replace": files}))
    command = ["go", "test", "-mod=readonly", "-count=1", "-json", "-overlay", str(overlay),
               "-run", "^TestMeasureSelectorWork$", "./internal/executor"]
    process = subprocess.run(command, cwd=ROOT, env={**os.environ, "GOWORK": "off"},
                             stdout=subprocess.PIPE, text=True)
    records = []
    for line in process.stdout.splitlines():
        event = json.loads(line)
        output = event.get("Output", "")
        if "SELECTOR_ROW " in output:
            records.append(json.loads(output.split("SELECTOR_ROW ", 1)[1]))
        if process.returncode:
            print(output, end="", flush=True)
    process.check_returncode()
    if not records:
        raise ValueError("no selector workload ran")
    if any(("Window" in row) != counted or ("Total" in row) != counted for row in records):
        raise ValueError("missing or unexpected selector counters")
    return records


if __name__ == "__main__":
    with tempfile.TemporaryDirectory() as temp:
        directory = Path(temp)
        stock = run(directory, counted=False)
        counted = run(directory, counted=True)
    outcomes = [{k: v for k, v in row.items() if k not in ("Window", "Total")} for row in counted]
    if stock != outcomes:
        raise ValueError("stock and counted outcome rows differ")
    for row in counted:
        print(json.dumps(row, separators=(",", ":")))
