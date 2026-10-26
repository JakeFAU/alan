"""Evaluation output must preserve rejection and unavailable provider accounting."""
import json
from pathlib import Path
import tempfile
import unittest
import normalize

ROOT=Path(__file__).resolve().parents[4]

class Normalization(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory();self.addCleanup(self.tmp.cleanup)
        self.path=Path(self.tmp.name)
        self.view=json.loads((ROOT/'docs/foundational/phase1f/evidence/live-phase1e/inspect.json').read_text())
        self.meta={'system':'baseline','case_id':'test','source':'retained-real-input'}
        (self.path/'summary.json').write_text(json.dumps(dict(elapsed_seconds=1,estimated_token_cost_usd=.003452,unresolved_cost_reservations=0,store_sha256='fixture',replay_pass=True)))
    def result(self):
        (self.path/'inspect.json').write_text(json.dumps(self.view));(self.path/'init-command.json').write_text(json.dumps(self.meta))
        return normalize.normalize(self.path)
    def test_rejected_diagnostic_is_not_delivered(self):
        self.meta['system']='investigation'
        self.view['Result']={'answer':{'Kind':'string','Text':''},'disposition':{'Kind':'string','Text':'stopped'},'final_candidate':{'Kind':'string','Text':'Unreviewed unsupported content.'}}
        result=self.result()
        self.assertEqual(result['final']['answer'],'')
        self.assertEqual(result['diagnostic_final_candidate'],'Unreviewed unsupported content.')
    def test_no_returned_model_still_counts_missing_usage(self):
        candidate=next(r['Candidate'] for r in self.view['Results'] if r['Candidate']['ReturnedModel'])
        candidate['ReturnedModel']='';candidate['Usage']=None
        result=self.result()
        self.assertEqual(result['usage']['missing_usage_results'],1)
        self.assertEqual(result['usage']['model_allowed_attempts'],2)
        self.assertEqual(result['usage']['known_results'],1)
    def test_unknown_model_attempt_is_not_zero_usage_result(self):
        candidate=next(r['Candidate'] for r in self.view['Results'] if r['Candidate']['ReturnedModel'])
        self.view['Results']=[r for r in self.view['Results'] if r['Candidate']['ID']!=candidate['ID']]
        self.view['Unknown']=[candidate['ID']];self.view['Result']=None
        result=self.result()
        self.assertEqual(result['execution_outcome'],'unknown')
        self.assertEqual(result['usage']['unknown_model_attempts'],1)
        self.assertEqual(result['usage']['known_results'],1)
        self.assertIsNone(result['final']['answer'])

if __name__=='__main__':unittest.main()
