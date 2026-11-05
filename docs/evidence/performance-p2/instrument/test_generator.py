"""Fail-closed source-anchor proofs; no builds/workloads or input mutation.

Run: P2_SQLITE_SOURCE=/absolute/original/sqlite python3 -m unittest discover \
  -s docs/evidence/performance-p2/instrument -p 'test_*.py' -v
"""
import ast
import os
from pathlib import Path
import re
import unittest

HERE=Path(__file__).resolve().parent


def definitions():
    tree=ast.parse((HERE/'generate.py').read_text())
    keep=[n for n in tree.body if isinstance(n,ast.FunctionDef) and n.name in {'replace','entry','sqlite_transform','sqlite_sql_transform'}]
    namespace={'re':re}
    exec(compile(ast.Module(body=keep,type_ignores=[]),'generate.py:definitions','exec'),namespace)
    return namespace


class GeneratorAnchors(unittest.TestCase):
    def test_exact_replacement_only(self):
        replace=definitions()['replace']
        self.assertEqual(replace('x anchor y','anchor','new'),'x new y')
        for source in ['absent','anchor anchor']:
            with self.assertRaisesRegex(RuntimeError,'anchor expected'):replace(source,'anchor','new')

    def test_function_anchor_missing_or_duplicate(self):
        entry=definitions()['entry'];signature='func target(ctx context.Context) error {\n'
        self.assertIn('fn.target.calls',entry(signature,'target'))
        for source in ['',signature+signature]:
            with self.assertRaisesRegex(RuntimeError,'ambiguous/missing'):entry(source,'target')

    def test_sqlite_required_anchor_missing_duplicate(self):
        root=os.environ.get('P2_SQLITE_SOURCE')
        if not root:self.skipTest('set P2_SQLITE_SOURCE to original pinned SQLite directory')
        original=(Path(root)/'sqlite3.go').read_text();transform=definitions()['sqlite_transform']
        expected=transform(original)
        self.assertIn('rc.probeScope',expected)
        self.assertIn('s.execSync(args,scope)',expected)
        anchor='static int\n_sqlite3_reset_clear(sqlite3_stmt* stmt)'
        for bad in [original.replace(anchor,'REMOVED',1),original+'\n'+anchor]:
            with self.assertRaisesRegex(RuntimeError,'anchor expected'):transform(bad)
        sql=(Path(root)/'sqlite3_sql.go').read_text();transform=definitions()['sqlite_sql_transform']
        self.assertIn('request_read_only',transform(sql))
        with self.assertRaisesRegex(RuntimeError,'anchor expected'):transform(sql+sql)


if __name__=='__main__':unittest.main()
