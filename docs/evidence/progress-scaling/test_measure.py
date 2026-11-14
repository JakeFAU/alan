"""Apparatus controls: ordering, custody, failed rows and strict driver edits."""
import json
from pathlib import Path
import sys
import tempfile
import unittest

import measure


class MeasureTest(unittest.TestCase):
    def test_all_six_orders_in_adjacent_blocks(self):
        rows = measure.timing_design((1, 8))
        self.assertEqual(len(rows), 42)
        self.assertEqual(len({row['name'] for row in rows}), 42)
        expected = [
            ['base-full', 'candidate-full', 'candidate-progress'],
            ['base-full', 'candidate-progress', 'candidate-full'],
            ['candidate-full', 'base-full', 'candidate-progress'],
            ['candidate-full', 'candidate-progress', 'base-full'],
            ['candidate-progress', 'base-full', 'candidate-full'],
            ['candidate-progress', 'candidate-full', 'base-full'],
        ]
        for bound in (1, 8):
            block = [r for r in rows if r['bound'] == bound]
            self.assertEqual([r['kind'] for r in block],
                             ['warmup'] * 3 + ['timing'] * 18)
            self.assertEqual([r['arm'] for r in block[:3]], expected[0])
            self.assertEqual([[r['arm'] for r in block[3 + i:6 + i]]
                              for i in range(0, 18, 3)], expected)

    def test_output_is_new_and_inside_owned_workspace(self):
        with tempfile.TemporaryDirectory() as tmp:
            workspace = Path(tmp) / 'plan'
            workspace.mkdir()
            sample = workspace / 'new'
            self.assertEqual(measure.reserve_output(sample, workspace), sample.resolve())
            with self.assertRaises(FileExistsError):
                measure.reserve_output(sample, workspace)
            with self.assertRaises(ValueError):
                measure.reserve_output(workspace / '..' / 'outside', workspace)
            (workspace / 'escape').symlink_to(Path(tmp))
            with self.assertRaises(ValueError):
                measure.reserve_output(workspace / 'escape' / 'outside', workspace)

    def test_report_binding_rejects_other_revision_mode_or_output(self):
        with tempfile.TemporaryDirectory() as tmp:
            store = Path(tmp) / 'store.sqlite'
            row = {'name': 'sample-1', 'bound': 8}
            expected = measure.report_identity(row, measure.BASE, 'full', store)
            self.assertEqual(expected['source_label'], measure.BASE)
            self.assertEqual(expected['execution_mode'], 'full')
            self.assertEqual(expected['store'], str(store))
            measure.check_identity(expected, row, measure.BASE, 'full', store)
            for key, value in [('source_label', '0' * 40), ('execution_mode', 'progress'),
                               ('store', '/tmp/another.sqlite'), ('sample', 'sample-2'), ('bound', 1)]:
                with self.subTest(key=key), self.assertRaises(ValueError):
                    measure.check_identity(dict(expected, **{key: value}), row, measure.BASE, 'full', store)

    def test_failed_child_retains_stdout_stderr_exit_and_row(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            recorder = measure.TaskRecorder(root, env={'PATH': '/usr/bin:/bin'})
            row = {'name': 'failed', 'bound': 1, 'kind': 'smoke', 'arm': 'base-full'}
            # This executable returns 17 after producing partial evidence.
            cmd = [sys.executable, '-c',
                   "import sys;print('partial stdout');print('failed check', file=sys.stderr);sys.exit(17)"]
            result = measure.capture_row(recorder, row, measure.BASE, 'full',
                                         Path(sys.executable), root / 'store.sqlite', cmd)
            gate = root / 'gates' / 'failed'
            self.assertEqual(result['exit'], 17)
            self.assertIn('failure', result)
            self.assertEqual((gate / 'stdout.log').read_text(), 'partial stdout\n')
            self.assertEqual((gate / 'stderr.log').read_text(), 'failed check\n')
            self.assertEqual(json.loads((gate / 'row.json').read_text()), result)
            self.assertEqual(result['stdout_sha256'], measure.sha(gate / 'stdout.log'))

    def test_driver_fails_closed_on_duplicated_or_missing_loop(self):
        source = (measure.DRIVER / 'main.go').read_text()
        generated = measure.transform_driver(source)
        self.assertNotIn('cuts.verify(', generated)
        self.assertNotIn('var base, previous resourceSnapshot', generated)
        self.assertIn('store.ContinuationBasis(ctx,', generated)
        self.assertIn('runObservation(ctx, store, opts)', generated)
        self.assertIn('reflect.DeepEqual(replayed.Roots, cut.Roots)', generated)
        for malformed in (source.replace('windowName := "execute_window"', ''),
                          source + '\nwindowName := "execute_window"\n'):
            with self.assertRaisesRegex(ValueError, 'anchor'):
                measure.transform_driver(malformed)


if __name__ == '__main__':
    unittest.main()
