import json
import os
import sys
import shutil
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from common import Recorder, copy_db, members, write
from cross_build import load_original


class Custody(unittest.TestCase):
    def test_complete_copy_and_refuse_overwrite(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            source, target = root / 'source', root / 'copy'
            for suffix in ('', '-wal', '-shm'):
                Path(str(source) + suffix).write_bytes(('original' + suffix).encode())
            copy_db(source, target, root / 'proof.json')
            self.assertEqual(members(source), members(target))
            self.assertTrue(json.loads((root / 'proof.json').read_text())['equal'])
            with self.assertRaises(FileExistsError):
                copy_db(source, target, root / 'must-not-exist.json')
            self.assertFalse((root / 'must-not-exist.json').exists())

    def test_failed_child_is_retained_and_cannot_be_overwritten(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            write(root / 'route-manifest.json', {'environment': {'PATH': os.environ['PATH']}})
            recorder = Recorder(root)
            with self.assertRaisesRegex(RuntimeError, 'stopped at child'):
                recorder.run('child', [sys.executable, '-c', 'print("retained"); raise SystemExit(7)'])
            gate = root / 'gates/child'
            self.assertEqual(json.loads((gate / 'exit.json').read_text())['exit'], 7)
            self.assertEqual((gate / 'stdout.log').read_text(), 'retained\n')
            with self.assertRaises(FileExistsError):
                recorder.run('child', [sys.executable, '-c', 'raise SystemExit(0)'])
            self.assertEqual(json.loads((gate / 'exit.json').read_text())['exit'], 7)

    def test_original_controller_import_creates_no_bytecode(self):
        previous = sys.dont_write_bytecode
        try:
            with tempfile.TemporaryDirectory() as d:
                root = Path(d)
                source = root / 'original.py'
                source.write_text('VALUE = 73\n')
                # Explicitly permit caches initially, so only the wrapper's guard
                # can prevent this real import from changing the source tree.
                sys.dont_write_bytecode = False
                module = load_original(source)
                self.assertEqual(module.VALUE, 73)
                self.assertEqual(list(root.iterdir()), [source])
                self.assertEqual(source.read_text(), 'VALUE = 73\n')
        finally:
            sys.dont_write_bytecode = previous

    def test_detect_source_change(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            (root / 'source').write_bytes(b'original')
            actual_copy = shutil.copyfileobj
            def mutate_after_copy(src, dst):
                actual_copy(src, dst)
                (root / 'source').write_bytes(b'changed during copy')
            with patch('common.shutil.copyfileobj', side_effect=mutate_after_copy):
                with self.assertRaisesRegex(ValueError, 'changed'):
                    copy_db(root / 'source', root / 'copy', root / 'proof.json')
            self.assertFalse(json.loads((root / 'proof.json').read_text())['equal'])


if __name__ == '__main__':
    unittest.main()
