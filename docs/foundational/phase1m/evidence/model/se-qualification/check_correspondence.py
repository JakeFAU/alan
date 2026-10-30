"""Postfreeze C_se mapping, reusing the reviewed finite C_es artifact checker.

No reducer, model exploration, or runtime execution is used to obtain expectations.
The exact narrowly changed expected receive order is explicit below.
"""
import base64, hashlib, json, pathlib, sys
legacy = pathlib.Path('docs/foundational/phase1l/evidence/model/task2b/check_correspondence.py')
source = legacy.read_text()
changes = [
    ("[('candidate', candidate, 'candidate'), ('s', successor, 'successor')]",
     "[('s', successor, 'successor'), ('candidate', candidate, 'candidate')]"),
    ("'/Code/2/Match/Source/Literal/Text'", "'/Code/3/Match/Source/Literal/Text'"),
    ("'outcome_consume e matched by original handle, template, provenance, generation, obligation','outcome_consume s matched by original handle, template, provenance, generation, obligation'",
     "'outcome_consume s matched by original handle, template, provenance, generation, obligation','outcome_consume e matched by original handle, template, provenance, generation, obligation'"),
    ("examples/invocationjoin/deployment.go", "examples/consumingpolicy/deployment.go"),
    ("examples/invocationjoin/consumer/main.go", "examples/consumingpolicy/consumer/main.go"),
]
for before, after in changes:
    assert source.count(before) == 1, before
    source = source.replace(before, after)
namespace = {}
exec(compile(source, str(legacy) + '[C_se explicit adaptation]', 'exec'), namespace)
result, artifacts = namespace['result'], namespace['x']
result['context'] = 'C_se'
result['checker_reuse'] = dict(source=str(legacy), sha256=hashlib.sha256(legacy.read_bytes()).hexdigest(), replacements=changes)
# Conservative byte estimates are for this finite pure stable profile. Values
# cannot acquire arbitrary strings: input/mode/result literals, original handles,
# hashes, and fixed protocol reasons are the only sources. The longest handle is
# bounded using a 64-character dynamic subject, generation1, sequence<=35, ordinal0.
handle_bytes = len(json.dumps(['f'*64, 1, 35, 0], separators=(',', ':')).encode())
assert handle_bytes < 256
scalar = dict(Kind='string', Number=-(2**63), Text='\x00'*256, Flag=False)
scalar_bytes = len(json.dumps(scalar, separators=(',', ':')).encode())
byte_records = {}
for binding, record in artifacts.items():
    programs = []
    for row in record['Provision']['Programs']:
        raw = base64.b64decode(row['Bytes'])
        p = json.loads(raw)
        registers = set((p['Author'].get('Inputs') or {}).get('Record') or {})
        max_payload, max_bindings = 0, 0
        for instruction in p['Code']:
            assert instruction['Op'] in ['receive', 'emit', 'branch', 'set', 'jump', 'end']
            assert not instruction.get('Collection')
            if instruction['Dst']: registers.add(instruction['Dst'])
            registers.update((instruction['Bind'] or {}).values())
            registers.update((instruction['Updates'] or {}).keys())
            max_bindings = max(max_bindings, len(instruction['Bind'] or {}))
            if instruction['Effect']:
                max_payload = max(max_payload, len(instruction['Effect']['Payload']))
        assert max(map(len, registers), default=0) <= 32
        assert len(registers) <= 16 and max_payload <= 3 and max_bindings <= 5
        projection_upper = 128 + len(registers) * (34 + scalar_bytes + 2)
        # resolveCompositionProgram changes at most four scalar strings per
        # instruction; original retained metadata/program bytes stay intact.
        bound_program_upper = len(raw) + len(p['Code']) * 4 * 256
        assert projection_upper < 65536 and bound_program_upper < 65536
        programs.append(dict(program_id=row['ID'], retained_program_bytes=len(raw),
                             register_count=len(registers), registers=sorted(registers),
                             max_emit_payload_fields=max_payload, max_receive_bindings=max_bindings,
                             projection_bytes_upper=projection_upper,
                             bound_program_bytes_upper=bound_program_upper))
    byte_records[binding] = programs
# At most five scalar payload fields in an outcome, three in a request/intent;
# each checked envelope has at most32 scalar value/string slots
# (including single-member links and <=3 captured operation handles). Budget512
# for keys/delimiters/integers outside those slots. This bound also deliberately
# overestimates the per-reduction one-intent array, admission and freeze records.
# Audited from the named source structs; a Value (including its Text/Kind) counts
# as one whole scalar slot, while a non-Value string counts as one slot. Counts
# overestimate this profile's actual nonempty values; numbers/keys use the margin.
slot_limits = dict(Observation=12, Intent=8, AdmissionDescriptor=11,
                   compositionInvocation=9, compositionAdmission=12,
                   compositionOutcome=9, compositionFreeze=7,
                   compositionReply=16)
assert max(slot_limits.values()) <= 32
envelope_upper = 512 + 32 * (scalar_bytes + 34 + 2)
assert envelope_upper < 65536
result['operational_bounds'] = dict(
    scope='Source-derived conservative bounds for the declared pure positive profile; not arbitrary user payloads, transient errors, or external adapters.',
    configured_steps=200, configured_dispatches=200, configured_capacity=1,
    default_byte_caps=dict(MaxProgramBytes=1048576, MaxDataBytes=1048576, MaxObservationBytes=1048576, MaxOutputBytes=1048576),
    maximum_original_handle_bytes=handle_bytes, scalar_text_bytes_upper=256,
    worst_escaped_scalar_bytes=scalar_bytes, envelope_scalar_slot_limits=slot_limits,
    protocol_envelope_bytes_upper=envelope_upper,
    conservative_common_bytes_upper=65536, programs=byte_records,
    dispatch_derivation='Per admitted singleton: initial invocation admission (request observation inserted atomically), one original reply-message dispatch, then ready original outcome settlement/delivery. pendingDeliveries omits admitted unready invocations and removes settled ones. Thus at most3 per original admission: A6/B9. Root settlement is not charged by report.Dispatches.',
    correction='The Phase1M task-1-report.md lines137-139 and evidence/reviews/task1-r1.md lines101-102 called request insertion a request dispatch. Admission creates that observation; the third counted dispatch is outcome settlement/delivery. Numeric6/9 is unchanged. Earlier wording is preserved, not silently edited.')
for path in ['internal/executor/composition_types.go','internal/executor/invocation_payload.go','internal/executor/store.go','schema/schema.go',str(pathlib.Path(__file__))]:
    result['source_sha256'][path] = hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()
pathlib.Path(sys.argv[2]).write_text(json.dumps(result,indent=2)+'\n')
print('C_se postfreeze: all25 parent instructions per binding, original A/B/helper/S exact; common conservative byte bound65536 <1048576; dispatch derivation corrected.')
print('final_output_sha256='+hashlib.sha256(pathlib.Path(sys.argv[2]).read_bytes()).hexdigest())
