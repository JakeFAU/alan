#!/usr/bin/env python3
"""Create a disposable selector counter overlay; never edit the source tree.

Counts loadProgram, boundCompositionProgram and Reduce at their real entries,
scoped strictly to a selector call. The baseline reference remains independent.
No timing from this overlay is performance evidence. Input files and patches
are retained for independent source/counter review.
"""
import argparse
import difflib
import hashlib
import json
from pathlib import Path
import subprocess


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def replace_once(text, old, new):
    if text.count(old) != 1:
        raise ValueError('expected exactly one anchor: ' + old)
    return text.replace(old, new)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    source, out = args.source.resolve(), args.out.resolve()
    if out == source or source in out.parents:
        raise ValueError('overlay must be outside the source tree')
    out.mkdir(parents=True, exist_ok=False)
    inputs = {}
    paths = subprocess.check_output(['git', '-C', str(source), 'ls-files', '-z']).split(b'\0')
    for name in paths:
        if not name:
            continue
        rel = name.decode()
        raw = (source / rel).read_bytes()
        inputs[rel] = digest(raw)
        dest = out / rel
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_bytes(raw)
    patches = []
    def edit(rel, anchor, addition):
        path = out / rel
        before = path.read_text()
        after = replace_once(before, anchor, anchor + addition)
        path.write_text(after)
        patches.extend(difflib.unified_diff(before.splitlines(True), after.splitlines(True), fromfile=rel, tofile=rel+'.probe'))
    for rel, anchor, count in [
        ('internal/executor/store.go', 'func loadRun(ctx context.Context, q queryer, id RunID) (RunState, error) {', 'visits'),
        ('internal/executor/store.go', 'func loadProgram(ctx context.Context, q queryer, id string, l Limits) (Program, error) {', 'loads'),
        ('internal/executor/invocation.go', 'func boundCompositionProgram(ctx context.Context, q queryer, p Program, run RunID, l Limits) (Program, error) {', 'bindings'),
        ('internal/executor/program.go', 'func Reduce(p Program, s Projection, o *Observation, id RunID, seq int64, l Limits) (Reduction, error) {', 'reductions'),
    ]:
        edit(rel, anchor, '\n p3ProbeAdd("' + count + '")\n')
    (out / 'internal/executor/p3_probe_test.go').write_text(PROBE)
    (out / 'p3-probe-inputs.json').write_text(json.dumps(inputs, indent=2, sort_keys=True)+'\n')
    (out / 'p3-probe.patch').write_text(''.join(patches))
    (out / 'p3-probe-generator.json').write_text(json.dumps({'path':str(Path(__file__).resolve()),'sha256':digest(Path(__file__).read_bytes()),'source':str(source),'head':subprocess.check_output(['git','-C',str(source),'rev-parse','HEAD'],text=True).strip()},indent=2)+'\n')
    subprocess.run(['gofmt','-w',str(out/'internal/executor/p3_probe_test.go'),*[str(out/p) for p in ('internal/executor/store.go','internal/executor/invocation.go','internal/executor/program.go')]],check=True)


PROBE = r'''package executor
import (
 "reflect"
 "sync"
 "testing"
)
var p3ProbeMu sync.Mutex
var p3ProbeActive bool
var p3ProbeCounts map[string]int
func p3ProbeAdd(kind string) {
 p3ProbeMu.Lock();defer p3ProbeMu.Unlock()
 if p3ProbeActive {p3ProbeCounts[kind]++}
}
func p3ProbeSelect(t *testing.T,s *Store,reference bool,last *RunID)(*readyRun,map[string]int){
 t.Helper()
 p3ProbeMu.Lock();p3ProbeActive=true;p3ProbeCounts=map[string]int{};p3ProbeMu.Unlock()
 var r *readyRun;var err error
 if reference{r,err=s.p3BaselineSelectReady(t.Context(),last)}else{r,err=s.selectReady(t.Context(),last)}
 p3ProbeMu.Lock();p3ProbeActive=false;counts:=p3ProbeCounts;p3ProbeMu.Unlock()
 if err!=nil{t.Fatal(err)}
 return r,counts
}
func TestP3ProbeRemainingWork(t *testing.T){
 for _,tc:=range []struct{name string;programs int;ready bool;bindings,reductions int}{
  {"shared_waiting",4,false,32,0}, {"distinct_waiting",32,false,32,0}, {"shared_late_ready",4,true,32,1},
 }{
  t.Run(tc.name,func(t *testing.T){
   s,_:=messageStore(t,p3WaitingProvision(t,32,tc.programs))
   if tc.ready{p3Admit(t,s,"r31","late",9)}
   want,baseline:=p3ProbeSelect(t,s,true,nil)
   got,candidate:=p3ProbeSelect(t,s,false,nil)
   if !reflect.DeepEqual(want,got){t.Fatalf("selection differs: %+v %+v",want,got)}
   for _,counts:=range []map[string]int{baseline,candidate}{
    if counts["bindings"]!=tc.bindings || counts["reductions"]!=tc.reductions {t.Fatalf("remaining work changed: %v",counts)}
   }
   if baseline["loads"]!=32{t.Fatalf("baseline calibration: %v",baseline)}
   // RED is asserted by the ordinary authorizer test, separately. This probe
   // accepts the pre-change baseline or exact predicted candidate count.
   if candidate["loads"]!=32 && candidate["loads"]!=tc.programs{t.Fatalf("unexplained loads: %v",candidate)}
   t.Logf("baseline=%v candidate=%v",baseline,candidate)
  })
 }
}
func TestP3ProbeSameCallBindings(t *testing.T){
 s,last,want:=p3SameCallBindingCut(t)
 for _,reference:=range []bool{true,false}{
  got,counts:=p3ProbeSelect(t,s,reference,&last)
  if !reflect.DeepEqual(got,want){t.Fatalf("same-call selection: %+v want %+v",got,want)}
  if counts["visits"]!=2 || counts["bindings"]!=2 || counts["reductions"]!=1{t.Fatalf("same-call remaining work: %v",counts)}
  if reference && counts["loads"]!=2 || !reference && counts["loads"]!=1 && counts["loads"]!=2{t.Fatalf("same-call loads: %v",counts)}
  t.Logf("reference=%t counts=%v",reference,counts)
 }
}
'''

if __name__ == '__main__':
    main()
