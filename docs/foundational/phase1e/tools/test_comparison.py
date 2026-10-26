"""Focused positive and negative controls using retained pre-correction evidence."""
import copy
import json
from pathlib import Path
import unittest

from comparison import compare_research, rows

ROOT = Path(__file__).resolve().parents[4]
DOC = ROOT / 'docs/foundational/phase1e'
RETAINED = DOC / 'evidence/tooling-correction'
SOURCE = json.loads((RETAINED / 'before-source-manifest.json').read_text())['source_sha256']


class ComparisonControls(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.baseline = rows(DOC / 'evidence/baseline/final-normal.records.jsonl.gz')
        cls.current = rows(RETAINED / 'final-normal.records.jsonl.gz')
        cls.events = rows(RETAINED / 'final-normal.go-test.jsonl.gz')

    def compare(self, current=None, events=None, mode='normal'):
        return compare_research(self.baseline, self.current if current is None else current,
                                ROOT, SOURCE, mode, self.events if events is None else events)

    def record(self, records, marker, index=0):
        return [r for r in records if r['Marker'] == marker][index]

    def test_retained_normal_and_race_accept_documented_provenance(self):
        for mode in ('normal', 'race'):
            result = self.compare(rows(RETAINED / f'final-{mode}.records.jsonl.gz'),
                                  rows(RETAINED / f'final-{mode}.go-test.jsonl.gz'), mode)
            self.assertEqual([r['ExactPayloads'] for r in result], [15, 0, 0, 0])
            self.assertTrue(result[-1]['PublicCheckpointProjectionEqual'])

    def test_reject_behavior_mutations(self):
        for marker, field, value in [('QUALIFICATION', 'States', -1),
                                     ('COMPOSITION', 'Transitions', -1),
                                     ('COMPOSITION', 'Alphabet', []),
                                     ('SUPERVISION', 'Trace', ['invented counterexample']),
                                     ('SUPERVISION', 'Violation', 'changed control finding')]:
            with self.subTest(marker=marker, field=field):
                changed = copy.deepcopy(self.current)
                self.record(changed, marker, 0 if marker == 'QUALIFICATION' else 1)['Evidence'][field] = value
                with self.assertRaises(ValueError):
                    self.compare(changed)

    def test_reject_reordering_and_missing_record(self):
        for remove in (False, True):
            changed = copy.deepcopy(self.current)
            indexes = [i for i, r in enumerate(changed) if r['Marker'] == 'COMPOSITION']
            if remove:
                del changed[indexes[1]]
            else:
                changed[indexes[1]], changed[indexes[2]] = changed[indexes[2]], changed[indexes[1]]
            with self.assertRaises(ValueError):
                self.compare(changed)

    def test_reject_public_projection_even_if_actual_and_expected_both_changed(self):
        changed = copy.deepcopy(self.current)
        checkpoint = self.record(changed, 'CONFORMANCE')['Evidence']['checkpoints'][0]
        checkpoint['actual']['Expected'] += 1
        checkpoint['expected']['Expected'] += 1
        with self.assertRaises(ValueError):
            self.compare(changed)

    def test_reject_model_stream_and_missing_projection_field(self):
        for model in (False, True):
            changed = copy.deepcopy(self.current)
            record = self.record(changed, 'CONFORMANCE')['Evidence']
            if model:
                record['model_trace'].reverse()
            else:
                del record['checkpoints'][0]['assertions']
            with self.assertRaises(ValueError):
                self.compare(changed)

    def test_reject_source_identity_and_manifest_inconsistency(self):
        for target in ('wrapper', 'record', 'manifest', 'file'):
            changed = copy.deepcopy(self.current)
            if target == 'wrapper':
                self.record(changed, 'QUALIFICATION')['SourceID'] = 'wrong'
            elif target == 'record':
                self.record(changed, 'COMPOSITION', 1)['Evidence']['SourceID'] = 'wrong'
            elif target == 'manifest':
                self.record(changed, 'COMPOSITION')['Evidence']['Manifest']['SHA256'] = 'wrong'
            else:
                self.record(changed, 'COMPOSITION')['Evidence']['Manifest']['Files'][0]['SHA256'] = 'wrong'
            with self.subTest(target=target), self.assertRaises(ValueError):
                self.compare(changed)

    def test_reject_missing_go_pass_event(self):
        with self.assertRaises(ValueError):
            self.compare(events=[])


if __name__ == '__main__':
    unittest.main(verbosity=2)
