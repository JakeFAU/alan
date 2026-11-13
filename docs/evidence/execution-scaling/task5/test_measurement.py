import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest

from custody import TaskRecorder, extract_source, require_copy
from measurement import schedule, check_schedule, validate_report
from overlays import insert_probes


class MachineryTests(unittest.TestCase):
    def test_design_order(self):
        rows = schedule()
        self.assertEqual(189, len(rows))
        self.assertEqual(189, len({r['name'] for r in rows}))
        self.assertEqual(21, sum(r['experiment'] == 'warmup' for r in rows))
        for changed in (rows[1:], rows + [rows[-1]], rows[::-1]):
            with self.assertRaisesRegex(ValueError, 'schedule'):
                check_schedule(changed)
        wrong = copy.deepcopy(rows)
        wrong[-1]['approval'] = 'targeted'
        with self.assertRaisesRegex(ValueError, 'schedule'):
            check_schedule(wrong)

    def test_invalid_probe_anchors(self):
        for source in ('func other() {\n}\n', 'func f() {\n}\n' * 2):
            with self.assertRaisesRegex(ValueError, 'unique function'):
                insert_probes(source, 'f', [('x', '', '1', 'always')])
        with self.assertRaisesRegex(ValueError, 'unique anchor'):
            insert_probes('func f() {\n}\n', 'f', [('x', 'missing', '1', 'always')])

    def test_recorder_preserves_failure_timeout_and_mismatch(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            source = root / 'source'
            source.mkdir()
            program = source / 'program.py'
            program.write_text('pass\n')
            rec = TaskRecorder(root, sources={'test': source}, tools=[Path(sys.executable)],
                               env={'PATH': '/usr/bin:/bin'})
            for name, code, timeout, expected in [
                ('nonzero', 'raise SystemExit(7)', 3, 7),
                ('timeout', 'import time; time.sleep(5)', 0.05, -9),
            ]:
                with self.assertRaises(RuntimeError):
                    rec.run(name, [sys.executable, '-c', code], timeout=timeout)
                raw = json.loads((rec.gates / name / 'outcome.json').read_text())
                self.assertEqual(expected, raw['exit'])
                self.assertEqual(name == 'timeout', raw['timeout'])
            program.write_text('print("changed")\n')
            with self.assertRaisesRegex(ValueError, 'frozen'):
                rec.run('source-tampered', [sys.executable, program])

    def test_binary_and_controller_change_rejected(self):
        for name in ('driver.bin', 'controller.py'):
            with self.subTest(name=name), tempfile.TemporaryDirectory() as d:
                root = Path(d)
                tool = root / name
                tool.write_text('frozen')
                rec = TaskRecorder(root, tools=[tool], env={'PATH': '/usr/bin:/bin'})
                tool.write_text('changed')
                with self.assertRaisesRegex(ValueError, 'frozen'):
                    rec.run('bad-tool', [sys.executable, '-c', 'pass'])

    def test_copy_required_and_copy_race(self):
        from unittest.mock import patch
        import shutil
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            source = root / 'source'
            source.write_bytes(b'main')
            with self.assertRaisesRegex(ValueError, 'sidecars'):
                require_copy(source, root / 'missing', root / 'missing.json', stopped=True)
            for suffix in ('-wal', '-shm'):
                Path(str(source) + suffix).touch()
            copied = root / 'copied'
            require_copy(source, copied, root / 'valid.json', stopped=True)
            self.assertTrue(Path(str(copied) + '-wal').is_file())
            original = shutil.copyfileobj
            def change(src, dst):
                original(src, dst)
                if src.name == str(source):
                    source.write_bytes(b'changed')
            with patch('common.shutil.copyfileobj', side_effect=change):
                with self.assertRaisesRegex(ValueError, 'changed'):
                    require_copy(source, root / 'racing', root / 'racing.json', stopped=True)

    def test_extract_rejects_unsafe_members_before_writing(self):
        import io
        import tarfile
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            archive = root / 'test.tar'
            for invalid in ('../escape', '/absolute'):
                with tarfile.open(archive, 'w') as out:
                    good = tarfile.TarInfo('valid')
                    good.size = 1
                    out.addfile(good, io.BytesIO(b'x'))
                    out.addfile(tarfile.TarInfo(invalid))
                with self.assertRaisesRegex(ValueError, 'unsafe'):
                    extract_source(archive, root / 'output')
                self.assertFalse((root / 'output' / 'valid').exists())


class ReportControls(unittest.TestCase):
    """Mutate actual excluded smoke rows. No synthetic successful driver proof."""
    @classmethod
    def setUpClass(cls):
        import os
        from custody import read
        from overlays import probe_keys, scopes
        value = os.environ.get('ALAN_TASK5_SMOKE_ROOT')
        if not value:
            raise unittest.SkipTest('supply a retained excluded smoke root')
        root = Path(value).resolve()
        binding = read(root / 'smoke-freeze.json')
        cls.rows = {}
        for record in binding['rows']:
            design = record['design']
            name = design['name']
            output = root / 'gates' / ('sample-' + name) / 'stdout.log'
            store = root / 'samples' / name / 'store/store.sqlite'
            cls.rows[name] = (read(output), design, record['revision'], store, scopes(),
                              probe_keys(design['side']))

    def mutate(self, name, change, reason):
        original, *binding = self.rows[name]
        validate_report(original, *binding)
        altered = copy.deepcopy(original)
        change(altered)
        self.assertNotEqual(original, altered)
        with self.assertRaisesRegex(ValueError, reason):
            validate_report(altered, *binding)

    def test_targeted_phase_and_execute_sum(self):
        name = 'excluded-smoke-b8-targeted'
        self.mutate(name, lambda r: next(s for s in r['stages'] if s['name'] == 'approval_targeted_basis')
                    .update(name='approval_snapshot_basis'), 'stage order')
        self.mutate(name, lambda r: r['execute_totals'].update(
            wall_ns=r['execute_totals']['wall_ns'] - sum(s['wall_ns'] for s in r['stages']
                                                        if s['name'] == 'approval_targeted_basis')),
                    'Execute sum mismatch')

    def test_primary_rejects_contamination(self):
        name = 'excluded-smoke-b8-targeted'
        for change in (
            lambda r: r.update(profiled=True),
            lambda r: r.update(instrumented=True),
            lambda r: r.update(approval_mode='snapshot'),
        ):
            self.mutate(name, change, 'runtime configuration')
        self.mutate(name, lambda r: r.update(execute_counter_registry={'forged': 1}), 'contamination')
        self.mutate(name, lambda r: r['prefixes'][-1]['direct'].update(counter_deltas={'forged': 1}),
                    'contamination')

    def test_false_or_truncated_success(self):
        name = 'excluded-smoke-b8-targeted'
        self.mutate(name, lambda r: r.update(invariants_ok=False), 'complete success proof')
        self.mutate(name, lambda r: r['prefixes'].pop(), 'exactly 8 prefixes')
        self.mutate(name, lambda r: r['prefixes'].reverse(), 'expected position')
        self.mutate(name, lambda r: r['prefixes'][2]['cut'].update(completed=1), 'returned-cut')
        self.mutate(name, lambda r: r.update(source_label='foreign'), 'identity mismatch')

    def test_diagnostic_accounting_is_checked_before_adapter(self):
        name = 'excluded-smoke-b8-compact'
        self.mutate(name, lambda r: r['execute_counter_registry'].update(
            {'sql.attribution.invalid_scope_events': 1}), 'incomplete')
        self.mutate(name, lambda r: r['execute_counter_registry'].update(
            {'sql.total.executions': 0}), 'partition')
        self.mutate(name, lambda r: r['prefixes'][0]['direct']['counter_deltas'].update(
            {'work.static.ready_candidates': 99999}), 'arithmetic')
        self.mutate(name, lambda r: r['prefixes'][0]['direct']['counter_deltas'].pop(
            'sql.scope_total.unknown.sqlite_row_returns'), 'vocabulary')
