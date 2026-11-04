"""Small cross-language golden tests. Full finite-report mutants are prove_comparison.py."""
import copy
import json
from pathlib import Path
import unittest

from finite_validation import fixed_exchange, go_json, retained_schema


class GoSerializationGolden(unittest.TestCase):
    def test_go_html_and_line_separator_escaping(self):
        self.assertEqual(go_json({'value': '<>&\u2028\u2029'}),
                         '{"value":"\\u003c\\u003e\\u0026\\u2028\\u2029"}')

    def test_retained_schema_go_hashes(self):
        self.assertEqual(retained_schema('station')['ID'], '4e68aa32fde1981a6bf248e478598dc9bf3d9a010e33a717874e7775661cb822')
        self.assertEqual(retained_schema('sample')['ID'], '5169443b0d17ff6cf587f0c298d3fa4c4ee4ad8754641292be3fb435badfb204')

    def test_actual_go_exchange_and_digest(self):
        contact = json.loads(Path(__file__).with_name('fixtures').joinpath('go-http-contact.json').read_text())
        namespace = 'e0d2f3ce6561f81d713f659724e96ebd2c962e7a05562c0d002ce5f48273e591'
        prepared, result = fixed_exchange(contact, namespace, 'http://127.0.0.1:64117/measure', 1)
        self.assertEqual(prepared['Digest'], 'ed5059af6f3db5e24fdb5cc91266eff09668ba475de6e77fdb6bbda7a0f65886')
        self.assertEqual(result['Digest'], 'cc70894bda5866fddf7cdff1ee1a2b6e4ffcfd587cce589268c2503b71ede8b2')
        changed = copy.deepcopy(contact)
        changed['request_wire'] = ' '+changed['request_wire']
        with self.assertRaisesRegex(ValueError, 'canonical HTTP'):
            fixed_exchange(changed, namespace, 'http://127.0.0.1:64117/measure', 1)


if __name__ == '__main__':
    unittest.main()
