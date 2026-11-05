"""Bind the unchanged public route's actual module copies before compilation."""
import hashlib
from pathlib import Path

from recorder import inventory, write

MODULE = 'docs/evidence/bounded-series/performance/module'
BINDINGS = {'external-bind-candidate': ('candidate-module', 'candidate'),
            'external-bind-reader': ('baseline-reader-module', 'baseline')}


class ExternalInputs:
    def __init__(self, recorder):
        self.recorder = recorder
        self.root = recorder.root

    def run(self, name, argv, cwd=None, extra=None, expected=0):
        if name not in BINDINGS:
            return self.recorder.run(name, argv, cwd, extra, expected)
        directory, target = BINDINGS[name]
        module = self.root / 'external' / directory
        origin = self.root / 'candidate' / MODULE
        target = self.root / target
        replacement = 'github.com/JakeFAU/alan=' + str(target)
        if Path(cwd) != module or list(map(str, argv)) != ['go', 'mod', 'edit', '-replace=' + replacement]:
            raise ValueError('unexpected external binding command')
        original, copied = inventory(origin), inventory(module)
        write(self.root / 'external' / (name + '-input.json'), {
            'frozen_origin': str(origin), 'actual_module': str(module),
            'frozen_files': original, 'copied_files_before_binding': copied})
        if copied != original:
            raise ValueError('external copy differs from frozen original before binding')
        result = self.recorder.run(name, argv, cwd, extra, expected)
        # go mod edit adds only this declared replacement to the frozen module.
        expected_mod = ((origin / 'go.mod').read_bytes().rstrip(b'\n') +
                        ('\n\nreplace github.com/JakeFAU/alan => ' + str(target) + '\n').encode())
        expected_files = dict(original)
        expected_files['go.mod'] = {'size': len(expected_mod),
                                    'sha256': hashlib.sha256(expected_mod).hexdigest()}
        bound = inventory(module)
        write(self.root / 'external' / (name + '-binding.json'), {
            'actual_module': str(module), 'replacement_target': str(target),
            'expected_bound_files': expected_files, 'bound_files': bound,
            'only_declared_replacement': bound == expected_files})
        if bound != expected_files:
            raise ValueError('external binding changed more than the declared replacement')
        self.recorder.sources['external-' + directory] = module
        return result
