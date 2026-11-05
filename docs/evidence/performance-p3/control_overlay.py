#!/usr/bin/env python3
"""Materialize separate diagnostic counters and uninstrumented selector controls.

Consumes the source-frozen candidate and existing selector_probe.py. No original
source is changed. The baseline selector is the independent test-only reference.
The protocol fixes samples before this generator or either control executes.
"""
import argparse
import difflib
import importlib.util
import json
from pathlib import Path
import shutil
import subprocess


def replace_once(text, old, new):
    if text.count(old) != 1:
        raise ValueError('expected one anchor: ' + old)
    return text.replace(old, new)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source', type=Path)
    parser.add_argument('probe', type=Path)
    parser.add_argument('output', type=Path)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=False)
    for name in ('timed', 'counted'):
        shutil.copytree(args.source, args.output/name)
    spec = importlib.util.spec_from_file_location('frozen_probe', args.probe)
    probe = importlib.util.module_from_spec(spec); spec.loader.exec_module(probe)
    counted = args.output/'counted'
    patches = []
    def edit(relative, anchor, addition):
        path = counted/relative
        before = path.read_text()
        after = replace_once(before, anchor, anchor+addition)
        path.write_text(after)
        patches.extend(difflib.unified_diff(before.splitlines(True),after.splitlines(True),fromfile=relative,tofile=relative+'.control'))
    for relative, anchor, kind in (
        ('internal/executor/store.go','func loadRun(ctx context.Context, q queryer, id RunID) (RunState, error) {','visits'),
        ('internal/executor/store.go','func loadProgram(ctx context.Context, q queryer, id string, l Limits) (Program, error) {','loads'),
        ('internal/executor/invocation.go','func boundCompositionProgram(ctx context.Context, q queryer, p Program, run RunID, l Limits) (Program, error) {','bindings'),
        ('internal/executor/program.go','func Reduce(p Program, s Projection, o *Observation, id RunID, seq int64, l Limits) (Reduction, error) {','reductions')):
        edit(relative,anchor,'\n p3ProbeAdd("'+kind+'")\n')
    for relative,anchor in (
        ('internal/executor/runner.go','\t\tp, ok := programs[run.ProgramID]'),
        ('internal/executor/readiness_reference_test.go','\t\tp, err := loadProgram(ctx, s.conn, run.ProgramID, s.limits)')):
        path=counted/relative;before=path.read_text()
        after=replace_once(before,anchor,'\t\tp3ProbeProgram(run.ProgramID)\n'+anchor)
        path.write_text(after)
        patches.extend(difflib.unified_diff(before.splitlines(True),after.splitlines(True),fromfile=relative,tofile=relative+'.control'))
    helper=replace_once(probe.PROBE,' "reflect"',' "reflect"\n "encoding/json"')
    helper=replace_once(helper,'t.Logf("baseline=%v candidate=%v",baseline,candidate)',
                       'raw,_:=json.Marshal(map[string]any{"case":tc.name,"baseline":baseline,"candidate":candidate});t.Logf("P3_CONTROL_COUNTER %s",raw)')
    helper += '''
func p3ProbeProgram(id string) {
 p3ProbeMu.Lock();defer p3ProbeMu.Unlock()
 if !p3ProbeActive{return}
 p3ProbeCounts["eligible_visits"]++
 if p3ProbeCounts["id:"+id]==0{p3ProbeCounts["distinct_programs"]++}
 p3ProbeCounts["id:"+id]++
}
'''
    (counted/'internal/executor/p3_control_probe_test.go').write_text(helper)
    (args.output/'timed/internal/executor/p3_control_timing_test.go').write_text(TIMING)
    for root in (counted,args.output/'timed'):
        changed=[str(p) for p in (root/'internal/executor').glob('p3_control*test.go')]
        if root==counted:
            changed += [str(root/'internal/executor'/name) for name in ('store.go','invocation.go','program.go','runner.go','readiness_reference_test.go')]
        subprocess.run(['gofmt','-w',*changed],check=True)
    (args.output/'counted.patch').write_text(''.join(patches))


TIMING = r'''package executor
import (
 "encoding/json"
 "os"
 "reflect"
 "runtime"
 "testing"
 "time"
)
func TestP3ControlTiming(t *testing.T) {
 name,impl:=os.Getenv("P3_CONTROL_CASE"),os.Getenv("P3_CONTROL_IMPL")
 programs:=4
 switch name {case "shared_waiting","shared_late_ready":case "distinct_waiting":programs=32;default:t.Fatal("unknown control case")}
 if impl!="baseline" && impl!="candidate"{t.Fatal("unknown implementation")}
 s,_:=messageStore(t,p3WaitingProvision(t,32,programs))
 if name=="shared_late_ready"{p3Admit(t,s,"r31","late",9)}
 selectOne:=func()(*readyRun,error){if impl=="baseline"{return s.p3BaselineSelectReady(t.Context(),nil)};return s.selectReady(t.Context(),nil)}
 // Exactly one excluded call primes the same fixture; no commits occur.
 want,err:=selectOne();if err!=nil{t.Fatal(err)}
 if (want!=nil)!=(name=="shared_late_ready"){t.Fatal("unexpected control readiness")}
 const iterations=20
 var before,after runtime.MemStats
 runtime.ReadMemStats(&before)
 start:=time.Now()
 for i:=0;i<iterations;i++{got,err:=selectOne();if err!=nil || !reflect.DeepEqual(got,want){t.Fatalf("control differs: %+v %v",got,err)}}
 elapsed:=time.Since(start)
 runtime.ReadMemStats(&after)
 raw,_:=json.Marshal(map[string]any{"case":name,"implementation":impl,"iterations":iterations,"excluded_calls":1,
  "wall_ns":elapsed.Nanoseconds(),"allocated_bytes":after.TotalAlloc-before.TotalAlloc,"mallocs":after.Mallocs-before.Mallocs,
  "gc_count":after.NumGC-before.NumGC,"go_version":runtime.Version(),"gomaxprocs":runtime.GOMAXPROCS(0),"selected":want!=nil})
 t.Logf("P3_CONTROL_TIMING %s",raw)
}
'''


if __name__=='__main__':main()
