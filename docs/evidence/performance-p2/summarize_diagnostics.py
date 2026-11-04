#!/usr/bin/env python3
"""Validate frozen P2 diagnostic archives and write compact, unnormalized evidence.

One raw report is decoded at a time. No executable, database, or profile is opened.
SQL counts describe work; they do not assign CPU or rank latency costs.
"""
import argparse
from collections import Counter, defaultdict
import hashlib
import json
from pathlib import Path
import re

from campaign_validation import (
    BOUNDS, ENVIRONMENT, METRIC_NAMES, PREFIX_CONFIGURATION, RESOURCE_KEYS,
    SINGLETON_PHASES, TRANSPORT, go_json, require,
    tooling_hashes, validate_meta, validate_report, validate_transport_proof,
    validate_vector,
)

HERE = Path(__file__).resolve().parent
SQL_METRICS = ('completed_prepares', 'invocation_attempts', 'step_calls',
               'executions', 'sqlite_row_returns', 'fullscan_steps', 'vm_steps',
               'step_errors', 'bind_errors')
ATTRIBUTION_EVENTS = ('family_overflow_events', 'cell_overflow_events', 'invalid_scope_events')
SCOPES = {'unknown', 'invalid', 'createStore', 'openStore', 'Replay', 'Inspect',
          'InspectExecution', 'Run', 'finishRun', 'selectReady', 'settleRoots',
          'Commit', 'AuthorizeIntent', 'ApproveContinuation', 'PrepareRoot',
          'SettleRoot', 'prepareAdapterWork', 'accountAdapter', 'recordAdapterResult',
          'settleAdapterLocked', 'recordDispatchAdmission', 'AdmitInput',
          'AdmitInvocation', 'prepareExternal', 'recordExternalReceipt', 'settleExternalLocked'}
CLASSES = {'unknown', 'read_write_capable', 'read_only_configured'}
POST_STATES = {'unknown', 'off', 'on'}
CELL = re.compile(r'^sql\.scope\.([^.]+)\.class\.([^.]+)\.post_autocommit\.([^.]+)\.template\.(.+)\.([^.]+)$')
OVERFLOW_FAMILY = '<template-cap-or-size-overflow>'
MEMBERSHIP_FAMILY = ('SELECT root_id FROM composition_static_members WHERE subject=? AND generation=? '
                     'UNION ALL SELECT root_id FROM composition_members WHERE subject=? AND generation=?')
SCOPE_NOTE = 'Declared semantic context scope, not nearest Go caller; post_autocommit is post-step state, not transaction identity.'
RESOURCE_NOTE = ('Process-wide resources, including HTTP handler when present; GC/CPU asynchronous. '
                 'RSS is cumulative process high-water, not an additive interval cost.')
INCLUSIVE_NOTE = ('Execute encloses run and approval stages. All named groups and coarse timers are '
                  'inclusive; parent and nested spans must never be summed as disjoint costs.')
RUNTIME = {'go_version': 'go1.27.3', 'goos': 'darwin', 'goarch': 'arm64',
           'gomaxprocs': 4, 'mem_profile_rate': 524288, 'profiled': False,
           'instrumented': False, 'schedule': 'ordinary', 'reopen_every': 0,
           'prefix_enabled': True}


def file_sha(path):
    """Hash large inputs without a second whole-report byte buffer."""
    digest = hashlib.sha256()
    with Path(path).open('rb') as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def read_json(path):
    with Path(path).open() as source:
        return json.load(source)


def valid_digest(value):
    return isinstance(value, str) and re.fullmatch('[0-9a-f]{64}', value) is not None


def flag_configuration(row):
    args = row.get('args', [])
    require(isinstance(args, list) and all(isinstance(a, str) for a in args), 'invalid diagnostic arguments')
    values, index = {}, 0
    permitted = {'mode', 'prefix', 'cpu-profile', 'alloc-profile', 'trace'}
    while index < len(args):
        text = args[index]
        require(text.startswith('-') and not text.startswith('--'), 'unsupported diagnostic flag: ' + text)
        key, separator, value = text[1:].partition('=')
        require(key in permitted and key not in values, 'unsupported/repeated diagnostic flag: ' + key)
        if not separator:
            require(index + 1 < len(args), 'missing diagnostic flag value')
            index += 1
            value = args[index]
        require(bool(value), 'empty diagnostic flag value')
        values[key] = value
        index += 1
    mode, prefix = values.get('mode', 'http'), values.get('prefix', 'true')
    require(mode in ('http', 'direct') and prefix in ('true', 'false'), 'invalid mode/prefix')
    profiled = any(key in values for key in ('cpu-profile', 'alloc-profile', 'trace'))
    require(row.get('profiled', False) is profiled, 'profile flags/row mismatch')
    require(type(row.get('instrumented', False)) is bool, 'invalid instrumentation flag')
    require(not (profiled and row.get('instrumented', False)), 'counter/coarse and profiles must be separate')
    require(mode != 'direct' or prefix == 'false', 'direct bridge requires disabled prefixes')
    runtime = dict(RUNTIME, profiled=profiled, instrumented=row.get('instrumented', False), prefix_enabled=prefix == 'true')
    require('expected_runtime' not in row or row['expected_runtime'] == runtime, 'frozen runtime differs from declared flags')
    require('expected_transport' not in row or row['expected_transport'] == TRANSPORT, 'frozen transport drift')
    return mode, runtime


def validate_rows(rows):
    require(isinstance(rows, list) and bool(rows), 'nonempty frozen schedule required')
    seen = set()
    actual_tools = tooling_hashes()
    for row in rows:
        name = row.get('name', '')
        require(isinstance(name, str) and re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._-]*', name) is not None
                and name not in seen, 'unsafe/duplicate sample name')
        seen.add(name)
        require(type(row.get('bound')) is int and row['bound'] in BOUNDS, 'unsupported bound')
        require(isinstance(row.get('source'), str) and bool(row['source']), 'missing source label')
        require(isinstance(row.get('binary'), str) and Path(row['binary']).is_absolute(), 'absolute recorded executable path required')
        require(valid_digest(row.get('expected_binary_sha256')), 'missing frozen executable hash')
        require(row.get('expected_environment') == ENVIRONMENT, 'unexpected frozen environment')
        require(row.get('expected_tooling_sha256') == actual_tools, 'tooling differs from frozen schedule')
        flag_configuration(row)
    return actual_tools


def validate_phases(report, n):
    required = Counter((phase, 0) for phase in SINGLETON_PHASES)
    required.update(('run', i) for i in range(1, n + 1))
    for phase in ('approval_snapshot_basis', 'approval_establishment'):
        required.update((phase, i) for i in range(2, n + 1))
    stages = report.get('stages', [])
    require(isinstance(stages, list) and Counter((s.get('name'), s.get('position', 0)) for s in stages) == required,
            'required phase multiplicity/positions mismatch')
    for stage in stages:
        require(not stage.get('error'), 'stage error: ' + stage['name'])
        require(type(stage.get('position', 0)) is int, 'invalid phase position')
        validate_vector(stage, METRIC_NAMES)
    require(next(s for s in stages if s['name'] == 'execute_window')['wall_ns'] > 0, 'nonpositive Execute duration')


def validate_final_identity(report, n):
    """Shared final fields, also retained when prefix collection is disabled."""
    contacts, evidence = report.get('contacts', []), report.get('evidence', [])
    require(len(contacts) == len(evidence) == n, 'contact/evidence count mismatch')
    seen = {key: set() for key in ('operation_id', 'root_id', 'result_digest', 'observation_id', 'prepared_digest')}
    keys, outcomes = set(), set()
    for k, (contact, item) in enumerate(zip(contacts, evidence), 1):
        run = {'Subject': 'series-' + str(k), 'Generation': 1}
        require(type(item.get('index')) is int and item['index'] == k and item.get('run') == run
                and type(contact.get('ordinal')) is int and contact['ordinal'] == k, 'final position/run identity mismatch')
        for field in seen:
            value = item.get(field)
            require(isinstance(value, str) and value and value not in seen[field], 'missing/duplicate final identity: ' + field)
            seen[field].add(value)
        p, candidate = item.get('prepared', {}), item.get('candidate', {})
        request = p.get('Request', {})
        require(request.get('ID') == item['operation_id'] == contact.get('operation_id') == candidate.get('ID')
                and p.get('Digest') == item['prepared_digest'] == contact.get('prepared_digest'), 'final request/digest mismatch')
        key = request.get('Key')
        require(isinstance(key, str) and key and key not in keys and key == contact.get('key') == candidate.get('Key'), 'final operation key mismatch')
        keys.add(key)
        require(request.get('Capability') == contact.get('capability') == 'meter'
                and request.get('Effect') == contact.get('effect') == 'read'
                and request.get('Payload') == {'station': {'Kind': 'string', 'Number': 0, 'Text': 'north', 'Flag': False}},
                'final workload payload mismatch')
        require(candidate.get('Disposition') == 'validated' and candidate.get('ResponseID') == item.get('response_id')
                == contact.get('response_id') == 'local-response:' + item['operation_id'], 'final result identity mismatch')
        selected = item.get('selected_at', {})
        require(selected.get('Run') == run and type(selected.get('Sequence')) is int and selected['Sequence'] > 0,
                'final selection mismatch')
        outcome = item.get('root_outcome', {})
        require(isinstance(outcome, dict) and isinstance(outcome.get('ID'), str) and outcome['ID']
                and outcome['ID'] not in outcomes and outcome.get('Caller') == run
                and outcome.get('InvocationID') == 'root:' + item['root_id']
                and outcome.get('OperationID') == 'root:' + item['root_id'] and outcome.get('TemplateID') == ''
                and outcome.get('Status') == 'succeeded' and outcome.get('Reason') == 'member_succeeded'
                and outcome.get('ResponsePresent') is False and not outcome.get('Response')
                and type(outcome.get('DecisionOrder')) is int and outcome['DecisionOrder'] > 0, 'final original root outcome mismatch')
        outcomes.add(outcome['ID'])
        if k == 1:
            require(not item.get('approval'), 'unexpected first approval')
        else:
            approval = item.get('approval', {})
            protected, previous = approval.get('Approval', {}), evidence[k - 2]
            require(protected.get('Target') == run and protected.get('PredecessorID') == previous['operation_id']
                    and protected.get('PreparedDigest') == previous['prepared_digest']
                    and protected.get('ResultDigest') == previous['result_digest'] and protected.get('Admitted') is True
                    and protected.get('Reason') == 'operator' and protected.get('ScopeDigest')
                    and approval.get('SuccessorOperationID') == item['operation_id']
                    and approval.get('SelectedAt', {}).get('Run') == run, 'final approval mismatch')
    membership = report.get('root_membership', [])
    require(membership == [{'position': k, 'run': e['run'], 'root_id': e['root_id']}
                           for k, e in enumerate(evidence, 1)], 'final root membership mismatch')


def direct_transport():
    config = dict(TRANSPORT)
    for key in ('dial_timeout_ns', 'tcp_keepalive_ns', 'idle_timeout_ns', 'response_header_timeout_ns',
                'server_read_timeout_ns', 'server_write_timeout_ns', 'max_conns_per_host',
                'max_header_bytes', 'expect_continue_timeout_ns'):
        config[key] = 0
    config.update(mode='direct', http_version='none', endpoint='deterministic-local-no-http',
                  protocol='measurement/v1', listen_address='', request_format='station-text',
                  response_format='historical-candidate-v1', max_request_bytes=64, max_response_bytes=256,
                  timeout_ns=0, reuse_connections=False, resource_scope='process-no-server')
    return config


def validate_direct(report):
    require(report.get('transport') == direct_transport(), 'direct transport configuration mismatch')
    require(not report.get('server_contacts'), 'direct bridge must not invent server contacts')
    for contact, item in zip(report['contacts'], report['evidence']):
        prepared, candidate = item['prepared'], item['candidate']
        require(prepared.get('Protocol') == contact.get('protocol') == 'measurement/v1'
                and prepared.get('Endpoint') == contact.get('endpoint') == 'deterministic-local-no-http'
                and prepared.get('Wire') == 'north' and prepared.get('MaxResponseBytes') == 256
                and prepared.get('Model') == '' and prepared.get('MaxOutputTokens') == 0, 'direct retained Prepared mismatch')
        freeze = dict(prepared, Digest='')
        require(hashlib.sha256(go_json(freeze).encode()).hexdigest() == prepared['Digest'], 'direct Prepared digest mismatch')
        require(candidate.get('Payload') == {'sample': {'Kind': 'string', 'Number': 0, 'Text': 'present', 'Flag': False}}
                and candidate.get('HTTPStatus') == 0 and candidate.get('RequestID') == ''
                and candidate.get('Response') == '' and candidate.get('ReturnedModel') == '' and candidate.get('Usage') is None,
                'direct retained Candidate mismatch')
        require(not contact.get('request_wire') and not contact.get('response_wire')
                and not contact.get('request_sha256') and not contact.get('response_sha256')
                and contact.get('response_complete') is False, 'direct bridge has fabricated HTTP wire evidence')


def validate_evidence(report, row):
    mode, runtime = flag_configuration(row)
    name, n = row['name'], row['bound']
    require(report.get('version') == 'alan/performance-p2/v1' and report.get('invariants_ok') is True
            and not report.get('error'), 'failed/incomplete report: ' + name)
    require(type(report.get('bound')) is int and report['bound'] == n and report.get('sample') == name
            and report.get('source_label') == row['source'], 'report identity mismatch')
    require(all(type(report.get(k)) is type(v) and report[k] == v for k, v in runtime.items()), 'runtime configuration mismatch')
    require(all(type(report.get(k)) is int and report[k] == n for k in ('completed', 'selected', 'spent'))
            and type(report.get('protected_approvals')) is int and report['protected_approvals'] == n - 1
            and type(report.get('reductions')) is int and report['reductions'] == 4 * n - 1, 'final workload count mismatch')
    require(report.get('runtime_metric_names') == METRIC_NAMES and report.get('prefix_configuration') == PREFIX_CONFIGURATION,
            'metric/prefix configuration mismatch')
    require(report.get('resource_scope') == 'process-wide including same-process HTTP server; GC/CPU asynchronous; RSS cumulative high-water gauge',
            'resource scope mismatch')
    require(type(report.get('process_wall_ns')) is int and report['process_wall_ns'] > 0, 'process wall time missing')
    validate_final_identity(report, n)
    validate_phases(report, n)
    if mode == 'http' and runtime['prefix_enabled']:
        # Reuse the complete, unmodified campaign validator; this also validates
        # returned cuts, contact bijection, prefix and residual arithmetic.
        strict_row = dict(row, expected_runtime=runtime, expected_transport=TRANSPORT)
        validate_report(report, strict_row)
        scope = 'full campaign validate_report (HTTP, prefix-enabled)'
    elif mode == 'http':
        require(not report.get('prefixes'), 'prefix-off emitted records')
        transport = report.get('transport', {})
        require(set(transport) == set(TRANSPORT) | {'endpoint'} and all(transport[k] == v for k, v in TRANSPORT.items()),
                'HTTP calibration transport mismatch')
        # validate_transport_proof checks prepared endpoints against this value;
        # enforce the same endpoint constraints as the strict path as well.
        validate_http_endpoint(transport.get('endpoint', ''))
        validate_transport_proof(report, n)
        scope = 'HTTP prefix-off: strict transport proof plus exact final identity/count/configuration/phases/resources; no prefix proof'
    else:
        require(not report.get('prefixes'), 'direct prefix-off emitted records')
        validate_direct(report)
        scope = ('Excluded direct contextual bridge: retained driver proof and exact final identity/count/configuration/phases/resources; '
                 'no independent endpoint journal and no prefix proof')
    return scope


def validate_http_endpoint(endpoint):
    import ipaddress
    from urllib.parse import urlsplit
    parsed = urlsplit(endpoint)
    require(parsed.scheme == 'http' and parsed.hostname is not None and ipaddress.ip_address(parsed.hostname).is_loopback
            and parsed.port is not None and parsed.path == '/measure' and not parsed.query and not parsed.fragment
            and parsed.username is None, 'loopback endpoint identity mismatch')


def validate_record(meta, report, row, hashes):
    base = [row['binary'], '-bound', str(row['bound']), '-dir', str(Path(report.get('store', '')).parent),
            '-sample', row['name'], '-source-label', row['source']]
    require(meta.get('command') == base + row.get('args', []), 'process arguments mismatch')
    # The campaign validator's command check is for ordinary rows only. Verify
    # the entire actual command above, then reuse every other strict meta check.
    ordinary_meta = dict(meta, command=base)
    validate_meta(ordinary_meta, report, row, hashes['schedule'], hashes['report'], hashes['progress'])


def checked_counters(counters):
    require(isinstance(counters, dict) and all(isinstance(k, str) and type(v) is int and 0 <= v < 2**64
                                             for k, v in counters.items()), 'negative/missing/invalid counter')


def zero_sql():
    return dict.fromkeys(SQL_METRICS, 0)


def add_metrics(target, source):
    for key in SQL_METRICS:
        target[key] += source[key]


def sql_partition(counters, include_cells=False):
    """Validate all nine partitions, including per-family scoped attribution.

    Families retain all nine keys in snapshots/deltas. Sparse scope cells omit
    zero values. An empty pre-SQL snapshot can omit all total metrics, but never
    only some totals or a metric of a retained family.
    """
    checked_counters(counters)
    families, scoped = {}, {}
    dimensions = defaultdict(zero_sql)
    scopes = defaultdict(zero_sql)
    membership = defaultdict(zero_sql)
    family_cells = defaultdict(dict)
    total, overflow = {}, {}
    events = {}
    driver = {}
    for key, value in counters.items():
        if key.startswith('sql.template.'):
            family, metric = key[len('sql.template.'):].rsplit('.', 1)
            require(family and metric in SQL_METRICS, 'unknown family metric')
            families.setdefault(family, {})[metric] = value
        elif key.startswith('sql.scope.'):
            match = CELL.fullmatch(key)
            require(match is not None, 'malformed SQL scope cell')
            scope, connection, post, family, metric = match.groups()
            require(scope in SCOPES and connection in CLASSES and post in POST_STATES and metric in SQL_METRICS,
                    'unknown SQL scope/class/post-state/metric')
            scoped.setdefault(family, zero_sql())[metric] += value
            dimensions[(scope, connection, post)][metric] += value
            scopes[scope][metric] += value
            if include_cells:
                family_cells[family].setdefault((scope, connection, post), zero_sql())[metric] += value
            if scope == 'finishRun' and family == MEMBERSHIP_FAMILY:
                membership[family][metric] += value
        elif key.startswith('sql.total.'):
            metric = key[len('sql.total.'):]
            require(metric in SQL_METRICS, 'unknown total SQL metric')
            total[metric] = value
        elif key.startswith('sql.scope_overflow.'):
            metric = key[len('sql.scope_overflow.'):]
            require(metric in SQL_METRICS, 'unknown overflow SQL metric')
            overflow[metric] = value
        elif key.startswith('sql.attribution.'):
            field = key[len('sql.attribution.'):]
            require(field in set(ATTRIBUTION_EVENTS) | {'family_count', 'cell_count', 'incomplete_events'}, 'unknown attribution field')
            events[field] = value
        elif key.startswith('sql.driver.'):
            driver[key[len('sql.driver.'):]] = value
        else:
            require(not key.startswith('sql.'), 'unknown SQL counter key')
    require(set(overflow) == set(SQL_METRICS), 'missing SQL overflow counters')
    require(set(events) == set(ATTRIBUTION_EVENTS) | {'family_count', 'cell_count', 'incomplete_events'}, 'missing SQL attribution counters')
    require(events['incomplete_events'] == sum(events[k] for k in ATTRIBUTION_EVENTS), 'incomplete event partition mismatch')
    if not total:
        require(not families and not scoped and not any(overflow.values()), 'missing SQL total counters')
        total = zero_sql()
    require(set(total) == set(SQL_METRICS), 'missing SQL total metric')
    family_sum, scope_sum = zero_sql(), zero_sql()
    for family, metrics in families.items():
        require(set(metrics) == set(SQL_METRICS), 'missing family SQL metric: ' + family)
        add_metrics(family_sum, metrics)
    for metrics in scoped.values():
        add_metrics(scope_sum, metrics)
    require(set(scoped) <= set(families), 'scope cell has no matching family')
    require(total == family_sum, 'total/family SQL partition mismatch')
    require(all(total[k] == scope_sum[k] + overflow[k] for k in SQL_METRICS), 'total/scoped SQL partition mismatch')
    require(all(scoped.get(family, {}).get(k, 0) <= metrics[k] for family, metrics in families.items() for k in SQL_METRICS),
            'family scoped partition exceeds family total')
    if not any(overflow.values()):
        require(all(metrics == scoped.get(family, zero_sql()) for family, metrics in families.items()), 'family scoped partition mismatch')
    # A nonzero overflow vector must have a matching event even if it occurred
    # in a nested/inclusive measurement. Never silently call it complete.
    require(not any(overflow.values()) or events['cell_overflow_events'] > 0, 'unmarked cell overflow')
    require(not any(scopes.get('invalid', {}).values()) or events['invalid_scope_events'] > 0, 'unmarked invalid scope')
    complete = events['incomplete_events'] == 0 and not any(overflow.values())
    if OVERFLOW_FAMILY in families and any(families[OVERFLOW_FAMILY].values()):
        # An already-overflowed family may be used in a later zero-event delta.
        complete = False
    unknown = scopes.get('unknown', zero_sql())
    ordered = [{'family': family, 'metrics': families[family]} for family in sorted(families)]
    if include_cells:
        for item in ordered:
            item['scope_connection_post_state'] = [
                {'scope': s, 'connection_class': c, 'post_autocommit': p, 'metrics': metrics}
                for (s, c, p), metrics in sorted(family_cells.get(item['family'], {}).items())]
    top = sorted(ordered, key=lambda item: (-item['metrics']['executions'], item['family']))[:10]
    return {'totals': total, 'families': ordered, 'top10': top, 'top10_by': 'executions descending, family text ascending',
            'attribution_events': events, 'scope_overflow': overflow,
            'attribution_complete': complete, 'top10_complete': complete,
            'known_semantic_scope_attribution_complete': complete and not any(unknown.values()),
            'unknown_scope': unknown, 'invalid_scope': scopes.get('invalid', zero_sql()),
            'scopes': dict(sorted(scopes.items())),
            'scope_connection_post_state': [{'scope': s, 'connection_class': c, 'post_autocommit': p, 'metrics': metrics}
                                            for (s, c, p), metrics in sorted(dimensions.items())],
            'finishRun_membership_families': [{'family': f, 'metrics': m} for f, m in sorted(membership.items())],
            'driver': dict(sorted(driver.items()))}


def all_vectors(report):
    yield from report.get('stages', [])
    if report.get('execute_totals'):
        yield report['execute_totals']
    for prefix in report.get('prefixes', []):
        yield prefix['direct']
        yield prefix['cumulative']
    if report.get('prefix_enabled'):
        yield report['prefix_residual']


def counter_mode(report):
    found = set()
    for vector in all_vectors(report):
        counters = vector.get('counter_deltas', {})
        checked_counters(counters)
        for key in counters:
            if key.startswith('sql.'):
                found.add('sql')
            elif key.startswith('coarse.'):
                found.add('coarse')
            else:
                require(key.startswith(('fn.', 'work.')), 'unknown instrumentation counter: ' + key)
                found.add('work')
    require(not ('sql' in found and 'coarse' in found), 'mixed SQL/coarse overlays')
    require(not ('coarse' in found and 'work' in found), 'mixed coarse/work overlays')
    mode = 'sql' if 'sql' in found else 'coarse' if 'coarse' in found else 'none'
    require((mode != 'none') == report.get('instrumented', False), 'instrumentation evidence/configuration mismatch')
    require(mode != 'none' or 'work' not in found, 'work counters missing SQL overlay')
    return mode


def run_attribution(counter_vectors):
    maxima = dict.fromkeys(ATTRIBUTION_EVENTS, 0)
    unknown = False
    overflow_family = False
    for counters in counter_vectors:
        for field in maxima:
            maxima[field] = max(maxima[field], counters.get('sql.attribution.' + field, 0))
        unknown = unknown or any(value > 0 and key.startswith('sql.scope.unknown.') for key, value in counters.items())
        overflow_family = overflow_family or any(value > 0 and key.startswith('sql.template.' + OVERFLOW_FAMILY + '.')
                                                  for key, value in counters.items())
    complete = not any(maxima.values()) and not overflow_family
    return {'attribution_complete': complete, 'top10_complete': complete,
            'known_semantic_scope_attribution_complete': complete and not unknown,
            'maximum_observed_vector_events': maxima, 'unknown_scope_observed': unknown,
            'overflow_family_observed': overflow_family,
            'event_policy': 'Any overflow/invalid event anywhere defeats complete run attribution; maxima are not a disjoint event sum.'}


def coarse_counters(counters):
    output = {}
    for key, value in counters.items():
        require(key.startswith('coarse.'), 'non-coarse counter in coarse overlay')
        name, field = key[len('coarse.'):].rsplit('.', 1)
        require(field in ('calls', 'inclusive_wall_ns') and name, 'unknown coarse counter field')
        output.setdefault(name, {})[field] = value
    require(all(set(fields) == {'calls', 'inclusive_wall_ns'} for fields in output.values()), 'missing coarse counter field')
    return dict(sorted(output.items()))


def compact_counters(counters, mode, detail=False, run_status=None):
    checked_counters(counters)
    output = {'mode': mode, 'sql': None}
    if mode == 'sql':
        sql = sql_partition(counters, include_cells=detail)
        if run_status is not None:
            for field in ('attribution_complete', 'top10_complete', 'known_semantic_scope_attribution_complete'):
                sql[field] = sql[field] and run_status[field]
        if detail:
            output['sql'] = sql
        else:
            output['sql'] = {k: sql[k] for k in ('totals', 'attribution_events', 'scope_overflow', 'attribution_complete',
                                                'known_semantic_scope_attribution_complete', 'unknown_scope', 'invalid_scope',
                                                'finishRun_membership_families', 'driver')}
        output['function_work'] = {k: v for k, v in sorted(counters.items()) if k.startswith(('fn.', 'work.'))}
    elif mode == 'coarse':
        output['coarse'] = coarse_counters(counters)
        output['coarse_note'] = 'Inclusive wall/call counters; nested spans overlap; no SQL inference.'
    else:
        require(not counters, 'normal/profile row has unexpected counters')
        output['sql_note'] = 'SQL counters were not collected; this does not mean zero SQL.'
    return output


def resource_vector(vector):
    return {key: vector[key] for key in RESOURCE_KEYS + ('process_maxrss_cumulative', 'runtime_metric_deltas')}


def vector_summary(vector, mode, run_status=None, detail=False):
    result = compact_counters(vector.get('counter_deltas', {}), mode, detail, run_status)
    result['resources'] = resource_vector(vector)
    return result


def stage_groups(stages, mode, run_status=None):
    groups = {}
    for stage in stages:
        name = stage['name']
        if name not in groups:
            groups[name] = {'count': 0, 'inclusive': True,
                            'enclosing_window': 'execute_window' if name in ('run', 'approval_snapshot_basis', 'approval_establishment') else None,
                            'resources': {k: 0 for k in RESOURCE_KEYS},
                            'runtime_metric_deltas': dict.fromkeys(METRIC_NAMES, 0),
                            'process_maxrss_cumulative_max': 0,
                            'sql_totals': zero_sql() if mode == 'sql' else None,
                            'function_work': {}, 'coarse': {}}
        group = groups[name]
        group['count'] += 1
        for key in RESOURCE_KEYS:
            group['resources'][key] += stage[key]
        for key, value in stage['runtime_metric_deltas'].items():
            group['runtime_metric_deltas'][key] += value
        group['process_maxrss_cumulative_max'] = max(group['process_maxrss_cumulative_max'], stage['process_maxrss_cumulative'])
        compact = compact_counters(stage.get('counter_deltas', {}), mode, run_status=run_status)
        if mode == 'sql':
            add_metrics(group['sql_totals'], compact['sql']['totals'])
            for key, value in compact['function_work'].items():
                group['function_work'][key] = group['function_work'].get(key, 0) + value
        elif mode == 'coarse':
            for boundary, fields in compact['coarse'].items():
                target = group['coarse'].setdefault(boundary, {'calls': 0, 'inclusive_wall_ns': 0})
                for key, value in fields.items():
                    target[key] += value
    return dict(sorted(groups.items()))


def summarize_report(report, meta, row):
    mode = counter_mode(report)
    run_status = run_attribution(v.get('counter_deltas', {}) for v in all_vectors(report)) if mode == 'sql' else None
    # Check every available vector, including the independently retained nested
    # Execute sum, without accumulating any full raw report across sizes.
    for vector in all_vectors(report):
        validate_vector(vector, METRIC_NAMES)
        if mode == 'sql':
            sql_partition(vector.get('counter_deltas', {}))
        elif mode == 'coarse':
            coarse_counters(vector.get('counter_deltas', {}))
    outer = next(s for s in report['stages'] if s['name'] == 'execute_window')
    counts = {k: report.get(k) for k in ('completed', 'selected', 'spent', 'protected_approvals', 'reductions', 'replay_history', 'replay_intents')}
    counts.update(contacts=len(report['contacts']), server_contacts=len(report.get('server_contacts', [])),
                  results=len(report['evidence']), roots=len(report.get('root_membership', [])))
    prefixes = []
    for prefix in report.get('prefixes', []):
        prefixes.append({key: prefix[key] for key in ('position', 'declaration_bound', 'retained_positions', 'contacts', 'server_contacts')})
        prefixes[-1].update(direct=vector_summary(prefix['direct'], mode, run_status),
                            cumulative=vector_summary(prefix['cumulative'], mode, run_status))
    return {'sample': row['name'], 'source_label': row['source'], 'declaration_bound': row['bound'],
            'transport_mode': report['transport']['mode'], 'prefix_enabled': report['prefix_enabled'],
            'counter_mode': mode, 'profiled': report['profiled'], 'normalization': 'none',
            'counts': counts, 'resource_note': RESOURCE_NOTE, 'inclusive_note': INCLUSIVE_NOTE,
            'sql_scope_note': SCOPE_NOTE,
            'sql_measure_note': 'sqlite_row_returns and fullscan_steps are not exact rows examined; no per-family CPU/latency cost is inferred.',
            'run_attribution': run_status,
            'execute': vector_summary(outer, mode, run_status, detail=True),
            'stages_by_name': stage_groups(report['stages'], mode, run_status),
            'prefixes': prefixes,
            'prefix_interval_policy': 'Direct is the adjacent same-process increment (including approval); cumulative is T(B,k) for fixed declaration B. No unrelated-sample subtraction.',
            'prefix_residual': vector_summary(report['prefix_residual'], mode, run_status) if report['prefix_enabled'] else None,
            'process': {key: meta.get(key) for key in ('seconds', 'user_cpu_seconds', 'system_cpu_seconds', 'process_peak_rss', 'process_peak_rss_unit', 'load_before', 'load_after')},
            'driver_process_wall_ns': report['process_wall_ns']}


def write_json(path, value):
    with Path(path).open('x') as output:
        json.dump(value, output, sort_keys=True, separators=(',', ':'), allow_nan=False)
        output.write('\n')


def write_summaries(schedule_path, results, out):
    schedule_path, results, out = map(Path, (schedule_path, results, out))
    # Refuse existing output even when its contents are empty. An unsuccessful
    # partial attempt remains inspectable and cannot silently be overwritten.
    require(not out.resolve().is_relative_to(results.resolve()), 'output must be outside immutable input results')
    out.mkdir(parents=True, exist_ok=False)
    rows = read_json(schedule_path)
    tool_hashes = validate_rows(rows)
    schedule_hash = file_sha(schedule_path)
    sources = {name: file_sha(HERE / name) for name in ('summarize_diagnostics.py', 'test_diagnostic_summary.py',
               'test_campaign_tooling.py', 'contract.md', 'instrument/README.md', 'instrument/build.sh',
               'instrument/generate.py', 'instrument/sqlite_metrics.go.txt', 'instrument/timing.py',
               'instrument/attribution.py', 'instrument/proof_test.go.txt', 'instrument/test_generator.py', 'instrument/verify.py')}
    index = {'version': 'alan/performance-p2/diagnostic-summary-v1', 'normalization': 'none',
             'schedule_sha256': schedule_hash, 'campaign_tooling_sha256': tool_hashes,
             'analysis_source_sha256': sources, 'samples': [],
             'validation_policy': 'Exact frozen hashes/configuration; strict HTTP proof; per-vector nine-metric SQL partitions; run-wide overflow invalidates complete attribution. Original binary paths are provenance only.'}
    for row in rows:
        name = row['name']
        report_path, meta_path, progress_path = (results / (name + suffix) for suffix in ('.json', '.meta.json', '.progress'))
        hashes = {'schedule': schedule_hash, 'report': file_sha(report_path), 'meta': file_sha(meta_path), 'progress': file_sha(progress_path)}
        meta = read_json(meta_path)
        report = read_json(report_path)
        scope = validate_evidence(report, row)
        validate_record(meta, report, row, hashes)
        result = summarize_report(report, meta, row)
        # Detect mutation during analysis as well as the retained runner hashes.
        require(file_sha(report_path) == hashes['report'] and file_sha(meta_path) == hashes['meta']
                and file_sha(progress_path) == hashes['progress'], 'input changed while analyzing: ' + name)
        result.update(input_sha256=hashes, validation_scope=scope, schedule_row=row)
        destination = out / (name + '.summary.json')
        write_json(destination, result)
        index['samples'].append({'sample': name, 'summary': destination.name, 'summary_sha256': file_sha(destination),
                                 'input_sha256': hashes, 'validation_scope': scope,
                                 'counter_mode': result['counter_mode'], 'run_attribution': result['run_attribution']})
        del report, meta, result
    require(file_sha(schedule_path) == schedule_hash, 'schedule changed while analyzing')
    write_json(out / 'index.json', index)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--schedule', type=Path, required=True)
    parser.add_argument('--results', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    try:
        write_summaries(args.schedule, args.results, args.out)
    except (OSError, ValueError, KeyError, TypeError, StopIteration) as error:
        parser.exit(1, 'diagnostic summary rejected: ' + str(error) + '\n')


if __name__ == '__main__':
    main()
