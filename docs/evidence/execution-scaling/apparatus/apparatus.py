#!/usr/bin/env python3
"""Hash-bound overlay and serial single-row recorder; Python standard library only."""
import argparse
import difflib
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import sys
import time

HERE = pathlib.Path(__file__).resolve().parent
P2 = HERE.parents[1] / 'performance-p2'
ENV = {'GOMAXPROCS': '4', 'GOGC': '100', 'GOMEMLIMIT': 'off', 'GOWORK': 'off',
       'GOTOOLCHAIN': 'local', 'CGO_ENABLED': '1', 'GOPROXY': 'off', 'GOSUMDB': 'off'}
BOUNDS = [1, 8, 16, 32, 64, 128, 256]


def sha(data):
    return hashlib.sha256(data).hexdigest()


def write(path, value):
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + '\n')


def inventory(root):
    return [{'path': str(p.relative_to(root)), 'sha256': sha(p.read_bytes())}
            for p in sorted(root.rglob('*')) if p.is_file() and not {'.git', '__pycache__'}.intersection(p.relative_to(root).parts)]


def check_inventory(root, expected):
    if inventory(root) != expected:
        raise ValueError('input inventory changed: ' + str(root))


def replace(text, old, new):
    if text.count(old) != 1:
        raise ValueError('anchor must occur exactly once: ' + old[:100])
    return text.replace(old, new)


def pinned_inputs():
    lock = json.loads((HERE / 'inputs.json').read_text())
    for row in lock:
        path = P2 / row['path']
        if sha(path.read_bytes()) != row['sha256']:
            raise ValueError('pinned P2 input changed: ' + row['path'])


def patch_driver(driver):
    path = driver / 'main.go'
    s = path.read_text()
    s = replace(s, 'type report struct {', 'type report struct {\n ApprovalMode string `json:"approval_mode"`\n ExecuteCounterRegistry map[string]uint64 `json:"execute_counter_registry,omitempty"`')
    s = replace(s, 'mode := flag.String("mode",', 'approval := flag.String("approval-mode", "snapshot", "snapshot or targeted retained basis")\n mode := flag.String("mode",')
    s = replace(s, 'if (*mode != "http"', 'if (*approval != "snapshot" && *approval != "targeted") || (*mode != "http"')
    s = replace(s, 'r.Transport = frozenTransport(*mode, *endpoint)', 'r.ApprovalMode = *approval\n r.Transport = frozenTransport(*mode, *endpoint)')
    s = replace(s, 'windowName := "execute_window"', '''if r.ApprovalMode == "targeted" {
 if _, ok := any(store).(continuationBasisReader); !ok { return errors.New("targeted continuation basis API unavailable on this source") }
 }
 windowName := "execute_window"''')
    old = '''if err = r.measure("approval_snapshot_basis", i, func() error {
					cut, e := aruntime.InspectExecution(ctx, path, aruntime.DefaultLimits())'''
    new = '''basisStage := "approval_snapshot_basis"
                if r.ApprovalMode == "targeted" { basisStage = "approval_targeted_basis" }
                if err = r.measure(basisStage, i, func() error {
                    if r.ApprovalMode == "targeted" {
                        var e error
                        basis, e = any(store).(continuationBasisReader).ContinuationBasis(ctx, aruntime.RunID{Subject: fmt.Sprintf("series-%d", i), Generation: 1})
                        return e
                    }
					cut, e := aruntime.InspectExecution(ctx, path, aruntime.DefaultLimits())'''
    s = replace(s, old, new)
    s = replace(s, 'var base, previous resourceSnapshot', 'var registryStart map[string]uint64\n if counterRegistry != nil { registryStart = counterRegistry() }\n var base, previous resourceSnapshot')
    s = replace(s, '\t\treturn nil\n\t}); err != nil {\n\t\treturn err\n\t}\n\tif r.PrefixEnabled {\n\t\tif err := r.verifyPrefixes();', '\t\tif counterRegistry != nil {\n var e error\n r.ExecuteCounterRegistry, e = subtractCounters(registryStart, counterRegistry())\n if e != nil {return e}\n } \n\t\treturn nil\n\t}); err != nil {\n\t\treturn err\n\t}\n\tif r.PrefixEnabled {\n\t\tif err := r.verifyPrefixes();')
    path.write_text(s)
    (driver / 'scaling_overlay.go').write_text('''package main
import ("context"; aruntime "github.com/JakeFAU/alan/runtime")
// An interface assertion keeps baseline builds valid without the candidate API.
type continuationBasisReader interface { ContinuationBasis(context.Context, aruntime.RunID) (aruntime.ContinuationBasis, error) }
var counterRegistry func() map[string]uint64
''')


def compact_counters(out):
    sqlite = out / 'sqlite' / 'alan_measurement.go'
    sqlite.write_text(sqlite.read_text() + (HERE / 'compact_sql.go.txt').read_text())
    shutil.copyfile(HERE / 'compact_sql_test.go.txt', out / 'sqlite' / 'scaling_compact_test.go')
    (out / 'driver' / 'instrument_hook.go').write_text('''package main
import (probe "github.com/JakeFAU/alan/measurecounter"; sqlite "github.com/mattn/go-sqlite3")
func init() {
 counterSnapshot=func() map[string]uint64 { m:=probe.Snapshot(); for k,v:=range sqlite.MeasurementCompactSnapshot() {m[k]=v}; return m }
 counterRegistry=sqlite.MeasurementSnapshot
}
''')


def generate(a):
    pinned_inputs()
    check_inventory(a.source, json.loads(a.source_manifest.read_text()))
    if a.out.exists() or a.out.resolve().is_relative_to(a.source.resolve()):
        raise ValueError('output must be new and outside source')
    source_before = inventory(a.source)
    profile = json.loads(a.anchors.read_text())
    if a.kind == 'counters':
        # Reuse precisely the frozen SQLite accounting, replacing only the Go
        # work-counter injection with explicit, reviewable per-source anchors.
        generator = (P2 / 'instrument' / 'generate.py').read_text()
        start = generator.index('functions={')
        end = generator.index("(a.out/'alan/measurecounter').mkdir()", start)
        injection = '''functions = {}
profile = json.loads(pathlib.Path(PROFILE_PATH).read_text())
for file, spec in profile.items():
 def transform(s, file=file, spec=spec):
    s=imported(s)
    for name in spec.get('entries', []): s=entry(s,name)
    for name,scope in SCOPES.get(file,[]):
        matches=list(re.finditer(r'^func (?:\\([^\\n]*\\) )?'+re.escape(name)+r'\\(ctx context.Context[^\\n]*\\{\\n',s,re.M))
        if len(matches)!=1:raise RuntimeError('scope signature: '+file+':'+name)
        m=matches[0];s=s[:m.end()]+f'\\tctx = scalingprobe.Scope(ctx, {scope})\\n'+s[m.end():]
    for row in spec.get('edits', []): s=replace(s,row['old'],row['new'])
    return s
 functions[file]=[]
 edit('alan/internal/executor/'+file,transform)
'''
        generator = generator[:start] + injection.replace('PROFILE_PATH', repr(str(a.anchors.resolve()))) + generator[end:]
        # __file__ intentionally points to the pinned original tool so its
        # template resolution and original tool-manifest identity stay exact.
        argv = sys.argv
        sys.argv = ['generate.py', '--alan', str(a.source), '--sqlite', str(a.sqlite),
                    '--driver', str(P2 / 'driver'), '--out', str(a.out)]
        try:
            exec(compile(generator, str(P2 / 'instrument' / 'generate.py'), 'exec'),
                 {'__file__': str(P2 / 'instrument' / 'generate.py'), '__name__': '__main__'})
        finally:
            sys.argv = argv
        (a.out / 'adapted-generator.py').write_text(generator)
        compact_counters(a.out)
    else:
        a.out.mkdir(parents=True)
        shutil.copytree(P2 / 'driver', a.out / 'driver')
    patch_driver(a.out / 'driver')
    subprocess.run(['gofmt', '-w', str(a.out / 'driver')], check=True)
    if a.kind == 'counters':
        subprocess.run(['gofmt', '-w', str(a.out / 'sqlite' / 'alan_measurement.go')], check=True)
    # Every overlay difference from the original driver/SQL accounting is retained.
    patches = []
    for p in sorted((a.out / 'driver').glob('*.go')):
        original = P2 / 'driver' / p.name
        old = original.read_text() if original.exists() else ''
        patches.extend(difflib.unified_diff(old.splitlines(True), p.read_text().splitlines(True), fromfile='driver/'+p.name+'.original', tofile='driver/'+p.name))
    (a.out / 'driver-overlay.patch').write_text(''.join(patches))
    check_inventory(a.source, source_before)
    source = a.out / 'alan' if a.kind == 'counters' else a.source.resolve()
    driver = a.out / 'driver'
    shutil.copyfile(driver / 'go.mod', driver / 'scaling.mod')
    shutil.copyfile(driver / 'go.sum', driver / 'scaling.sum')
    env = dict(os.environ, **ENV)
    cmds = [['go', 'mod', 'edit', '-modfile=scaling.mod', '-replace=github.com/JakeFAU/alan='+str(source)]]
    if a.kind == 'counters':
        cmds[0].append('-replace=github.com/mattn/go-sqlite3='+str((a.out / 'sqlite').resolve()))
    cmds.append(['go', 'build', '-mod=readonly', '-modfile=scaling.mod', '-o', str((a.out / 'driver.bin').resolve()), '.'])
    for cmd in cmds:
        subprocess.run(cmd, cwd=driver, env=env, check=True)
    write(a.out / 'overlay.json', {'kind': a.kind, 'source': str(a.source.resolve()), 'source_manifest_sha256': sha(a.source_manifest.read_bytes()),
          'source_input_inventory': source_before, 'tools': inventory(HERE), 'anchors_sha256': sha(a.anchors.read_bytes()),
          'p2_inputs': json.loads((HERE / 'inputs.json').read_text()), 'environment': ENV, 'commands': cmds,
          'generated_inventory': inventory(a.out), 'binary_sha256': sha((a.out / 'driver.bin').read_bytes())})


def validate(row, expected_bound=None, expected_approval='snapshot'):
    if not isinstance(row, dict) or not row or row.get('error') or row.get('invariants_ok') is not True:
        raise ValueError('empty/failed/incomplete report')
    b = row.get('bound')
    if b not in BOUNDS or (expected_bound is not None and b != expected_bound):
        raise ValueError('wrong bound')
    if row.get('approval_mode', 'snapshot') != expected_approval:
        raise ValueError('wrong approval mode')
    if row.get('reductions') != 4*b-1 or any(row.get(k) != b for k in ['selected','completed','spent']):
        raise ValueError('invalid exact workload counts')
    if row.get('protected_approvals') != b-1 or len(row.get('contacts', [])) != b or len(row.get('evidence', [])) != b:
        raise ValueError('invalid independent evidence counts')
    if row.get('prefix_enabled') is not True or len(row.get('prefixes', [])) != b:
        raise ValueError('missing required prefix proof')
    if not any(s.get('name') == 'execute_window' and not s.get('error') and s.get('wall_ns', 0) > 0 for s in row.get('stages', [])):
        raise ValueError('missing/failed Execute')
    return row


def record(a):
    if a.out.exists():
        raise ValueError('row directory exists; no replacement/retry')
    if a.bound not in BOUNDS or a.approval not in ['snapshot','targeted']:
        raise ValueError('invalid mode/bound')
    a.out.mkdir(parents=True)
    binary_hash = sha(a.binary.read_bytes())
    controller_hash = sha(pathlib.Path(__file__).read_bytes())
    source_before = inventory(a.source)
    check_inventory(a.source, json.loads(a.source_manifest.read_text()))
    cmd = [str(a.binary.resolve()), '-bound='+str(a.bound), '-dir='+str((a.out / 'store').resolve()),
           '-sample='+a.out.name, '-source-label='+a.label, '-mode=direct', '-prefix=true']
    if a.overlay:
        cmd.append('-approval-mode='+a.approval)
    # Whitelist only local build/runtime paths; do not inherit credentials.
    env = {k: os.environ[k] for k in ['PATH','HOME','TMPDIR'] if k in os.environ}
    env.update(ENV)
    started = time.time()
    with (a.out / 'stdout.json').open('wb') as stdout, (a.out / 'stderr.log').open('wb') as stderr:
        result = subprocess.run(cmd, cwd=a.out, env=env, stdout=stdout, stderr=stderr)
    failure = None
    try:
        if result.returncode:
            raise ValueError('nonzero process exit '+str(result.returncode))
        validate(json.loads((a.out / 'stdout.json').read_text()), a.bound, a.approval)
        check_inventory(a.source, source_before)
        if sha(a.binary.read_bytes()) != binary_hash:
            raise ValueError('binary changed during row')
        if sha(pathlib.Path(__file__).read_bytes()) != controller_hash:
            raise ValueError('recorder changed during row')
    except Exception as e:
        failure = str(e)
    write(a.out / 'row.json', {'command': cmd, 'cwd': str(a.out.resolve()), 'environment': env,
          'started_unix': started, 'ended_unix': time.time(), 'exit_code': result.returncode,
          'validation_error': failure, 'source_manifest_sha256': sha(a.source_manifest.read_bytes()),
          'source_unchanged': inventory(a.source) == source_before, 'binary_sha256_before': binary_hash,
          'binary_sha256_after': sha(a.binary.read_bytes()), 'stdout_sha256': sha((a.out / 'stdout.json').read_bytes()),
          'stderr_sha256': sha((a.out / 'stderr.log').read_bytes()), 'recorder_sha256': controller_hash, 'recorder_sha256_after': sha(pathlib.Path(__file__).read_bytes())})
    if failure:
        raise ValueError(failure)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    subs = p.add_subparsers(dest='command', required=True)
    g = subs.add_parser('generate')
    for name in ['source','source-manifest','out']:
        g.add_argument('--'+name, type=pathlib.Path, required=True)
    g.add_argument('--kind', choices=['ordinary','counters'], required=True)
    g.add_argument('--anchors', type=pathlib.Path, default=HERE / 'baseline-anchors.json')
    g.add_argument('--sqlite', type=pathlib.Path)
    r = subs.add_parser('record')
    for name in ['binary','source','source-manifest','out']:
        r.add_argument('--'+name, type=pathlib.Path, required=True)
    r.add_argument('--bound', type=int, required=True)
    r.add_argument('--label', required=True)
    r.add_argument('--approval', choices=['snapshot','targeted'], default='snapshot')
    r.add_argument('--overlay', action='store_true')
    v = subs.add_parser('validate')
    v.add_argument('report', type=pathlib.Path)
    a = p.parse_args()
    if a.command == 'generate':
        if a.kind == 'counters' and not a.sqlite:
            p.error('counter overlay requires --sqlite')
        generate(a)
    elif a.command == 'record':
        record(a)
    else:
        validate(json.loads(a.report.read_text()))


if __name__ == '__main__':
    main()
