#!/usr/bin/env python3
"""Metadata-only regression with a synthetic Go command, not runtime qualification."""
from pathlib import Path
import gzip
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest

TOOL = Path(os.environ.get('ALAN_METADATA_TEST_TOOL', str(Path(__file__).with_name('qualify.py')))).resolve()
SOURCE = '0' * 64
EVENT = (json.dumps({'Action': 'pass', 'Package': 'synthetic/consumer',
                    'Test': 'TestFaultyOrdinaryCollectorReachesIndependentMappingMonitor'}) + '\n').encode()


class QualificationMetadataTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix='alan-metadata-test-')
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name).resolve()
        self.doc = self.root / 'docs/foundational/phase1g'
        (self.doc / 'tools').mkdir(parents=True)
        shutil.copyfile(TOOL, self.doc / 'tools/qualify.py')
        (self.doc / 'source-manifest.json').write_text(json.dumps({
            'files': [], 'source_sha256': SOURCE,
            'platform': 'deliberately-different-freeze-host',
            'go_version': 'go version deliberately-different-freeze-toolchain'}))
        self.out = self.root / 'output'
        self.out.mkdir()
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        # The fake executable checks both command invocations receive the same
        # selected environment/cwd. It never compiles or executes Go sources.
        fake = self.bin / 'go'
        fake.write_text('#!' + sys.executable + '\n' + '''import json, os, pathlib, sys
assert os.environ['GOTOOLCHAIN'] == 'local'
assert os.environ['GOFLAGS'] == '-mod=readonly'
assert 'OPENAI_API_KEY' not in os.environ
assert pathlib.Path.cwd() == pathlib.Path(os.environ['EXPECTED_ROOT'])
with open(os.environ['CALLS'], 'a') as f:
    f.write(json.dumps(sys.argv[1:]) + '\\n')
if sys.argv[1:] == ['version']:
    print('go version executing-toolchain synthetic/host')
else:
    assert sys.argv[1:] == ['-C', 'examples/worktree/consumer', 'test', '-json', '-count=1', '-timeout=240s', './...']
    print(json.dumps({'Action': 'pass', 'Package': 'synthetic/consumer', 'Test': 'TestFaultyOrdinaryCollectorReachesIndependentMappingMonitor'}))
''')
        fake.chmod(0o755)
        self.env = dict(os.environ, PATH=str(self.bin) + os.pathsep + os.environ['PATH'],
                        GOTOOLCHAIN='must-be-overridden', GOFLAGS='must-be-overridden',
                        OPENAI_API_KEY='synthetic-not-a-credential', EXPECTED_ROOT=str(self.root),
                        CALLS=str(self.root / 'calls.jsonl'), ALAN_QUALIFICATION_OUT=str(self.out),
                        PYTHONDONTWRITEBYTECODE='1')

    def run_tool(self, parse=False):
        # Parse on a deliberately different host and forbid a version probe.
        host = 'different-parser-host' if parse else 'executing-host'
        if parse:
            self.env['PATH'] = str(self.bin)
            (self.bin / 'go').write_text('#!' + sys.executable + '\nraise AssertionError("parse-existing must not invoke go")\n')
        code = ('import runpy, sys; from unittest.mock import patch; '
                'sys.argv = ' + repr(['qualify.py', 'consumer'] + (['--parse-existing'] if parse else [])) + '; '
                '\nwith patch("platform.platform", return_value=' + repr(host) + '):\n'
                '    runpy.run_path(' + repr(str(self.doc / 'tools/qualify.py')) + ', run_name="__main__")\n')
        proc = subprocess.run([sys.executable, '-c', code], cwd=self.root,
                              env=self.env, capture_output=True)
        evidence = os.environ.get('ALAN_METADATA_TEST_EVIDENCE')
        if evidence:
            dest = Path(evidence) / self._testMethodName
            shutil.copytree(self.root, dest)
            (dest / 'stdout.txt').write_bytes(proc.stdout)
            (dest / 'stderr.txt').write_bytes(proc.stderr)
            (dest / 'result.json').write_text(json.dumps({'ExitCode': proc.returncode,
                'SyntheticCommandOnly': True, 'ParserHost': host}, indent=2) + '\n')
        self.assertEqual(proc.returncode, 0, proc.stderr.decode())
        return json.loads((self.out / 'final-consumer.command.json').read_text())

    def test_fresh_execution_uses_executing_environment(self):
        meta = self.run_tool()
        self.assertEqual(meta['Platform'], 'executing-host')
        self.assertEqual(meta['GoVersion'], 'go version executing-toolchain synthetic/host')
        calls = [json.loads(line) for line in (self.root / 'calls.jsonl').read_text().splitlines()]
        self.assertEqual(calls[0], ['version'])
        self.assertEqual(calls[1], meta['Command'][1:])
        self.assertEqual(meta['Environment']['GOTOOLCHAIN'], 'local')
        self.assertEqual(meta['Environment']['GOFLAGS'], '-mod=readonly')
        self.assertTrue(meta['CredentialAbsent'])

    def test_parse_existing_preserves_recorded_execution_environment(self):
        raw = self.out / 'final-consumer.go-test.jsonl.gz'
        raw.write_bytes(gzip.compress(EVENT, mtime=0))
        original = dict(SourceID=SOURCE, StartedUTC='original-start', ElapsedSeconds=123.5,
                        ExitCode=0, Stderr='', Command=['original', 'command'],
                        Directory='/original/execution/directory', Platform='original-executing-host',
                        GoVersion='go version original-executing-toolchain', CredentialAbsent=True,
                        Environment={key: 'original-' + key for key in
                            ('GOTOOLCHAIN', 'GOFLAGS', 'GOCACHE', 'GOPROXY', 'GOSUMDB')},
                        RawSHA256=hashlib.sha256(EVENT).hexdigest())
        (self.out / 'final-consumer.command.json').write_text(json.dumps(original))
        meta = self.run_tool(parse=True)
        for key in ('Platform', 'GoVersion', 'Directory', 'Command', 'Environment',
                    'StartedUTC', 'ElapsedSeconds', 'CredentialAbsent', 'RawSHA256'):
            self.assertEqual(meta[key], original[key], key)
        self.assertEqual(gzip.decompress(raw.read_bytes()), EVENT)


if __name__ == '__main__':
    unittest.main()
