"""Unchanged monitors reject causal-identity and spending faults without live calls."""
import copy
import json
from pathlib import Path
import tempfile
import unittest
import live_control as c

ROOT=Path(__file__).resolve().parents[4]

class Controls(unittest.TestCase):
    def setUp(self):
        self.view=json.loads((ROOT/'docs/foundational/phase1f/evidence/live-phase1e/inspect.json').read_text())
        row=self.view['Snapshot']['Tables']['adapter_attempts'][0]
        self.attempt=json.loads(row['descriptor'])
        intent=self.attempt['Intent']
        auth=next(json.loads(r['result']) for r in self.view['Snapshot']['Tables']['attempts']
                  if r['id']==self.attempt['AuthorizationID'])
        self.event=dict(PID=123,Store='/tmp/test.sqlite',Boundary='adapter.authorized',Phase='pre-transport',
                        Token='test',Intent=intent['ID'],Attempt=auth['ID'],Subject=intent['Run']['Subject'],
                        Generation=intent['Run']['Generation'],Epoch=auth['Source']['Epoch'],Sequence=intent['Sequence'])
    def check(self,event=None):
        return c.validate_boundary(event or self.event,self.view,123,'/tmp/test.sqlite',{'test'})
    def test_retained_boundary(self):
        self.assertEqual(self.check(),self.attempt)
    def test_epoch_substitution(self):
        e=dict(self.event);e['Epoch']+=1
        with self.assertRaisesRegex(ValueError,'epoch/sequence'):self.check(e)
    def test_sequence_substitution(self):
        e=dict(self.event);e['Sequence']+=1
        with self.assertRaisesRegex(ValueError,'sequence'):self.check(e)
    def test_uncommitted_intent(self):
        self.view['Snapshot']['Tables']['adapter_attempts']=[]
        with self.assertRaisesRegex(ValueError,'missing'):self.check()
    def test_repeated_token(self):
        tokens={'test'}
        c.validate_boundary(self.event,self.view,123,'/tmp/test.sqlite',tokens)
        with self.assertRaisesRegex(ValueError,'token'):
            c.validate_boundary(self.event,self.view,123,'/tmp/test.sqlite',tokens)
    def test_reservation_and_rejection(self):
        p=self.attempt['Prepared'];self.assertGreater(c.reservation(p),.05)
        p=copy.deepcopy(p);p['Model']='another-model'
        with self.assertRaisesRegex(ValueError,'model'):c.reservation(p)
    def test_spending_limit_and_duplicate(self):
        run={'operations':[]};ledger={'maximum_total_usd':10,'controlled_runs':[run],
                                   'reservations':[{'unknown_spend_reservation_usd':9.9}]}
        with self.assertRaisesRegex(ValueError,'limit'):c.add_operation(ledger,run,dict(id='one',reserved_usd=.2))
        self.assertEqual(run['operations'],[])
        c.add_operation(ledger,run,dict(id='one',reserved_usd=.05))
        with self.assertRaisesRegex(ValueError,'duplicate'):c.add_operation(ledger,run,dict(id='one',reserved_usd=.01))
    def test_unknown_keeps_reservation(self):
        run={'operations':[dict(id='missing-result',reserved_usd=.2)]}
        c.reconcile(run,self.view)
        self.assertEqual(c.charge({'controlled_runs':[run]}),.2)
        self.assertNotIn('budget_charge_usd',run['operations'][0])
    def test_actual_usage_reconciliation(self):
        run={'operations':[dict(id=self.event['Intent'],reserved_usd=c.reservation(self.attempt['Prepared']))]}
        c.reconcile(run,self.view)
        self.assertEqual(run['operations'][0]['status'],'usage-known')
        self.assertAlmostEqual(run['operations'][0]['estimated_token_cost_usd'],.001686)

class UsageAccounting(unittest.TestCase):
    setUp = Controls.setUp

    def probe(self, usage):
        # The review's original candidate and reservation, on an in-memory copy.
        candidate = dict(ID='probe', ReturnedModel=c.MODEL,
                         Response='{"service_tier":"default"}', Usage=usage,
                         RequestID='original-request', ResponseID='original-response')
        self.view['Snapshot']['Tables']['adapter_results'] = [
            {'descriptor': json.dumps({'Candidate': candidate})}]
        return {'operations': [dict(id='probe', reserved_usd=.2)]}

    def assert_retained(self, run):
        c.reconcile(run, self.view)
        op = run['operations'][0]
        self.assertEqual(c.charge({'controlled_runs': [run]}), .2)
        self.assertEqual(op['status'], 'malformed-usage-reservation-retained')
        self.assertTrue(op['cost_reason'])
        self.assertEqual(op['request_id'], 'original-request')
        self.assertEqual(op['response_id'], 'original-response')
        self.assertEqual(c.unresolved_reservations(run), .2)

    def test_original_review_negative_total(self):
        self.assert_retained(self.probe(dict(InputTokens=0, OutputTokens=0,
                                            TotalTokens=-1, Details={})))

    def test_original_review_inconsistent_total(self):
        self.assert_retained(self.probe(dict(InputTokens=1, OutputTokens=1,
                                            TotalTokens=666, Details={})))

    def test_invalid_token_fields_retain_reservation(self):
        for field in ('InputTokens', 'OutputTokens', 'TotalTokens'):
            for invalid in (True, False, 1.0, 1.5, '1', -1):
                with self.subTest(field=field, value=invalid):
                    usage = dict(InputTokens=1, OutputTokens=1, TotalTokens=2, Details={})
                    usage[field] = invalid
                    self.assert_retained(self.probe(usage))

    def test_zero_billable_usage_retains_reservation(self):
        self.assert_retained(self.probe(dict(InputTokens=0, OutputTokens=0,
                                            TotalTokens=0, Details={})))

    def test_invalid_cache_counts_retain_reservation(self):
        for field in ('cached_tokens', 'cache_write_tokens'):
            for invalid in (True, False, 1.0, 1.5, '1', -1, None, 11):
                with self.subTest(field=field, value=invalid):
                    usage = dict(InputTokens=10, OutputTokens=1, TotalTokens=11,
                                 Details={'input_tokens_details': {field: invalid}})
                    self.assert_retained(self.probe(usage))
        usage = dict(InputTokens=10, OutputTokens=1, TotalTokens=11,
                     Details={'input_tokens_details': {'cached_tokens': 6, 'cache_write_tokens': 5}})
        self.assert_retained(self.probe(usage))

    def test_malformed_accounting_shapes_retain_reservation(self):
        for usage in (True, 5, 'usage', [],
                      dict(InputTokens=1, OutputTokens=1, TotalTokens=2, Details=True),
                      dict(InputTokens=1, OutputTokens=1, TotalTokens=2,
                           Details={'input_tokens_details': 'cache'})):
            with self.subTest(usage=usage):
                self.assert_retained(self.probe(usage))

    def test_absent_accounting_has_reason_and_preserves_ids(self):
        for usage in (None, {}, dict(InputTokens=1, OutputTokens=1)):
            with self.subTest(usage=usage):
                run = self.probe(usage)
                c.reconcile(run, self.view)
                op = run['operations'][0]
                self.assertEqual(c.charge({'controlled_runs': [run]}), .2)
                self.assertEqual(op['status'], 'unknown-cost-reservation-retained')
                self.assertTrue(op['cost_reason'])
                self.assertEqual(op['request_id'], 'original-request')
                self.assertEqual(op['response_id'], 'original-response')

    def test_invalid_accounting_cannot_reuse_previous_discount(self):
        run = self.probe(dict(InputTokens=0, OutputTokens=0, TotalTokens=-1, Details={}))
        run['operations'][0].update(status='usage-known', budget_charge_usd=0,
                                    estimated_token_cost_usd=0)
        self.assert_retained(run)
        self.assertNotIn('estimated_token_cost_usd', run['operations'][0])
        ledger = dict(maximum_total_usd=.3, controlled_runs=[run])
        with self.assertRaisesRegex(ValueError, 'limit'):
            c.add_operation(ledger, run, dict(id='next', reserved_usd=.2))

    def test_valid_cache_usage_cost_and_complete_ids(self):
        run = self.probe(dict(InputTokens=100, OutputTokens=20, TotalTokens=120,
                             Details={'input_tokens_details': {'cached_tokens': 30,
                                                                'cache_write_tokens': 10}}))
        c.reconcile(run, self.view)
        op = run['operations'][0]
        self.assertEqual(op['status'], 'usage-known')
        # 60 ordinary*2 + 30 cached*.1 + 10 written*5 + 20 output*10, per million.
        self.assertEqual(op['estimated_token_cost_usd'], .000373)
        self.assertEqual(op['budget_charge_usd'], .000875)
        self.assertEqual(op['request_id'], 'original-request')
        self.assertEqual(op['response_id'], 'original-response')
        self.assertEqual(c.unresolved_reservations(run), 0)

    def test_summary_counts_all_unresolved_reservations(self):
        run = {'operations': [dict(reserved_usd=.2, status='released'),
                              dict(reserved_usd=.3, status='unknown-cost-reservation-retained'),
                              dict(reserved_usd=.4, status='malformed-usage-reservation-retained'),
                              dict(reserved_usd=.5, status='usage-known'),
                              dict(reserved_usd=0, status='local')]}
        self.assertAlmostEqual(c.unresolved_reservations(run), .9)

    def test_reconciliation_error_leaves_durable_reservation(self):
        run = self.probe(dict(InputTokens=1, OutputTokens=1, TotalTokens=2, Details={}))
        candidate = json.loads(self.view['Snapshot']['Tables']['adapter_results'][0]['descriptor'])
        candidate['Candidate']['ReturnedModel'] = 'unapproved-model'
        self.view['Snapshot']['Tables']['adapter_results'][0]['descriptor'] = json.dumps(candidate)
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'ledger.json'
            c.write_json(path, {'controlled_runs': [run]})
            with self.assertRaisesRegex(ValueError, 'returned model'):
                c.reconcile(run, self.view)
            self.assertEqual(c.charge(json.loads(path.read_text())), .2)
            self.assertEqual(c.charge({'controlled_runs': [run]}), .2)

if __name__=='__main__':unittest.main()
