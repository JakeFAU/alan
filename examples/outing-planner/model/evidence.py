"""Independent raw schema9 evidence; model API is injected from captured bytes."""
import base64
import hashlib
import json
import pathlib
import sqlite3

InvariantError = MODEL["InvariantError"]
require = MODEL["require"]

# Exact physical schema9, pinned from the supported source DDL. This is data,
# not a call into the production reader or verifier. Changes fail closed.
STORE_SCHEMA = r"""
CREATE TABLE metadata (singleton INTEGER PRIMARY KEY CHECK(singleton=1), schema_version INTEGER NOT NULL, semantics TEXT NOT NULL, key_version INTEGER NOT NULL, namespace TEXT NOT NULL);
CREATE TABLE programs (id TEXT PRIMARY KEY, semantics TEXT NOT NULL, bytes BLOB NOT NULL);
CREATE TABLE subjects (id TEXT PRIMARY KEY, generation INTEGER NOT NULL CHECK(generation>0), owner TEXT NOT NULL, revision INTEGER NOT NULL CHECK(revision>0), retired INTEGER NOT NULL CHECK(retired IN (0,1)), retry_cancelled INTEGER NOT NULL CHECK(retry_cancelled IN (0,1)));
CREATE TABLE permissions (kind TEXT NOT NULL, source TEXT NOT NULL, target TEXT NOT NULL, operation TEXT NOT NULL, revision INTEGER NOT NULL, obligation TEXT NOT NULL, PRIMARY KEY(kind,source,target,operation,revision,obligation));
CREATE TABLE runs (subject TEXT NOT NULL REFERENCES subjects(id), generation INTEGER NOT NULL CHECK(generation>0), program_id TEXT NOT NULL REFERENCES programs(id), initial BLOB NOT NULL, epoch INTEGER NOT NULL CHECK(epoch>0), sequence INTEGER NOT NULL CHECK(sequence>=0), projection BLOB NOT NULL, PRIMARY KEY(subject,generation));
CREATE TABLE history (subject TEXT NOT NULL, generation INTEGER NOT NULL, sequence INTEGER NOT NULL CHECK(sequence>0), epoch INTEGER NOT NULL CHECK(epoch>0), source_revision INTEGER NOT NULL CHECK(source_revision>0), selected_observation TEXT REFERENCES observations(id), successor BLOB NOT NULL, intents BLOB NOT NULL, PRIMARY KEY(subject,generation,sequence), FOREIGN KEY(subject,generation) REFERENCES runs(subject,generation));
CREATE TABLE observations (id TEXT PRIMARY KEY, subject TEXT NOT NULL, generation INTEGER NOT NULL, descriptor BLOB NOT NULL, consumed_sequence INTEGER, FOREIGN KEY(subject,generation) REFERENCES runs(subject,generation), FOREIGN KEY(subject,generation,consumed_sequence) REFERENCES history(subject,generation,sequence) DEFERRABLE INITIALLY DEFERRED);
CREATE TABLE intents (id TEXT PRIMARY KEY, subject TEXT NOT NULL, generation INTEGER NOT NULL, sequence INTEGER NOT NULL, ordinal INTEGER NOT NULL CHECK(ordinal>=0), source_revision INTEGER NOT NULL CHECK(source_revision>0), descriptor BLOB NOT NULL, UNIQUE(subject,generation,sequence,ordinal), FOREIGN KEY(subject,generation,sequence) REFERENCES history(subject,generation,sequence));
CREATE TABLE terminal_obligations (subject TEXT NOT NULL, generation INTEGER NOT NULL, sequence INTEGER NOT NULL, outcome BLOB NOT NULL, destinations BLOB NOT NULL, delivery_status BLOB NOT NULL, operation_id TEXT REFERENCES lifecycle_results(id) DEFERRABLE INITIALLY DEFERRED, source_revision INTEGER NOT NULL CHECK(source_revision>0), PRIMARY KEY(subject,generation), FOREIGN KEY(subject,generation) REFERENCES runs(subject,generation));
CREATE TABLE admissions (id TEXT PRIMARY KEY, domain TEXT NOT NULL CHECK(domain IN ('input','message','terminal','lifecycle','receipt','adapter')), transport_id TEXT NOT NULL, descriptor BLOB NOT NULL, result BLOB NOT NULL, observation_id TEXT REFERENCES observations(id), UNIQUE(domain,transport_id));
CREATE TABLE message_acknowledgments (intent_id TEXT PRIMARY KEY REFERENCES intents(id), admission_id TEXT NOT NULL REFERENCES admissions(id));
CREATE TABLE attempts (id TEXT PRIMARY KEY, intent_id TEXT NOT NULL REFERENCES intents(id), descriptor BLOB NOT NULL, result BLOB NOT NULL);
CREATE TABLE lifecycle_results (id TEXT PRIMARY KEY, decision_sequence INTEGER NOT NULL UNIQUE CHECK(decision_sequence>0), descriptor BLOB NOT NULL, result BLOB NOT NULL);
CREATE TABLE run_origins (subject TEXT NOT NULL, generation INTEGER NOT NULL, kind TEXT NOT NULL CHECK(kind IN ('root','retry')), operation_id TEXT REFERENCES lifecycle_results(id) DEFERRABLE INITIALLY DEFERRED, predecessor_generation INTEGER, descriptor BLOB NOT NULL, PRIMARY KEY(subject,generation), FOREIGN KEY(subject,generation) REFERENCES runs(subject,generation));
CREATE UNIQUE INDEX one_static_root ON run_origins(subject) WHERE kind='root';
CREATE TABLE external_attempts (intent_id TEXT PRIMARY KEY REFERENCES intents(id), descriptor BLOB NOT NULL);
CREATE TABLE external_receipts (intent_id TEXT PRIMARY KEY REFERENCES external_attempts(intent_id), descriptor BLOB NOT NULL);
CREATE TABLE corpus_assets (id TEXT PRIMARY KEY, descriptor BLOB NOT NULL);
CREATE TABLE adapter_attempts (intent_id TEXT PRIMARY KEY REFERENCES intents(id), descriptor BLOB NOT NULL);
CREATE TABLE adapter_results (intent_id TEXT PRIMARY KEY REFERENCES adapter_attempts(intent_id), descriptor BLOB NOT NULL);
CREATE TABLE settlements (id TEXT PRIMARY KEY, intent_id TEXT NOT NULL REFERENCES intents(id), descriptor BLOB NOT NULL, observation_id TEXT REFERENCES observations(id));

CREATE TABLE composition_profile (singleton INTEGER PRIMARY KEY CHECK(singleton=1), version TEXT NOT NULL, descriptor BLOB NOT NULL);
CREATE TABLE composition_roots (id TEXT PRIMARY KEY, descriptor BLOB NOT NULL);
CREATE TABLE composition_static_members (subject TEXT NOT NULL, generation INTEGER NOT NULL CHECK(generation=1), root_id TEXT NOT NULL REFERENCES composition_roots(id), descriptor BLOB NOT NULL, PRIMARY KEY(subject,generation), FOREIGN KEY(subject,generation) REFERENCES runs(subject,generation));
CREATE TABLE composition_templates (id TEXT PRIMARY KEY, descriptor BLOB NOT NULL);
CREATE TABLE composition_facts (decision_order INTEGER PRIMARY KEY CHECK(decision_order>0), id TEXT NOT NULL UNIQUE, kind TEXT NOT NULL CHECK(kind IN ('execution','lifecycle','admission','reservation','reply','freeze','outcome','authorization','external_result')), descriptor BLOB NOT NULL);
CREATE TABLE composition_invocations (id TEXT PRIMARY KEY, operation_id TEXT NOT NULL UNIQUE REFERENCES intents(id), template_id TEXT NOT NULL REFERENCES composition_templates(id), root_id TEXT NOT NULL REFERENCES composition_roots(id), descriptor BLOB NOT NULL, admission BLOB NOT NULL);
CREATE TABLE composition_bindings (id TEXT PRIMARY KEY, invocation_id TEXT NOT NULL UNIQUE REFERENCES composition_invocations(id), descriptor BLOB NOT NULL);
CREATE TABLE composition_members (subject TEXT NOT NULL, generation INTEGER NOT NULL CHECK(generation=1), invocation_id TEXT NOT NULL REFERENCES composition_invocations(id), root_id TEXT NOT NULL REFERENCES composition_roots(id), member TEXT NOT NULL, descriptor BLOB NOT NULL, PRIMARY KEY(subject,generation), UNIQUE(invocation_id,member), FOREIGN KEY(subject,generation) REFERENCES runs(subject,generation));
CREATE TABLE composition_results (invocation_id TEXT PRIMARY KEY REFERENCES composition_invocations(id), decision_order INTEGER NOT NULL UNIQUE REFERENCES composition_facts(decision_order), descriptor BLOB NOT NULL);
CREATE TABLE composition_charges (operation_id TEXT NOT NULL REFERENCES intents(id), kind TEXT NOT NULL CHECK(kind IN ('invocation','effect')), root_id TEXT NOT NULL REFERENCES composition_roots(id), decision_order INTEGER NOT NULL REFERENCES composition_facts(decision_order), descriptor BLOB NOT NULL, PRIMARY KEY(operation_id,kind));

CREATE TABLE composition_deployment(singleton INTEGER PRIMARY KEY CHECK(singleton=1), descriptor BLOB NOT NULL);
CREATE TABLE composition_permissions(singleton INTEGER PRIMARY KEY CHECK(singleton=1), descriptor BLOB NOT NULL);
CREATE TABLE composition_reservations(operation_id TEXT PRIMARY KEY REFERENCES intents(id), root_id TEXT NOT NULL REFERENCES composition_roots(id), descriptor BLOB NOT NULL);
CREATE TABLE composition_root_publications(scope_id TEXT PRIMARY KEY, root_id TEXT NOT NULL REFERENCES composition_roots(id), decision_order INTEGER NOT NULL UNIQUE REFERENCES composition_facts(decision_order) DEFERRABLE INITIALLY DEFERRED, descriptor BLOB NOT NULL);
CREATE TABLE composition_closures(scope_id TEXT PRIMARY KEY, root_id TEXT NOT NULL REFERENCES composition_roots(id), decision_order INTEGER NOT NULL UNIQUE REFERENCES composition_facts(decision_order), descriptor BLOB NOT NULL);
CREATE TABLE composition_publications(operation_id TEXT PRIMARY KEY REFERENCES composition_admission_decisions(operation_id), descriptor BLOB NOT NULL);
CREATE TABLE composition_outcome_deliveries(operation_id TEXT PRIMARY KEY REFERENCES composition_publications(operation_id), observation_id TEXT NOT NULL UNIQUE REFERENCES observations(id));
CREATE TABLE composition_admission_decisions(operation_id TEXT PRIMARY KEY REFERENCES intents(id), root_id TEXT NOT NULL REFERENCES composition_roots(id), descriptor BLOB NOT NULL);
CREATE TABLE composition_requests(invocation_id TEXT PRIMARY KEY REFERENCES composition_invocations(id), observation_id TEXT NOT NULL UNIQUE REFERENCES observations(id), descriptor BLOB NOT NULL);
CREATE TABLE composition_replies(operation_id TEXT PRIMARY KEY REFERENCES intents(id), invocation_id TEXT NOT NULL REFERENCES composition_invocations(id), descriptor BLOB NOT NULL);
CREATE TABLE composition_freezes(scope_id TEXT PRIMARY KEY, root_id TEXT NOT NULL REFERENCES composition_roots(id), descriptor BLOB NOT NULL);
CREATE TABLE adapter_dispatch_admissions (
	intent_id TEXT PRIMARY KEY NOT NULL REFERENCES adapter_attempts(intent_id),
	prepared_digest TEXT NOT NULL,
	after_fact INTEGER NOT NULL REFERENCES composition_facts(decision_order));"""


def raw_cell(value):
    """Typed length-delimited bytes; no JSON coercion of SQL values."""
    import struct
    if value is None:
        tag, body = b'N', b''
    elif type(value) is int:
        tag, body = b'I', str(value).encode('ascii')
    elif type(value) is float:
        tag, body = b'F', struct.pack('>d', value)
    elif type(value) is str:
        tag, body = b'T', value.encode('utf-8')
    elif type(value) is bytes:
        tag, body = b'B', value
    else:
        raise InvariantError('unsupported SQL cell type')
    return tag + len(body).to_bytes(8, 'big') + body


def raw_row_digest(values):
    import hashlib
    return hashlib.sha256(b'outing-raw-row/v1\x00' + b''.join(raw_cell(v) for v in values)).hexdigest()


def schema_manifest(connection):
    objects = connection.execute("SELECT type,name,tbl_name,sql FROM sqlite_master ORDER BY type,name").fetchall()
    tables = [row[1] for row in objects if row[0] == 'table']
    metadata = {}
    for table in tables:
        # Names were read from sqlite_master; quote embedded quotes, never interpolate literals.
        quoted = '"' + table.replace('"', '""') + '"'
        indexes = list(connection.execute('PRAGMA index_list(' + quoted + ')'))
        metadata[table] = {
            'columns': list(connection.execute('PRAGMA table_xinfo(' + quoted + ')')),
            'foreign_keys': list(connection.execute('PRAGMA foreign_key_list(' + quoted + ')')),
            'indexes': sorted((r[1], r[2], r[3], r[4]) for r in indexes),
            'index_columns': {r[1]: list(connection.execute('PRAGMA index_xinfo("' + r[1].replace('"', '""') + '")')) for r in indexes},
        }
    return objects, metadata


def read_raw_store(path):
    """Enumerate one read-only consistent cut, including committed WAL frames.

    Inventory is evidence of coverage, not a semantic correspondence verdict.
    No connection survives this function; the returned rows all share one cut.
    """
    import pathlib
    with sqlite3.connect(':memory:') as expected:
        expected.executescript(STORE_SCHEMA)
        wanted_objects, wanted_metadata = schema_manifest(expected)
    connection = sqlite3.connect(pathlib.Path(path).resolve().as_uri() + '?mode=ro', uri=True)
    try:
        connection.execute('PRAGMA query_only=ON')
        connection.execute('BEGIN')
        objects, metadata = schema_manifest(connection)
        require([(r[0], r[1], r[2]) for r in objects] == [(r[0], r[1], r[2]) for r in wanted_objects], 'schema objects: exact schema9 / 42 tables required')
        for actual, wanted in zip(objects, wanted_objects):
            normalize = lambda sql: ' '.join((sql or '').split())
            require(normalize(actual[3]) == normalize(wanted[3]), 'schema definition: ' + actual[1])
        require(metadata == wanted_metadata, 'schema columns, keys, indexes or foreign keys')
        require(connection.execute('PRAGMA user_version').fetchone()[0] == 9, 'schema version: require 9')
        data, inventory = {}, {}
        for table, info in sorted(metadata.items()):
            columns = [r[1] for r in info['columns']]
            primary = [r[1] for r in sorted(info['columns'], key=lambda r: r[5]) if r[5]]
            quoted = '"' + table + '"'
            values = list(connection.execute('SELECT * FROM ' + quoted + ' ORDER BY ' + ','.join('"' + name + '"' for name in primary)))
            data[table] = [dict(zip(columns, row)) for row in values]
            inventory[table] = {'columns': columns, 'primary_key': primary, 'count': len(values),
                'rows': [{'key': {key: row[key] for key in primary}, 'sha256': raw_row_digest(row.values())} for row in data[table]]}
        foreign_errors = list(connection.execute('PRAGMA foreign_key_check'))
        require(not foreign_errors, 'raw foreign-key links: ' + repr(foreign_errors[:3]))
        connection.rollback()
        return {'rows': data, 'inventory': inventory, 'schema': {'version': 9, 'tables': len(data), 'objects': objects, 'metadata': metadata}}
    finally:
        connection.close()


def go_json(value):
    """Go compact JSON for already ordered structs/maps; raw record order is retained."""
    return (json.dumps(value, ensure_ascii=False, separators=(',', ':'), allow_nan=False)
            .replace('&', '\\u0026').replace('<', '\\u003c').replace('>', '\\u003e')
            .replace('\u2028', '\\u2028').replace('\u2029', '\\u2029')).encode('utf-8')


def identity(domain, value):
    return hashlib.sha256(go_json([domain, value])).hexdigest()


def decoded(raw):
    def pairs(items):
        result = {}
        for key, value in items:
            require(key not in result, 'duplicate JSON field: ' + key)
            result[key] = value
        return result
    return json.loads(raw, object_pairs_hook=pairs)


def run_id(subject, generation=1):
    return {'Subject': subject, 'Generation': generation}


def value(item):
    kind = 'bool' if type(item) is bool else 'int' if type(item) is int else 'list' if type(item) is list else 'string'
    result = {'Kind': kind, 'Number': item if kind == 'int' else 0,
              'Text': item if kind == 'string' else '', 'Flag': item if kind == 'bool' else False}
    if kind == 'list':
        result['List'] = item
    return result


class Evidence:
    """Closed fixture-domain reconstruction, with explicit per-row coverage."""
    def __init__(self, raw, configuration, fixture, manifest):
        self.raw = raw
        self.tables = raw['rows']
        self.configuration = configuration
        self.fixture = fixture
        self.manifest = manifest
        self.coverage = {t: {} for t in self.tables}
        self.events = []
        self.facts = {}
        self.history_order = {}
        self.symbols = {}

    def key(self, table, row):
        return tuple(row[k] for k in self.raw['inventory'][table]['primary_key'])

    def cover(self, table, row, rule):
        key = self.key(table, row)
        require(key not in self.coverage[table], 'duplicate row classification: ' + table)
        self.coverage[table][key] = rule

    def one(self, table):
        require(len(self.tables[table]) == 1, 'exact singleton: ' + table)
        return self.tables[table][0]

    def rows_equal(self, table, expected, rule):
        actual = self.tables[table]
        require(sorted(actual, key=lambda r: self.key(table, r)) == sorted(expected, key=lambda r: self.key(table, r)), rule)
        for row in actual:
            self.cover(table, row, rule)

    def descriptor_rows(self, table, expected, rule):
        actual = [{k: decoded(v) if k in ('descriptor', 'admission') else v for k, v in row.items()} for row in self.tables[table]]
        require(sorted(actual, key=lambda r: self.key(table, r)) == sorted(expected, key=lambda r: self.key(table, r)), rule)
        for row in self.tables[table]:
            self.cover(table, row, rule)

    def static(self):
        metadata = self.one('metadata')
        require(metadata['singleton'] == 1 and metadata['schema_version'] == 9 and metadata['semantics'] == 'alan/1d/v1' and metadata['key_version'] == 2,
                'supported metadata schema/semantics/key')
        self.namespace = metadata['namespace']
        require(len(self.namespace) == 64 and all(c in '0123456789abcdef' for c in self.namespace), 'namespace identity')
        self.cover('metadata', metadata, 'fixed schema9 metadata')
        for table in ('lifecycle_results', 'external_attempts', 'external_receipts', 'corpus_assets', 'composition_closures'):
            require(not self.tables[table], 'unsupported nonempty category: ' + table)
        require(self.manifest['version'] == 1, 'manifest version')
        require(hashlib.sha256(self.configuration).hexdigest() == self.manifest['configuration_sha256'], 'configuration manifest digest')
        self.config = decoded(self.configuration)
        require(self.fixture['version'] == 1 and self.fixture['reference_time'] == self.config['reference_time'], 'fixture reference time')
        deployment = decoded(self.one('composition_deployment')['descriptor'])
        matches = [d for d in self.manifest['deployments'] if {'Subjects': d['subjects'], 'Runs': d['initial_runs']} == deployment]
        require(len(matches) == 1, 'exact pre-created deployment manifest')
        self.deployment = matches[0]
        self.variant = {'assess-first': 'direct', 'delegated': 'delegated', 'confirm-first': 'confirm-first'}.get(self.deployment['weather'])
        require(self.variant is not None, 'supported weather deployment')
        self.descriptor_rows('composition_deployment', [{'singleton': 1, 'descriptor': deployment}], 'frozen initial deployment')
        self.descriptor_rows('composition_permissions', [{'singleton': 1, 'descriptor': self.deployment['permissions']}], 'frozen initial permissions')
        initial = self.deployment['initial_runs']
        require([r['ID'] for r in initial] == [run_id('outing'), run_id('weather')], 'two original roots')
        self.initial = {r['ID']['Subject']: r for r in initial}
        require(self.initial['outing']['Projection']['Data']['configuration'] == value(self.configuration.decode('utf-8')), 'original configuration bytes')
        normalized = self.initial['outing']['Projection']['Data']['normalized_candidates']['List']
        expected_candidates = [{k: value(v) for k, v in sorted(c.items())} for c in self.manifest['candidates']]
        require(normalized == expected_candidates, 'complete frozen normalized candidates')
        self.candidates = [{k: v for k, v in c.items() if k not in ('eligible', 'reason')} for c in normalized if c['eligible']['Flag']]
        require(len(self.candidates) <= 2, 'unsupported candidate domain: require zero through two')
        self.proposal_id = self.initial['outing']['Projection']['Data']['proposal_id']
        self.weather_data = self.initial['weather']['Projection']['Data']
        require(self.deployment['helper_admission'] in (0, 1) if self.variant == 'delegated' else self.deployment['helper_admission'] == 0, 'declared helper allowance')
        expected_permissions = [
            {'Kind': 'message', 'Source': 'outing', 'Target': 'weather', 'Operation': 'message', 'Revision': 1, 'Obligation': obligation}
            for obligation in ('outing-decision', 'outing-proposal')]
        expected_permissions.append({'Kind': 'message', 'Source': 'weather', 'Target': 'outing', 'Operation': 'message', 'Revision': 1, 'Obligation': 'outing-assessment'})
        if self.variant != 'delegated':
            expected_permissions.append({'Kind': 'adapter', 'Source': 'weather', 'Target': 'open-meteo-hourly', 'Operation': 'forecast', 'Revision': 1, 'Obligation': 'forecast'})
        require(sorted(self.deployment['permissions'], key=lambda p: go_json(p)) == sorted(expected_permissions, key=lambda p: go_json(p)), 'fixed example initial permission set')
        self.templates = self.deployment['composition']['Templates'] or []
        require(len(self.templates) == (1 if self.variant == 'delegated' else 0), 'exact helper catalogue')
        program_ids = {r['ProgramID'] for r in initial}
        program_ids.update(m['ProgramID'] for t in self.templates for m in t['Members'].values())
        artifacts = {}
        for a in self.manifest['artifacts']:
            body = base64.b64decode(a['canonical_base64'], validate=True)
            require(hashlib.sha256(body).hexdigest() == a['program_id'] and decoded(body)['Semantics'] == a['semantics'], 'manifest canonical program identity/semantics')
            require(a['program_id'] not in artifacts or artifacts[a['program_id']] == (a['semantics'], body), 'conflicting manifest artifact')
            artifacts[a['program_id']] = (a['semantics'], body)
        require(program_ids <= artifacts.keys(), 'manifest missing original program')
        self.rows_equal('programs', [{'id': pid, 'semantics': artifacts[pid][0], 'bytes': artifacts[pid][1]} for pid in program_ids], 'exact raw program inventory')
        self.programs = {pid: decoded(artifacts[pid][1]) for pid in program_ids}
        self.roots = {}
        for root in self.deployment['composition']['Roots']:
            weather = root['Run']['Subject'] == 'weather'
            delegated = weather and self.variant == 'delegated'
            limits = {'MaxInvocations': self.deployment['helper_admission'] if delegated else 0, 'MaxCreatedRuns': int(delegated),
                'MaxDepth': int(delegated), 'MaxEffects': len(self.candidates) if weather else 0, 'MaxInFlight': 1}
            require(root['Limits'] == limits and root['Members'] == [root['Run']] and root['Capabilities'] == (['open-meteo-hourly'] if weather else None)
                and root['Templates'] == ([self.templates[0]['ID']] if delegated else None), 'exact fixed resource envelope and allowlists')
            expected = dict(root, ID=identity('alan/resource-profile/v1', [self.namespace, root['Run']]), Version='alan/resource-profile/v1')
            self.roots[root['Run']['Subject']] = expected
        profile = {'Version': 'alan/composition/v1', 'Roots': sorted(self.roots.values(), key=lambda r: r['ID']), 'Templates': self.templates or None, 'RecordDispatchAdmissions': True}
        self.descriptor_rows('composition_profile', [{'singleton': 1, 'version': 'alan/composition/v1', 'descriptor': profile}], 'exact composition profile')
        self.descriptor_rows('composition_roots', [{'id': r['ID'], 'descriptor': r} for r in self.roots.values()], 'exact root limits and capabilities')
        self.descriptor_rows('composition_templates', [{'id': t['ID'], 'descriptor': t} for t in self.templates], 'exact template catalogue')
        self.descriptor_rows('composition_static_members', [{'subject': name, 'generation': 1, 'root_id': r['ID'], 'descriptor': {'RootID': r['ID'], 'Run': run_id(name)}} for name, r in self.roots.items()], 'exact static root membership')
        self.descriptor_rows('run_origins', [{'subject': s['ID'], 'generation': 1, 'kind': 'root', 'operation_id': None, 'predecessor_generation': None,
            'descriptor': {'Subject': s, 'ProgramID': self.initial[s['ID']]['ProgramID'], 'Initial': self.initial[s['ID']]['Projection']}} for s in self.deployment['subjects']], 'exact root origins')

    def histories(self):
        self.runs = {r['subject']: r for r in self.tables['runs']}
        require(len(self.runs) == len(self.tables['runs']), 'one generation per run')
        self.intents = {}
        self.history = {}
        self.selected = {}
        expected_intents = []
        for subject, run in self.runs.items():
            require(run['generation'] == 1 and run['epoch'] >= 1 and run['sequence'] >= 0, 'fixed generation and valid epoch')
            require(run['program_id'] in self.programs, 'run program from exact manifest')
            rows = sorted((r for r in self.tables['history'] if r['subject'] == subject), key=lambda r: r['sequence'])
            require([r['sequence'] for r in rows] == list(range(1, run['sequence'] + 1)), 'contiguous complete run history')
            previous = decoded(run['initial'])
            previous_epoch = 1
            for row in rows:
                seq = row['sequence']
                require(row['epoch'] >= previous_epoch, 'run history epochs never decrease')
                previous_epoch = row['epoch']
                require(row['generation'] == 1 and row['source_revision'] == 1 and 1 <= row['epoch'] <= run['epoch'], 'history source authority')
                require(previous['Status'] == 'open', 'history after terminal predecessor')
                instructions = self.programs[run['program_id']]['Code']
                require(type(previous['PC']) is int and 0 <= previous['PC'] < len(instructions), 'history predecessor program counter')
                instruction = instructions[previous['PC']]
                intents = decoded(row['intents'])
                selection = row['selected_observation']
                successor = decoded(row['successor'])
                require((selection is not None) == (instruction['Op'] == 'receive'), 'observable selection cannot be hidden as pure step')
                require(len(intents) == (1 if instruction['Op'] == 'emit' else 0), 'observable operation cannot be hidden as pure step')
                require((successor['Status'] != 'open') == (instruction['Op'] == 'end'), 'terminal transition versus pure step')
                for ordinal, intent in enumerate(intents):
                    require(intent['ID'] == go_json([subject, 1, seq, ordinal]).decode() and intent['Run'] == run_id(subject) and intent['Sequence'] == seq and intent['Ordinal'] == ordinal,
                            'original intent identity and producing history')
                    require(intent['Kind'] in ('message', 'adapter', 'invocation') and 'Authority' not in intent, 'unsupported intent kind/authority')
                    require(intent['ID'] not in self.intents, 'duplicate original intent')
                    self.intents[intent['ID']] = intent
                    expected_intents.append({'id': intent['ID'], 'subject': subject, 'generation': 1, 'sequence': seq, 'ordinal': ordinal, 'source_revision': 1, 'descriptor': go_json(intent)})
                if selection is not None:
                    require(selection not in self.selected, 'duplicate observation selection')
                    self.selected[selection] = (subject, seq)
                self.history[(subject, seq)] = row
                self.cover('history', row, 'contiguous projection/operation/selection history')
                previous = successor
            require(previous == decoded(run['projection']), 'final history successor equals projection')
        require(sum(len([r for r in self.tables['history'] if r['subject'] == s]) for s in self.runs) == len(self.tables['history']), 'orphan history')
        self.rows_equal('intents', expected_intents, 'history/intents complete bijection')
        self.observations = {}
        for row in self.tables['observations']:
            observation = decoded(row['descriptor'])
            require(observation['ID'] == row['id'] and observation['Run'] == run_id(row['subject'], row['generation']), 'observation row identity')
            expected_selection = None if row['consumed_sequence'] is None else (row['subject'], row['consumed_sequence'])
            require(self.selected.get(row['id']) == expected_selection, 'observation/history selection bijection')
            self.observations[row['id']] = observation
        require(set(self.selected) <= self.observations.keys(), 'selected observation missing')

    def index_facts(self):
        for expected_order, row in enumerate(self.tables['composition_facts'], 1):
            fact = decoded(row['descriptor'])
            require(row['decision_order'] == expected_order == fact['Order'] and row['id'] == fact['ID'] and row['kind'] == fact['Kind'], 'contiguous composition fact order and identity')
            require(fact['Kind'] in ('execution', 'authorization', 'admission', 'reservation', 'reply', 'freeze', 'outcome', 'external_result'), 'unsupported composition fact kind')
            key = (fact['Kind'], fact['Run']['Subject'], fact['Sequence']) if fact['Kind'] == 'execution' else (fact['Kind'], fact['OperationID'])
            require(key not in self.facts, 'duplicate source composition fact')
            self.facts[key] = fact
            if fact['Kind'] == 'execution':
                self.history_order[(fact['Run']['Subject'], fact['Sequence'])] = expected_order

    def fact_order(self, kind, operation):
        key = (kind,) + (operation if isinstance(operation, tuple) else (operation,))
        require(key in self.facts, 'missing ' + kind + ' fact for original ' + repr(operation))
        return self.facts[key]['Order']

    def add_event(self, stage, operation, actor, target='', candidate=-1, binding='', disposition='', order=None, after=0, before=None):
        event = MODEL['Event'](stage, operation, actor, target, candidate, binding, disposition)
        require(all(e['event'] != event for e in self.events), 'duplicate observable boundary: ' + stage)
        self.events.append({'event': event, 'order': order, 'after': after, 'before': before})

    def event_for_intent(self, intent, suffix, order=None, **extra):
        symbol = self.symbols[intent['ID']]
        actor = self.actors[intent['Run']['Subject']]
        if symbol.startswith('forecast:'):
            candidate = int(symbol.split(':')[-1])
            self.add_event('forecast.' + suffix, symbol, actor, candidate=candidate, order=order, **extra)
        elif symbol == MODEL['CALL']:
            self.add_event('invocation.' + suffix, symbol, 'weather', 'helper', order=order, **extra)
        else:
            target = 'outing' if actor == 'weather' else 'weather'
            self.add_event('message.' + suffix, symbol, actor, target, binding=MODEL['BINDING'] if actor == 'helper' else '', order=order, **extra)

    def bind_operations(self):
        calls = [i for i in self.intents.values() if i['Kind'] == 'invocation']
        require(len(calls) <= (1 if self.variant == 'delegated' else 0), 'one original helper invocation')
        self.call = calls[0] if calls else None
        self.actors = {'outing': 'outing', 'weather': 'weather'}
        self.member = None
        self.admission = None
        self.invocation = None
        self.binding = None
        if self.call:
            call = self.call
            require(call['Run'] == run_id('weather') and call['Target'] == self.templates[0]['ID'] and call['Generation'] == 1 and call['Obligation'] == 'assess', 'original invocation caller/template/obligation')
            expected_request = {'allowed_weather_codes': self.weather_data['allowed_weather_codes'], 'candidates': value(self.candidates), 'proposal_id': self.proposal_id, 'units': self.weather_data['units']}
            require(call['Payload'] == expected_request, 'original helper request payload')
            self.symbols[call['ID']] = MODEL['CALL']
            self.invocation = {'ID': identity('alan/invocation/v1', [self.namespace, call['ID']]), 'OperationID': call['ID'], 'TemplateID': self.templates[0]['ID'],
                'RootID': self.roots['weather']['ID'], 'ParentInvocationID': '', 'Caller': run_id('weather'), 'SourceRevision': 1, 'SourceSequence': call['Sequence'], 'Depth': 1, 'Inputs': expected_request}
            decisions = self.tables['composition_admission_decisions']
            require(len(decisions) <= 1, 'one original admission decision')
            if decisions:
                allowed = self.deployment['helper_admission'] == 1
                member = run_id(identity('alan/invocation-member/v1', [self.invocation['ID'], 'worker']))
                binding = {'Protocol': 'alan/composition/v1', 'ID': '', 'Version': 'alan/binding/v1', 'Namespace': self.namespace,
                    'InvocationID': self.invocation['ID'], 'OperationID': call['ID'], 'TemplateID': call['Target'], 'RootID': self.roots['weather']['ID'],
                    'Caller': run_id('weather'), 'Depth': 1, 'Members': {'worker': member}, 'Programs': {'worker': self.templates[0]['Members']['worker']['ProgramID']}}
                # Binding identity hashes its complete descriptor with ID empty.
                binding['ID'] = identity('alan/binding/v1', binding)
                self.admission = {'Invocation': self.invocation, 'Accepted': allowed, 'Reason': 'accepted' if allowed else 'invocation_limit',
                    'BindingID': binding['ID'] if allowed else '', 'Members': [member] if allowed else None}
                self.descriptor_rows('composition_admission_decisions', [{'operation_id': call['ID'], 'root_id': self.roots['weather']['ID'], 'descriptor': self.admission}], 'exact original helper admission')
                if allowed:
                    self.member, self.binding = member, binding
                    self.actors[member['Subject']] = 'helper'
        elif self.tables['composition_admission_decisions']:
            raise InvariantError('admission without original invocation')
        root = self.roots['weather']['ID']
        accepted = self.member is not None
        self.descriptor_rows('composition_invocations', [{'id': self.invocation['ID'], 'operation_id': self.call['ID'], 'template_id': self.call['Target'], 'root_id': root,
            'descriptor': self.invocation, 'admission': self.admission}] if accepted else [], 'accepted invocation row bijection')
        self.descriptor_rows('composition_bindings', [{'id': self.binding['ID'], 'invocation_id': self.invocation['ID'], 'descriptor': self.binding}] if accepted else [], 'exact original helper binding')
        member_descriptor = None
        if accepted:
            prototype = self.templates[0]['Members']['worker']
            member_descriptor = {'Run': self.member, 'InvocationID': self.invocation['ID'], 'RootID': root, 'Member': 'worker',
                'ProgramID': prototype['ProgramID'], 'Initial': prototype['Initial'], 'Owner': 'invocation:' + self.invocation['ID']}
        self.descriptor_rows('composition_members', [{'subject': self.member['Subject'], 'generation': 1, 'invocation_id': self.invocation['ID'], 'root_id': root, 'member': 'worker', 'descriptor': member_descriptor}] if accepted else [], 'exact dynamic member provenance')
        expected_subjects = [{k.lower().replace('retrycancelled', 'retry_cancelled'): v for k, v in s.items()} for s in self.deployment['subjects']]
        if accepted:
            expected_subjects.append({'id': self.member['Subject'], 'generation': 1, 'owner': member_descriptor['Owner'], 'revision': 1, 'retired': False, 'retry_cancelled': False})
        self.rows_equal('subjects', expected_subjects, 'exact subjects and unchanged authority')
        expected_runs = dict(self.initial)
        if accepted:
            expected_runs[self.member['Subject']] = {'ID': self.member, 'ProgramID': member_descriptor['ProgramID'], 'Projection': member_descriptor['Initial']}
        require(set(self.runs) == set(expected_runs), 'exact run inventory; no hidden child')
        for name, initial_run in expected_runs.items():
            row = self.runs[name]
            require(row['program_id'] == initial_run['ProgramID'] and decoded(row['initial']) == initial_run['Projection'], 'original run program and initial projection')
            self.cover('runs', row, 'exact root/member initial state and final history')
        permissions = [{k.lower(): v for k, v in p.items()} for p in self.deployment['permissions']]
        if accepted:
            permissions.append({'kind': 'adapter', 'source': self.member['Subject'], 'target': 'open-meteo-hourly', 'operation': 'forecast', 'revision': 1, 'obligation': 'forecast'})
        self.rows_equal('permissions', permissions, 'exact initial and bound helper grants')
        forecasts = sorted((i for i in self.intents.values() if i['Kind'] == 'adapter'), key=lambda i: i['Sequence'])
        require(len(forecasts) <= len(self.candidates), 'hidden extra original forecast')
        self.forecasts = forecasts
        owner = self.member if self.variant == 'delegated' else run_id('weather')
        for index, intent in enumerate(forecasts):
            require(owner is not None and intent['Run'] == owner, 'original forecast owner')
            candidate = self.candidates[index]
            payload = {k: candidate[k] for k in ('end_hour', 'latitude', 'longitude', 'start_hour')}
            payload['units'] = self.weather_data['units']
            require(intent['Target'] == 'open-meteo-hourly' and intent['Generation'] == 1 and intent['Obligation'] == 'forecast' and intent['Payload'] == payload, 'original forecast request/candidate ordering')
            self.symbols[intent['ID']] = 'forecast:' + self.actors[owner['Subject']] + ':' + str(index)
        message_specs = {'outing-proposal': ('outing', 'weather', 'proposal'), 'outing-assessment': ('weather', 'outing', 'assessment'), 'outing-decision': ('outing', 'weather', 'decision'), 'assess': ('helper', 'weather', 'reply')}
        for intent in self.intents.values():
            if intent['Kind'] == 'message':
                require(intent['Obligation'] in message_specs, 'unsupported message obligation')
                source, target, name = message_specs[intent['Obligation']]
                require(self.actors.get(intent['Run']['Subject']) == source and intent['Target'] == target and intent['Generation'] == 1, 'original public/bound message endpoints')
                symbol = 'message:' + source + ':' + name
                require(symbol not in self.symbols.values(), 'duplicate original message/reply')
                self.symbols[intent['ID']] = symbol
        require(len(self.symbols) == len(self.intents), 'complete original operation classification')
        for intent in self.intents.values():
            self.event_for_intent(intent, 'commit', self.fact_order('execution', (intent['Run']['Subject'], intent['Sequence'])))

    def authorizations(self):
        self.auth = {}
        for row in self.tables['attempts']:
            operation = row['intent_id']
            require(operation in self.intents and operation not in self.auth, 'one first authorization per original')
            intent = self.intents[operation]
            expected_id = go_json(['authorization', operation]).decode()
            require(row['id'] == expected_id and decoded(row['descriptor']) == {'Intent': intent, 'Revision': 1}, 'authorization exact original intent/source revision')
            auth = decoded(row['result'])
            order = self.fact_order('authorization', operation)
            source = auth['Source']
            subject = intent['Run']['Subject']
            require(source['Run'] == intent['Run'] and source['Revision'] == 1 and intent['Sequence'] <= source['Sequence'] <= self.runs[subject]['sequence'], 'authorization source snapshot coordinates')
            require(1 <= source['Epoch'] <= self.runs[subject]['epoch'], 'authorization source epoch')
            prefix = [seq for (name, seq), rank in self.history_order.items() if name == subject and rank < order]
            require(source['Sequence'] == max(prefix, default=0), 'authorization Source.Sequence must match global fact prefix')
            latest = self.history[(subject, source['Sequence'])]
            require(source['Epoch'] >= latest['epoch'], 'authorization source epoch precedes latest local commit')
            next_row = self.history.get((subject, source['Sequence'] + 1))
            require(next_row is None or source['Epoch'] <= next_row['epoch'], 'authorization source epoch exceeds later local commit')
            projection = decoded(latest['successor'])
            ended = projection['Status'] != 'open'
            bound = self.symbols[operation] == 'message:helper:reply' and self.member is not None
            allowed = not ended or bound
            expected = {'ID': expected_id, 'IntentID': operation, 'Allowed': allowed, 'Reason': 'authorized' if allowed else 'source_closed', 'Source': source, 'ClosureOperation': ''}
            require(auth == expected, 'source authorization disposition/bound-reply exception')
            require(allowed or self.symbols[operation] == 'message:outing:decision', 'unsupported closed-source operation')
            self.auth[operation] = auth
            self.cover('attempts', row, 'immutable first source authorization at exact fact prefix')
            extra = {'disposition': 'allow' if allowed else 'deny'} if intent['Kind'] in ('message', 'invocation') else {}
            self.event_for_intent(intent, 'authorize', order, **extra)

    def observation(self, expected, producer):
        require(expected['ID'] in self.observations and self.observations[expected['ID']] == expected, producer + ': exact original observation')
        row = next(r for r in self.tables['observations'] if r['id'] == expected['ID'])
        self.cover('observations', row, producer)
        return self.selected.get(expected['ID'])

    def request(self):
        if self.admission:
            require(self.call['ID'] in self.auth and self.auth[self.call['ID']]['Allowed'], 'helper admission requires allowed original authorization')
            accepted = self.member is not None
            self.add_event('invocation.admit', MODEL['CALL'], 'weather', 'helper', binding=MODEL['BINDING'] if accepted else '',
                disposition='accepted' if accepted else 'invocation_limit', order=self.fact_order('admission', self.call['ID']))
        expected_rows = []
        if self.member:
            observation_id = identity('alan/invocation-request/v1', self.invocation['ID'])
            request = {'ID': observation_id, 'Run': self.member, 'Kind': 'message', 'Provenance': 'runtime-invocation-request', 'Source': 'weather', 'SourceGeneration': 1,
                'Obligation': 'assess', 'Operation': self.call['ID'], 'Payload': self.call['Payload']}
            expected_rows = [{'invocation_id': self.invocation['ID'], 'observation_id': observation_id, 'descriptor': request}]
            selection = self.observation(request, 'atomic helper request admission')
            if selection:
                self.add_event('invocation.request.select', MODEL['CALL'], 'helper', 'weather', binding=MODEL['BINDING'], order=self.fact_order('execution', selection))
        self.descriptor_rows('composition_requests', expected_rows, 'accepted request/member atomic group')

    def messages(self):
        self.message_admissions = {}
        self.expected_replies = []
        for row in self.tables['admissions']:
            if row['domain'] == 'adapter':
                continue
            require(row['domain'] == 'message', 'unsupported admission domain')
            d, result = decoded(row['descriptor']), decoded(row['result'])
            operation = d['IntentID']
            require(operation in self.intents and self.intents[operation]['Kind'] == 'message', 'message admission original')
            require(operation not in self.message_admissions, 'unsupported domain: multiple transports per original')
            require(operation in self.auth, 'message admission without first source authorization')
            intent, auth = self.intents[operation], self.auth[operation]
            transport = row['transport_id']
            expected = {'TransportID': transport, 'Kind': 'message', 'LogicalID': operation, 'Source': intent['Run'], 'Target': intent['Target'], 'Generation': 1,
                'Obligation': intent['Obligation'], 'Operation': '', 'Payload': intent['Payload'], 'IntentID': operation, 'TerminalSequence': 0}
            require(d == expected and transport and row['id'] == go_json(['admission', 'message', transport]).decode(), 'exact original message admission')
            reply = self.symbols[operation] == 'message:helper:reply'
            accepted = result['Accepted']
            reason = 'recorded_reply' if reply else 'accepted'
            if not auth['Allowed']:
                require(not accepted, 'denied source admitted')
                reason = 'source_denied:source_closed'
            elif not accepted:
                require(not reply, 'bound reply refusal outside fixture domain')
                reason = 'target_closed'
            observation_id = go_json(['observation', 'message', operation]).decode() if accepted and not reply else ''
            require(result == {'TransportID': transport, 'Accepted': accepted, 'Reason': reason, 'Bound': run_id(intent['Target']), 'ObservationID': observation_id}
                    and row['observation_id'] == (observation_id or None), 'message admission/observation disposition')
            self.message_admissions[operation] = row
            self.cover('admissions', row, 'one original public or bound reply transport')
            if reply:
                reply_descriptor = {'InvocationID': self.invocation['ID'], 'RootID': self.roots['weather']['ID'], 'Intent': intent, 'Authorization': auth, 'Accepted': True, 'Reason': 'recorded_reply'}
                self.expected_replies.append({'operation_id': operation, 'invocation_id': self.invocation['ID'], 'descriptor': reply_descriptor})
            order = self.fact_order('reply', operation) if reply else None
            self.event_for_intent(intent, 'admit', order, disposition='accepted' if accepted else reason, after=self.fact_order('authorization', operation))
            if observation_id:
                observation = {'ID': observation_id, 'Run': run_id(intent['Target']), 'Kind': 'message', 'Provenance': 'ordinary-message', 'Source': intent['Run']['Subject'],
                    'SourceGeneration': 1, 'Obligation': intent['Obligation'], 'Operation': '', 'Payload': intent['Payload']}
                selection = self.observation(observation, 'ordinary message admission')
                if selection:
                    self.add_event('message.select', self.symbols[operation], intent['Target'], intent['Run']['Subject'], order=self.fact_order('execution', selection))
        self.descriptor_rows('composition_replies', self.expected_replies, 'bound reply admission/reply atomic group')
        for row in self.tables['message_acknowledgments']:
            operation = row['intent_id']
            require(operation in self.message_admissions and row['admission_id'] == self.message_admissions[operation]['id'], 'original transport acknowledgment')
            self.cover('message_acknowledgments', row, 'raw runner acknowledgment distinct from reply diagnostic')
            self.event_for_intent(self.intents[operation], 'acknowledge')

    def fixture_response(self, candidate):
        """Derive native response and suitability from frozen fixture bytes alone."""
        import decimal
        import math
        import re
        from email.message import Message as HeaderMessage
        fields = ('latitude', 'longitude', 'start_hour', 'end_hour')
        matches = [entry for entry in self.fixture['entries'] if all(entry[k] == candidate[k]['Text'] for k in fields)]
        require(len(matches) == 1, 'one fixture response per frozen request tuple')
        entry = matches[0]
        require(type(entry['status']) is int and 200 <= entry['status'] <= 599 and type(entry['body']) is str and len(entry['body'].encode()) <= 16384, 'supported bounded fixture response')
        disposition, payload = 'protocol_invalid', None
        if entry['status'] != 200:
            return entry, 'http_rejected', None, 'unavailable'
        headers = {k.lower(): v for k, v in entry['headers'].items()}
        require(len(headers) == len(entry['headers']), 'duplicate fixture header')
        media = HeaderMessage()
        media['content-type'] = headers.get('content-type', '')
        if (media.get_content_type() != 'application/json' or media.get_content_charset() not in (None, 'utf-8')
                or headers.get('content-encoding', 'identity').lower() != 'identity' or 'content-range' in headers):
            return entry, disposition, payload, 'unavailable'
        class Number(str):
            pass
        def pairs(items):
            require(len(dict(items)) == len(items), 'duplicate fixture body key')
            return dict(items)
        def numeric(number, low, high):
            return (isinstance(number, Number) and len(number) <= 32 and re.fullmatch(r'-?(0|[1-9][0-9]{0,2})(\.[0-9]+)?([eE][+-]?[0-9]{1,2})?', number)
                    and decimal.Decimal(low) <= decimal.Decimal(number) <= decimal.Decimal(high))
        try:
            body = json.loads(entry['body'], parse_float=Number, parse_int=Number, object_pairs_hook=pairs)
            zone = ('GMT', 'UTC', 'Etc/GMT', 'Etc/UTC')
            if body.get('error', False) is not False or body['utc_offset_seconds'] != Number('0') or body['timezone'] not in zone or body.get('timezone_abbreviation', 'GMT') not in zone:
                return entry, disposition, payload, 'unavailable'
            lat, lon = body['latitude'], body['longitude']
            if not numeric(lat, -90, 90) or not numeric(lon, -180, 180):
                return entry, disposition, payload, 'unavailable'
            qlat, qlon = float(candidate['latitude']['Text']), float(candidate['longitude']['Text'])
            radians = math.pi / 180
            haversine = math.sin((qlat - float(lat)) * radians / 2) ** 2 + math.cos(qlat * radians) * math.cos(float(lat) * radians) * math.sin((qlon - float(lon)) * radians / 2) ** 2
            if haversine > math.sin(50 / (2 * 6371)) ** 2:
                return entry, disposition, payload, 'unavailable'
            unit = '°C' if self.config['units'] == 'celsius' else '°F'
            units = body['hourly_units']
            if units.get('time') != 'iso8601' or units.get('temperature_2m') != unit or units.get('weather_code') != 'wmo code':
                return entry, disposition, payload, 'unavailable'
            hourly = body['hourly']
            times, temperatures, codes = hourly['time'], hourly['temperature_2m'], hourly['weather_code']
            wanted_times = [candidate[k]['Text'][:-4] for k in ('start_hour', 'end_hour')]
            if len(times) != 2 or len(temperatures) != 2 or len(codes) != 2 or sorted(times) != wanted_times:
                return entry, disposition, payload, 'unavailable'
            low, high = (-150, 150) if unit == '°C' else (-238, 302)
            if any(t is not None and not numeric(t, low, high) for t in temperatures):
                return entry, disposition, payload, 'unavailable'
            if any(c is not None and (not isinstance(c, Number) or not re.fullmatch(r'-?(0|[1-9][0-9]*)', c)) for c in codes):
                return entry, disposition, payload, 'unavailable'
            valid_codes = (0, 1, 2, 3, 45, 48, 51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 71, 73, 75, 77, 80, 81, 82, 85, 86, 95, 96, 97, 99)
            if any(t is None for t in temperatures) or any(c is None or int(c) not in valid_codes for c in codes):
                return entry, 'unavailable', payload, 'unavailable'
            first, last = times.index(wanted_times[0]), times.index(wanted_times[1])
            payload = {k: value(v) for k, v in sorted({'source': 'controlled Open-Meteo fixture', 'grid_latitude': str(lat), 'grid_longitude': str(lon),
                'temperature_unit': unit, 'start_hour': candidate['start_hour']['Text'], 'end_hour': candidate['end_hour']['Text'],
                'start_temperature': str(temperatures[first]), 'end_temperature': str(temperatures[last]), 'start_weather_code': int(codes[first]), 'end_weather_code': int(codes[last])}.items())}
            allowed = {row['code']['Number'] for row in self.weather_data['allowed_weather_codes']['List']}
            suitability = 'suitable' if int(codes[first]) in allowed and int(codes[last]) in allowed else 'unsuitable'
            return entry, 'validated', payload, suitability
        except (ValueError, TypeError, KeyError, InvariantError, decimal.InvalidOperation):
            return entry, disposition, None, 'unavailable'

    def fixture_outcomes(self):
        self.fixture_results = [self.fixture_response(c) for c in self.candidates]
        self.outcomes = tuple(result[3] for result in self.fixture_results)

    def adapters(self):
        from urllib.parse import urlencode
        self.adapter_attempts, self.adapter_results = {}, {}
        expected_reservations, expected_charges = [], []
        expected_adapter_admissions, expected_settlements = [], []
        if self.member:
            order = self.fact_order('admission', self.call['ID'])
            charge = {'RootID': self.roots['weather']['ID'], 'OperationID': self.call['ID'], 'Kind': 'invocation', 'Invocations': 1, 'CreatedRuns': 1, 'Effects': 0, 'DecisionOrder': order}
            expected_charges.append({'operation_id': self.call['ID'], 'kind': 'invocation', 'root_id': self.roots['weather']['ID'], 'decision_order': order, 'descriptor': charge})
        attempts = {r['intent_id']: r for r in self.tables['adapter_attempts']}
        results = {r['intent_id']: r for r in self.tables['adapter_results']}
        dispatches = {r['intent_id']: r for r in self.tables['adapter_dispatch_admissions']}
        settlements = {r['intent_id']: r for r in self.tables['settlements']}
        forecast_ids = {i['ID'] for i in self.forecasts}
        require(set(attempts) <= forecast_ids and set(results) <= set(attempts) and set(dispatches) <= set(attempts) and set(settlements) <= set(results), 'complete original adapter chain; missing durable result or stray operation')
        require(len(settlements) == len(self.tables['settlements']), 'duplicate original adapter settlement')
        for index, intent in enumerate(self.forecasts):
            operation = intent['ID']
            if operation not in attempts:
                continue
            require(operation in self.auth and self.auth[operation]['Allowed'], 'adapter requires original allowed authorization')
            attempt = decoded(attempts[operation]['descriptor'])
            declaration = self.programs[self.runs[intent['Run']['Subject']]['program_id']]['Author']['Effects']['forecast']
            request = {'ID': operation, 'Key': go_json([2, self.namespace, operation]).decode(), 'Capability': 'open-meteo-hourly', 'Effect': 'forecast',
                'Input': declaration['Input'], 'Output': declaration['Output'], 'Payload': intent['Payload'], 'Assets': []}
            query = {'cell_selection': 'nearest', 'end_hour': intent['Payload']['end_hour']['Text'][:-4], 'hourly': 'temperature_2m,weather_code',
                'latitude': intent['Payload']['latitude']['Text'], 'longitude': intent['Payload']['longitude']['Text'], 'start_hour': intent['Payload']['start_hour']['Text'][:-4],
                'temperature_unit': self.config['units'], 'timeformat': 'iso8601', 'timezone': 'GMT'}
            prepared = {'Request': request, 'Digest': '', 'Protocol': 'open-meteo/hourly/v1', 'Endpoint': self.config['endpoint'], 'Model': '', 'MaxOutputTokens': 0, 'MaxResponseBytes': 16384,
                'Wire': self.config['endpoint'] + '?' + urlencode(sorted(query.items()))}
            prepared['Digest'] = hashlib.sha256(go_json(prepared)).hexdigest()
            expected_attempt = {'Intent': intent, 'AuthorizationID': self.auth[operation]['ID'], 'Prepared': prepared, 'Allowed': True, 'Reason': 'authorized'}
            require(attempt == expected_attempt, 'original adapter prepared request/wire/schema')
            self.adapter_attempts[operation] = attempt
            self.cover('adapter_attempts', attempts[operation], 'original fixture request and immutable adapter attempt')
            reservation_order = self.fact_order('reservation', operation)
            reservation = {'OperationID': operation, 'RootID': self.roots['weather']['ID'], 'Kind': 'adapter', 'Run': intent['Run'], 'Sequence': intent['Sequence'],
                'Allowed': True, 'Reason': 'authorized', 'AttemptDigest': identity('alan/attempt-row/v1', attempt)}
            expected_reservations.append({'operation_id': operation, 'root_id': self.roots['weather']['ID'], 'descriptor': reservation})
            charge = {'RootID': self.roots['weather']['ID'], 'OperationID': operation, 'Kind': 'effect', 'Invocations': 0, 'CreatedRuns': 0, 'Effects': 1, 'DecisionOrder': reservation_order}
            expected_charges.append({'operation_id': operation, 'kind': 'effect', 'root_id': self.roots['weather']['ID'], 'decision_order': reservation_order, 'descriptor': charge})
            self.event_for_intent(intent, 'reserve', reservation_order)
            if operation not in dispatches:
                require(operation not in results, 'durable result without dispatch gate')
                continue
            dispatch = dispatches[operation]
            require(dispatch['prepared_digest'] == prepared['Digest'] and reservation_order <= dispatch['after_fact'] <= len(self.facts), 'original dispatch prepared digest/fact prefix')
            self.cover('adapter_dispatch_admissions', dispatch, 'dispatch gate only; no endpoint contact inference')
            self.event_for_intent(intent, 'dispatch', after=dispatch['after_fact'], before=dispatch['after_fact'] + 1)
            if operation not in results:
                continue
            result = decoded(results[operation]['descriptor'])
            entry, disposition, payload, _ = self.fixture_results[index]
            candidate = {'ID': operation, 'Key': request['Key'], 'Disposition': disposition, 'Payload': payload, 'HTTPStatus': entry['status'], 'RequestID': '', 'ResponseID': '', 'ReturnedModel': '', 'Response': entry['body'], 'Usage': None}
            defaults = {name: value(0 if field['Kind'] == 'int' else '') for name, field in declaration['Output']['Record'].items()}
            values = dict(payload if payload is not None else defaults, disposition=value(disposition), valid=value(disposition == 'validated'))
            values = dict(sorted(values.items()))
            expected_result = {'Digest': '', 'Candidate': candidate, 'Values': values, 'Disposition': disposition}
            expected_result['Digest'] = hashlib.sha256(go_json(expected_result)).hexdigest()
            require(result == expected_result, 'original native fixture response and derived values')
            self.adapter_results[operation] = result
            self.cover('adapter_results', results[operation], 'native fixture response, exact result digest and suitability')
            self.event_for_intent(intent, 'result', self.fact_order('external_result', operation))
            if operation not in settlements:
                continue
            transport = go_json(['adapter-delivery', operation]).decode()
            observation_id = go_json(['observation', 'adapter', operation]).decode()
            descriptor = {'TransportID': transport, 'Kind': 'adapter', 'LogicalID': operation, 'Source': run_id('open-meteo-hourly'), 'Target': intent['Run']['Subject'], 'Generation': 1,
                'Obligation': 'forecast', 'Operation': operation, 'Payload': values, 'IntentID': operation, 'TerminalSequence': 0}
            admission_result = {'TransportID': transport, 'Accepted': True, 'Reason': 'settled', 'Bound': intent['Run'], 'ObservationID': observation_id}
            expected_adapter_admissions.append({'id': go_json(['admission', 'adapter', transport]).decode(), 'domain': 'adapter', 'transport_id': transport,
                'descriptor': go_json(descriptor), 'result': go_json(admission_result), 'observation_id': observation_id})
            expected_settlements.append({'id': go_json(['adapter-outcome', operation]).decode(), 'intent_id': operation, 'descriptor': result, 'observation_id': observation_id})
            observation = {'ID': observation_id, 'Run': intent['Run'], 'Kind': 'adapter', 'Provenance': 'runtime-adapter', 'Source': 'open-meteo-hourly', 'SourceGeneration': 1,
                'Obligation': 'forecast', 'Operation': operation, 'Payload': values}
            selection = self.observation(observation, 'atomic adapter settlement')
            self.event_for_intent(intent, 'settle', after=self.fact_order('external_result', operation))
            if selection:
                self.event_for_intent(intent, 'select', self.fact_order('execution', selection))
        actual_adapter_admissions = [r for r in self.tables['admissions'] if r['domain'] == 'adapter']
        require(sorted(actual_adapter_admissions, key=lambda r: r['id']) == sorted(expected_adapter_admissions, key=lambda r: r['id']), 'adapter settlement/admission atomic bijection')
        for row in actual_adapter_admissions:
            self.cover('admissions', row, 'adapter settlement/admission atomic group')
        self.descriptor_rows('settlements', expected_settlements, 'original result/settlement bijection')
        self.descriptor_rows('composition_reservations', expected_reservations, 'original adapter reservation bijection')
        self.descriptor_rows('composition_charges', expected_charges, 'exact original ordered charge inventory')

    def payloads(self):
        assessment = []
        for index, intent in enumerate(self.forecasts):
            assessment.append(dict(sorted(dict(self.candidates[index], forecast_operation=value(intent['ID']), weather_status=value(self.outcomes[index])).items())))
        self.response = {'assessments': value(assessment), 'proposal_id': self.proposal_id}
        outcome, selected = MODEL['policy'](self.outcomes)
        for intent in self.intents.values():
            if intent['Kind'] != 'message':
                continue
            symbol = self.symbols[intent['ID']]
            if symbol.endswith(':proposal'):
                expected = {'candidates': value(self.candidates), 'proposal_id': self.proposal_id}
                require(intent['Payload'] == expected, 'original proposal frozen candidates')
            elif symbol.endswith((':assessment', ':reply')):
                require(len(self.forecasts) == len(self.candidates) and intent['Payload'] == self.response, 'exact ordered assessment/original handles')
                if symbol.endswith(':assessment') and self.variant == 'delegated':
                    deliveries = self.tables['composition_outcome_deliveries']
                    selection = self.selected.get(deliveries[0]['observation_id']) if len(deliveries) == 1 else None
                    require(selection is not None and selection[0] == 'weather' and selection[1] < intent['Sequence']
                        and self.fact_order('execution', selection) < self.fact_order('execution', ('weather', intent['Sequence'])),
                        'public assessment before original invocation outcome selection')
            else:
                expected = {'candidates': value(self.candidates), 'outcome': value(outcome), 'proposal_id': self.proposal_id,
                    'selection': value([assessment[selected]] if selected >= 0 else [])}
                require(intent['Payload'] == expected, 'original decision deterministic policy/selection')

    def terminals(self):
        expected_terminals, expected_freezes = [], []
        self.scopes = {}
        for name, run in self.runs.items():
            actor = self.actors[name]
            root = self.roots['outing' if actor == 'outing' else 'weather']['ID']
            scope = self.invocation['ID'] if actor == 'helper' else 'root:' + root
            self.scopes[scope] = (name, root)
            projection = decoded(run['projection'])
            if projection['Status'] == 'open':
                continue
            require(projection['Status'] in ('succeeded', 'failed'), 'unsupported terminal status')
            expected_terminals.append({'subject': name, 'generation': 1, 'sequence': run['sequence'], 'outcome': go_json(projection),
                'destinations': b'[]', 'delivery_status': b'{}', 'operation_id': None, 'source_revision': 1})
            fact = self.facts[('execution', name, run['sequence'])]
            order = self.fact_order('freeze', scope)
            require(order == fact['Order'] + 1, 'atomic terminal execution/freeze adjacency')
            operations = [i for i in self.intents.values() if i['Run']['Subject'] == name]
            freeze = {'InvocationID': scope, 'Trigger': fact['ID'], 'Run': run_id(name), 'Sequence': run['sequence'], 'DecisionOrder': order, 'Status': projection['Status'],
                'ReplyIntents': sorted(i['ID'] for i in operations if self.symbols[i['ID']] == 'message:helper:reply'),
                'InvocationIntents': sorted(i['ID'] for i in operations if i['Kind'] == 'invocation'), 'ExternalIntents': sorted(i['ID'] for i in operations if i['Kind'] == 'adapter')}
            expected_freezes.append({'scope_id': scope, 'root_id': root, 'descriptor': freeze})
            if actor == 'helper':
                self.add_event('helper.end_freeze', MODEL['CALL'], 'helper', binding=MODEL['BINDING'], disposition=projection['Status'], order=order)
            else:
                self.add_event('root.end_freeze', 'root:' + actor, actor, disposition=projection['Status'], order=order)
        self.rows_equal('terminal_obligations', expected_terminals, 'exact original terminal obligations with no routes')
        self.descriptor_rows('composition_freezes', expected_freezes, 'complete frozen originals including committed reply debt')
        expected_publications, expected_results, expected_deliveries = [], [], []
        if self.tables['composition_publications']:
            require(self.admission is not None and len(self.tables['composition_publications']) == 1, 'one publication for original admitted/refused call')
            accepted = self.member is not None
            order = self.fact_order('outcome', self.call['ID'])
            publication = {'ID': identity('alan/invocation-outcome/v1', self.invocation['ID']), 'InvocationID': self.invocation['ID'], 'OperationID': self.call['ID'],
                'TemplateID': self.call['Target'], 'Caller': run_id('weather'), 'Status': 'succeeded' if accepted else 'rejected', 'Reason': 'member_succeeded' if accepted else 'invocation_limit',
                'ResponsePresent': accepted, 'Response': self.response if accepted else None, 'DecisionOrder': order}
            expected_publications.append({'operation_id': self.call['ID'], 'descriptor': publication})
            if accepted:
                expected_results.append({'invocation_id': self.invocation['ID'], 'decision_order': order, 'descriptor': publication})
            observation_id = identity('alan/invocation-result-observation/v1', publication['ID'])
            payload = {'status': value(publication['Status']), 'reason': value(publication['Reason']), 'response_present': value(accepted)}
            if accepted:
                payload.update({'response.' + k: v for k, v in self.response.items()})
            observation = {'ID': observation_id, 'Run': run_id('weather'), 'Kind': 'invocation', 'Provenance': 'runtime-invocation', 'Source': self.call['Target'], 'SourceGeneration': 1,
                'Obligation': 'assess', 'Operation': self.call['ID'], 'Payload': dict(sorted(payload.items()))}
            selection = self.observation(observation, 'atomic invocation outcome publication/delivery')
            expected_deliveries.append({'operation_id': self.call['ID'], 'observation_id': observation_id})
            binding = MODEL['BINDING'] if accepted else ''
            self.add_event('invocation.publish_deliver', MODEL['CALL'], 'weather', 'weather', binding=binding, disposition=publication['Status'], order=order)
            if selection:
                self.add_event('invocation.outcome.select', MODEL['CALL'], 'weather', binding=binding, order=self.fact_order('execution', selection))
        self.descriptor_rows('composition_publications', expected_publications, 'original invocation publication response')
        self.descriptor_rows('composition_results', expected_results, 'accepted result copy iff accepted publication')
        self.rows_equal('composition_outcome_deliveries', expected_deliveries, 'atomic original outcome delivery link')
        expected_roots = []
        for row in self.tables['composition_root_publications']:
            require(row['scope_id'] in self.scopes and row['scope_id'].startswith('root:'), 'original root publication scope')
            name, root = self.scopes[row['scope_id']]
            status = decoded(self.runs[name]['projection'])['Status']
            require(status in ('succeeded', 'failed'), 'root publication before terminal')
            order = self.fact_order('outcome', row['scope_id'])
            publication = {'ID': identity('alan/root-outcome/v1', root), 'InvocationID': row['scope_id'], 'OperationID': row['scope_id'], 'TemplateID': '', 'Caller': run_id(name),
                'Status': status, 'Reason': 'member_succeeded' if status == 'succeeded' else 'member_failed', 'ResponsePresent': False, 'Response': None, 'DecisionOrder': order}
            expected_roots.append({'scope_id': row['scope_id'], 'root_id': root, 'decision_order': order, 'descriptor': publication})
            self.add_event('root.publish', 'root:' + name, name, disposition=status, order=order)
        self.descriptor_rows('composition_root_publications', expected_roots, 'original root publication')

    def validate_facts(self):
        expected = {}
        def add(kind, operation, subject, sequence, root, source):
            key = (kind, subject, sequence) if kind == 'execution' else (kind, operation)
            require(key in self.facts, 'missing source-bound fact: ' + repr(key))
            identity_input = [run_id(subject), sequence] if kind == 'execution' else operation
            fact = {'Version': 'alan/composition-fact/v1', 'ID': identity('alan/fact/' + kind + '/v1', identity_input), 'Kind': kind,
                'Order': self.facts[key]['Order'], 'RootIDs': [root], 'Run': run_id(subject), 'Sequence': sequence, 'OperationID': operation,
                'RecordDigest': identity('alan/' + ('history' if kind == 'execution' else kind) + '-row/v1', source)}
            expected[key] = fact
        for (subject, seq), row in self.history.items():
            root = self.roots['outing' if subject == 'outing' else 'weather']['ID']
            source = {'Run': run_id(subject), 'Sequence': seq, 'Epoch': row['epoch'], 'Revision': row['source_revision'], 'Selected': row['selected_observation'] or '',
                'Successor': decoded(row['successor']), 'Intents': decoded(row['intents'])}
            add('execution', '', subject, seq, root, source)
        for kind, table, keyfield, descriptor_field in (('authorization', 'attempts', 'intent_id', 'result'), ('external_result', 'adapter_results', 'intent_id', 'descriptor'),
                ('admission', 'composition_admission_decisions', 'operation_id', 'descriptor'), ('reservation', 'composition_reservations', 'operation_id', 'descriptor'),
                ('reply', 'composition_replies', 'operation_id', 'descriptor'), ('freeze', 'composition_freezes', 'scope_id', 'descriptor'),
                ('outcome', 'composition_publications', 'operation_id', 'descriptor'), ('outcome', 'composition_root_publications', 'scope_id', 'descriptor')):
            for row in self.tables[table]:
                source = decoded(row[descriptor_field])
                operation = row[keyfield]
                if kind == 'freeze':
                    subject, seq, root = source['Run']['Subject'], source['Sequence'], row['root_id']
                elif operation.startswith('root:'):
                    subject, seq, root = source['Caller']['Subject'], 0, row['root_id']
                else:
                    intent = self.intents[operation]
                    subject, seq = intent['Run']['Subject'], intent['Sequence']
                    root = self.roots['outing' if subject == 'outing' else 'weather']['ID']
                add(kind, operation, subject, seq, root, source)
        for subject in self.runs:
            orders = [self.history_order[(subject, seq)] for seq in range(1, self.runs[subject]['sequence'] + 1)]
            require(orders == sorted(orders), 'run-local execution facts must preserve local sequence order')
        require(expected == self.facts, 'complete source-bound composition fact/digest bijection')
        for row in self.tables['composition_facts']:
            self.cover('composition_facts', row, 'exact source digest and globally ordered fact')

    def operand(self, operand, data):
        require(operand['Offset'] is None, 'unsupported continuation operand offset')
        result = data.get(operand['Ref']) if operand['Ref'] else operand['Literal']
        require(result is not None, 'unresolved continuation operand: ' + operand['Ref'])
        return result

    def receive_match(self, subject, projection, instruction):
        match = instruction['Match']
        result = {'Kind': match['Kind'], 'Provenance': match['Provenance']}
        for source, target, kind, field in (('Source', 'Source', 'string', 'Text'), ('Generation', 'SourceGeneration', 'int', 'Number'),
                ('Obligation', 'Obligation', 'string', 'Text'), ('Operation', 'Operation', 'string', 'Text')):
            resolved = self.operand(match[source], projection['Data'])
            require(resolved['Kind'] == kind, 'receive operand kind: ' + source)
            result[target] = resolved[field]
        if self.actors[subject] == 'helper' and result['Kind'] == 'message' and result['Source'] == 'caller':
            result.update(Source='weather', Provenance='runtime-invocation-request', Operation=self.call['ID'])
        return result

    def historical_receives(self):
        for observation_id, (subject, sequence) in self.selected.items():
            before = decoded(self.runs[subject]['initial']) if sequence == 1 else decoded(self.history[(subject, sequence - 1)]['successor'])
            instruction = self.programs[self.runs[subject]['program_id']]['Code'][before['PC']]
            require(instruction['Op'] == 'receive', 'historical selected receive boundary')
            match = self.receive_match(subject, before, instruction)
            observation = self.observations[observation_id]
            require(observation['Run'] == run_id(subject) and all(observation[k] == v for k, v in match.items()),
                    'historical receive exact match/original correlation: ' + subject + ':' + str(sequence))

    def next_boundary(self, subject, projection):
        """Walk only the declared pure opcode subset, stopping before any event.

        This resolves the current continuation, not a general VM replay or proof
        of historical pure data updates. Bounds and unsupported opcodes fail closed.
        """
        import copy
        current = copy.deepcopy(projection)
        program = self.programs[self.runs[subject]['program_id']]['Code']
        pure_path = []
        for _ in range(4096):
            pc, data = current['PC'], current['Data']
            require(current['Status'] == 'open' and type(pc) is int and 0 <= pc < len(program), 'continuation program counter/status')
            instruction = program[pc]
            op = instruction['Op']
            if op == 'receive':
                return dict(boundary='receive', **self.receive_match(subject, current, instruction)), pure_path
            if op == 'end':
                return {'boundary': 'end', 'status': instruction['Outcome']}, pure_path
            if op == 'emit':
                effect = instruction['Effect']
                resolved = {field: self.operand(effect[field], data) for field in ('Target', 'Generation', 'Obligation')}
                require(all(resolved[field]['Kind'] == kind for field, kind in (('Target', 'string'), ('Generation', 'int'), ('Obligation', 'string'))),
                        'continuation emit operand kinds')
                target = resolved['Target']['Text']
                if self.actors[subject] == 'helper' and target == 'caller':
                    target = 'weather'
                return {'boundary': 'emit', 'Kind': effect['Kind'], 'Target': target,
                    'Generation': resolved['Generation']['Number'], 'Obligation': resolved['Obligation']['Text'],
                    'Payload': {k: self.operand(v, data) for k, v in effect['Payload'].items()}}, pure_path
            pure_path.append(pc)
            current['PC'] = pc + 1
            cursors = current.setdefault('Cursors', {})
            if op == 'set':
                data[instruction['Dst']] = copy.deepcopy(self.operand(instruction['A'], data))
            elif op == 'branch':
                equal = self.operand(instruction['A'], data) == self.operand(instruction['B'], data)
                current['PC'] = instruction['Target'] if equal else instruction['Otherwise']
            elif op == 'jump':
                current['PC'] = instruction['Target']
            elif op == 'foreach':
                snapshot = self.operand(instruction['A'], data)
                require(snapshot['Kind'] == 'list', 'continuation foreach list')
                cursors[str(pc)] = {'Snapshot': copy.deepcopy(snapshot), 'Index': 0}
            elif op in ('next', 'loop'):
                key = str(instruction['Target'])
                require(key in cursors, 'continuation durable cursor')
                cursor = cursors[key]
                rows = cursor['Snapshot']['List']
                require(type(cursor['Index']) is int and 0 <= cursor['Index'] <= len(rows), 'continuation cursor index')
                if op == 'loop':
                    current['PC'] = instruction['Target'] + 1
                elif cursor['Index'] == len(rows):
                    current['PC'] = program[instruction['Target']]['Otherwise']
                    del cursors[key]
                else:
                    data.update({destination: copy.deepcopy(rows[cursor['Index']][field]) for field, destination in instruction['Bind'].items()})
                    cursor['Index'] += 1
            elif op == 'append':
                collection = data[instruction['Dst']]
                require(collection['Kind'] == 'list', 'continuation append list')
                collection['List'].append({field: copy.deepcopy(self.operand(operand, data)) for field, operand in instruction['Updates'].items()})
            else:
                require(False, 'unsupported pure continuation opcode: ' + op)
            current['Cursors'] = {k: v for k, v in cursors.items() if int(k) < current['PC'] < program[int(k)]['Otherwise']}
        require(False, 'bounded pure continuation exceeded')

    def expected_boundary(self, config, state, actor):
        position = getattr(state, actor)
        def receive(kind, provenance, source, obligation, operation=''):
            return {'boundary': 'receive', 'Kind': kind, 'Provenance': provenance, 'Source': source, 'SourceGeneration': 1, 'Obligation': obligation, 'Operation': operation}
        def emit(kind, target, obligation, payload):
            return {'boundary': 'emit', 'Kind': kind, 'Target': target, 'Generation': 1, 'Obligation': obligation, 'Payload': payload}
        def message(name):
            if name == 'proposal':
                return emit('message', 'weather', 'outing-proposal', {'candidates': value(self.candidates), 'proposal_id': self.proposal_id})
            if name == 'decision':
                outcome, selected = MODEL['policy'](self.outcomes)
                assessments = self.response['assessments']['List']
                return emit('message', 'weather', 'outing-decision', {'candidates': value(self.candidates), 'proposal_id': self.proposal_id,
                    'outcome': value(outcome), 'selection': value([assessments[selected]] if selected >= 0 else [])})
            return emit('message', 'weather' if name == 'reply' else 'outing', 'assess' if name == 'reply' else 'outing-assessment', self.response)
        if position in ('ready_end', 'ready_fail'):
            status = 'succeeded' if actor == 'helper' else 'failed' if position == 'ready_fail' or MODEL['policy'](self.outcomes)[0] == 'unavailable' else 'succeeded'
            return {'boundary': 'end', 'status': status}
        if actor == 'outing':
            if position == 'start':
                return message('proposal')
            if position == 'ready_decision':
                return message('decision')
            if position == 'await_assessment':
                return receive('message', 'ordinary-message', 'weather', 'outing-assessment')
        elif actor == 'weather':
            if position == 'await_proposal':
                return receive('message', 'ordinary-message', 'outing', 'outing-proposal')
            if position in ('await_decision', 'await_decision_first'):
                return receive('message', 'ordinary-message', 'outing', 'outing-decision')
            if position == 'await_helper':
                return receive('invocation', 'runtime-invocation', self.templates[0]['ID'], 'assess', self.call['ID'])
            if position == 'ready_assessment':
                return message('assessment')
            if position == 'work' and config.variant == 'delegated':
                return emit('invocation', self.templates[0]['ID'], 'assess', {'allowed_weather_codes': self.weather_data['allowed_weather_codes'],
                    'candidates': value(self.candidates), 'proposal_id': self.proposal_id, 'units': self.weather_data['units']})
        elif position == 'await_request':
            return receive('message', 'runtime-invocation-request', 'weather', 'assess', self.call['ID'])
        if position == 'work':
            pending = next((f for f in state.forecasts if f.stage < 7), None)
            if pending is None:
                return message('reply' if actor == 'helper' else 'assessment')
            if pending.stage:
                original = next(op for op, symbol in self.symbols.items() if symbol == pending.operation)
                return receive('adapter', 'runtime-adapter', 'open-meteo-hourly', 'forecast', original)
            candidate = self.candidates[pending.candidate]
            return emit('adapter', 'open-meteo-hourly', 'forecast', dict({k: candidate[k] for k in ('end_hour', 'latitude', 'longitude', 'start_hour')}, units=self.weather_data['units']))
        require(False, 'unsupported concrete/model continuation: ' + actor + ':' + position)

    def final_continuations(self, config, state):
        relation = {}
        for subject, actor in self.actors.items():
            projection = decoded(self.runs[subject]['projection'])
            position = getattr(state, actor)
            if projection['Status'] != 'open':
                require(projection['Status'] == position, 'concrete final status/model mismatch')
                relation[subject] = {'model_position': position, 'status': position}
                continue
            try:
                actual, pure_path = self.next_boundary(subject, projection)
                expected = self.expected_boundary(config, state, actor)
                require(actual == expected, 'resolved boundary differs: ' + json.dumps({'actual': actual, 'expected': expected}, sort_keys=True))
            except (InvariantError, KeyError, TypeError, IndexError) as error:
                raise InvariantError('concrete final continuation/model boundary: ' + subject + ':' + position + ': ' + str(error)) from error
            relation[subject] = {'model_position': position, 'retained_pc': projection['PC'], 'pure_pc_path': pure_path, 'next_boundary': actual}
        return relation

    def concrete_debts(self):
        """Derive the protocol debt ledger from raw originals, independently of State."""
        debt = []
        acknowledgments = {r['intent_id'] for r in self.tables['message_acknowledgments']}
        selected_operations = {o['Operation'] for o in self.observations.values() if o['ID'] in self.selected}
        selected_messages = {r['id'] for r in self.tables['admissions'] if r['observation_id'] in self.selected}
        selected_symbols = set()
        for intent in self.intents.values():
            if intent['Kind'] != 'message':
                continue
            operation, symbol = intent['ID'], self.symbols[intent['ID']]
            admission = self.message_admissions.get(operation)
            if operation not in self.auth:
                debt.append(symbol + ':authorization')
            if not admission:
                debt.append(symbol + ':admission')
            if operation not in acknowledgments:
                debt.append(symbol + ':acknowledgment')
            if symbol == 'message:helper:reply':
                if not admission:
                    debt.append(symbol + ':recorded_reply')
            elif admission and admission['id'] in selected_messages:
                selected_symbols.add(symbol)
            else:
                debt.append(symbol + ':selection')
        if self.call:
            if not self.admission:
                debt.append(MODEL['CALL'] + ':admission')
            if not self.tables['composition_publications']:
                debt.append(MODEL['CALL'] + ':publication_delivery')
            outcome = self.tables['composition_outcome_deliveries']
            if not outcome or outcome[0]['observation_id'] not in self.selected:
                debt.append(MODEL['CALL'] + ':selection')
            if self.member and self.tables['composition_requests'][0]['observation_id'] not in self.selected:
                debt.append(MODEL['CALL'] + ':request_selection')
        dispatches = {r['intent_id'] for r in self.tables['adapter_dispatch_admissions']}
        settlements = {r['intent_id'] for r in self.tables['settlements']}
        for intent in self.forecasts:
            op, symbol = intent['ID'], self.symbols[intent['ID']]
            next_stage = ('authorize' if op not in self.auth else 'reserve' if op not in self.adapter_attempts else 'dispatch' if op not in dispatches
                else 'result' if op not in self.adapter_results else 'settle' if op not in settlements else 'select' if op not in selected_operations else '')
            if next_stage:
                debt.append(symbol + ':' + next_stage)
        if self.variant != 'confirm-first' and not (self.admission and not self.member):
            debt += ['candidate:' + str(index) + ':forecast' for index in range(len(self.forecasts), len(self.candidates))]
        if 'message:weather:assessment' not in selected_symbols:
            debt.append('outing:assessment_receive')
        if 'message:outing:decision' not in selected_symbols:
            debt.append('weather:decision_receive')
        for subject, actor in self.actors.items():
            if decoded(self.runs[subject]['projection'])['Status'] == 'open':
                debt.append(actor + ':local_end')
        published = {r['root_id'] for r in self.tables['composition_root_publications']}
        for actor, root in self.roots.items():
            if root['ID'] not in published:
                debt.append('root:' + actor + ':publication')
        return tuple(sorted(debt))

    def correspondence(self):
        """Find a compatible model placement, without fabricating a total chronology."""
        from dataclasses import asdict
        config = MODEL['Config'](self.variant, self.outcomes, self.deployment['helper_admission'], 'broad')
        events = self.events
        prerequisites = [0] * len(events)
        for index, current in enumerate(events):
            for other_index, other in enumerate(events):
                if index == other_index:
                    continue
                earlier = (current['order'] is not None and other['order'] is not None and other['order'] < current['order'])
                earlier |= other['order'] is not None and other['order'] <= current['after']
                earlier |= other['before'] is not None and current['order'] is not None and current['order'] >= other['before']
                if earlier:
                    prerequisites[index] |= 1 << other_index
        all_bits = (1 << len(events)) - 1
        seen = set()
        witness = {'consumed': 0}
        def visit(state, consumed, path):
            if consumed == all_bits:
                return state, path
            key = (state, consumed)
            if key in seen:
                return None
            seen.add(key)
            if bin(consumed).count('1') > witness['consumed']:
                witness.update(consumed=bin(consumed).count('1'), path=path, state=MODEL['state_view'](config, state))
            available = {entry['event']: index for index, entry in enumerate(events) if not consumed & (1 << index) and prerequisites[index] & consumed == prerequisites[index]}
            for event, successor in MODEL['successors'](config, state):
                if event not in available:
                    continue
                index = available[event]
                result = visit(successor, consumed | (1 << index), path + [index])
                if result is not None:
                    return result
            return None
        matched = visit(MODEL['initial'](config), 0, [])
        require(matched is not None, 'no legal partial-order model path; witness=' + json.dumps(witness, sort_keys=True))
        self.state, path = matched
        MODEL['validate_state'](config, self.state)
        continuations = self.final_continuations(config, self.state)
        concrete_debts = self.concrete_debts()
        require(concrete_debts == MODEL['debts'](config, self.state), 'concrete final debt/model ledger')
        for table, rows in self.tables.items():
            require(len(self.coverage[table]) == len(rows), 'unclassified raw rows: ' + table)
            for item in self.raw['inventory'][table]['rows']:
                key = tuple(item['key'][k] for k in self.raw['inventory'][table]['primary_key'])
                item['rule'] = self.coverage[table][key]
        graph = MODEL['explore'](config)
        require(self.state in graph.states, 'concrete state absent from independent graph')
        index = graph.states.index(self.state)
        complete = MODEL['successful_mark'](config, self.state)
        diagnostic = MODEL['diagnostic_terminal'](config, self.state)
        unknown = [f.operation for f in self.state.forecasts if f.stage == 4]
        refusal = self.state.invocation.admission == 'refused' or any(m.authorization == 'deny' for m in self.state.messages)
        classification = 'complete' if complete else 'diagnostic_terminal' if diagnostic else 'unknown_forecast' if unknown else 'refused' if refusal else 'confirm_first_cycle' if self.variant == 'confirm-first' and self.state.messages[0].selected else 'pending'
        authorization = 'violated' if any(m.authorization == 'deny' for m in self.state.messages) else 'pending' if any(m.committed and not m.authorization and not m.binding for m in self.state.messages) else 'verified_for_retained_originals'
        return {'accepted': True, 'classification': classification, 'config': asdict(config), 'semantic_answer': list(self.state.decision),
            'successful_completion': complete, 'can_complete_under_known_fixture': index in graph.completing,
            'state': MODEL['state_view'](config, self.state), 'debts': MODEL['debts'](config, self.state),
            'original_mapping': dict(sorted(self.symbols.items())), 'charges': [decoded(r['descriptor']) for r in sorted(self.tables['composition_charges'], key=lambda r: r['decision_order'])],
            'events': [dict(event=asdict(e['event']), fact_order=e['order'], after_fact=e['after'], before_fact=e['before']) for e in events],
            'compatible_model_placement': path, 'placement_claim': 'One legal partial-order placement, not an observed total chronology.',
            'inventory': self.raw['inventory'], 'preclose_authorization': authorization,
            'final_continuations': continuations, 'concrete_debts': concrete_debts,
            'decision_sole_pending_at_commit': 'not established without separately captured decision-commit cut',
            'endpoint_contact': 'not established; raw dispatch gates do not prove transport contact',
            'unknown_result': 'Missing durable result is an epistemic gap. A selected no-later-result/no-retry continuation requires an explicit environment alternative.',
            'bounded_domain': 'Schema9, exact frozen program/deployment/candidates, at most two candidates and one helper, fixed authority, one transport per original. No compiler proof, calendar proof, universal refinement or general congruence.'}


    def public_wait(self, subject):
        row = self.runs[subject]
        projection = decoded(row['projection'])
        if projection['Status'] != 'open':
            return None
        instruction = self.programs[row['program_id']]['Code'][projection['PC']]
        if instruction['Op'] != 'receive':
            return None
        match = instruction['Match']
        wait = {'Kind': match['Kind'], 'Provenance': match['Provenance'], 'Source': '', 'SourceGeneration': 0, 'Obligation': '', 'OperationID': '', 'Resolved': False, 'MatchingObservationIDs': None}
        for source, target, kind in (('Source', 'Source', 'Text'), ('Generation', 'SourceGeneration', 'Number'), ('Obligation', 'Obligation', 'Text'), ('Operation', 'OperationID', 'Text')):
            operand = match[source]
            v = projection['Data'].get(operand['Ref']) if operand['Ref'] else operand['Literal']
            if v is None:
                return wait
            wait[target] = v[kind]
        wait['Resolved'] = True
        if self.actors[subject] == 'helper' and wait['Kind'] == 'message' and wait['Source'] == 'caller':
            wait.update(Source='weather', Provenance='runtime-invocation-request', OperationID=self.call['ID'])
        matching = [o['ID'] for o in self.observations.values() if o['Run'] == run_id(subject) and o['ID'] not in self.selected and
            all(wait[a] == o[b] for a, b in (('Kind', 'Kind'), ('Provenance', 'Provenance'), ('Source', 'Source'), ('SourceGeneration', 'SourceGeneration'), ('Obligation', 'Obligation'), ('OperationID', 'Operation')))]
        wait['MatchingObservationIDs'] = sorted(matching) or None
        return wait

    def public_pending(self):
        pending = []
        def add(kind, subject, operation='', observation='', depends='', run=None, admissions=None):
            pending.append({'Kind': kind, 'Run': run_id(subject), 'OperationID': operation, 'ObservationID': observation, 'DependsOn': depends, 'DependsOnRun': run, 'AdmissionIDs': admissions})
        for subject in self.runs:
            wait = self.public_wait(subject)
            if wait:
                matching = wait['MatchingObservationIDs'] or []
                kind = 'unresolved_receive' if not wait['Resolved'] else 'matching_observation' if matching else 'waiting_' + wait['Kind']
                add(kind, subject, wait['OperationID'], matching[0] if matching else '', wait['OperationID'])
        if self.call:
            publication = self.tables['composition_publications']
            if not publication:
                add('invocation_awaiting_outcome' if self.admission else 'invocation_awaiting_admission', 'weather', self.call['ID'], depends=self.call['ID'])
                if self.admission:
                    dependencies = []
                    if self.member:
                        if decoded(self.runs[self.member['Subject']]['projection'])['Status'] == 'open':
                            dependencies.append(('invocation_member', '', self.member))
                        else:
                            recorded_replies = {r['operation_id'] for r in self.tables['composition_replies']}
                            for intent in self.intents.values():
                                if self.symbols[intent['ID']] == 'message:helper:reply' and intent['ID'] not in recorded_replies:
                                    dependencies.append(('invocation_reply_admission', intent['ID'], None))
                            for intent in self.forecasts:
                                if intent['ID'] not in self.adapter_results:
                                    dependencies.append(('invocation_external_outcome', intent['ID'], None))
                    if not dependencies:
                        dependencies.append(('invocation_publication', self.call['ID'], None))
                    for kind, original, member in dependencies:
                        add(kind, 'weather', self.call['ID'], depends=original, run=member)
            else:
                obs = self.tables['composition_outcome_deliveries'][0]['observation_id']
                if obs not in self.selected and decoded(self.runs['weather']['projection'])['Status'] == 'open':
                    add('published_unconsumed', 'weather', self.call['ID'], obs, obs)
        settlements = {r['intent_id'] for r in self.tables['settlements']}
        self.effect_stages = {}
        for intent in self.forecasts:
            operation = intent['ID']
            stage = 'settled' if operation in settlements else 'result_unsettled' if operation in self.adapter_results else 'reserved_unknown' if operation in self.adapter_attempts else 'authorized' if operation in self.auth else 'committed'
            self.effect_stages[operation] = stage
            if stage != 'settled':
                add('effect_' + stage, intent['Run']['Subject'], operation, depends=operation)
        self.root_pending = {}
        publications = {r['root_id'] for r in self.tables['composition_root_publications']}
        for name, root in self.roots.items():
            if root['ID'] in publications:
                self.root_pending[name] = ''
                continue
            status = decoded(self.runs[name]['projection'])['Status']
            depends, reason = root['ID'], ''
            if status == 'open':
                reason = 'members'
                add('root_member', name, run=run_id(name))
            elif status == 'succeeded' and name == 'weather':
                if self.call and self.admission is None:
                    reason, depends = 'invocation_admission', self.call['ID']
                elif self.member and not self.tables['composition_publications']:
                    reason, depends = 'descendant_outcome', self.call['ID']
                else:
                    unresolved = sorted(i['ID'] for i in self.forecasts if i['ID'] not in self.adapter_results)
                    if unresolved:
                        reason, depends = 'external_outcome', unresolved[0]
            self.root_pending[name] = reason
            add('root_awaiting_publication', name, depends=depends)
        acknowledged = {r['intent_id'] for r in self.tables['message_acknowledgments']}
        # Public diagnostics intentionally suppress this one raw runner debt.
        public_acknowledged = acknowledged | {r['operation_id'] for r in self.tables['composition_replies']}
        for intent in self.intents.values():
            if intent['Kind'] == 'message' and intent['ID'] not in public_acknowledged:
                admission = self.message_admissions.get(intent['ID'])
                add('message_awaiting_acknowledgment' if admission else 'message_awaiting_delivery', intent['Run']['Subject'], intent['ID'], depends=intent['ID'], admissions=[admission['id']] if admission else None)
        return pending

    def report_coverage(self, report):
        def collection(name, actual, expected, key):
            require(type(actual) is list and len(actual) == len(expected), 'report ' + name + ' coverage')
            actual_index = {key(x): x for x in actual}
            require(len(actual_index) == len(actual) and set(actual_index) == {key(x) for x in expected}, 'report ' + name + ' identity coverage')
            for wanted in expected:
                found = actual_index[key(wanted)]
                require(all(found.get(k) == v for k, v in wanted.items()), 'report ' + name + ' raw correspondence')
        require(report['namespace'] == self.namespace and report['configuration'] == self.config and report['candidates'] == self.manifest['candidates'] and report['weather_variant'] == self.deployment['weather'], 'report frozen input coverage')
        collection('operations', report['operations'], list(self.intents.values()), lambda r: r['ID'])
        permissions = [{k.title(): v for k, v in r.items()} for r in self.tables['permissions']]
        collection('permissions', report['permissions'], permissions, lambda r: tuple(r[k] for k in ('Kind', 'Source', 'Target', 'Operation', 'Revision', 'Obligation')))
        expected_history = []
        for (subject, sequence), row in self.history.items():
            before = decoded(self.runs[subject]['initial']) if sequence == 1 else decoded(self.history[(subject, sequence - 1)]['successor'])
            after = decoded(row['successor'])
            expected_history.append({'Run': run_id(subject), 'Sequence': sequence, 'Epoch': row['epoch'], 'SourceRevision': 1, 'BeforePC': before['PC'], 'AfterPC': after['PC'],
                'Instruction': self.programs[self.runs[subject]['program_id']]['Code'][before['PC']]['Op'], 'Status': after['Status'], 'SelectedObservationID': row['selected_observation'] or '',
                'OperationIDs': [i['ID'] for i in decoded(row['intents'])] or None})
        collection('history', report['history'], expected_history, lambda r: (r['Run']['Subject'], r['Sequence']))
        expected_admissions = []
        for row in self.tables['admissions']:
            d, result = decoded(row['descriptor']), decoded(row['result'])
            expected_admissions.append(dict(result, ID=row['id'], Domain=row['domain'], LogicalID=d['LogicalID'], Target=d['Target'], RequestedGeneration=d['Generation'], Source=d['Source'],
                Obligation=d['Obligation'], Correlation=d['Operation'], SourceOperationID=d['IntentID']))
        collection('admissions', report['admissions'], expected_admissions, lambda r: r['ID'])
        expected_observations = []
        for observation in self.observations.values():
            selection = self.selected.get(observation['ID'])
            provenance = observation['Provenance']
            origin = {'ordinary-message': 'message', 'runtime-adapter': 'adapter', 'runtime-invocation-request': 'invocation_request', 'runtime-invocation': 'invocation_outcome'}[provenance]
            admissions = sorted(r['id'] for r in self.tables['admissions'] if r['observation_id'] == observation['ID'])
            operation = observation['Operation']
            if provenance == 'ordinary-message':
                operation = decoded(next(r['descriptor'] for r in self.tables['admissions'] if r['observation_id'] == observation['ID']))['IntentID']
            producer = self.intents[operation] if origin in ('message', 'invocation_request') else None
            expected_observations.append(dict(observation, SelectedAt={'Run': run_id(selection[0]), 'Sequence': selection[1]} if selection else None,
                Origin=origin, SourceOperationID=operation, Admissions=admissions or None, ProducerRun=producer['Run'] if producer else None,
                ProducerReduction={'Run': producer['Run'], 'Sequence': producer['Sequence']} if producer else None))
        collection('observations', report['observations'], expected_observations, lambda r: r['ID'])
        expected_pending = self.public_pending()
        collection('pending', report['pending'], expected_pending, lambda r: json.dumps(r, sort_keys=True, separators=(',', ':')))
        expected_runs = []
        for name, row in self.runs.items():
            operations = sorted((i for i in self.intents.values() if i['Run']['Subject'] == name), key=lambda i: (i['Sequence'], i['Ordinal']))
            projection = decoded(row['projection'])
            actor = self.actors[name]
            subject_row = next(r for r in self.tables['subjects'] if r['id'] == name)
            subject_view = {'ID': name, 'Generation': 1, 'Owner': subject_row['owner'], 'Revision': 1, 'Retired': False, 'RetryCancelled': False}
            expected_runs.append({'ID': run_id(name), 'ProgramID': row['program_id'], 'Epoch': row['epoch'], 'Sequence': row['sequence'], 'Revision': 1, 'Initial': decoded(row['initial']), 'Projection': projection,
                'RootID': self.roots['outing' if actor == 'outing' else 'weather']['ID'], 'OwningInvocationID': self.invocation['ID'] if actor == 'helper' else '',
                'OriginOperationID': self.call['ID'] if actor == 'helper' else '', 'Origin': 'invocation' if actor == 'helper' else 'root',
                'OperationIDs': [i['ID'] for i in operations] or None, 'InvocationIDs': sorted(i['ID'] for i in operations if i['Kind'] == 'invocation') or None,
                'EffectIDs': sorted(i['ID'] for i in operations if i['Kind'] == 'adapter') or None, 'Wait': self.public_wait(name),
                'Semantics': self.programs[row['program_id']]['Semantics'], 'Subject': subject_view, 'Predecessor': None,
                'Instruction': self.programs[row['program_id']]['Code'][projection['PC']]['Op'] if projection['Status'] == 'open' else '',
                'LocalState': 'open' if projection['Status'] == 'open' else 'terminal_with_obligations' if any(p['Run'] == run_id(name) for p in expected_pending) else 'terminal_settled',
                'UnresolvedConsequences': sorted(i['ID'] for i in operations if i['Kind'] == 'adapter' and i['ID'] in self.adapter_attempts and (i['ID'] not in self.adapter_results or self.adapter_results[i['ID']]['Disposition'] != 'validated')) or None})
        collection('runs', report['runs'], expected_runs, lambda r: r['ID']['Subject'])
        expected_effects = []
        for intent in self.forecasts:
            operation = intent['ID']
            result = self.adapter_results.get(operation)
            settlement = next((r for r in self.tables['settlements'] if r['intent_id'] == operation), None)
            expected = {'OperationID': operation, 'Source': intent['Run'], 'Sequence': intent['Sequence'], 'SourceRevision': 1, 'Kind': 'adapter', 'Capability': 'open-meteo-hourly', 'Name': 'forecast',
                'State': self.effect_stages[operation], 'ResultDisposition': result['Disposition'] if result else '', 'Authorization': self.auth.get(operation),
                'AdapterAttempt': self.adapter_attempts.get(operation), 'AdapterResult': result, 'ExternalAttempt': None, 'ExternalReceipt': None,
                'Settlement': {'ID': settlement['id'], 'ObservationID': settlement['observation_id']} if settlement else None,
                'AdditionalReceipts': None, 'ReadOpportunities': None, 'ReadsOfOriginal': None, 'Usage': None,
                'ReceiptEvidenceState': 'known' if result and result['Disposition'] == 'validated' else 'none', 'Consequence': 'receipt_retained' if result and result['Disposition'] == 'validated' else 'unresolved' if operation in self.adapter_attempts else 'no_dispatch',
                'Dispatch': {'Minimum': int(result is not None), 'Maximum': int(operation in self.adapter_attempts)}, 'AuthorityAdmissions': None, 'DispatchAdmission': None}
            dispatch = next((r for r in self.tables['adapter_dispatch_admissions'] if r['intent_id'] == operation), None)
            if dispatch:
                expected['DispatchAdmission'] = {'OperationID': operation, 'PreparedDigest': dispatch['prepared_digest'], 'AfterFact': dispatch['after_fact']}
            expected_effects.append(expected)
        collection('effects', report['effects'], expected_effects, lambda r: r['OperationID'])
        if self.variant == 'delegated':
            delegated = report['delegated']
            require(delegated['helper_admission'] == self.deployment['helper_admission'] and delegated['helper_template_id'] == self.templates[0]['ID'], 'report helper deployment')
            expected_invocations = []
            if self.call:
                publication = decoded(self.tables['composition_publications'][0]['descriptor']) if self.tables['composition_publications'] else None
                freeze = next((decoded(r['descriptor']) for r in self.tables['composition_freezes'] if r['scope_id'] == self.invocation['ID']), None)
                outcome_id = self.tables['composition_outcome_deliveries'][0]['observation_id'] if publication else ''
                selected = self.selected.get(outcome_id)
                replies = [decoded(r['descriptor']) for r in self.tables['composition_replies']]
                inv_stage = 'consumed' if selected else 'published' if publication else 'frozen' if freeze else 'admitted' if self.member else 'refused' if self.admission else 'authorized' if self.call['ID'] in self.auth else 'committed'
                dependencies = [p for p in expected_pending if p['Run'] == run_id('weather') and p['OperationID'] == self.call['ID'] and p['Kind'] in ('invocation_member', 'invocation_reply_admission', 'invocation_external_outcome', 'invocation_publication')]
                expected_invocations.append({'State': inv_stage, 'Replies': replies or None, 'ResponseOperationID': replies[0]['Intent']['ID'] if publication and self.member else '',
                    'ParentSelection': 'selected' if selected else 'unconsumed' if publication else 'unpublished', 'Dependencies': dependencies or None, 'OperationID': self.call['ID'], 'TemplateID': self.call['Target'], 'Caller': run_id('weather'), 'Sequence': self.call['Sequence'], 'SourceRevision': 1,
                    'Inputs': self.call['Payload'], 'Authorization': self.auth.get(self.call['ID']), 'Admission': self.admission, 'Freeze': freeze, 'Closure': None, 'Outcome': publication,
                    'OutcomeObservationID': outcome_id, 'RequestObservationID': self.tables['composition_requests'][0]['observation_id'] if self.member else '',
                    'SelectedAt': {'Run': run_id('weather'), 'Sequence': selected[1]} if selected else None})
            collection('invocations', delegated['invocations'], expected_invocations, lambda r: r['OperationID'])
            expected_roots = []
            for name, root in self.roots.items():
                charges = [decoded(r['descriptor']) for r in self.tables['composition_charges'] if r['root_id'] == root['ID']]
                total = {'RootID': root['ID'], 'OperationID': '', 'Kind': '', 'Invocations': sum(c['Invocations'] for c in charges), 'CreatedRuns': sum(c['CreatedRuns'] for c in charges), 'Effects': sum(c['Effects'] for c in charges), 'DecisionOrder': 0}
                publication = next((decoded(r['descriptor']) for r in self.tables['composition_root_publications'] if r['root_id'] == root['ID']), None)
                unresolved = sorted(i['ID'] for i in self.forecasts if name == 'weather' and i['ID'] not in self.adapter_results)
                expected_roots.append({'RootID': root['ID'], 'Run': run_id(name), 'LocalStatus': decoded(self.runs[name]['projection'])['Status'], 'Limits': root['Limits'], 'Charges': total,
                    'Outcome': publication, 'Pending': self.root_pending[name], 'PendingInvocations': [self.call['ID']] if name == 'weather' and self.call and not self.tables['composition_publications'] and (not self.admission or self.member) else [],
                    'UnresolvedExternal': unresolved, 'UnknownExternal': [op for op in unresolved if op in self.adapter_attempts]})
            collection('roots', delegated['roots'], expected_roots, lambda r: r['RootID'])
        return {'verified': True, 'scope': 'Complete raw-linked report collections and pending entries; narrative prose and presentation fields are not an oracle.'}



def check_store(store, configuration, fixture, manifest, report=None):
    source_paths = {'configuration': pathlib.Path(configuration), 'fixture': pathlib.Path(fixture), 'manifest': pathlib.Path(manifest)}
    inputs = {name: path.read_bytes() for name, path in source_paths.items()}
    evidence = Evidence(read_raw_store(store), inputs['configuration'], decoded(inputs['fixture']), decoded(inputs['manifest']))
    for stage in ('static', 'histories', 'index_facts', 'bind_operations', 'historical_receives', 'fixture_outcomes', 'payloads', 'authorizations', 'request', 'messages', 'adapters', 'terminals', 'validate_facts'):
        getattr(evidence, stage)()
    result = evidence.correspondence()
    result['inputs'] = {name: {'path': str(source_paths[name].resolve()), 'sha256': hashlib.sha256(raw).hexdigest()} for name, raw in inputs.items()}
    result['store_path'] = str(pathlib.Path(store).resolve())
    result['schema_manifest'] = evidence.raw['schema']
    result['raw_cut_sha256'] = hashlib.sha256(b'outing-raw-cut/v1\x00' + go_json({table: [row['sha256'] for row in inventory['rows']] for table, inventory in evidence.raw['inventory'].items()})).hexdigest()
    result['manifest_source'] = evidence.manifest['source']
    acknowledgments = {row['intent_id'] for row in evidence.tables['message_acknowledgments']}
    result['raw_runner_message_debt'] = sorted(i['ID'] for i in evidence.intents.values() if i['Kind'] == 'message' and i['ID'] not in acknowledgments)
    decision = next((i for i in evidence.intents.values() if evidence.symbols[i['ID']] == 'message:outing:decision'), None)
    weather_wait = evidence.public_wait('weather')
    decision_cut = bool(decision and evidence.runs['outing']['sequence'] == decision['Sequence'] and decision['ID'] not in evidence.auth
        and result['raw_runner_message_debt'] == [decision['ID']] and weather_wait and weather_wait['Resolved']
        and weather_wait['Kind'] == 'message' and weather_wait['Source'] == 'outing' and weather_wait['Obligation'] == 'outing-decision'
        and all(f.stage == 7 for f in evidence.state.forecasts) and (evidence.variant != 'delegated' or evidence.state.invocation.selected)
        and decoded(evidence.runs['outing']['projection'])['Status'] == decoded(evidence.runs['weather']['projection'])['Status'] == 'open')
    result['decision_commit_cut'] = {'verified': decision_cut, 'original_decision': decision['ID'] if decision else None,
        'claim': 'This complete captured cut establishes sole pending internal message delivery only when verified; a later Store cannot establish this historical premise.'}
    if decision_cut:
        result['decision_sole_pending_at_commit'] = 'verified in this captured cut; final-run attribution still requires capture/source integrity and prefix binding'
    if report:
        result['report_coverage'] = evidence.report_coverage(report if isinstance(report, dict) else decoded(pathlib.Path(report).read_bytes()))
    return result


def compare_stores(baseline, candidate, configuration, fixture, manifest, replacement=None):
    exported = decoded(pathlib.Path(replacement).read_bytes()) if replacement else None
    pair = [check_store(path, configuration, fixture, manifest, exported[role]['report'] if exported else None) for role, path in (('baseline', baseline), ('candidate', candidate))]
    require(pair[0]['config']['variant'] == 'direct', 'comparison baseline must be direct')
    require(pair[0]['inputs'] == pair[1]['inputs'], 'comparison frozen input identity')
    frozen = decoded(pathlib.Path(manifest).read_bytes())
    candidates = [c for c in frozen['candidates'] if c['eligible']]
    originals = []
    for index, row in enumerate(candidates):
        operations = []
        for checked in pair:
            operations.append(next((op for op, symbol in checked['original_mapping'].items() if symbol.startswith('forecast:') and symbol.endswith(':' + str(index))), ''))
        mapping = {'candidate_id': row['id']}
        # The export omits absent originals; retained identities remain exact.
        mapping.update({name: operation for name, operation in zip(('baseline_operation_id', 'candidate_operation_id'), operations) if operation})
        originals.append(mapping)
    equal = pair[0]['semantic_answer'] == pair[1]['semantic_answer']
    positive = equal and all(p['successful_completion'] for p in pair)
    comparison = {'positive_replacement': positive, 'baseline_cost': dict(zip(('invocations', 'created_runs', 'effects'), pair[0]['state']['charges'])),
        'candidate_cost': dict(zip(('invocations', 'created_runs', 'effects'), pair[1]['state']['charges'])), 'forecast_operations': originals}
    if exported:
        require(exported['derived_config_sha256'] == pair[0]['inputs']['configuration']['sha256'] and exported['fixture_sha256'] == pair[0]['inputs']['fixture']['sha256'], 'replacement frozen input digests')
    if exported and pair[1]['config']['variant'] == 'delegated':
        require(exported['delegated_comparison'] == comparison, 'replacement report original mapping and separate actual costs')
    return {'accepted': True, 'classification': 'successful_bounded_replacement' if positive else 'not_successful_replacement',
        'semantic_answers_equal': equal, 'comparison': comparison, 'baseline': pair[0], 'candidate': pair[1],
        'claim': 'Both complete raw cuts independently checked. Equal final answers alone do not establish completion; costs remain distinct.'}
