"""Discriminate copied-input drift, incorrect binding and real build-time writes."""
import json
from pathlib import Path
import shutil
import sys
import tempfile
import unittest

from external_inputs import ExternalInputs, MODULE
from recorder import Recorder


class ExternalInputTests(unittest.TestCase):
    def fixture(self, root):
        origin = root / 'candidate' / MODULE
        origin.mkdir(parents=True)
        (origin / 'go.mod').write_text('module example.org/qualification-input\n\ngo 1.27.3\n')
        (origin / 'input.go').write_text('package input\n')
        module = root / 'external/candidate-module'
        shutil.copytree(origin, module)
        recorder = Recorder(root)
        adapter = ExternalInputs(recorder)
        argv = ['go', 'mod', 'edit', '-replace=github.com/JakeFAU/alan=' + str(root / 'candidate')]
        return origin, module, recorder, adapter, argv

    def test_bound_copy_is_registered_and_real_child_mutation_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            origin, module, recorder, adapter, argv = self.fixture(root)
            adapter.run('external-bind-candidate', argv, module)
            self.assertEqual(recorder.sources['external-candidate-module'], module)
            proof = json.loads((root / 'external/external-bind-candidate-binding.json').read_text())
            self.assertTrue(proof['only_declared_replacement'])
            self.assertEqual((origin / 'input.go').read_text(), 'package input\n')
            # This real child stands in for a compiler mutating an input while it runs.
            with self.assertRaisesRegex(ValueError, 'source/tool bytes changed'):
                adapter.run('mutating-build', [sys.executable, '-c',
                    'from pathlib import Path; import sys; Path(sys.argv[1]).write_text("changed")',
                    module / 'input.go'], module)
            gate = root / 'gates/mutating-build'
            self.assertEqual(json.loads((gate / 'outcome.json').read_text())['exit'], 0)
            self.assertTrue((gate / 'postmetadata-error.json').exists())

    def test_non_module_file_drift_before_binding_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            _, module, recorder, adapter, argv = self.fixture(root)
            (module / 'input.go').write_text('changed before binding\n')
            with self.assertRaisesRegex(ValueError, 'differs from frozen original'):
                adapter.run('external-bind-candidate', argv, module)
            self.assertFalse((root / 'gates/external-bind-candidate').exists())
            self.assertEqual(recorder.sources, {})

    def test_wrong_replace_is_rejected_before_execution(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            _, module, recorder, adapter, argv = self.fixture(root)
            argv[-1] = '-replace=github.com/JakeFAU/alan=' + str(root / 'wrong')
            with self.assertRaisesRegex(ValueError, 'unexpected external binding command'):
                adapter.run('external-bind-candidate', argv, module)
            self.assertFalse((root / 'gates/external-bind-candidate').exists())
            self.assertEqual(recorder.sources, {})


if __name__ == '__main__':
    unittest.main()
