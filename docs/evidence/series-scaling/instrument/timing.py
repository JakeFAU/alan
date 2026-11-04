#!/usr/bin/env python3
"""Generate a separate coarse inclusive-wall diagnostic; no SQL/per-row probes."""
import argparse
import difflib
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess

HERE = Path(__file__).resolve().parent
# A finite key set. Root preparation/scope/descriptor helpers are deliberately
# absent: timers cover coarse owner/reader boundaries rather than root-loop work.
BOUNDARIES = {
    'runner.go': [('Run', 'Store.Run', True),
                  ('selectReady', 'Store.selectReady', True),
                  ('pendingDeliveries', 'pendingDeliveries', False),
                  ('finishRun', 'Store.finishRun', True)],
    'composition_root.go': [('settleRoots', 'Store.settleRoots', True)],
    'composition_inspect.go': [('readRootCompletions', 'readRootCompletions', False)],
    'replay.go': [('validateLedger', 'validateLedger', False),
                  ('validateOrdinaryLedger', 'validateOrdinaryLedger', False)],
    'composition_store.go': [('validateCompositionStore', 'validateCompositionStore', False)],
    'composition_replay.go': [('validateCompositionProtocol', 'validateCompositionProtocol', False)],
    'inspection_execution.go': [('InspectExecution', 'InspectExecution', False)],
    'messages.go': [('AuthorizeIntent', 'Store.AuthorizeIntent', True)],
    'adapters.go': [('prepareAdapterWork', 'Store.prepareAdapterWork', True),
                    ('transportAdapter', 'transportAdapter', False),
                    ('accountAdapter', 'Store.accountAdapter', True),
                    ('recordAdapterResult', 'Store.recordAdapterResult', True),
                    ('settleAdapter', 'Store.settleAdapter', True)],
}

def digest(data):
    return hashlib.sha256(data).hexdigest()

def manifest(root):
    return [{'path': str(p.relative_to(root)), 'sha256': digest(p.read_bytes())}
            for p in sorted(root.rglob('*'))
            if p.is_file() and '.git' not in p.relative_to(root).parts]

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--alan', type=Path, required=True)
    parser.add_argument('--driver', type=Path, required=True)
    parser.add_argument('--sqlite', type=Path, required=True,
                        help='Original pinned SQLite module; never instrumented or edited')
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--build', action='store_true', help='Also build the copied driver')
    args = parser.parse_args()
    args.out = args.out.resolve()
    if args.out.exists():
        raise SystemExit('output must be new')
    if 'github.com/mattn/go-sqlite3 v1.14.52' not in (args.alan/'go.mod').read_text():
        raise SystemExit('expected frozen SQLite v1.14.52 pin absent')
    args.out.mkdir(parents=True)
    originals = {}
    for name in ['alan', 'driver']:
        src = getattr(args, name).resolve()
        originals[name] = {p['path']: p['sha256'] for p in manifest(src)}
        shutil.copytree(src, args.out/name, ignore=shutil.ignore_patterns('.git'))
        for path in [args.out/name, *(args.out/name).rglob('*')]:
            path.chmod(path.stat().st_mode | (0o700 if path.is_dir() else 0o200))
    changes = []
    names = []
    source_originals = {}
    for file, boundaries in BOUNDARIES.items():
        path = args.out/'alan/internal/executor'/file
        old = path.read_text()
        source_originals[str(path.relative_to(args.out))] = old
        source = old
        if source.count('import (') != 1:
            raise RuntimeError('import anchor ambiguous/absent: '+file)
        source = source.replace('import (', 'import (\n scalingtiming "github.com/JakeFAU/alan/measuretiming"', 1)
        for function, label, method in boundaries:
            receiver = r'\(s \*Store\) ' if method else ''
            pattern = r'^func '+receiver+re.escape(function)+r'\([^\n]*\{\n'
            matches = list(re.finditer(pattern, source, re.M))
            if len(matches) != 1:
                raise RuntimeError('signature anchor ambiguous/absent: '+file+':'+function)
            match = matches[0]
            index = len(names)
            names.append(label)
            source = source[:match.end()]+f'\tdefer scalingtiming.Start({index})()\n'+source[match.end():]
        path.write_text(source)
        changes.append(str(path.relative_to(args.out)))
    timing = args.out/'alan/measuretiming'
    timing.mkdir()
    labels = ','.join(json.dumps(name) for name in names)
    (timing/'timing.go').write_text('''// Package measuretiming exists only in generated measurement source copies.
package measuretiming
import ("sync"; "time")
var names = [...]string{'''+labels+'''}
type metric struct { calls, inclusiveWallNS uint64 }
var metrics [len(names)]metric
var mu sync.Mutex
// Start captures time at entry; the returned completion is evaluated at return.
// This is inclusive wall time, not CPU time. Nested/concurrent spans overlap.
func Start(index int) func() {
 started := time.Now()
 return func() {
  elapsed := uint64(time.Since(started).Nanoseconds())
  mu.Lock(); metrics[index].calls++; metrics[index].inclusiveWallNS += elapsed; mu.Unlock()
 }
}
func Snapshot() map[string]uint64 {
 mu.Lock(); defer mu.Unlock()
 out := make(map[string]uint64, 2*len(names))
 for i,name := range names {
  out["coarse."+name+".calls"] = metrics[i].calls
  out["coarse."+name+".inclusive_wall_ns"] = metrics[i].inclusiveWallNS
 }
 return out
}
''')
    (args.out/'driver/instrument_hook.go').write_text('''package main
import "github.com/JakeFAU/alan/measuretiming"
func init() { counterSnapshot = measuretiming.Snapshot }
''')
    subprocess.run(['gofmt', '-w', str(timing), str(args.out/'driver/instrument_hook.go'),
                    *[str(args.out/p) for p in changes]], check=True)
    added = ['alan/measuretiming/timing.go', 'driver/instrument_hook.go']
    transformations = []
    for rel in changes+added:
        old = source_originals.get(rel, '')
        new = (args.out/rel).read_text()
        transformations.append({'path': rel,
            'original_sha256': digest(old.encode()) if rel in source_originals else None,
            'modified_sha256': digest(new.encode()),
            'patch': ''.join(difflib.unified_diff(old.splitlines(True), new.splitlines(True),
                fromfile=rel+'.original', tofile=rel))})
    (args.out/'timing-transformations.json').write_text(json.dumps(transformations, indent=2)+'\n')
    (args.out/'timing.patch').write_text(''.join(change['patch'] for change in transformations))
    for name in ['alan', 'driver']:
        (args.out/(name+'-original-manifest.json')).write_text(json.dumps(originals[name], indent=2, sort_keys=True)+'\n')
        (args.out/(name+'-modified-manifest.json')).write_text(json.dumps(manifest(args.out/name), indent=2)+'\n')
    metadata = {'measurement_only': True, 'mode': 'coarse-inclusive-wall',
        'built': args.build, 'sql_instrumented': False, 'per_row_instrumented': False,
        'generator_sha256': digest(Path(__file__).read_bytes()),
        'inputs': {name: str(getattr(args, name).resolve()) for name in ['alan', 'driver', 'sqlite']},
        'driver_main_sha256': digest((args.driver/'main.go').read_bytes()),
        'original_sqlite_go_sha256': digest((args.sqlite/'sqlite3.go').read_bytes()),
        'original_sqlite_binding_sha256': digest((args.sqlite/'sqlite3-binding.c').read_bytes()),
        'boundaries': names,
        'definitions': {'calls': 'Completed invocations; error returns included.',
            'inclusive_wall_ns': 'Monotonic elapsed wall nanoseconds from entry through existing deferred cleanup, before aggregate mutex. Nested/concurrent spans overlap; do not sum.',
            'retention': 'Fixed array and two output counters per static boundary; one mutex acquisition per completed span, no per-SQL/per-row/per-descriptor probes.'}}
    env = dict(os.environ, GOWORK='off', GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off')
    if args.build:
        driver = args.out/'driver'
        shutil.copyfile(driver/'go.mod', driver/'timing.mod')
        shutil.copyfile(driver/'go.sum', driver/'timing.sum')
        subprocess.run(['go', 'mod', 'edit', '-modfile=timing.mod',
                        '-replace=github.com/JakeFAU/alan='+str(args.out/'alan'),
                        '-replace=github.com/mattn/go-sqlite3='+str(args.sqlite.resolve())],
                       cwd=driver, env=env, check=True)
        subprocess.run(['go', 'build', '-mod=readonly', '-modfile=timing.mod',
                        '-o', str(args.out/'driver.bin'), '.'], cwd=driver, env=env, check=True)
        metadata['executable_sha256'] = digest((args.out/'driver.bin').read_bytes())
        metadata['modfile_sha256'] = digest((driver/'timing.mod').read_bytes())
    (args.out/'timing-generation.json').write_text(json.dumps(metadata, indent=2)+'\n')
    print(args.out)

if __name__ == '__main__':
    main()
