import copy
import importlib.util
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path
import unittest

HERE = Path(__file__).resolve().parent

def module(name):
    spec = importlib.util.spec_from_file_location(name, HERE/(name+'.py'))
    m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m)
    return m

class ToolingTests(unittest.TestCase):
    def test_anchor_fails_closed(self):
        replace = module('instrument').replace
        self.assertEqual(replace('abc', 'b', 'x'), 'axc')
        for s in ('ac', 'abbc'):
            with self.assertRaises(ValueError): replace(s, 'b', 'x')

    def test_distribution_reconciles_early_and_empty_returns(self):
        c = {'p3.distribution.waiting.visits_0.distinct_0': 1,
             'p3.distribution.selected.visits_3.distinct_1': 2,
             'p3.calls': 3, 'p3.eligible_visits': 6, 'p3.distinct_programs': 2,
             'p3.reusable_visits': 4, 'p3.program_id.example.visits': 6,
             'p3.load.success': 6, 'p3.decode.calls': 6, 'p3.validate.calls': 6,
             'p3.binding.calls': 6, 'work.reduce.selectReady': 2}
        check = module('summarize').counter_check
        self.assertEqual(check(c)['repeat_visits'], 4)
        with self.assertRaises(ValueError): check({})
        for key in ('p3.calls','p3.eligible_visits','p3.distinct_programs','p3.reusable_visits','p3.binding.calls','p3.program_id.example.visits','p3.decode.calls','p3.validate.calls','p3.load.success','work.reduce.selectReady'):
            bad = copy.deepcopy(c); bad[key] += 1
            with self.assertRaises(ValueError): check(bad)
        bad = copy.deepcopy(c); bad['p3.load.failure'] = 1
        with self.assertRaises(ValueError): check(bad)

    def test_failed_child_outcome_survives_missing_binary_without_credentials(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp); binary = root/'child'
            binary.write_text('#!'+sys.executable+'\nfrom pathlib import Path\nimport sys\nprint("{}",flush=True)\nPath(sys.argv[0]).unlink()\nsys.exit(7)\n')
            binary.chmod(0o700)
            schedule = root/'schedule.json'
            schedule.write_text(json.dumps([{'name':'failed','bound':1,'binary':str(binary),'source':'test','args':['-mode=direct']}]))
            env = dict(os.environ, P3_SYNTHETIC_CREDENTIAL='must-not-be-recorded-p3-secret', CODEX_SYNTHETIC_IDENTITY='must-not-be-recorded-p3-identity')
            result = subprocess.run([sys.executable,str(HERE.parent/'performance-p2/run_schedule.py'),str(schedule),str(root/'out')], env=env,capture_output=True,text=True)
            self.assertNotEqual(result.returncode,0)
            raw=(root/'out/failed.meta.json').read_text(); meta=json.loads(raw)
            self.assertEqual(meta['exit'],7)
            self.assertIn('No such file',meta['validation_error'])
            self.assertIn('process_peak_rss',meta)
            self.assertNotIn('must-not-be-recorded',raw)
            self.assertTrue((root/'out/failed.json').exists())

    def test_schedule_is_complete_and_balanced(self):
        rows = json.loads((HERE/'schedule.json').read_text())['rows']
        self.assertEqual(len(rows), 98)
        for bound in (1,8,16,32,64,128,256):
            selected = [r for r in rows if r['bound'] == bound]
            self.assertEqual([r['variant'] for r in selected[:2]], list('AB'))
            self.assertTrue(all(r['excluded']=='warmup' for r in selected[:2]))
            for n, order in enumerate(('AB','BA','AB','BA','AB','BA'), 1):
                pair = [r for r in selected if r.get('pair') == n]
                self.assertEqual(''.join(r['variant'] for r in pair), order)
                self.assertTrue(all(r['excluded'] is False for r in pair))

if __name__ == '__main__': unittest.main()
