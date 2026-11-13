import importlib.util
import json
import pathlib
import tempfile
import unittest

HERE = pathlib.Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('apparatus', HERE / 'apparatus.py')
a = importlib.util.module_from_spec(spec)
spec.loader.exec_module(a)


class ApparatusTests(unittest.TestCase):
    def valid(self):
        return {'bound': 1, 'invariants_ok': True, 'reductions': 3, 'selected': 1, 'completed': 1,
                'spent': 1, 'protected_approvals': 0, 'contacts': [{}], 'evidence': [{}],
                'prefix_enabled': True, 'prefixes': [{}], 'stages': [{'name': 'execute_window', 'wall_ns': 1}]}

    def test_reject_empty_failed_and_partial_rows(self):
        for row in [{}, None, dict(self.valid(), error='failed'), dict(self.valid(), invariants_ok=False),
                    dict(self.valid(), stages=[]), dict(self.valid(), prefixes=[]), dict(self.valid(), bound=2),
                    dict(self.valid(), protected_approvals=1), dict(self.valid(), contacts=[])]:
            with self.subTest(row=row), self.assertRaises(ValueError):
                a.validate(row)
        a.validate(self.valid())

    def test_reject_wrong_approval_mode(self):
        with self.assertRaises(ValueError):
            a.validate(dict(self.valid(), approval_mode='unknown'))
        with self.assertRaises(ValueError):
            a.validate(dict(self.valid(), approval_mode='targeted'))

    def test_exact_anchor_rejects_missing_or_duplicated(self):
        for source in ['missing', 'anchor anchor']:
            with self.assertRaises(ValueError):
                a.replace(source, 'anchor', 'new')
        self.assertEqual(a.replace('anchor', 'anchor', 'new'), 'new')

    def test_changed_input_is_rejected(self):
        with tempfile.TemporaryDirectory() as d:
            p = pathlib.Path(d)
            (p / 'source.go').write_text('original')
            frozen = a.inventory(p)
            a.check_inventory(p, frozen)
            (p / 'source.go').write_text('changed')
            with self.assertRaises(ValueError):
                a.check_inventory(p, frozen)

    def test_cli_rejects_invalid_mode(self):
        import subprocess
        import sys
        result = subprocess.run([sys.executable, str(HERE / 'apparatus.py'), 'generate', '--kind=invalid'], capture_output=True, text=True)
        self.assertEqual(result.returncode, 2)
        self.assertIn('invalid choice', result.stderr)

    def test_frozen_p2_tools(self):
        a.pinned_inputs()

    def test_baseline_work_anchors_are_exact(self):
        import subprocess
        for file, spec in json.loads((HERE / 'baseline-anchors.json').read_text()).items():
            text = subprocess.check_output(['git', 'show', 'd59eaf2976849e450635d252c7bca62eeccbef2e:internal/executor/'+file], cwd=a.P2, text=True)
            for row in spec['edits']:
                text = a.replace(text, row['old'], row['new'])

    def test_driver_preserves_independent_proof(self):
        import shutil
        with tempfile.TemporaryDirectory() as d:
            driver = pathlib.Path(d)
            for p in (a.P2 / 'driver').glob('*'):
                if p.is_file():
                    shutil.copyfile(p, driver / p.name)
            proof = (driver / 'prefix_proof.go').read_bytes()
            a.patch_driver(driver)
            self.assertEqual(proof, (driver / 'prefix_proof.go').read_bytes())
            text = (driver / 'main.go').read_text()
            self.assertIn('any(store).(continuationBasisReader)', text)
            self.assertIn('r.measure("approval_establishment"', text)
            self.assertIn('cuts.verify(rr, i, true)', text)


if __name__ == '__main__':
    unittest.main()
