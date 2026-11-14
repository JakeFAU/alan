"""Controls for comparing the public weather Inspect and Replay projections."""
import importlib.util
from pathlib import Path
import unittest


# The inherited P3 helpers also include a top-level qualify.py.
spec = importlib.util.spec_from_file_location('progress_qualify', Path(__file__).with_name('qualify.py'))
qualify = importlib.util.module_from_spec(spec)
spec.loader.exec_module(qualify)
compare_weather_readers = qualify.compare_weather_readers


class WeatherReaderTest(unittest.TestCase):
    def test_common_projection_and_distinct_details(self):
        inspect = {'namespace': 'original', 'opportunities_spent': 1, 'reads': [
            {'operation_id': 'original-effect', 'retained_response': 'recorded',
             'approval_observation_id': 'original-approval', 'request_key': 'original-request',
             'result_digest': 'original-result'}],
            'history': [{'sequence': 1}], 'pending': ['unresolved']}
        replay = {'namespace': 'original', 'opportunities_spent': 1, 'reads': [
            {key: value for key, value in inspect['reads'][0].items() if key != 'retained_response'}],
            'replayed_commits': 1}
        compare_weather_readers(inspect, replay)
        with self.assertRaises(ValueError):
            compare_weather_readers(inspect, dict(replay, replayed_commits=2))
        with self.assertRaises(ValueError):
            compare_weather_readers(inspect, dict(replay, opportunities_spent=0))
        for field in ('operation_id', 'approval_observation_id', 'request_key', 'result_digest'):
            with self.subTest(field=field), self.assertRaises(ValueError):
                compare_weather_readers(inspect, dict(replay, reads=[dict(replay['reads'][0], **{field: 'foreign'})]))
        with self.assertRaises(ValueError):
            compare_weather_readers(inspect, dict(replay, reads=inspect['reads']))
