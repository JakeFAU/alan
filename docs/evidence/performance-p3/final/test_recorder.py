"""Actual child/timeout/privacy failures, not mocked outcome assertions."""
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

from recorder import Recorder, environment


class RecorderTests(unittest.TestCase):
    def test_failed_child_status_survives_deleted_executable(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            child = root / 'child'
            child.write_text('#!/bin/sh\nrm -- "$0"\nexit 23\n')
            child.chmod(0o755)
            with self.assertRaises(FileNotFoundError):
                Recorder(root).run('failed', [child], timeout=10)
            gate = root / 'gates/failed'
            self.assertEqual(json.loads((gate / 'outcome.json').read_text())['exit'], 23)
            self.assertTrue((gate / 'postmetadata-error.json').exists())
            with self.assertRaises(FileExistsError):
                Recorder(root).run('failed', ['/bin/true'])

    def test_timeout_is_not_success(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with self.assertRaises(RuntimeError):
                Recorder(root).run('timeout', [sys.executable, '-c', 'import time; time.sleep(30)'], timeout=0.1)
            outcome = json.loads((root / 'gates/timeout/outcome.json').read_text())
            self.assertTrue(outcome['timeout'])
            self.assertEqual(outcome['exit'], -9)

    def test_credentials_are_not_inherited_or_recorded(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            secret = 'synthetic-credential-not-for-recording'
            with patch.dict(os.environ, {'P3_SYNTHETIC_SECRET': secret}):
                env = environment(root)
                path = Recorder(root, env=env).run('privacy', [sys.executable, '-c',
                    'import os; assert "P3_SYNTHETIC_SECRET" not in os.environ'], timeout=10)
            self.assertEqual(path.read_text(), '')
            for file in (root / 'gates/privacy').iterdir():
                self.assertNotIn(secret, file.read_text())


if __name__ == '__main__':
    unittest.main()
