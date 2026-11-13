"""Historical owner control against a genuine retained K01 capture."""
from copy import deepcopy
import os
from pathlib import Path
import unittest

from custody import read
from qualification import verify_k01_owner


class HistoricalOwnerTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        value = os.environ.get('ALAN_TASK5_K01_ROOT')
        if not value:
            raise unittest.SkipTest('supply the retained excluded historical-control root')
        root = Path(value)
        cls.known = read(root / 'gates/owner-supported-format-inspect/stdout.log')
        owner = root / 'historical-controls/owner-positive'
        cls.opened = read(owner / 'open.json')
        cls.run_result = read(owner / 'run.json')

    def test_accepts_quiescent_owner_with_original_open_run(self):
        verify_k01_owner(self.known, self.opened, self.run_result)

    def test_rejects_mismatched_original_state(self):
        changes = (
            ('namespace', lambda k, o, r: o.update(Namespace='different-store')),
            ('missing run', lambda k, o, r: o['Runs'].pop('b')),
            ('generation', lambda k, o, r: o['Runs']['b']['ID'].update(Generation=2)),
            ('sequence', lambda k, o, r: o['Runs']['a'].update(Sequence=0)),
            ('renewal', lambda k, o, r: o['Runs']['b'].update(Epoch=o['Runs']['b']['Epoch'] - 1)),
            ('projection', lambda k, o, r: o['Runs']['a']['Projection'].update(Data={})),
            ('independent reference', lambda k, o, r: k['Tables']['runs'][0].update(projection='{"Status":"open"}')),
        )
        self.reject_mutations(changes)

    def test_rejects_wrong_open_runs_and_work(self):
        changes = (
            ('globally complete', lambda k, o, r: r['Report'].update(OpenRuns=[])),
            ('wrong open run', lambda k, o, r: r['Report'].update(OpenRuns=[o['Runs']['a']['ID']])),
            ('duplicate open run', lambda k, o, r: r['Report']['OpenRuns'].append(o['Runs']['b']['ID'])),
            ('not quiescent', lambda k, o, r: r['Report'].update(Quiescent=False)),
            ('new step', lambda k, o, r: r['Report'].update(Steps=1)),
            ('new dispatch', lambda k, o, r: r['Report'].update(Dispatches=1)),
            ('missing roots', lambda k, o, r: r['Report'].update(Roots=[])),
            ('run error', lambda k, o, r: r.update(Error='failed')),
        )
        self.reject_mutations(changes)

    def reject_mutations(self, changes):
        for label, change in changes:
            with self.subTest(label=label):
                known, opened, run = deepcopy((self.known, self.opened, self.run_result))
                change(known, opened, run)
                with self.assertRaises(ValueError):
                    verify_k01_owner(known, opened, run)
