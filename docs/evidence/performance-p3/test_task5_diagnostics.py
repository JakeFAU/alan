"""Adversarial checks of the compact Task5 diagnostic summary, not new workloads."""
import copy
import json
import os
from pathlib import Path
import unittest
from extract_task5 import readiness
from control_overlay import replace_once


class DiagnosticSummaryTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.result=json.loads(Path(os.environ['P3_TASK5_RESULTS']).read_text())

    def test_all_seven_mechanisms_and_negative_distributions(self):
        self.assertEqual(set(self.result['mechanism']),{'1','8','16','32','64','128','256'})
        for bound,row in self.result['mechanism'].items():
            for variant,source in row['variants'].items():
                with self.subTest(bound=bound,variant=variant):
                    counters=dict(source['counters'],**source['readiness']['distribution'])
                    counters.update({'p3.program_id.'+key+'.visits':value for key,value in source['readiness']['program_id_visits'].items()})
                    self.assertEqual(readiness(counters),source['readiness'])
                    for key in ('p3.calls','p3.eligible_visits','p3.distinct_programs','p3.reusable_visits'):
                        bad=copy.deepcopy(counters);bad[key]+=1
                        with self.assertRaises(ValueError):readiness(bad)
                    bad=copy.deepcopy(counters);bad['p3.load.failure']=1
                    with self.assertRaises(ValueError):readiness(bad)
                    bad=copy.deepcopy(counters);key=next(k for k in bad if k.startswith('p3.program_id.'));bad[key]+=1
                    with self.assertRaises(ValueError):readiness(bad)
                    bad=copy.deepcopy(counters);bad['p3.distribution.invalid']=1
                    with self.assertRaises(ValueError):readiness(bad)

    def test_all_work_counters_preserved(self):
        for row in self.result['mechanism'].values():
            before,after=[row['variants'][v]['counters'] for v in ('A','B')]
            self.assertEqual({k:v for k,v in before.items() if k.startswith('work.')},
                             {k:v for k,v in after.items() if k.startswith('work.')})
            self.assertEqual(row['eliminated_sql'],2*row['eliminated_loads'])

    def test_all_primary_rows_preserved(self):
        design=json.loads((Path(__file__).resolve().parent/'schedule.json').read_text())['rows']
        rows=self.result['all_98_rows']
        self.assertEqual([r['name'] for r in rows],[r['name'] for r in design])
        self.assertEqual(len(rows),98)
        self.assertEqual(sum(bool(r['excluded']) for r in rows),14)
        self.assertTrue(all(r['exit']==0 and len(r['report_sha256'])==64 for r in rows))
        for key in ('primary_timing_gate','primary_allocation_lower'):self.assertTrue(self.result['primary'][key])
        for key in ('guard_violation','rss_resolution_required'):self.assertFalse(self.result['primary'][key])

    def test_control_protocol_and_actual_visits(self):
        self.assertEqual(set(self.result['controls']),{'shared_waiting','distinct_waiting','shared_late_ready'})
        for name,control in self.result['controls'].items():
            for variant in ('baseline','candidate'):
                self.assertEqual(len(control['samples'][variant]),6)
                self.assertTrue(all(s['iterations']==20 and s['excluded_calls']==1 for s in control['samples'][variant]))
                counts=control['counter_evidence'][variant]
                ids={k:v for k,v in counts.items() if k.startswith('id:')}
                self.assertEqual(sum(ids.values()),32)
                self.assertEqual(len(ids),32 if name=='distinct_waiting' else 4)
                self.assertEqual(counts['bindings'],32)
                self.assertEqual(counts.get('reductions',0),int(name=='shared_late_ready'))

    def test_overlay_anchor_rejects_missing_or_duplicate(self):
        self.assertEqual(replace_once('abc','b','x'),'axc')
        for text in ('ac','abbc'):
            with self.assertRaises(ValueError):replace_once(text,'b','x')


if __name__=='__main__':unittest.main(verbosity=2)
