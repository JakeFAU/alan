#!/usr/bin/env python3
"""Generate measurement-only ALAN and SQLite copies; never edits input trees."""
import argparse, difflib, hashlib, json, pathlib, re, shutil, subprocess

HERE = pathlib.Path(__file__).resolve().parent
p = argparse.ArgumentParser()
p.add_argument('--alan', required=True, type=pathlib.Path)
p.add_argument('--sqlite', required=True, type=pathlib.Path)
p.add_argument('--driver', required=True, type=pathlib.Path)
p.add_argument('--out', required=True, type=pathlib.Path)
a = p.parse_args()
if a.out.exists():
    raise SystemExit('output must not already exist')
a.out.mkdir(parents=True)
for name, src in [('alan', a.alan), ('sqlite', a.sqlite), ('driver', a.driver)]:
    shutil.copytree(src, a.out/name, ignore=shutil.ignore_patterns('.git'))
    (a.out/name).chmod((a.out/name).stat().st_mode | 0o700)
    for path in (a.out/name).rglob('*'):
        path.chmod(path.stat().st_mode | (0o700 if path.is_dir() else 0o200))
changes=[]
def sha(b): return hashlib.sha256(b).hexdigest()
def edit(rel, transform):
    path=a.out/rel
    old=path.read_text()
    new=transform(old)
    if old == new: raise RuntimeError('unchanged transformation: '+rel)
    path.write_text(new)
    changes.append(dict(path=rel, original_sha256=sha(old.encode()), modified_sha256=sha(new.encode()),
                        patch=''.join(difflib.unified_diff(old.splitlines(True),new.splitlines(True),fromfile=rel+'.original',tofile=rel))))
def replace(s, old, new, n=1):
    if s.count(old)!=n: raise RuntimeError(f'anchor expected {n} occurrences, got {s.count(old)}: {old[:120]}')
    return s.replace(old,new)
def imported(s):
    return replace(s,'import (','import (\n scalingprobe "github.com/JakeFAU/alan/measurecounter"')
def entry(s,name,key=None):
    matches=list(re.finditer(r'^func (?:\([^\n]*\) )?'+re.escape(name)+r'\([^\n]*\{\n',s,re.M))
    if len(matches)!=1: raise RuntimeError('signature anchor ambiguous/missing: '+name)
    m=matches[0]
    return s[:m.end()]+f'\tscalingprobe.Add("fn.{key or name}.calls", 1)\n'+s[m.end():]
functions={
 'composition_types.go':[('descriptor','compositionProfile.descriptor')],
 'composition_store.go':[(x,None) for x in ['invocationEnabled','readCompositionProfile','retainCompositionProfile','validateCompositionStore']],
 'composition_root.go':[(x,None) for x in ['PrepareRoot','SettleRoot','settleRoots','prepareRoot']],
 'invocation_outcome.go':[('prepareCompositionScope',None)],
 'runner.go':[('selectReady',None),('finishRun',None)],
 'composition_inspect.go':[('readRootCompletions',None)],
 'replay.go':[(x,None) for x in ['Replay','validateLedger','validateOrdinaryLedger']],
 'composition_replay.go':[('validateCompositionProtocol',None)],
 'composition_reservation.go':[('compositionProtocol',None)],
 'inspection_execution.go':[('InspectExecution',None)],
 'inspect.go':[('scanInspectionTable',None)],
 'program.go':[('Reduce',None)],
 'store.go':[],
}
for file, names in functions.items():
 def transform(s,file=file,names=names):
    s=imported(s)
    for name,key in names: s=entry(s,name,key)
    if file=='composition_types.go':
        s=replace(s,'err := decode([]byte(p.raw), &c)','scalingprobe.Add("fn.compositionProfile.descriptor.input_bytes", uint64(len(p.raw)))\n\terr := decode([]byte(p.raw), &c)')
    if file=='composition_store.go':
        s=replace(s,'raw, err := json.Marshal(c)\n\tif err != nil {\n\t\treturn nil, err\n\t}', 'raw, err := json.Marshal(c)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n scalingprobe.Add("fn.retainCompositionProfile.initial_marshal_bytes",uint64(len(raw)))')
    if file=='composition_inspect.go':
        s=replace(s,'for _, id := range roots {','for _, id := range roots {\n scalingprobe.Add("work.readRootCompletions.roots",1)')
        s=replace(s,'for _, operation := range sortedKeys(intents) {','for _, operation := range sortedKeys(intents) {\n scalingprobe.Add("work.readRootCompletions.root_intent_pairs",1)')
    if file=='runner.go':
        for sig,key in [('func (s *Store) pendingDeliveries(ctx context.Context) ([]pendingDelivery, error) {','pendingDeliveries.owner'),('func pendingDeliveries(ctx context.Context, q queryer) ([]pendingDelivery, error) {','pendingDeliveries.reader')]:
            s=replace(s,sig,sig+'\n scalingprobe.Add("fn.'+key+'.calls",1)')
        s=replace(s,'for i := 0; i < len(ids); i++ {','for i := 0; i < len(ids); i++ {\n scalingprobe.Add("work.selectReady.runs_considered",1)')
        s=replace(s,'for _, observationID := range observationIDs {','for _, observationID := range observationIDs {\n scalingprobe.Add("work.selectReady.observations_considered",1)')
        s=replace(s,'if _, err = Reduce(p, run.Projection, nil, id, run.Sequence+1, s.limits); err != nil {','scalingprobe.Add("reduce.selectReady",1)\n\t\t\tif _, err = Reduce(p, run.Projection, nil, id, run.Sequence+1, s.limits); err != nil {')
        s=replace(s,'_, err = Reduce(p, run.Projection, &observation.Observation, id, run.Sequence+1, s.limits)','scalingprobe.Add("reduce.selectReady",1)\n\t\t\t_, err = Reduce(p, run.Projection, &observation.Observation, id, run.Sequence+1, s.limits)')
    if file=='replay.go':
        s=replace(s,'result, e := Reduce(p, projection, selected, id, h.seq, limits)','scalingprobe.Add("reduce.validateOrdinaryLedger",1)\n\t\t\tresult, e := Reduce(p, projection, selected, id, h.seq, limits)')
    if file=='store.go':
        s=replace(s,'reduction, err := Reduce(p, r.Projection, observation, r.ID, r.Sequence+1, s.limits)','scalingprobe.Add("reduce.Commit",1)\n\treduction, err := Reduce(p, r.Projection, observation, r.ID, r.Sequence+1, s.limits)')
    if file=='inspect.go':
        s=replace(s,'records = append(records, record)','scalingprobe.Add("materialized."+table+".rows",1)\n\t\tfor _, value := range record {\n switch v := value.(type) { case string: scalingprobe.Add("materialized."+table+".text_blob_bytes",uint64(len(v))); case []byte: scalingprobe.Add("materialized."+table+".text_blob_bytes",uint64(len(v))) }\n }\n\t\trecords = append(records, record)')
    return s
 edit('alan/internal/executor/'+file,transform)
(a.out/'alan/measurecounter').mkdir()
(a.out/'alan/measurecounter/counter.go').write_text('''// Package measurecounter exists only in generated measurement source copies.
package measurecounter
import ("sync"; "sync/atomic")
var counters sync.Map
func Add(key string, n uint64) { v,_ := counters.LoadOrStore(key, new(atomic.Uint64)); v.(*atomic.Uint64).Add(n) }
func Snapshot() map[string]uint64 { out:=map[string]uint64{}; counters.Range(func(k,v any) bool {out[k.(string)]=v.(*atomic.Uint64).Load(); return true}); return out }
''')
# SQLite edits live in the copied driver only. Status counters are reset after
# every actual C step, preventing a cached pending SQLITE_ROW being counted twice.
def sqlite_transform(s):
    s=replace(s,'static int\n_sqlite3_reset_clear(sqlite3_stmt* stmt)', '''typedef struct { int runs; int fullscan; int vm; } alan_probe_metrics;
static alan_probe_metrics alan_probe_take(sqlite3_stmt *stmt) {
  alan_probe_metrics m;
  m.runs = sqlite3_stmt_status(stmt, SQLITE_STMTSTATUS_RUN, 1);
  m.fullscan = sqlite3_stmt_status(stmt, SQLITE_STMTSTATUS_FULLSCAN_STEP, 1);
  m.vm = sqlite3_stmt_status(stmt, SQLITE_STMTSTATUS_VM_STEP, 1);
  return m;
}
static int
_sqlite3_reset_clear(sqlite3_stmt* stmt)''')
    s=replace(s,'long long* changes, const char** pzTail)','long long* changes, const char** pzTail, alan_probe_metrics* metrics, int* prepared)',1)
    s=replace(s,'  sqlite3_finalize(stmt);\n  *pzTail = tail;','  *prepared = 1;\n  *metrics = alan_probe_take(stmt);\n  sqlite3_finalize(stmt);\n  *pzTail = tail;')
    s=replace(s,'\trepreps C.int\n','\trepreps C.int\n probeTemplate string\n')
    s=replace(s,'ss := &SQLiteStmt{c: c, s: s, t: t}','ss := &SQLiteStmt{c: c, s: s, t: t}\n if s != nil { ss.probeTemplate = probeNormalize(C.GoString(C.sqlite3_sql(s))); probeAdd(ss.probeTemplate,"completed_prepares",1) }')
    s=replace(s,'\t\tvar tail *C.char\n\t\tpquery := C.CString(query)','\t\tvar tail *C.char\n var metrics C.alan_probe_metrics\n var prepared C.int\n originalQuery := query\n\t\tpquery := C.CString(query)')
    s=replace(s,'C._sqlite3_exec_no_args(c.db, pquery, C.int(len(query)), &rowid, &changes, &tail)','C._sqlite3_exec_no_args(c.db, pquery, C.int(len(query)), &rowid, &changes, &tail, &metrics, &prepared)')
    s=replace(s,'\t\tC.free(unsafe.Pointer(pquery))','''        if prepared != 0 {
 consumed := len(originalQuery)-len(query)
 key := probeNormalize(originalQuery[:consumed])
 probeAdd(key,"completed_prepares",1)
 probeRecord(key, int(metrics.runs), int(metrics.fullscan), int(metrics.vm), rv == C.SQLITE_ROW)
 }
\t\tC.free(unsafe.Pointer(pquery))''')
    s=replace(s,'rv := C._sqlite3_step_row_internal(s.s, &rowid, &changes)','rv := C._sqlite3_step_row_internal(s.s, &rowid, &changes)\n s.probeStep(rv)')
    s=replace(s,'rv := C._sqlite3_step_internal(rc.s.s)','rv := C._sqlite3_step_internal(rc.s.s)\n rc.s.probeStep(rv)')
    s=replace(s,'rv = C._sqlite3_step_columns(s.s, &ncol, &repreps)','rv = C._sqlite3_step_columns(s.s, &ncol, &repreps)\n s.probeStep(rv)',2)
    s=replace(s,'return C._sqlite3_step_internal(rc.s.s)','rv := C._sqlite3_step_internal(rc.s.s)\n rc.s.probeStep(rv)\n return rv')
    s+='''
func (s *SQLiteStmt) probeStep(rv C.int) {
 m := C.alan_probe_take(s.s)
 probeRecord(s.probeTemplate,int(m.runs),int(m.fullscan),int(m.vm),rv == C.SQLITE_ROW)
}
'''
    return s
edit('sqlite/sqlite3.go',sqlite_transform)
(a.out/'sqlite/alan_measurement.go').write_text((HERE/'sqlite_metrics.go.txt').read_text())
(a.out/'driver/instrument_hook.go').write_text('''package main
import (probe "github.com/JakeFAU/alan/measurecounter"; sqlite "github.com/mattn/go-sqlite3")
func init() {counterSnapshot=func() map[string]uint64 { m:=probe.Snapshot(); for k,v:=range sqlite.MeasurementSnapshot() {m[k]=v}; return m }}
''')
# Preserve a source manifest, exact patches and generated additions before build.
for name in ['alan','sqlite','driver']:
    manifest=[]
    for path in sorted((a.out/name).rglob('*')):
        if path.is_file(): manifest.append({'path':str(path.relative_to(a.out/name)), 'sha256':sha(path.read_bytes())})
    (a.out/(name+'-modified-manifest.json')).write_text(json.dumps(manifest,indent=2)+'\n')
    original=getattr(a, name)
    original_manifest=[{'path':str(path.relative_to(original)), 'sha256':sha(path.read_bytes())} for path in sorted(original.rglob('*')) if path.is_file() and '.git' not in path.relative_to(original).parts]
    (a.out/(name+'-original-manifest.json')).write_text(json.dumps(original_manifest,indent=2)+'\n')
(a.out/'transformations.json').write_text(json.dumps(changes,indent=2)+'\n')
(a.out/'measurement.patch').write_text(''.join(c['patch'] for c in changes))
(a.out/'generation.json').write_text(json.dumps({'measurement_only':True,'inputs':{k:str(getattr(a,k).resolve()) for k in ['alan','sqlite','driver']},'generator_sha256':sha(pathlib.Path(__file__).read_bytes()),'sqlite_metrics_sha256':sha((HERE/'sqlite_metrics.go.txt').read_bytes())},indent=2)+'\n')
subprocess.run(['gofmt','-w',str(a.out/'alan/measurecounter'),*[str(a.out/'alan/internal/executor'/f) for f in functions],str(a.out/'sqlite/sqlite3.go'),str(a.out/'sqlite/alan_measurement.go'),str(a.out/'driver/instrument_hook.go')],check=True)
# Record exact post-gofmt bytes too; pre-gofmt patches plus deterministic gofmt are reproducible.
for name in ['alan','sqlite','driver']:
    manifest=[{'path':str(path.relative_to(a.out/name)),'sha256':sha(path.read_bytes())} for path in sorted((a.out/name).rglob('*')) if path.is_file()]
    (a.out/(name+'-formatted-manifest.json')).write_text(json.dumps(manifest,indent=2)+'\n')
print(a.out)
