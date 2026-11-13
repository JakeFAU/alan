"""Controls for the frozen P5 schedule, evidence reread and full-report cuts."""
import copy
from pathlib import Path
import sys
from tempfile import TemporaryDirectory
import unittest

import campaign
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
