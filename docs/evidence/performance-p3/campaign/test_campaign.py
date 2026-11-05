"""Adversarial tooling tests; real tiny driver fixtures are supplied explicitly."""
import copy
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from analyze import analyze, compare, guard, improvement_above_five_percent
from custody import bound_row, expected_custody, fingerprint, sha, validate_plan
from run import run_one, validate_meta, validate_records
from validation import validate_report


class DirectProofTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.binding=json.loads(Path(os.environ['P3_TOOLING_BINDING']).read_text())
        cls.fixture=json.loads(Path(os.environ['P3_TOOLING_FIXTURE']).read_text())
        cls.row={'name':cls.fixture['sample'],'bound':cls.fixture['bound'],'source':cls.fixture['source_label'],
                 'expected_runtime':cls.binding['runtime']}

    def test_frozen_plan(self):
        validate_plan(self.binding)

    def test_real_direct_fixture(self):
        validate_report(self.fixture,self.row)

    def test_metadata_negatives(self):
        path=Path(os.environ['P3_TOOLING_FIXTURE'])
        plan_path=Path(os.environ['P3_TOOLING_FIXTURE_PLAN'])
        plan=json.loads(plan_path.read_text())
        meta=json.loads(path.with_suffix('.meta.json').read_text())
        row=meta['row']
        def validate(m):
            validate_meta(m,self.fixture,plan,row,sha(plan_path),sha(path),sha(path.with_suffix('.progress')))
        validate(meta)
        start=json.loads(path.with_suffix('.start.json').read_text())
        outcome=json.loads(path.with_suffix('.outcome.json').read_text())
        validate_records(meta,start,outcome)
        for s,o in (({},outcome),(start,{}),(start,dict(outcome,exit=23))):
            with self.assertRaises(ValueError):validate_records(meta,s,o)
        for key,value in {'exit':23,'validation_error':'failed','binding_sha256':'wrong','row':{},
                          'report_sha256':'wrong','progress_sha256':'wrong','custody_before':'wrong',
                          'custody_after':'wrong','command':[],'cwd':'/wrong','environment':{},
                          'reported_configuration':{},'process_peak_rss_unit':'KiB', 'load_after':[],
                          'seconds':float('nan')}.items():
            with self.subTest(key=key):
                bad=copy.deepcopy(meta);bad[key]=value
                with self.assertRaises(ValueError):validate(bad)

    def test_missing_or_extra_campaign_rows_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            plan=copy.deepcopy(self.binding);plan['output']=tmp
            with patch('analyze.validate_plan'), patch('analyze.snapshot',return_value=expected_custody(plan)):
                with self.assertRaisesRegex(ValueError,'missing/extra'):analyze(plan,'test')
                (Path(tmp)/'undeclared-row.json').write_text('{}')
                with self.assertRaisesRegex(ValueError,'missing/extra'):analyze(plan,'test')

    def test_changed_custody_prevents_launch(self):
        for kind in ('sources','build_inputs','binaries','tools','dependencies','go','python'):
            with self.subTest(kind=kind), tempfile.TemporaryDirectory() as tmp:
                plan=copy.deepcopy(self.binding);plan['output']=tmp
                row=bound_row(plan,plan['design']['rows'][0])
                bad=copy.deepcopy(expected_custody(plan));bad[kind]={}
                with patch('run.os.fork',side_effect=AssertionError('must not launch')):
                    with self.assertRaisesRegex(ValueError,'custody differs'):run_one(plan,row,'test',custody_reader=lambda p:bad)
                self.assertEqual(list(Path(tmp).iterdir()),[])

    def test_negative_direct_proofs(self):
        cases={
            'protocol':lambda r:r['transport'].__setitem__('protocol','HTTP'),
            'endpoint':lambda r:r['transport'].__setitem__('endpoint','localhost'),
            'journal':lambda r:r['server_contacts'].append(r['contacts'][0]),
            'capability':lambda r:r['contacts'][0].__setitem__('capability','wrong'),
            'effect':lambda r:r['contacts'][0].__setitem__('effect','wrong'),
            'contact_protocol':lambda r:r['contacts'][0].__setitem__('protocol','wrong'),
            'contact_endpoint':lambda r:r['contacts'][0].__setitem__('endpoint','wrong'),
            'contact_key':lambda r:r['contacts'][0].__setitem__('key','wrong'),
            'payload':lambda r:r['evidence'][0]['prepared']['Request']['Payload']['station'].__setitem__('Text','wrong'),
            'prepared_digest':lambda r:r['evidence'][0]['prepared'].__setitem__('Digest','0'*64),
            'result_digest':lambda r:r['evidence'][0].__setitem__('result_digest','0'*64),
            'http_result':lambda r:r['evidence'][0]['candidate'].__setitem__('HTTPStatus',200),
            'response':lambda r:r['evidence'][0]['candidate'].__setitem__('Response','fictional HTTP'),
            'approval_scope':lambda r:r['evidence'][1]['approval']['Approval'].__setitem__('ScopeDigest','0'*64),
            'approval':lambda r:r['evidence'][1]['approval']['Approval'].__setitem__('PredecessorID','wrong'),
            'drop_prefix':lambda r:r['prefixes'].pop(),
            'duplicate_prefix':lambda r:r['prefixes'].__setitem__(1,r['prefixes'][0]),
            'reorder_prefix':lambda r:r['prefixes'].reverse(),
            'prefix_http':lambda r:r['prefixes'][0].__setitem__('server_contacts',1),
            'wrong_counts':lambda r:r.__setitem__('reductions',0),
            'missing_phase':lambda r:r['stages'].pop(),
            'profile':lambda r:r.__setitem__('profiled',True),
            'instrument':lambda r:r.__setitem__('instrumented',True),
            'counters':lambda r:r['stages'][0].__setitem__('counter_deltas',{'hidden':1}),
            'prefix_math':lambda r:r['prefixes'][0]['direct'].__setitem__('mallocs',0),
            'outer_label_rewrite':lambda r:r.__setitem__('resource_scope','process-no-server'),
        }
        for name,mutate in cases.items():
            with self.subTest(name=name):
                r=copy.deepcopy(self.fixture);mutate(r)
                with self.assertRaises((ValueError,KeyError)):validate_report(r,self.row)

    def test_schedule_and_identity_negatives(self):
        for name,mutate in {
            'missing':lambda p:p['design']['rows'].pop(),
            'duplicate':lambda p:p['design']['rows'].__setitem__(3,p['design']['rows'][2]),
            'reorder':lambda p:p['design']['rows'].reverse(),
            'GOSUMDB':lambda p:p['environment'].__setitem__('GOSUMDB','on'),
            'source':lambda p:p['variants']['B'].__setitem__('commit','wrong'),
            'tools':lambda p:p['tool_hashes'].__setitem__(next(iter(p['tool_hashes'])),'0'*64),
            'profile':lambda p:p['runtime'].__setitem__('profiled',True),
        }.items():
            with self.subTest(name=name):
                plan=copy.deepcopy(self.binding);mutate(plan)
                with self.assertRaises(ValueError):validate_plan(plan)

    def test_failure_survives_postexit_custody_error(self):
        with tempfile.TemporaryDirectory() as tmp:
            plan=copy.deepcopy(self.binding);plan['cwd']=tmp;plan['output']=tmp
            binary=Path(tmp)/'fail.sh';binary.write_text('#!/bin/sh\nexit 23\n');binary.chmod(0o755)
            plan['variants']['A']['binary']=str(binary)
            row=bound_row(plan,plan['design']['rows'][0]);row['name']='tooling-failed-child'
            expected=expected_custody(plan)
            with patch('run.snapshot',side_effect=[expected,OSError('postexit source disappeared')]):
                with self.assertRaises(ValueError):run_one(plan,row,'tooling',custody_reader=__import__('run').snapshot)
            outcome=json.loads((Path(tmp)/(row['name']+'.outcome.json')).read_text())
            meta=json.loads((Path(tmp)/(row['name']+'.meta.json')).read_text())
            self.assertEqual(outcome['exit'],23);self.assertEqual(meta['exit'],23)
            self.assertIn('postexit source disappeared',meta['validation_error'])

    def test_guard_boundaries_and_median_interpretation(self):
        self.assertFalse(improvement_above_five_percent([100]*6,[95]*6))
        self.assertTrue(improvement_above_five_percent([100]*6,[94]*6))
        self.assertFalse(guard(.001,.002,.001));self.assertTrue(guard(.001,.00201,.001))
        self.assertFalse(guard(1,1.099,.001));self.assertTrue(guard(1,1.101,.001))
        result=compare([1,2,3,40,50,60],[.5,1,2,30,31,32])
        self.assertEqual(result['median_reduction'],1-16/21.5)
        self.assertNotEqual(result['median_reduction'],1-result['paired_candidate_over_baseline']['median'])
        self.assertTrue(result['all_six_positive'])
        self.assertFalse(compare([1]*6,[1,.9,.9,.9,.9,.9])['all_six_positive'])


if __name__=='__main__':unittest.main(verbosity=2)
