"""Controls for the P7 source transforms, row binding and decision rule."""
from pathlib import Path
import tempfile
import unittest

import measure as m


class MeasureTests(unittest.TestCase):
    def test_schedule_has_adjacent_balanced_pairs_and_no_duplicate_ids(self):
        rows = m.schedule()
        self.assertEqual(len(rows), 98)
        self.assertEqual(len({r['name'] for r in rows}), 98)
        for i, n in enumerate(m.BOUNDS):
            segment = rows[14*i:14*(i+1)]
            self.assertEqual({r['bound'] for r in segment}, {n})
            self.assertEqual([r['kind'] for r in segment], ['warmup']*2 + ['timing']*12)
            for block in range(1, 7):
                pair = segment[2*block:2*block+2]
                self.assertEqual([r['block'] for r in pair], [block, block])
                self.assertEqual(tuple(r['arm'] for r in pair), m.ARMS[::(1 if block % 2 else -1)])

    def test_reorder_refuses_missing_duplicate_and_nonadjacent_anchors(self):
        changed = m.reorder('before\n' + m.UPDATE + m.INSERT + 'after\n')
        self.assertEqual(changed, 'before\n' + m.INSERT + m.UPDATE + 'after\n')
        for source in (m.UPDATE, m.UPDATE + m.INSERT + m.UPDATE, m.UPDATE + '\n' + m.INSERT):
            with self.subTest(source=source), self.assertRaises(ValueError):
                m.reorder(source)

    def test_timed_derivative_only_wraps_five_actual_calls_in_pinned_generator(self):
        generated = m.timer_generator()
        self.assertEqual(generated.count(r'p7Start := rc.s.p7Start()\n'), 2)
        self.assertEqual(generated.count(r'p7Start := s.p7Start()\n'), 2)
        self.assertIn(r"p7Finish(p7Start)\n s.probeStep(rv,rc.probeScope)',2)", generated)
        original = m.p6.counter_generator()
        with self.assertRaises(ValueError):
            m.timer_generator(original.replace('rv := C._sqlite3_step_row_internal(s.s, &rowid, &changes)', 'wrong anchor'))

    def test_validation_rejects_missing_failed_or_rebound_rows(self):
        design = m.schedule()[:2]
        configs = {arm: dict(label=arm, digest=arm) for arm in m.ARMS}
        rows = [dict(design=r, revision=r['arm'], binary_sha256=r['arm'], mode='progress', exit=0) for r in design]
        m.check_rows(design, rows, configs)
        for bad in (rows[:-1], rows[::-1], [dict(rows[0], revision='other'), rows[1]],
                    [dict(rows[0], failure='preserved failure'), rows[1]]):
            with self.assertRaises(ValueError):
                m.check_rows(design, bad, configs)

    def test_decision_requires_floor_direction_and_saving_above_mad(self):
        self.assertTrue(m.advance([1000]*6, [980, 990, 980, 975, 990, 1010]))
        self.assertFalse(m.advance([1000]*6, [999]*6))
        self.assertFalse(m.advance([1000]*6, [980, 980, 980, 980, 1020, 1020]))
        self.assertFalse(m.advance([1000]*6, [980, 939, 940, 999, 999, 1100]))

    def test_reserve_refuses_other_plan_and_reuse(self):
        with tempfile.TemporaryDirectory() as d:
            path = Path(d)
            child = path / 'campaign'
            self.assertEqual(m.reserve(child, path), child.resolve())
            with self.assertRaises(FileExistsError):
                m.reserve(child, path)
            with self.assertRaises(ValueError):
                m.reserve(path.parent / 'elsewhere', path)


if __name__ == '__main__':
    unittest.main()
