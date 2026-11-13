import json
import pathlib
import tempfile
import unittest

from compare_counters import compare


class CounterComparisonTests(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = pathlib.Path(directory.name)
        self.reports = {
            (bound, kind): self.report(bound, kind)
            for bound in (1, 8) for kind in ('original', 'compact')
        }

    def report(self, bound, kind):
        totals = {
            'sql.total.completed_prepares': 0,
            'sql.total.invocation_attempts': 0,
            'sql.total.step_calls': 0,
            'sql.total.executions': 3 * bound,
            'sql.total.sqlite_row_returns': 0,
            'sql.total.fullscan_steps': 0,
            'sql.total.vm_steps': 0,
            'sql.total.step_errors': 0,
            'sql.total.bind_errors': 0,
        }
        registry = dict(totals, **{
            'sql.scope.read.class.select.executions': 2 * bound,
            'sql.scope_overflow.executions': bound,
        })
        key = ('sql.scope.read.class.select.executions' if kind == 'original'
               else 'sql.scope_total.read.executions')
        return {
            'bound': bound,
            'invariants_ok': True,
            'prefix_enabled': True,
            'stages': [{'name': 'execute_window', 'counter_deltas': totals}],
            'prefixes': [
                {'position': position, 'declaration_bound': bound,
                 'retained_positions': position,
                 'direct': {'counter_deltas': {key: 2}},
                 'cumulative': {'counter_deltas': {key: 2 * position}}}
                for position in range(1, bound + 1)
            ],
            'execute_counter_registry': registry,
        }

    def compare_reports(self):
        for (bound, kind), report in self.reports.items():
            path = self.root / f'gate-{kind}-counters-b{bound}'
            path.mkdir(exist_ok=True)
            (path / 'stdout.json').write_text(json.dumps(report))
        return compare(self.root)

    def test_complete_reports_compare_all_prefixes(self):
        result = self.compare_reports()
        self.assertTrue(result['ok'])
        self.assertEqual(
            [(row['bound'], row['counter_equalities']) for row in result['results']],
            [(1, 11), (8, 25)],
        )

    def test_rejects_truncated_prefixes_even_when_lengths_agree(self):
        for kind in ('compact', 'original'):
            self.reports[8, kind]['prefixes'].pop()
            with self.subTest(truncated_through=kind), self.assertRaises(ValueError):
                self.compare_reports()

    def test_rejects_reordered_prefixes_even_when_orders_agree(self):
        for kind in ('original', 'compact'):
            prefixes = self.reports[8, kind]['prefixes']
            prefixes[0], prefixes[1] = prefixes[1], prefixes[0]
        with self.assertRaises(ValueError):
            self.compare_reports()

    def test_rejects_missing_duplicate_and_invalid_positions(self):
        prefix = self.reports[8, 'original']['prefixes'][1]
        for position in (None, 1, 0, 9, 2.0):
            with self.subTest(position=position):
                if position is None:
                    del prefix['position']
                else:
                    prefix['position'] = position
                with self.assertRaises(ValueError):
                    self.compare_reports()

    def test_rejects_wrong_report_bound_even_when_declarations_agree(self):
        for kind in ('original', 'compact'):
            self.reports[8, kind] = self.report(1, kind)
        with self.assertRaises(ValueError):
            self.compare_reports()

    def test_rejects_report_bound_disagreement(self):
        self.reports[8, 'original']['bound'] = 1
        with self.assertRaises(ValueError):
            self.compare_reports()

    def test_rejects_missing_report_bound(self):
        report = self.reports[8, 'original']
        del report['bound']
        with self.assertRaises(ValueError):
            self.compare_reports()

    def test_rejects_non_integer_report_bound(self):
        for kind in ('original', 'compact'):
            self.reports[1, kind]['bound'] = True
        with self.assertRaises(ValueError):
            self.compare_reports()

    def test_rejects_wrong_prefix_declaration_bound(self):
        self.reports[8, 'compact']['prefixes'][3]['declaration_bound'] = 1
        with self.assertRaises(ValueError):
            self.compare_reports()

    def test_rejects_wrong_retained_position_count(self):
        self.reports[8, 'compact']['prefixes'][3]['retained_positions'] = 3
        with self.assertRaises(ValueError):
            self.compare_reports()

    def test_rejects_disabled_prefix_proofs(self):
        self.reports[1, 'original']['prefix_enabled'] = False
        with self.assertRaises(ValueError):
            self.compare_reports()

    def test_rejects_missing_prefix_proofs(self):
        del self.reports[1, 'original']['prefixes']
        with self.assertRaises(ValueError):
            self.compare_reports()

    def test_still_rejects_mismatched_common_prefix_counters(self):
        prefix = self.reports[8, 'compact']['prefixes'][-1]
        for field, actual in (('direct', 2), ('cumulative', 16)):
            with self.subTest(field=field):
                deltas = prefix[field]['counter_deltas']
                deltas['sql.scope_total.read.executions'] = actual + 1
                with self.assertRaisesRegex(ValueError, 'counter mismatch'):
                    self.compare_reports()
                deltas['sql.scope_total.read.executions'] = actual

    def test_still_rejects_wrong_execute_partition(self):
        registry = self.reports[8, 'compact']['execute_counter_registry']
        registry['sql.scope_overflow.executions'] = 9
        with self.assertRaisesRegex(ValueError, 'partition mismatch'):
            self.compare_reports()


if __name__ == '__main__':
    unittest.main()
