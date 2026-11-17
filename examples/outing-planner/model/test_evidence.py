"""Independent raw-reader controls; fixture arguments are supplied by the Go harness."""
import pathlib
import sqlite3
import tempfile
import types
import unittest
import os
import json

SOURCE = pathlib.Path(__file__).with_name('check.py')
model = types.ModuleType('outing_evidence_tests')
import sys
sys.modules[model.__name__] = model
exec(compile(SOURCE.read_bytes(), str(SOURCE), 'exec'), model.__dict__)
evidence = types.ModuleType('outing_raw_tests')
evidence.MODEL = model.__dict__
exec(compile(SOURCE.with_name('evidence.py').read_bytes(), 'evidence.py', 'exec'), evidence.__dict__)
model = evidence


class RawInventoryTests(unittest.TestCase):
    def test_typed_digest_distinguishes_null_empty_integer_text_and_blob(self):
        values = (None, '', 0, '0', b'', b'0', 0.0)
        digests = [model.raw_row_digest((v,)) for v in values]
        self.assertEqual(len(digests), len(set(digests)))

    def test_schema_rejects_extra_table_even_when_empty(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = pathlib.Path(tmp) / 'cut.db'
            with sqlite3.connect(str(path)) as db:
                db.executescript(model.STORE_SCHEMA)
                db.execute('PRAGMA user_version=9')
                db.execute('CREATE TABLE hidden (id TEXT PRIMARY KEY)')
            with self.assertRaisesRegex(model.InvariantError, 'schema objects'):
                model.read_raw_store(path)

    def test_schema_rejects_changed_constraint(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = pathlib.Path(tmp) / 'cut.db'
            with sqlite3.connect(str(path)) as db:
                db.executescript(model.STORE_SCHEMA.replace('CHECK(epoch>0)', 'CHECK(epoch>=0)'))
                db.execute('PRAGMA user_version=9')
            with self.assertRaisesRegex(model.InvariantError, 'schema definition'):
                model.read_raw_store(path)


    def test_live_wal_capture_preserves_complete_logical_cut(self):
        with tempfile.TemporaryDirectory() as directory:
            source = pathlib.Path(directory) / 'live.db'
            target = pathlib.Path(directory) / 'captured.db'
            with sqlite3.connect(str(source)) as writer:
                writer.execute('PRAGMA journal_mode=WAL')
                writer.executescript(model.STORE_SCHEMA)
                writer.execute('PRAGMA user_version=9')
                writer.execute("INSERT INTO metadata VALUES(1,9,'alan/1d/v1',2,?)", ('0' * 64,))
                writer.commit()
                before = model.read_raw_store(source)['inventory']
                capture_store(source, target)
                self.assertEqual(before, model.read_raw_store(source)['inventory'])
                self.assertEqual(before, model.read_raw_store(target)['inventory'])
                self.assertEqual(len(before), 42)
                self.assertFalse(target.with_suffix('.db-wal').exists())
                with self.assertRaisesRegex(ValueError, 'must be fresh'):
                    capture_store(source, target)



@unittest.skipUnless(os.environ.get('OUTING_RAW_FIXTURE'), 'Go capture directory not supplied')
class CopiedStoreControls(unittest.TestCase):
    def test_changed_same_shape_reply_candidate(self):
        original = pathlib.Path(os.environ['OUTING_RAW_FIXTURE'])
        with tempfile.TemporaryDirectory() as tmp:
            path = pathlib.Path(tmp) / 'mutation.db'
            with sqlite3.connect(original.joinpath('candidate.db').as_uri() + '?mode=ro', uri=True) as source, sqlite3.connect(str(path)) as target:
                source.backup(target)
                rows = target.execute('SELECT id,subject,sequence,descriptor FROM intents').fetchall()
                for operation, subject, sequence, raw in rows:
                    intent = json.loads(raw)
                    if intent['Kind'] == 'message' and intent['Obligation'] == 'assess':
                        intent['Payload']['assessments']['List'][0]['park_id'] = model.value('same-shaped-wrong-park')
                        target.execute('UPDATE intents SET descriptor=? WHERE id=?', (model.go_json(intent), operation))
                        target.execute('UPDATE history SET intents=? WHERE subject=? AND sequence=?', (model.go_json([intent]), subject, sequence))
                target.commit()
            x = model.Evidence(model.read_raw_store(path), original.joinpath('derived-configuration.json').read_bytes(),
                json.loads(original.joinpath('fixture.json').read_bytes()), json.loads(original.joinpath('deployment-manifest.json').read_bytes()))
            for name in ('static', 'histories', 'index_facts', 'bind_operations'):
                getattr(x, name)()
            x.outcomes = tuple(x.fixture_response(c)[3] for c in x.candidates)
            with self.assertRaisesRegex(model.InvariantError, 'exact ordered assessment/original handles'):
                x.payloads()

@unittest.skipUnless(os.environ.get('OUTING_RAW_FIXTURE'), 'Go capture directory not supplied')
class CompleteEvidenceControls(unittest.TestCase):
    def check(self, path, report=None):
        root = pathlib.Path(os.environ['OUTING_RAW_FIXTURE'])
        return model.check_store(path, root / 'derived-configuration.json', root / 'fixture.json', root / 'deployment-manifest.json', report)

    def copy(self, name):
        root = pathlib.Path(os.environ['OUTING_RAW_FIXTURE'])
        archive = pathlib.Path(os.environ.get('OUTING_MUTATION_DIR', tempfile.gettempdir()))
        archive.mkdir(parents=True, exist_ok=True)
        directory = pathlib.Path(tempfile.mkdtemp(prefix=name + '-', dir=str(archive)))
        target = directory / 'mutation.db'
        capture_store(root / 'candidate.db', target)
        return target

    def mutate_original(self, connection, intent):
        connection.execute('UPDATE intents SET descriptor=? WHERE id=?', (model.go_json(intent), intent['ID']))
        row = connection.execute('SELECT intents FROM history WHERE subject=? AND sequence=?', (intent['Run']['Subject'], intent['Sequence'])).fetchone()
        intents = json.loads(row[0])
        intents[intent['Ordinal']] = intent
        connection.execute('UPDATE history SET intents=? WHERE subject=? AND sequence=?', (model.go_json(intents), intent['Run']['Subject'], intent['Sequence']))

    def test_copied_semantic_mutations(self):
        import hashlib
        cases = {
            'inert_program': 'exact raw program inventory',
            'hidden_forecast': 'hidden extra original forecast',
            'hidden_charge': 'exact original ordered charge inventory',
            'hidden_child': 'exact subjects and unchanged authority',
            'wrong_owner': 'original adapter prepared request/wire/schema',
            'wrong_request': 'original helper request payload',
            'wrong_original_response': 'original native fixture response and derived values',
            'same_shape_candidate': 'exact ordered assessment/original handles',
            'same_shape_handle': 'exact ordered assessment/original handles',
            'duplicate_reply': 'duplicate original message/reply',
            'duplicate_observation': 'unclassified raw rows: observations',
            'removed_result': 'missing durable result or stray operation',
            'cross_run_sequence': 'authorization Source.Sequence must match global fact prefix',
            'premature_assessment': 'public assessment before original invocation outcome selection',
        }
        original = pathlib.Path(os.environ['OUTING_RAW_FIXTURE']) / 'candidate.db'
        original_hash = hashlib.sha256(original.read_bytes()).hexdigest()
        for name, invariant in cases.items():
            with self.subTest(name=name):
                path = self.copy(name)
                with sqlite3.connect(str(path)) as db:
                    intents = [json.loads(row[0]) for row in db.execute('SELECT descriptor FROM intents')]
                    forecasts = sorted((i for i in intents if i['Kind'] == 'adapter'), key=lambda i: i['Sequence'])
                    reply = next(i for i in intents if i['Kind'] == 'message' and i['Obligation'] == 'assess')
                    call = next(i for i in intents if i['Kind'] == 'invocation')
                    if name == 'inert_program':
                        semantics, raw = db.execute('SELECT semantics,bytes FROM programs LIMIT 1').fetchone()
                        program = json.loads(raw)
                        program['Code'][0]['Dst'] = 'unused_but_valid_different_name'
                        raw = model.go_json(program)
                        db.execute('INSERT INTO programs VALUES(?,?,?)', (hashlib.sha256(raw).hexdigest(), semantics, raw))
                    elif name in ('hidden_forecast', 'duplicate_reply'):
                        intent = forecasts[0] if name == 'hidden_forecast' else reply
                        subject, sequence = intent['Run']['Subject'], intent['Sequence'] - 1
                        # Insert one valid original into an otherwise pure local row.
                        # Keep valid identities and one intent per emitting step, so this
                        # exercises inventory/semantic multiplicity rather than a parser.
                        successor = json.loads(db.execute('SELECT successor FROM history WHERE subject=? AND sequence=?', (subject, sequence)).fetchone()[0])
                        prior = json.loads(db.execute('SELECT successor FROM history WHERE subject=? AND sequence=?', (subject, sequence - 1)).fetchone()[0])
                        prior['PC'] = successor['PC']
                        extra = dict(intent, Sequence=sequence, ID=model.go_json([subject, 1, sequence, 0]).decode())
                        db.execute('UPDATE history SET successor=? WHERE subject=? AND sequence=?', (model.go_json(prior), subject, sequence - 1))
                        db.execute('INSERT INTO intents VALUES(?,?,?,?,?,?,?)', (extra['ID'], subject, 1, sequence, 0, 1, model.go_json(extra)))
                        db.execute('UPDATE history SET intents=? WHERE subject=? AND sequence=?', (model.go_json([extra]), subject, sequence))
                    elif name == 'hidden_charge':
                        root, order, raw = db.execute("SELECT root_id,decision_order,descriptor FROM composition_charges WHERE kind='invocation'").fetchone()
                        charge = json.loads(raw)
                        charge['OperationID'] = forecasts[0]['ID']
                        db.execute('INSERT INTO composition_charges VALUES(?,?,?,?,?)', (forecasts[0]['ID'], 'invocation', root, order, model.go_json(charge)))
                    elif name == 'hidden_child':
                        db.execute("INSERT INTO subjects VALUES('extra-retained-child',1,'invocation:extra',1,0,0)")
                        pid, initial = db.execute("SELECT program_id,initial FROM runs WHERE subject NOT IN ('weather','outing')").fetchone()
                        db.execute("INSERT INTO runs VALUES('extra-retained-child',1,?,?,1,0,?)", (pid, initial, initial))
                    elif name == 'wrong_owner':
                        operation, raw = db.execute('SELECT intent_id,descriptor FROM adapter_attempts LIMIT 1').fetchone()
                        attempt = json.loads(raw)
                        attempt['Intent']['Run'] = model.run_id('weather')
                        db.execute('UPDATE adapter_attempts SET descriptor=? WHERE intent_id=?', (model.go_json(attempt), operation))
                    elif name == 'wrong_request':
                        call['Payload']['units'] = model.value('fahrenheit')
                        self.mutate_original(db, call)
                    elif name == 'wrong_original_response':
                        rows = db.execute('SELECT intent_id,descriptor FROM adapter_results ORDER BY intent_id').fetchall()
                        self.assertEqual(len(rows), 2)
                        for (operation, _), (_, raw) in zip(rows, reversed(rows)):
                            db.execute('UPDATE adapter_results SET descriptor=? WHERE intent_id=?', (raw, operation))
                    elif name in ('same_shape_candidate', 'same_shape_handle'):
                        row = reply['Payload']['assessments']['List'][0]
                        if name == 'same_shape_candidate':
                            row['park_id'] = reply['Payload']['assessments']['List'][1]['park_id']
                        else:
                            row['forecast_operation'] = model.value(forecasts[1]['ID'])
                        self.mutate_original(db, reply)
                    elif name == 'duplicate_observation':
                        _, subject, generation, raw, _ = db.execute('SELECT * FROM observations LIMIT 1').fetchone()
                        observation = json.loads(raw)
                        observation['ID'] = model.identity('duplicate-observation-control', observation['ID'])
                        db.execute('INSERT INTO observations VALUES(?,?,?,?,NULL)', (observation['ID'], subject, generation, model.go_json(observation)))
                    elif name == 'removed_result':
                        db.execute('DELETE FROM adapter_results WHERE intent_id=?', (forecasts[0]['ID'],))
                    elif name == 'cross_run_sequence':
                        row = db.execute('SELECT id,result FROM attempts WHERE intent_id=?', (call['ID'],)).fetchone()
                        auth = json.loads(row[1])
                        auth['Source']['Sequence'] = forecasts[1]['Sequence']
                        db.execute('UPDATE attempts SET result=? WHERE id=?', (model.go_json(auth), row[0]))
                    elif name == 'premature_assessment':
                        assessment = next(i for i in intents if i['Obligation'] == 'outing-assessment')
                        outcome_id = db.execute('SELECT observation_id FROM composition_outcome_deliveries').fetchone()[0]
                        selected = db.execute('SELECT consumed_sequence FROM observations WHERE id=?', (outcome_id,)).fetchone()[0]
                        facts = []
                        for order, raw in db.execute("SELECT decision_order,descriptor FROM composition_facts WHERE kind='execution'"):
                            fact = json.loads(raw)
                            if fact['Run']['Subject'] == 'weather' and fact['Sequence'] in (selected, assessment['Sequence']):
                                facts.append((order, fact))
                        self.assertEqual(len(facts), 2)
                        for (order, _), (_, fact) in zip(facts, reversed(facts)):
                            fact['Order'] = order
                            db.execute('UPDATE composition_facts SET id=?,descriptor=? WHERE decision_order=?', ('temporary-' + str(order), model.go_json(fact), order))
                        for (order, _), (_, fact) in zip(facts, reversed(facts)):
                            db.execute('UPDATE composition_facts SET id=? WHERE decision_order=?', (fact['ID'], order))
                    db.commit()
                with self.assertRaisesRegex(model.InvariantError, invariant) as caught:
                    self.check(path)
                path.with_suffix('.mutation.json').write_text(json.dumps({'control': name, 'original_sha256': original_hash, 'expected_invariant': invariant, 'actual': str(caught.exception)}, indent=2) + '\n')
        self.assertEqual(hashlib.sha256(original.read_bytes()).hexdigest(), original_hash)

    def test_report_omitted_reply_and_changed_effect_consequence(self):
        root = pathlib.Path(os.environ['OUTING_RAW_FIXTURE'])
        report = json.loads((root / 'candidate-report.json').read_bytes())
        for name in ('reply', 'consequence'):
            with self.subTest(name=name):
                changed = json.loads(json.dumps(report))
                if name == 'reply':
                    changed['delegated']['invocations'][0]['Replies'] = []
                    invariant = 'report invocations raw correspondence'
                else:
                    changed['effects'][0]['Consequence'] = 'no_dispatch'
                    invariant = 'report effects raw correspondence'
                with self.assertRaisesRegex(model.InvariantError, invariant):
                    self.check(root / 'candidate.db', changed)

    def test_report_omissions_and_same_shape_changes(self):
        root = pathlib.Path(os.environ['OUTING_RAW_FIXTURE'])
        report = json.loads((root / 'candidate-report.json').read_bytes())
        for collection in ('operations', 'runs', 'effects', 'observations', 'admissions', 'permissions', 'history'):
            with self.subTest(collection=collection):
                changed = json.loads(json.dumps(report))
                changed[collection].pop()
                with self.assertRaisesRegex(model.InvariantError, 'report ' + collection + ' coverage'):
                    self.check(root / 'candidate.db', changed)
        for collection in ('invocations', 'roots'):
            changed = json.loads(json.dumps(report))
            changed['delegated'][collection].pop()
            with self.assertRaisesRegex(model.InvariantError, 'report ' + collection + ' coverage'):
                self.check(root / 'candidate.db', changed)


@unittest.skipUnless(os.environ.get('OUTING_RAW_PENDING_FIXTURE'), 'pending Go capture directory not supplied')
class PendingReportControls(unittest.TestCase):
    def test_omitted_or_altered_pending_original(self):
        root = pathlib.Path(os.environ['OUTING_RAW_PENDING_FIXTURE'])
        report = json.loads((root / 'candidate-report.json').read_bytes())
        def check(changed):
            return model.check_store(root / 'candidate.db', root / 'derived-configuration.json', root / 'fixture.json', root / 'deployment-manifest.json', changed)
        self.assertEqual(check(report)['classification'], 'unknown_forecast')
        for change in ('omitted', 'altered'):
            with self.subTest(change=change):
                altered = json.loads(json.dumps(report))
                index = next(i for i, row in enumerate(altered['pending']) if row['Kind'] == 'effect_reserved_unknown')
                if change == 'omitted':
                    altered['pending'].pop(index)
                    invariant = 'report pending coverage'
                else:
                    altered['pending'][index]['DependsOn'] = report['delegated']['invocation_operation_id']
                    invariant = 'report pending identity coverage'
                with self.assertRaisesRegex(model.InvariantError, invariant):
                    check(altered)


def capture_store(source_path, destination):
    """Capture a consistent raw cut; only the fresh destination changes journal mode."""
    source_path, destination = pathlib.Path(source_path).resolve(), pathlib.Path(destination).resolve()
    if destination.exists():
        raise ValueError('capture destination must be fresh')
    source = sqlite3.connect(source_path.as_uri() + '?mode=ro', uri=True)
    target = sqlite3.connect(str(destination))
    try:
        source.execute('PRAGMA query_only=ON')
        source.execute('BEGIN')
        objects, metadata = model.schema_manifest(source)
        source.backup(target)
        target.execute('PRAGMA journal_mode=DELETE')
        if model.schema_manifest(target) != (objects, metadata):
            raise ValueError('backup changed schema metadata')
        inventory = {}
        for table, info in sorted(metadata.items()):
            keys = [r[1] for r in sorted(info['columns'], key=lambda r: r[5]) if r[5]]
            quoted = lambda name: '"' + name.replace('"', '""') + '"'
            query = 'SELECT * FROM ' + quoted(table) + ' ORDER BY ' + ','.join(quoted(k) for k in keys)
            before, after = source.execute(query).fetchall(), target.execute(query).fetchall()
            if before != after:
                raise ValueError('backup changed complete logical rows: ' + table)
            inventory[table] = [model.raw_row_digest(row) for row in before]
        source.rollback()
    finally:
        source.close()
        target.close()
    import hashlib
    capture = {'source': str(source_path), 'destination': str(destination), 'sqlite': sqlite3.sqlite_version,
        'method': 'mode=ro source transaction plus SQLite backup; DELETE journal mode only on fresh destination',
        'complete_source_and_destination_rows_equal': True, 'inventory': inventory,
        'destination_sha256': hashlib.sha256(destination.read_bytes()).hexdigest()}
    destination.with_suffix(destination.suffix + '.capture.json').write_text(json.dumps(capture, indent=2) + '\n')


if __name__ == '__main__':
    if len(sys.argv) == 4 and sys.argv[1] == 'capture':
        capture_store(sys.argv[2], sys.argv[3])
    else:
        unittest.main()
