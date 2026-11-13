"""Controls for the frozen P5 schedule, evidence reread and full-report cuts."""
import copy
import json
from pathlib import Path
import signal
import sys
from tempfile import TemporaryDirectory
import unittest

import campaign
from campaign_qualification import controlled
from qualification import verify_direct_reports


class ScheduleTest(unittest.TestCase):
    def test_exact_balanced_serial_schedule(self):
        rows = campaign.designs('timing')
        self.assertEqual(len(rows), 98)
        for bound in (1, 8, 16, 32, 64, 128, 256):
            selected = [r for r in rows if r['bound'] == bound]
            self.assertEqual([r['side'] for r in selected[:2]], ['baseline', 'candidate'])
            self.assertTrue(all(r['approval'] == 'targeted' and r['kind'] == 'ordinary' for r in selected))
            self.assertEqual(''.join(r['arm'] for r in selected[2:]), 'ABBAABBAABBA')
        self.assertTrue(all(r['kind'] == 'counters' for r in campaign.designs('counters')))
        self.assertEqual(len(campaign.designs('counters')), 4)


class EvidenceTest(unittest.TestCase):
    def setUp(self):
        folder = TemporaryDirectory()
        self.addCleanup(folder.cleanup)
        self.root = Path(folder.name)
        self.plan = {'sources': {side: {'commit': side * 4} for side in campaign.SIDES}}
        self.binaries = {}
        for side in ('baseline-ordinary', 'baseline-counters', 'candidate-ordinary', 'candidate-counters'):
            path = self.root / side
            path.symlink_to(sys.executable)
            self.binaries[side] = str(path)
        self.rows = campaign.row_records(self.root, self.plan, 'smoke', self.binaries)

    def test_rejects_missing_reordered_duplicate_or_altered_frozen_rows(self):
        for mutant in (self.rows[:-1], self.rows[::-1], self.rows + self.rows[:1]):
            with self.subTest(length=len(mutant)), self.assertRaises(ValueError):
                campaign.require_rows(self.root, self.plan, 'smoke', self.binaries, mutant)
        for field, value in [('revision', 'foreign'), ('binary_sha256', 'foreign'),
                             ('argv', ['wrong command'])]:
            mutant = copy.deepcopy(self.rows)
            mutant[0][field] = value
            with self.subTest(field=field), self.assertRaises(ValueError):
                campaign.require_rows(self.root, self.plan, 'smoke', self.binaries, mutant)
        self.assertEqual(campaign.require_rows(self.root, self.plan, 'smoke', self.binaries, self.rows),
                         self.rows)

    def test_unexpected_exit_is_retained_without_replacement(self):
        # A real child exits 7. The frozen order is still attempted exactly once.
        (self.root / 'samples').mkdir()
        rec = campaign.TaskRecorder(self.root, env={'PATH': '/usr/bin:/bin'})
        rows = copy.deepcopy(self.rows)
        for record in rows:
            record['argv'] = [sys.executable, '-B', str(campaign.HERE / 'sample_usage.py'),
                              sys.executable, '-B', '-c', 'raise SystemExit(7)']
        with self.assertRaises(ValueError):
            campaign.execute_rows(rec, 'smoke', rows)
        for record in rows:
            name = record['design']['name']
            self.assertEqual(campaign.read(rec.gates / ('sample-' + name) / 'outcome.json')['exit'], 7)
        self.assertEqual(len(list(rec.gates.iterdir())), len(rows))
        with self.assertRaises(FileExistsError):
            campaign.execute_rows(rec, 'smoke', rows)
        with self.assertRaises(ValueError):
            campaign.read_rows(rec, 'smoke', rows)

    def test_build_cannot_replace_frozen_source_with_coherent_foreign_manifest(self):
        source, other = self.root / 'source', self.root / 'foreign'
        for path in (source, other):
            path.mkdir()
            (path / 'input.go').write_text('same content')
        env = {'PATH': '/usr/bin:/bin'}
        parent = campaign.TaskRecorder(self.root, env=env, sources={'candidate': source})
        foreign = campaign.TaskRecorder(self.root, env=env, sources={'candidate': other})
        campaign.separate_inputs(foreign, 'timing', [])
        digest = campaign.sha(self.root / 'timing-inputs.json')
        with self.assertRaisesRegex(ValueError, 'dropped or replaced a frozen source'):
            campaign.load_inputs(self.root, env, 'timing', digest, parent)

    def test_bound_files_reject_drift_without_binding_unrelated_results(self):
        env = {'PATH': '/usr/bin:/bin'}
        source = self.root / 'source'
        source.mkdir()
        entry = source / 'driver.py'
        entry.write_text('original')
        report = self.root / 'build-info.log'
        report.write_text('build info')
        rec = campaign.TaskRecorder(self.root, env=env, tools=[entry])
        campaign.separate_inputs(rec, 'timing', [report])
        digest = campaign.sha(self.root / 'timing-inputs.json')
        (source / 'results.md').write_text('additional evidence')
        campaign.load_inputs(self.root, env, 'timing', digest, rec)
        report.write_text('changed build info')
        with self.assertRaisesRegex(ValueError, 'bound archive or build evidence changed'):
            campaign.load_inputs(self.root, env, 'timing', digest, rec)
        report.write_text('build info')
        entry.write_text('changed executable input')
        with self.assertRaisesRegex(ValueError, 'frozen inputs changed'):
            campaign.load_inputs(self.root, env, 'timing', digest, rec)

    def test_usage_with_driver_progress_and_duplicate_or_missing_marker(self):
        rec = campaign.TaskRecorder(self.root, env={'PATH': '/usr/bin:/bin'})
        wrapper = [sys.executable, '-B', str(campaign.HERE / 'sample_usage.py'), sys.executable, '-B', '-c']
        rec.run('rss', [*wrapper, "import sys; print('sample=b1 stage=run done', file=sys.stderr)"])
        stderr = rec.gates / 'rss/stderr.log'
        text = stderr.read_text()
        self.assertGreater(campaign.process_peak(stderr, 1), 1)
        for mutant in (text + text.splitlines()[-1] + '\n', text.splitlines()[0] + '\n',
                       text + 'unexpected tail\n'):
            stderr.write_text(mutant)
            with self.assertRaises(ValueError):
                campaign.process_peak(stderr, 1)
        stderr.write_text(text)
        with self.assertRaises(ValueError):
            campaign.process_peak(stderr, 10**15)

    def test_usage_wrapper_preserves_signal_and_timeout(self):
        rec = campaign.TaskRecorder(self.root, env={'PATH': '/usr/bin:/bin'})
        wrapper = [sys.executable, '-B', str(campaign.HERE / 'sample_usage.py'), sys.executable, '-B', '-c']
        with self.assertRaises(RuntimeError):
            rec.run('signal', [*wrapper, 'import os, signal; os.kill(os.getpid(), signal.SIGTERM)'])
        with self.assertRaises(RuntimeError):
            rec.run('timeout', [*wrapper, 'import time; time.sleep(30)'], timeout=0.1)
        terminated = campaign.read(rec.gates / 'signal/outcome.json')
        timeout = campaign.read(rec.gates / 'timeout/outcome.json')
        self.assertEqual((terminated['exit'], terminated['timeout']), (-signal.SIGTERM, False))
        self.assertEqual((timeout['exit'], timeout['timeout']), (-signal.SIGKILL, True))

    def test_controlled_json_reassembles_fragmented_test_output(self):
        log = self.root / 'test2json.jsonl'
        case = {'case': 'repeated', 'evidence': 'x' * 2100}
        value = '    evidence.go:13: M4_CONTROLLED ' + json.dumps(case) + '\n'
        other = '    unrelated.go:4: ordinary output\n'
        chunks = [('TestRecovery', value[:1024]), ('TestOther', other),
                  ('TestRecovery', value[1024:2048]), ('TestRecovery', value[2048:])]
        events = [json.dumps({'Action': 'output', 'Package': 'weather', 'Test': name, 'Output': text})
                  for name, text in chunks]
        log.write_text('\n'.join(events))
        self.assertEqual(controlled(log, 'M4_CONTROLLED'), {'repeated': case})
        log.write_text('\n'.join(events[:-1]))
        with self.assertRaisesRegex(ValueError, 'truncated'):
            controlled(log, 'M4_CONTROLLED')

    def test_direct_report_checks_every_field_at_every_cut(self):
        rr = dict.fromkeys(('Steps', 'Dispatches', 'Quiescent', 'OpenRuns',
                            'UnresolvedIntents', 'PendingDeliveries', 'UnknownExternal',
                            'UnknownAdapters', 'Roots', 'RejectedAdmissions'))
        rr.update(Quiescent=True, Roots=[{}] * 8)
        report = {'ok': True, 'cuts': [{'label': 'after-run', 'position': n,
                  'run_report': copy.deepcopy(rr), 'snapshot': {'Roots': rr['Roots']}}
                  for n in range(1, 9)]}
        verify_direct_reports(report, 0)
        del report['cuts'][-1]['run_report']['RejectedAdmissions']
        with self.assertRaises(ValueError):
            verify_direct_reports(report, 0)


if __name__ == '__main__':
    unittest.main()
