#!/usr/bin/env python3
"""Add P3 call-local diagnostics to a fresh P2 overlay; never touch source inputs."""
import argparse
import difflib
import hashlib
import json
from pathlib import Path
import subprocess


def replace(source, old, new):
    if source.count(old) != 1:
        raise ValueError('expected one anchor: ' + old)
    return source.replace(old, new)


def readiness_visit_anchor(source):
    # Baseline visits load directly; candidate visits look up before a miss can
    # load. Count both at the same eligible boundary, including candidate hits.
    anchors = ('\t\tp, err := loadProgram(ctx, s.conn, run.ProgramID, s.limits)',
               '\t\tp, ok := programs[run.ProgramID]')
    matches = [anchor for anchor in anchors if source.count(anchor)]
    if len(matches) != 1 or source.count(matches[0]) != 1:
        raise ValueError('expected one supported readiness visit anchor')
    return matches[0]


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ('alan', 'sqlite', 'driver', 'out'):
        p.add_argument('--' + name, type=Path, required=True)
    a = p.parse_args()
    generator = a.alan / 'docs/evidence/performance-p2/instrument/generate.py'
    subprocess.run(['python3', str(generator), '--alan', str(a.alan), '--sqlite', str(a.sqlite),
                    '--driver', str(a.driver), '--out', str(a.out)], check=True)
    originals = {}
    def edit(rel, fn):
        path = a.out / rel
        originals[rel] = path.read_text()
        path.write_text(fn(originals[rel]))
    def runner(s):
        s = replace(s, 'func (s *Store) selectReady(ctx context.Context, last *RunID) (*readyRun, error) {',
                    'func (s *Store) selectReady(ctx context.Context, last *RunID) (selected *readyRun, selectedError error) {\n p3visits := 0\n p3ids := map[string]bool{}\n defer func() { scalingprobe.Readiness(p3visits, len(p3ids), selected != nil, selectedError != nil) }()')
        anchor = readiness_visit_anchor(s)
        s = replace(s, anchor,
                    '\t\tp3visits++\n p3ids[run.ProgramID] = true\n scalingprobe.Add("p3.program_id."+run.ProgramID+".visits",1)\n' + anchor)
        s = replace(s, '\t\tp, err = boundCompositionProgram(ctx, s.conn, p, id, s.limits)',
                    '\t\tscalingprobe.Add("p3.binding.calls",1)\n p, err = boundCompositionProgram(ctx, s.conn, p, id, s.limits)')
        s = replace(s, '\t\t\t_, err = Reduce(p, run.Projection, &observation.Observation, id, run.Sequence+1, s.limits)',
                    '\t\t\tscalingprobe.Add("p3.observation.attempts",1)\n _, err = Reduce(p, run.Projection, &observation.Observation, id, run.Sequence+1, s.limits)')
        s = replace(s, '\t\t\treturn &readyRun{authority: run.Authority(), observation: observationID}, nil',
                    '\t\t\tscalingprobe.Add("p3.observation.selected",1)\n return &readyRun{authority: run.Authority(), observation: observationID}, nil')
        return s
    edit('alan/internal/executor/runner.go', runner)
    def store(s):
        s = replace(s, 'func loadProgram(ctx context.Context, q queryer, id string, l Limits) (Program, error) {',
                    '''func loadProgram(ctx context.Context, q queryer, id string, l Limits) (out Program, outErr error) {
 if scalingprobe.IsScope(ctx,9) { defer func(){ if outErr == nil {scalingprobe.Add("p3.load.success",1)} else {scalingprobe.Add("p3.load.failure",1)} }() }
 return p3LoadProgram(ctx,q,id,l)
}
func p3LoadProgram(ctx context.Context, q queryer, id string, l Limits) (Program, error) {''')
        head, s = s.split('func p3LoadProgram(', 1)
        s = replace(s, '\tif err := decode(a.Bytes, &p); err != nil {',
                    '\tif scalingprobe.IsScope(ctx,9) {scalingprobe.Add("p3.decode.calls",1); scalingprobe.Add("p3.decode.bytes",uint64(len(a.Bytes)))}\n if err := decode(a.Bytes, &p); err != nil {')
        s = replace(s, '\treturn p, validateProgram(p)\n}',
                    '\tif scalingprobe.IsScope(ctx,9) {scalingprobe.Add("p3.validate.calls",1)}\n return p, validateProgram(p)\n}')
        return head + 'func p3LoadProgram(' + s
    edit('alan/internal/executor/store.go', store)
    helper = a.out / 'alan/measurecounter/readiness.go'
    helper.write_text('''package measurecounter
import "strconv"
// Distribution is counted at every return, including errors and early selections.
func Readiness(visits, distinct int, selected, failed bool) {
 Add("p3.calls",1); Add("p3.eligible_visits",uint64(visits)); Add("p3.distinct_programs",uint64(distinct)); Add("p3.reusable_visits",uint64(visits-distinct))
 suffix := "waiting"; if selected {suffix="selected"}; if failed {suffix="error"}
 Add("p3.distribution."+suffix+".visits_"+strconv.Itoa(visits)+".distinct_"+strconv.Itoa(distinct),1)
}
''')
    subprocess.run(['gofmt', '-w', str(helper), *[str(a.out / rel) for rel in originals]], check=True)
    changes = []
    for rel, old in originals.items():
        new = (a.out / rel).read_text()
        changes.append({'path': rel, 'before_sha256': hashlib.sha256(old.encode()).hexdigest(),
                        'after_sha256': hashlib.sha256(new.encode()).hexdigest(),
                        'patch': ''.join(difflib.unified_diff(old.splitlines(True), new.splitlines(True), fromfile=rel+'.p2', tofile=rel+'.p3'))})
    (a.out / 'p3-transformations.json').write_text(json.dumps(changes, indent=2)+'\n')
    (a.out / 'p3-instrument.patch').write_text(''.join(c['patch'] for c in changes)+helper.read_text())
    (a.out / 'p3-instrument-identity.json').write_text(json.dumps({'generator_sha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest(), 'parent_generator_sha256': hashlib.sha256(generator.read_bytes()).hexdigest()}, indent=2)+'\n')


if __name__ == '__main__':
    main()
