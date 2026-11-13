"""Acknowledged crash cuts with complete pre-reader archives and opposite-build resumes."""
import difflib
import json
from pathlib import Path
import re
import secrets
import signal
import subprocess

from custody import expect_tests, inventory, load, read, require_copy, sha, write
from overlays import replace
from qualification import ROUTES, compare

WEATHER = (('series', '^TestWeatherSeriesProcess$'),
           ('continuation', '^TestWeatherContinuationProcess$'),
           ('recovery', '^TestWeatherProcessRecovery$'),
           ('public', '^TestWeatherSeries(Eight|OriginalAndLegacyPrograms|Corruption)$'),
           ('predecessors', '^TestWeatherContinuationPredecessors$'))


def build_weather(rec, side, directory):
    # Retain the P3 public-consumer provenance overlay. Its expected inputs
    # use sibling source names and a separate output; give it explicit aliases.
    holder = directory / (side + '-weather')
    holder.mkdir()
    (holder / side).symlink_to(rec.sources[side], target_is_directory=True)

    class TemplateRecorder:
        root = holder
        env = dict(rec.env, GOFLAGS='-buildvcs=false')

    desc = ROUTES.weather_overlay(TemplateRecorder(), side)
    mapping = read(desc)['Replace']
    source = rec.sources[side] / 'examples/weather-agent'
    overlay = desc.parent
    # Replace the legacy series-only insertion with common kill instrumentation,
    # so the original assertions and exact ACK or endpoint witnesses stay intact.
    series = overlay / 'series_process_test.go'
    original_series = (source / series.name).read_text()
    if original_series.count('owner.kill(t, witness)') != 1:
        raise ValueError('series stopped archive anchor drift')
    text = series.read_text()
    text = replace(text, '\n\t\t\tpreserveSeries(t, db, tc.name+"-raw-stop")', '')
    start = 'before := capture(db)'
    text = replace(text, start, start + '''
 if strings.Contains(name, "-raw-stop-") && len(before) != 3 {
     t.Fatalf("reaped owner missing expected sidecars: %v", before)
 }''')
    if '"strings"' not in text:
        text = replace(text, 'import (', 'import (\n "strings"')
    series.write_text(text)

    # Copy the actual reaped Store immediately, before the caller can run any
    # reader. weatherOwner.start retains db explicitly in this overlay.
    barrier = source / 'barrier_process_test.go'
    new_barrier = overlay / barrier.name
    text = barrier.read_text()
    text = replace(text, 'type weatherOwner struct {', 'type weatherOwner struct {\n task5Store string')
    matches = list(re.finditer(r'&weatherOwner\{', text))
    if len(matches) != 1:
        raise ValueError('weather owner constructor anchor changed')
    text = replace(text, '&weatherOwner{', '&weatherOwner{task5Store: db, ')
    marker = 't.Logf("SIGKILL pid=%d witness=%s", c.cmd.Process.Pid, witness)'
    text = replace(text, marker, marker + '''
 if witness == "" { t.Fatal("missing acknowledged boundary or endpoint witness") }
 if dir := os.Getenv("ALAN_SERIES_EVIDENCE_DIR"); dir != "" {
     name := strings.ReplaceAll(t.Name(), "/", "-") + fmt.Sprintf("-raw-stop-%d", c.cmd.Process.Pid)
     preserveSeries(t, c.task5Store, name)
     record, err := json.Marshal(map[string]any{"pid": c.cmd.Process.Pid, "signal": int(s.Signal()), "reaped": c.reaped, "witness": witness})
     if err != nil { t.Fatal(err) }
     if err = os.WriteFile(filepath.Join(dir, name+".kill.json"), record, 0600); err != nil { t.Fatal(err) }
 }''')
    for package in ('strings', 'path/filepath'):
        if '"' + package + '"' not in text:
            text = replace(text, 'import (', 'import (\n "' + package + '"')
    new_barrier.write_text(text)
    # Re-map every overlay file to the real compiled source path.
    replacements = {str(source / Path(path).name): value for path, value in mapping.items()}
    replacements[str(barrier)] = str(new_barrier)
    rec.run('weather-format-' + side, ['gofmt', '-w', *replacements.values()])
    for original, patched in replacements.items():
        before, after = Path(original).read_text(), Path(patched).read_text()
        (overlay / (Path(original).name + '.task5.patch')).write_text(''.join(difflib.unified_diff(
            before.splitlines(True), after.splitlines(True), fromfile=original, tofile=patched)))
    bound = overlay / 'task5-overlay.json'
    write(bound, {'Replace': replacements})
    rec.bind(side + '-weather-overlay', overlay)
    binary = directory / (side + '-weather.test')
    rec.run('weather-build-' + side, ['go', 'test', '-c', '-mod=readonly', '-buildvcs=false', '-tags=integration',
            '-overlay=' + str(bound), '-o', binary, './examples/weather-agent'], rec.sources[side])
    rec.bind_tool(binary)


def weather_expected(source, section):
    folder = source / 'examples/weather-agent'
    if section == 'continuation':
        text = (folder / 'continuation_process_test.go').read_text()
        return {'TestWeatherContinuationProcess/' + name for name in re.findall(r'^\s*\{name: "([^"]+)"', text, re.M)}
    if section == 'series':
        text = (folder / 'series_process_test.go').read_text()
        return {'TestWeatherSeriesProcess/' + name for name in re.findall(r'^\s*\{"([^"]+)"', text, re.M)}
    if section == 'recovery':
        return {'TestWeatherProcessRecovery/two_interrupted_recoveries',
                'TestWeatherProcessRecovery/accepted_and_lost_then_explicitly_reobserved',
                'TestWeatherProcessRecovery/completed_then_explicitly_refreshed'}
    if section == 'public':
        return {'TestWeatherSeriesEight', 'TestWeatherSeriesOriginalAndLegacyPrograms', 'TestWeatherSeriesCorruption'}
    return {'TestWeatherContinuationPredecessors'}


def verify_weather_stops(folder, expected, kind):
    expected_db = sorted(folder.glob('*-raw-stop-*.sqlite'))
    if kind in ('series', 'continuation', 'recovery') and not expected_db:
        raise ValueError('weather route has no sealed reaped stores')
    names = []
    for db in expected_db:
        proof = read(Path(str(db) + '.copy.json'))
        kill = read(db.with_suffix('.kill.json'))
        snapshot = {suffix: sha(str(db) + suffix) for suffix in ('', '-wal', '-shm')}
        if (not proof['equal'] or proof['source_before'] != snapshot or proof['source_after'] != snapshot or
                proof['copy'] != snapshot or not kill['reaped'] or kill['signal'] != int(signal.SIGKILL) or
                kill['pid'] <= 0 or not kill['witness']):
            raise ValueError('invalid complete-copy or real process reap evidence: ' + str(db))
        names.append(db.name.split('-raw-stop-')[0])
    if kind in ('series', 'continuation') and not {t.replace('/', '-') for t in expected} <= set(names):
        raise ValueError('missing stopped capture for one or more named cut')
    if kind == 'recovery' and names.count('TestWeatherProcessRecovery-two_interrupted_recoveries') != 3:
        raise ValueError('expected three independently reaped attempts in repeated recovery')


def weather(rec, plan, *, excluded=False):
    tools = rec.root / 'builds/qualification'
    out = rec.root / ('excluded-weather-smoke' if excluded else 'weather-recovery')
    out.mkdir()
    for side in (('candidate',) if excluded else ('baseline', 'candidate')):
        source = rec.sources[side]
        binary = tools / (side + '-weather.test')
        sections = [('excluded-stop', '^TestWeatherProcessRecovery$/^after_request_commit$')] if excluded else WEATHER
        for name, filt in sections:
            folder = out / (side + '-' + name)
            folder.mkdir()
            archives = folder / 'archives'
            consumers = folder / 'public-consumers'
            report = rec.run(side + '-weather-' + name, ['go', 'tool', 'test2json', '-p', 'weather-' + side,
                        binary, '-test.v=test2json', '-test.count=1', '-test.timeout=25m', '-test.run=' + filt],
                        source / 'examples/weather-agent',
                        {'ALAN_SERIES_EVIDENCE_DIR': str(archives), 'ALAN_P3_CONSUMER_EVIDENCE_DIR': str(consumers),
                         })
            required = ({'TestWeatherProcessRecovery/after_request_commit'} if excluded else weather_expected(source, name))
            if not required:
                raise ValueError('source-backed weather filter has no required tests: ' + name)
            expect_tests(report, required)
            verify_weather_stops(archives, required, 'series' if excluded else name)
            retained = [p for p in consumers.rglob('weather-agent') if p.is_file()]
            if not retained and name != 'public':
                raise ValueError('missing actual independent public consumer binary')
            for consumer in retained:
                if not (consumer.parent / 'go.mod').is_file() or not (consumer.parent / 'go.sum').is_file():
                    raise ValueError('missing actual independent public consumer module')
            write(folder / 'raw-inventory.json', inventory(folder))
    if not excluded:
        # Inspect and Replay run on separately copied stopped inputs. Compare
        # by source, including the available prior P3 stopped/final archives.
        for source_side in ('baseline', 'candidate'):
            for folder in sorted(out.glob(source_side + '-*')):
                archives = folder / 'archives'
                sealed = inventory(archives)
                for index, db in enumerate(sorted(archives.glob('*.sqlite')), 1):
                    name = folder.name + '-archive-' + str(index)
                    branches, reports = [], []
                    for reader in ('baseline', 'candidate'):
                        branch = out / 'readers' / name / reader / 'store.sqlite'
                        branch.parent.mkdir(parents=True)
                        require_copy(db, branch, branch.parent / 'copy.json')
                        reports.append(rec.run(name + '-' + reader, [tools / (reader + '-direct'),
                            '-mode', 'generic-reader', '-db', branch, '-source-label', plan['sources'][reader]['commit']]))
                        branches.append(branch)
                    compare(rec, out, tools / 'direct-tools', name + '-compare', reports, branches)
                if inventory(archives) != sealed:
                    raise ValueError('sealed weather original changed')


def cross_build(rec, plan):
    tools = rec.root / 'builds/qualification'
    root = rec.root / 'cross-build-recovery'
    root.mkdir()
    original = load('task5_compare_stores', rec.sources['candidate'] / 'tools/compare_stores.py')
    for writer, reader in [('baseline', 'candidate'), ('candidate', 'baseline')]:
        for boundary, phase in [('execution.staged', 'pre-commit'), ('execution.committed', 'post-commit')]:
            name = 'cross-' + writer + '-' + phase
            case = root / name
            case.mkdir()
            source = case / 'writer.sqlite'
            binary = tools / (writer + '-alan')
            rec.run(name + '-init', [binary, 'init', '--db', source, '--demo', 'nested'])
            tables = read(rec.run(name + '-initial', [binary, 'inspect', '--db', source]))['Tables']
            subject = next(r for r in tables['subjects'] if r['id'] == 'inner')
            run = next(r for r in tables['runs'] if r['subject'] == 'inner' and r['generation'] == subject['generation'])
            event = {'Scenario': 'correspondence', 'Token': secrets.token_hex(16), 'Boundary': boundary,
                     'Phase': phase, 'Store': str(source), 'Subject': 'inner', 'Generation': run['generation'],
                     'Epoch': run['epoch'] + 1, 'Sequence': 7,
                     'Intent': json.dumps(['inner', run['generation'], 7, 0], separators=(',', ':'))}
            argv = ['sh', '-c', 'exec 3>&1 4<&0; exec "$@"', 'barrier', str(binary), 'run', '--db', str(source)]
            environment = dict(rec.env, GOFLAGS='-buildvcs=false', ALAN_BARRIER=json.dumps(event))
            before = rec.snapshot()
            write(case / 'command.json', {'argv': argv, 'cwd': str(case), 'environment': environment,
                   'source': plan['sources'][writer], 'binary_sha256': sha(binary), 'inputs': before})
            with (case / 'writer-stderr.log').open('xb') as err:
                with subprocess.Popen(argv, cwd=case, env=environment, stdin=subprocess.PIPE,
                                      stdout=subprocess.PIPE, stderr=err, start_new_session=True) as child:
                    try:
                        ack = original.read_acknowledgement(child.stdout)
                        write(case / 'ack.json', ack)
                        if any(ack.get(key) != value for key, value in dict(event, PID=child.pid).items()):
                            raise ValueError('wrong genuine barrier acknowledgement')
                    finally:
                        child.kill()
                        status = child.wait(timeout=30)
                        write(case / 'reap.json', {'pid': child.pid, 'exit': status, 'expected': -signal.SIGKILL})
            if status != -signal.SIGKILL:
                raise ValueError('writer was not reaped after real SIGKILL')
            # First action on the database after reap is complete-copy custody.
            sealed = case / 'sealed/store.sqlite'
            require_copy(source, sealed, case / 'seal-copy.json', stopped=True)
            sealed_identity = inventory(sealed.parent)
            write(case / 'inputs-after.json', rec.snapshot())
            outputs, reports, dbs = [], [], []
            for side in (writer, reader):
                clone = case / side / 'store.sqlite'
                clone.parent.mkdir(parents=True)
                require_copy(sealed, clone, clone.parent / 'copy.json', stopped=True)
                outputs.append(rec.run(name + '-' + side + '-resume', [tools / (side + '-alan'), 'run', '--db', clone]))
                reports.append(rec.run(name + '-' + side + '-read', [tools / (side + '-direct'), '-mode', 'generic-reader',
                    '-db', clone, '-source-label', plan['sources'][side]['commit']]))
                dbs.append(clone)
            if outputs[0].read_bytes() != outputs[1].read_bytes():
                raise ValueError('opposite-build RunReports differ')
            result = read(outputs[0])
            if not result['Quiescent'] or result['OpenRuns']:
                raise ValueError('opposite-build original computation did not complete')
            compare(rec, root, tools / 'direct-tools', name + '-exact', reports, dbs)
            if inventory(sealed.parent) != sealed_identity:
                raise ValueError('sealed stopped Store changed during resume/read')
            write(case / 'result.json', {'ok': True, 'writer': writer, 'reader': reader, 'boundary': boundary,
                                        'sealed_source': sealed_identity, 'run_report_sha256': sha(outputs[0])})


def recovery(rec, plan):
    weather(rec, plan)
    cross_build(rec, plan)
    write(rec.root / 'recovery-index.json', {name: inventory(rec.root / name)
          for name in ('weather-recovery', 'cross-build-recovery')})
