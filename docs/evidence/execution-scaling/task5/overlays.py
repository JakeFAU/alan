"""Disposable, unique-anchor transforms of the pinned public driver and probes."""
from collections import defaultdict
import difflib
from pathlib import Path
import re
import shutil

from custody import APPARATUS, P2, read, sha, write
from apparatus import compact_counters, patch_driver, pinned_inputs, replace


def insert_probes(source, function, probes):
    matches = list(re.finditer(r'^func (?:\([^\n]*\) )?' + re.escape(function) +
                               r'\([^\n]*\{\n', source, re.M))
    if len(matches) != 1:
        raise ValueError('expected unique function: ' + function)
    start = matches[0].end()
    end = source.index('\n}', start - 1)
    body = source[start:end]
    for key, anchor, value, when in probes:
        call = f'scalingprobe.Add("{key}", uint64({value}))'
        if when != 'always':
            call = ('if ' + when + ' { ' + call + ' }')
        if not anchor:
            body = call + '\n' + body
        else:
            target = anchor[1:]
            if body.count(target) != 1:
                raise ValueError(f'expected unique anchor: {function}/{key}: {target}')
            if anchor[0] == '+':
                body = body.replace(target, target + '\n' + call)
            elif anchor[0] == '-':
                body = body.replace(target, call + '\n' + target)
            else:
                raise ValueError('missing anchor direction: ' + anchor)
    return source[:start] + body + source[end:]


def patch(path, old, new, result):
    path.write_text(new)
    result.write_text(''.join(difflib.unified_diff(old.splitlines(True), new.splitlines(True),
                                                fromfile=path.name + '.original', tofile=path.name)))


def build_module(rec, name, source, module, binary, sqlite=None):
    mod = module / 'task5.mod'
    shutil.copyfile(module / 'go.mod', mod)
    shutil.copyfile(module / 'go.sum', mod.with_suffix('.sum'))
    args = ['go', 'mod', 'edit', '-modfile=' + str(mod),
            '-replace=github.com/JakeFAU/alan=' + str(source)]
    if sqlite:
        args.append('-replace=github.com/mattn/go-sqlite3=' + str(sqlite))
    rec.run(name + '-bind', args, module)
    rec.bind(name + '-module', module)
    rec.run(name + '-build', ['go', 'build', '-buildvcs=false', '-mod=readonly',
                            '-modfile=' + str(mod), '-o', binary, '.'], module)
    rec.bind_tool(binary)
    rec.run(name + '-build-info', ['go', 'version', '-m', binary], module)


def scopes():
    content = (P2 / 'instrument/sqlite_metrics.go.txt').read_text()
    match = re.search(r'probeScopeNames = \[\.\.\.\]string\{([^\n]+)\}', content)
    return tuple(re.findall(r'"([^"]+)"', match[1]))


def grouped_probes(profile):
    result = defaultdict(lambda: defaultdict(list))
    for file, function, key, anchor, value, when in profile:
        result[file][function].append((key, anchor, value, when))
    return {file: dict(functions) for file, functions in result.items()}


def profile(label):
    common = read(Path(__file__).with_name('work-common.json'))
    specific = read(Path(__file__).with_name('work-' + label + '.json'))
    return common + specific


def probe_keys(label):
    baseline = read(APPARATUS / 'baseline-anchors.json')
    keys = {key for _, _, key, _, _, _ in profile(label)}
    keys.update(('work.report.pending_intent_visits', 'work.scheduler.pending_intent_visits',
                 'work.report.pending_terminal_rows', 'work.scheduler.pending_terminal_rows',
                 'work.report.pending_sort_comparisons', 'work.scheduler.pending_sort_comparisons'))
    for spec in baseline.values():
        keys.update('fn.' + name + '.calls' for name in spec['entries'])
        keys.update(k for edit in spec['edits'] for k in re.findall(r'scalingprobe.Add\("([^"]+)"', edit['new']))
    return keys


def generate(rec, name, source, out, label, kind):
    pinned_inputs()
    measurement = kind != 'ordinary'
    if measurement:
        original = P2 / 'instrument/generate.py'
        code = original.read_text()
        code = replace(code, 'HERE = pathlib.Path(__file__).resolve().parent',
                       'HERE = pathlib.Path(' + repr(str(original.parent)) + ')')
        if kind == 'counters':
            # The original generator still owns SQLite accounting and SCOPES.
            # The exact executor transforms are retained by its patch manifest.
            anchors = read(APPARATUS / 'baseline-anchors.json')
            pending, = [edit for edit in anchors['runner.go']['edits'] if 'pendingDeliveries.intents_considered' in edit['new']]
            pending['new'] = replace(pending['new'],
                'scalingprobe.Add("work.pendingDeliveries.intents_considered",1)',
                'if scalingprobe.IsScope(ctx, 8) { scalingprobe.Add("work.report.pending_intent_visits", 1) } else { scalingprobe.Add("work.scheduler.pending_intent_visits", 1) }')
            if label == 'candidate':
                spec = anchors['composition_inspect.go']
                spec['edits'] = []
            start = code.index('functions={')
            end = code.index("(a.out/'alan/measurecounter').mkdir()", start)
            code = code[:start] + f'''from overlays import insert_probes
anchors = {dict(anchors)!r}
probes = {grouped_probes(profile(label))!r}
functions = sorted(set(anchors) | set(probes))
for file in functions:
 def transform(s, file=file):
    s = imported(s)
    spec = anchors.get(file, {{}})
    for name in spec.get('entries', []): s = entry(s, name)
    for name, scope in SCOPES.get(file, []):
        matches = list(re.finditer(r'^func (?:\\([^\\n]*\\) )?'+re.escape(name)+r'\\(ctx context.Context[^\\n]*\\{{\\n', s, re.M))
        if len(matches) != 1: raise ValueError('unique scope signature: '+file+':'+name)
        m = matches[0]
        s = s[:m.end()] + f'\\tctx = scalingprobe.Scope(ctx, {{scope}})\\n' + s[m.end():]
    for function, edits in probes.get(file, {{}}).items():
        s = insert_probes(s, function, edits)
    for row in spec.get('edits', []): s = replace(s, row['old'], row['new'])
    if file == 'runner.go':
        s = replace(s, 'for rows.Next() {{\\n\\t\\tvar source RunID',
                    'for rows.Next() {{\\n if scalingprobe.IsScope(ctx, 8) {{ scalingprobe.Add("work.report.pending_terminal_rows", 1) }} else {{ scalingprobe.Add("work.scheduler.pending_terminal_rows", 1) }}\\n var source RunID')
        s = replace(s, 'sort.Slice(pending, func(i, j int) bool {{ return pending[i].key < pending[j].key }})',
                    'sort.Slice(pending, func(i, j int) bool {{ if scalingprobe.IsScope(ctx, 8) {{ scalingprobe.Add("work.report.pending_sort_comparisons", 1) }} else {{ scalingprobe.Add("work.scheduler.pending_sort_comparisons", 1) }}; return pending[i].key < pending[j].key }})')
    return s
 edit('alan/internal/executor/'+file, transform)
''' + code[end:]
        generated = out.parent / (name + '-generator.py')
        generated.write_text(code)
        rec.bind_tool(generated)
        rec.run(name + '-generate', [rec.env['TASK5_PYTHON'], '-B', generated, '--alan', source,
                                    '--sqlite', rec.sources['sqlite'], '--driver', P2 / 'driver', '--out', out],
                extra={'PYTHONPATH': str(Path(__file__).parent)})
        if kind == 'counters':
            compact_counters(out)
        else:
            hook = out / 'driver/instrument_hook.go'
            hook.write_text(replace(hook.read_text(), 'func init() {',
                                    'func init() {counterRegistry = sqlite.MeasurementSnapshot;'))
        rec.bind(name + '-instrumented-alan', out / 'alan')
    else:
        out.mkdir()
        shutil.copytree(P2 / 'driver', out / 'driver')

    driver = out / 'driver'
    patch_driver(driver)
    main = driver / 'main.go'
    text = main.read_text()
    # Keep execute_window distinct from the sum of measured inner stages.
    text = replace(text, 'name == "approval_snapshot_basis" ||',
                   'name == "approval_snapshot_basis" || name == "approval_targeted_basis" ||')
    main.write_text(text)
    rec.run(name + '-format-driver', ['gofmt', '-w', driver])
    if measurement:
        for stem in ('generation.json', 'transformations.json', 'measurement.patch'):
            rec.bind_tool(out / stem)
        rec.run(name + '-format-sqlite', ['gofmt', '-w', out / 'sqlite/alan_measurement.go'])
        rec.bind(name + '-instrumented-sqlite', out / 'sqlite')
    differences = []
    for path in sorted(driver.glob('*.go')):
        orig = P2 / 'driver' / path.name
        differences.extend(difflib.unified_diff(
            orig.read_text().splitlines(True) if orig.exists() else [], path.read_text().splitlines(True),
            fromfile='original/' + path.name, tofile='overlay/' + path.name))
    diff = out / 'driver-overlay.patch'
    diff.write_text(''.join(differences))
    rec.bind_tool(diff)
    binary = out / 'driver.bin'
    build_module(rec, name, out / 'alan' if measurement else source, driver, binary,
                 sqlite=out / 'sqlite' if measurement else None)
    return binary


def direct_overlay(module, original):
    main = module / 'main.go'
    old = main.read_text()
    text = replace(old, '"os"\n', '"os"\n\t"reflect"\n')
    text = replace(text, 'type cut struct {', 'type cut struct {\n RunReport *aruntime.RunReport `json:"run_report"`')
    text = replace(text, 'type report struct {', 'type report struct {\n approvalMode string')
    text = replace(text, 'func (r *report) execute(', '''type continuationBasisReader interface {
 ContinuationBasis(context.Context, aruntime.RunID) (aruntime.ContinuationBasis, error)
}
func (r *report) captureRun(ctx context.Context, decl author.Series, m *meter, label string, position int, rr aruntime.RunReport) error {
 if err := r.capture(ctx, decl, m, label, position); err != nil { return err }
 last := &r.Cuts[len(r.Cuts)-1]
 if !rr.Quiescent || len(rr.Roots) != r.Bound || !reflect.DeepEqual(rr.Roots, last.Snapshot.Roots) ||
     rr.OpenRuns == nil || rr.UnresolvedIntents == nil || rr.PendingDeliveries == nil || rr.UnknownExternal == nil || rr.UnknownAdapters == nil {
     return fmt.Errorf("%s: incomplete RunReport or roots differ from typed cut", label)
 }
 last.RunReport = &rr
 return nil
}
func (r *report) execute(''')
    text = replace(text, 'opts := aruntime.RunOptions{', '''basisReader, targeted := any(store).(continuationBasisReader)
 if r.approvalMode == "targeted" && !targeted { return errors.New("targeted basis API unavailable") }
 opts := aruntime.RunOptions{''')
    start = '''s, e := aruntime.InspectExecution(ctx, r.Store, aruntime.DefaultLimits())
			if e != nil {
				return e
			}
			basis, e := s.ContinuationBasis(aruntime.RunID{Subject: fmt.Sprintf("series-%d", i), Generation: 1})'''
    text = replace(text, start, '''target := aruntime.RunID{Subject: fmt.Sprintf("series-%d", i), Generation: 1}
 var basis aruntime.ContinuationBasis
 var e error
 if r.approvalMode == "targeted" {
     basis, e = basisReader.ContinuationBasis(ctx, target)
 } else {
     s, err := aruntime.InspectExecution(ctx, r.Store, aruntime.DefaultLimits())
     if err != nil { return err }
     basis, e = s.ContinuationBasis(target)
 }''')
    for label, indent in [('after-run', '\t\t'), ('after-interval-reopen', '\t\t\t')]:
        start = indent + 'if _, err = store.Run(ctx, opts); err != nil {\n' + indent + '\treturn err\n' + indent + '}\n'
        start += indent + f'if err = r.capture(ctx, decl, m, "{label}", i); err != nil {{'
        new = indent + 'rr, runErr := store.Run(ctx, opts)\n' + indent + 'if runErr != nil { return runErr }\n'
        new += indent + f'if err = r.captureRun(ctx, decl, m, "{label}", i, rr); err != nil {{'
        text = replace(text, start, new)
    text = replace(text, 'if err = r.captureRun(ctx, decl, m, "after-interval-reopen",', '''basisReader, _ = any(store).(continuationBasisReader)
 if err = r.captureRun(ctx, decl, m, "after-interval-reopen",''')
    text = replace(text, 'mode := flag.String("mode",', 'approval := flag.String("approval-mode", "snapshot", "snapshot or targeted owner basis")\n mode := flag.String("mode",')
    text = replace(text, 'if (*mode !=', 'if (*approval != "snapshot" && *approval != "targeted") || (*mode !=')
    text = replace(text, 'r := report{Version:', 'r := report{approvalMode: *approval, Version:')
    main.write_text(text)
    return (original / 'main.go').read_text()
