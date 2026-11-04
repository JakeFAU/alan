#!/usr/bin/env python3
"""Small synthetic evidence tests. No Go process, profile, or database is used."""
import copy
import hashlib
import json
from pathlib import Path
import tempfile
import unittest

import summarize_diagnostics as summary
from test_campaign_tooling import CampaignProof


def sql_fixture():
    counters = {}
    for metric in summary.SQL_METRICS:
        counters['sql.total.' + metric] = 3
        counters['sql.template.SELECT a.' + metric] = 1
        counters['sql.template.SELECT b.' + metric] = 2
        counters['sql.scope.Run.class.read_write_capable.post_autocommit.on.template.SELECT a.' + metric] = 1
        counters['sql.scope.unknown.class.read_only_configured.post_autocommit.off.template.SELECT b.' + metric] = 2
        counters['sql.scope_overflow.' + metric] = 0
    for key in summary.ATTRIBUTION_EVENTS:
        counters['sql.attribution.' + key] = 0
    counters['sql.attribution.incomplete_events'] = 0
    counters['sql.attribution.family_count'] = 2
    counters['sql.attribution.cell_count'] = 2
    return counters


class SQLPartitions(unittest.TestCase):
    def test_all_nine_metrics_exact_and_unknown_separate(self):
        result = summary.sql_partition(sql_fixture())
        self.assertEqual(result['totals'], dict.fromkeys(summary.SQL_METRICS, 3))
        self.assertTrue(result['attribution_complete'])
        self.assertFalse(result['known_semantic_scope_attribution_complete'])
        self.assertEqual(result['unknown_scope']['step_calls'], 2)
        self.assertEqual([r['family'] for r in result['top10']], ['SELECT b', 'SELECT a'])
        detailed = summary.sql_partition(sql_fixture(), include_cells=True)
        self.assertEqual(detailed['families'][0]['scope_connection_post_state'][0]['scope'], 'Run')

    def test_per_family_misattribution_rejected_even_when_total_balances(self):
        c = sql_fixture()
        suffix = '.executions'
        a = 'sql.scope.Run.class.read_write_capable.post_autocommit.on.template.SELECT a' + suffix
        b = 'sql.scope.unknown.class.read_only_configured.post_autocommit.off.template.SELECT b' + suffix
        c[a], c[b] = 2, 1
        with self.assertRaisesRegex(ValueError, 'family scoped partition'):
            summary.sql_partition(c)

    def test_each_metric_partition_checked(self):
        for metric in summary.SQL_METRICS:
            c = sql_fixture()
            c['sql.total.' + metric] += 1
            with self.subTest(metric=metric), self.assertRaises(ValueError):
                summary.sql_partition(c)

    def test_negative_missing_and_boolean_rejected(self):
        for mutation in ('negative', 'missing', 'bool'):
            c = sql_fixture()
            key = 'sql.total.executions'
            if mutation == 'missing':
                del c[key]
            else:
                c[key] = -1 if mutation == 'negative' else True
            with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                summary.sql_partition(c)

    def test_invalid_and_overflow_preserved_as_incomplete(self):
        for event in summary.ATTRIBUTION_EVENTS:
            c = sql_fixture()
            c['sql.attribution.' + event] = 1
            c['sql.attribution.incomplete_events'] = 1
            result = summary.sql_partition(c)
            self.assertFalse(result['attribution_complete'])
            self.assertFalse(result['top10_complete'])
        c = sql_fixture()
        c['sql.attribution.cell_overflow_events'] = 1
        c['sql.attribution.incomplete_events'] = 1
        for metric in summary.SQL_METRICS:
            c['sql.scope_overflow.' + metric] = 1
            c['sql.scope.unknown.class.read_only_configured.post_autocommit.off.template.SELECT b.' + metric] -= 1
        self.assertEqual(summary.sql_partition(c)['scope_overflow']['executions'], 1)

    def test_run_wide_overflow_not_erased_by_later_zero_delta(self):
        c = sql_fixture()
        c['sql.attribution.family_overflow_events'] = 1
        c['sql.attribution.incomplete_events'] = 1
        status = summary.run_attribution([c, sql_fixture()])
        self.assertFalse(status['attribution_complete'])
        later = summary.compact_counters(sql_fixture(), 'sql', detail=True, run_status=status)
        self.assertFalse(later['sql']['top10_complete'])

    def test_integer_counts_remain_exact_above_float_range(self):
        c = sql_fixture()
        count = 2**53 + 7
        for metric in summary.SQL_METRICS:
            c['sql.total.' + metric] = count + 2
            c['sql.template.SELECT a.' + metric] = count
            c['sql.scope.Run.class.read_write_capable.post_autocommit.on.template.SELECT a.' + metric] = count
        self.assertEqual(summary.sql_partition(c)['totals']['executions'], count + 2)

    def test_top10_ties_use_family_text(self):
        c = sql_fixture()
        for metric in summary.SQL_METRICS:
            c['sql.total.' + metric] = 2
            c['sql.template.SELECT b.' + metric] = 1
            c['sql.scope.unknown.class.read_only_configured.post_autocommit.off.template.SELECT b.' + metric] = 1
        self.assertEqual([r['family'] for r in summary.sql_partition(c)['top10']], ['SELECT a', 'SELECT b'])

    def test_membership_is_independently_measured_sql(self):
        c = sql_fixture()
        family = summary.MEMBERSHIP_FAMILY
        for metric in summary.SQL_METRICS:
            c['sql.template.' + family + '.' + metric] = c.pop('sql.template.SELECT a.' + metric)
            c['sql.scope.finishRun.class.read_write_capable.post_autocommit.on.template.' + family + '.' + metric] = c.pop(
                'sql.scope.Run.class.read_write_capable.post_autocommit.on.template.SELECT a.' + metric)
        result = summary.sql_partition(c)
        self.assertEqual(result['finishRun_membership_families'][0]['metrics']['executions'], 1)


class DiagnosticEvidence(unittest.TestCase):
    def setUp(self):
        self.proof = CampaignProof(methodName='test_schedule_coverage_and_adjacent_ab_ba')
        self.proof.setUp()
        self.addCleanup(self.proof.doCleanups)
        self.row = copy.deepcopy(self.proof.rows[0])
        self.report = self.proof.report(self.row)
        self.report['maxrss_unit'] = 'bytes'

    def test_parent_and_nested_are_separate_inclusive_groups(self):
        groups = summary.stage_groups(self.report['stages'], 'none')
        self.assertEqual(groups['execute_window']['count'], 1)
        self.assertEqual(groups['run']['count'], 1)
        self.assertTrue(groups['execute_window']['inclusive'])
        self.assertEqual(groups['run']['enclosing_window'], 'execute_window')
        self.assertNotIn('total', groups)

    def test_adjacent_and_cumulative_preserved_and_arithmetic_checked(self):
        row = next(r for r in self.proof.rows if r['bound'] == 8)
        report = self.proof.report(row)
        summary.validate_evidence(report, row)
        result = summary.summarize_report(report, {'process_peak_rss': 999}, row)
        self.assertEqual(result['prefixes'][1]['direct']['resources']['wall_ns'], 1)
        self.assertEqual(result['prefixes'][1]['cumulative']['resources']['wall_ns'], 2)
        self.assertEqual(result['prefixes'][1]['declaration_bound'], 8)
        report['prefixes'][1]['direct']['wall_ns'] = 2
        with self.assertRaisesRegex(ValueError, 'arithmetic'):
            summary.validate_evidence(report, row)

    def test_no_sql_is_absent_measurement_and_coarse_is_separate(self):
        self.assertEqual(summary.counter_mode(self.report), 'none')
        self.assertIsNone(summary.summarize_report(self.report, {}, self.row)['execute']['sql'])
        self.report['stages'][0]['counter_deltas'] = {
            'coarse.Store.Run.calls': 1, 'coarse.Store.Run.inclusive_wall_ns': 5}
        self.report['instrumented'] = True
        self.row['instrumented'] = True
        self.row['expected_runtime']['instrumented'] = True
        self.assertEqual(summary.counter_mode(self.report), 'coarse')
        self.assertEqual(summary.compact_counters(self.report['stages'][0]['counter_deltas'], 'coarse')['coarse']['Store.Run']['calls'], 1)

    def test_report_identity_and_configuration_rejected(self):
        for key, value in [('sample', 'other'), ('go_version', 'go0'), ('reductions', 7), ('instrumented', True)]:
            report = copy.deepcopy(self.report)
            report[key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                summary.validate_evidence(report, self.row)

    def test_prefix_off_keeps_strict_transport_proof(self):
        self.row.pop('expected_runtime')
        self.row['args'] = ['-prefix=false']
        self.report['prefix_enabled'], self.report['prefixes'] = False, []
        self.report['server_contacts'][0]['response_id'] = 'forged-response'
        with self.assertRaises(ValueError):
            summary.validate_evidence(self.report, self.row)

    def test_archive_validation_never_reads_original_executable(self):
        meta = self.proof.write(self.row, self.report)
        Path(self.row['binary']).unlink()
        hashes = {'schedule': summary.file_sha(self.proof.schedule),
                  'report': meta['report_sha256'], 'progress': meta['progress_sha256']}
        summary.validate_record(meta, self.report, self.row, hashes)
        for field in ('schedule_sha256', 'report_sha256', 'progress_sha256', 'binary_sha256_after'):
            broken = dict(meta, **{field: '0' * 64})
            with self.subTest(field=field), self.assertRaises(ValueError):
                summary.validate_record(broken, self.report, self.row, hashes)
        broken = copy.deepcopy(meta)
        broken['command'].append('-prefix=false')
        with self.assertRaisesRegex(ValueError, 'arguments'):
            summary.validate_record(broken, self.report, self.row, hashes)

    def test_declared_diagnostic_flags_are_exact_command_suffix(self):
        self.row['args'] = ['-cpu-profile', '/old/archive/cpu.pb.gz']
        self.row['profiled'] = True
        self.row['expected_runtime']['profiled'] = True
        self.report['profiled'] = True
        meta = self.proof.write(self.row, self.report)
        meta['command'] += self.row['args']
        hashes = {'schedule': summary.file_sha(self.proof.schedule),
                  'report': meta['report_sha256'], 'progress': meta['progress_sha256']}
        summary.validate_evidence(self.report, self.row)
        summary.validate_record(meta, self.report, self.row, hashes)

    def test_prefix_off_has_no_fabricated_prefixes(self):
        self.row.pop('expected_runtime')
        self.row['args'] = ['-prefix=false']
        self.report['prefix_enabled'] = False
        self.report['prefixes'] = []
        summary.validate_evidence(self.report, self.row)
        self.assertEqual(summary.summarize_report(self.report, {}, self.row)['prefixes'], [])

    def test_direct_is_explicitly_excluded_and_has_no_endpoint_journal(self):
        self.row.pop('expected_runtime')
        self.row['args'] = ['-mode=direct', '-prefix=false']
        report = self.report
        report['prefix_enabled'], report['prefixes'], report['server_contacts'] = False, [], []
        report['transport'] = summary.direct_transport()
        item, contact = report['evidence'][0], report['contacts'][0]
        prepared = item['prepared']
        prepared.update(Protocol='measurement/v1', Endpoint='deterministic-local-no-http', Wire='north', MaxResponseBytes=256, Digest='')
        prepared['Digest'] = hashlib.sha256(summary.go_json(prepared).encode()).hexdigest()
        item['prepared_digest'] = prepared['Digest']
        item['candidate'].update(HTTPStatus=0, RequestID='', Response='')
        contact.update(protocol=prepared['Protocol'], endpoint=prepared['Endpoint'], prepared_digest=prepared['Digest'],
                       request_wire='', response_wire='', request_sha256='', response_sha256='', response_complete=False)
        self.assertIn('Excluded direct contextual bridge', summary.validate_evidence(report, self.row))
        report['server_contacts'] = [contact]
        with self.assertRaises(ValueError):
            summary.validate_evidence(report, self.row)

    def test_small_cli_core_writes_index_and_keeps_inputs_immutable(self):
        self.proof.schedule.write_text(json.dumps([self.row]))
        self.proof.write(self.row, self.report)
        Path(self.row['binary']).unlink()
        before = {path.name: summary.file_sha(path) for path in self.proof.samples.iterdir()}
        output = self.proof.root / 'summary'
        summary.write_summaries(self.proof.schedule, self.proof.samples, output)
        index = json.loads((output / 'index.json').read_text())
        self.assertEqual(len(index['samples']), 1)
        self.assertEqual(index['normalization'], 'none')
        self.assertEqual(before, {path.name: summary.file_sha(path) for path in self.proof.samples.iterdir()})
        self.assertTrue((output / (self.row['name'] + '.summary.json')).is_file())

    def test_new_output_required_and_path_traversal_rejected(self):
        with tempfile.TemporaryDirectory() as existing:
            with self.assertRaises(FileExistsError):
                summary.write_summaries(self.proof.schedule, self.proof.samples, Path(existing))
        bad = dict(self.row, name='../escape')
        with self.assertRaises(ValueError):
            summary.validate_rows([bad])


if __name__ == '__main__':
    unittest.main()
